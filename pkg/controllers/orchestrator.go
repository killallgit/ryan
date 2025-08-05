package controllers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/agents"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
)

// OrchestratorController bridges between the TUI and the agent orchestrator
type OrchestratorController struct {
	orchestrator       *agents.Orchestrator
	streamManager      *StreamManager
	progressDisplay    *ProgressDisplay
	toolRegistry       *tools.Registry
	conversation       chat.Conversation
	activeStreams      map[string]*AgentStream
	lastPromptTokens   int
	lastResponseTokens int
	log                *logger.Logger
	mu                 sync.RWMutex
}

// NewOrchestratorController creates a new orchestrator controller
func NewOrchestratorController(conversation chat.Conversation, toolRegistry *tools.Registry) (*OrchestratorController, error) {
	// Create orchestrator
	orchestrator := agents.NewOrchestrator()

	// Register built-in agents
	if err := orchestrator.RegisterBuiltinAgents(toolRegistry); err != nil {
		return nil, fmt.Errorf("failed to register agents: %w", err)
	}

	controller := &OrchestratorController{
		orchestrator:       orchestrator,
		streamManager:      NewStreamManager(),
		progressDisplay:    NewProgressDisplay(),
		toolRegistry:       toolRegistry,
		conversation:       conversation,
		activeStreams:      make(map[string]*AgentStream),
		lastPromptTokens:   0,
		lastResponseTokens: 0,
		log:                logger.WithComponent("orchestrator_controller"),
	}

	return controller, nil
}

// SendUserMessage processes a user message through the orchestrator
func (oc *OrchestratorController) SendUserMessage(content string) (chat.Message, error) {
	oc.log.Info("Processing user message", "content_preview", truncateString(content, 100))

	// Add user message to conversation
	userMsg := chat.NewUserMessage(content)
	oc.conversation.Tree.AddMessage(userMsg, oc.conversation.Tree.ActiveContext)

	// Create context for this request
	ctx := context.Background()

	// Execute through orchestrator
	result, err := oc.orchestrator.Execute(ctx, content, map[string]interface{}{
		"conversation": oc.conversation,
		"tools":        oc.toolRegistry,
	})

	if err != nil {
		oc.log.Error("Orchestrator execution failed", "error", err)
		errorMsg := chat.NewAssistantMessage(fmt.Sprintf("I encountered an error: %v", err))
		oc.conversation.Tree.AddMessage(errorMsg, oc.conversation.Tree.ActiveContext)
		return errorMsg, err
	}

	// Convert result to assistant message
	response := oc.formatAgentResult(result)
	assistantMsg := chat.NewAssistantMessage(response)
	oc.conversation.Tree.AddMessage(assistantMsg, oc.conversation.Tree.ActiveContext)

	return assistantMsg, nil
}

// StartStreaming starts a streaming response through the orchestrator
func (oc *OrchestratorController) StartStreaming(ctx context.Context, content string) (<-chan StreamingUpdate, error) {
	oc.log.Info("Starting streaming execution", "content_preview", truncateString(content, 100))

	// Create update channel
	updates := make(chan StreamingUpdate, 100)

	// Create agent stream
	streamID := generateStreamID()
	stream := &AgentStream{
		ID:      streamID,
		Updates: updates,
		Context: ctx,
		Started: time.Now(),
	}

	oc.mu.Lock()
	oc.activeStreams[streamID] = stream
	oc.mu.Unlock()

	// Start orchestrator execution in background
	go func() {
		defer close(updates)
		defer oc.removeStream(streamID)

		// Send initial update
		updates <- StreamingUpdate{
			Type:     StreamStarted,
			StreamID: streamID,
			Content:  "🤖 Analyzing your request...\n",
		}

		// Create execution context with progress channel
		progressChan := make(chan agents.ProgressUpdate, 100)
		execOptions := map[string]interface{}{
			"conversation": oc.conversation,
			"tools":        oc.toolRegistry,
			"progress":     progressChan,
			"streaming":    true,
		}

		// Start progress monitoring
		go oc.monitorProgress(progressChan, updates)

		// Execute through orchestrator
		result, err := oc.orchestrator.Execute(ctx, content, execOptions)

		if err != nil {
			updates <- StreamingUpdate{
				Type:     StreamError,
				StreamID: streamID,
				Content:  fmt.Sprintf("\n❌ Error: %v\n", err),
			}
			return
		}

		// Stream the final result
		oc.streamResult(result, updates)
	}()

	return updates, nil
}

// monitorProgress monitors agent execution progress and sends updates
func (oc *OrchestratorController) monitorProgress(progressChan <-chan agents.ProgressUpdate, updates chan<- StreamingUpdate) {
	for progress := range progressChan {
		update := StreamingUpdate{
			Type:    ChunkReceived,
			Content: oc.formatProgress(progress),
		}

		select {
		case updates <- update:
		default:
			// Channel full, skip
		}
	}
}

// streamResult streams the agent result to the update channel
func (oc *OrchestratorController) streamResult(result agents.AgentResult, updates chan<- StreamingUpdate) {
	// Stream summary
	updates <- StreamingUpdate{
		Type:    ChunkReceived,
		Content: fmt.Sprintf("\n## Summary\n%s\n", result.Summary),
	}

	// Stream details in chunks
	if result.Details != "" {
		updates <- StreamingUpdate{
			Type:    ChunkReceived,
			Content: "\n## Details\n",
		}

		// Split details into lines and stream
		lines := strings.Split(result.Details, "\n")
		for _, line := range lines {
			updates <- StreamingUpdate{
				Type:    ChunkReceived,
				Content: line + "\n",
			}

			// Small delay for effect
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Stream metadata
	if result.Metadata.AgentName != "" {
		updates <- StreamingUpdate{
			Type: ChunkReceived,
			Content: fmt.Sprintf("\n---\n📊 Execution time: %v\n🤖 Agents used: %s\n📁 Files processed: %d\n",
				result.Metadata.Duration,
				result.Metadata.AgentName,
				len(result.Metadata.FilesProcessed)),
		}
	}

	// Send completion
	updates <- StreamingUpdate{
		Type:    MessageComplete,
		Content: "\n✅ Task completed\n",
	}
}

// formatAgentResult formats an agent result for display
func (oc *OrchestratorController) formatAgentResult(result agents.AgentResult) string {
	var parts []string

	// Add summary
	if result.Summary != "" {
		parts = append(parts, fmt.Sprintf("**Summary:** %s", result.Summary))
	}

	// Add details
	if result.Details != "" {
		parts = append(parts, "", "**Details:**", result.Details)
	}

	// Add metadata if verbose
	if oc.isVerbose() && result.Metadata.Duration > 0 {
		parts = append(parts, "", fmt.Sprintf("*Execution time: %v*", result.Metadata.Duration))
	}

	return strings.Join(parts, "\n")
}

// formatProgress formats a progress update for display
func (oc *OrchestratorController) formatProgress(progress agents.ProgressUpdate) string {
	icon := "⏳"
	switch progress.Status {
	case "completed":
		icon = "✅"
	case "failed":
		icon = "❌"
	case "running":
		icon = "🔄"
	}

	return fmt.Sprintf("%s %s: %s", icon, progress.Agent, progress.Message)
}

// removeStream removes a stream from active streams
func (oc *OrchestratorController) removeStream(streamID string) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	delete(oc.activeStreams, streamID)
}

// isVerbose checks if verbose mode is enabled
func (oc *OrchestratorController) isVerbose() bool {
	// Could be configured via settings
	return false
}

// GetToolRegistry returns the tool registry
func (oc *OrchestratorController) GetToolRegistry() *tools.Registry {
	return oc.toolRegistry
}

// GetOrchestrator returns the agent orchestrator
func (oc *OrchestratorController) GetOrchestrator() *agents.Orchestrator {
	return oc.orchestrator
}

// GetHistory returns the conversation history
func (oc *OrchestratorController) GetHistory() []chat.Message {
	// Get messages from the active context
	return oc.conversation.Tree.GetContextMessages(oc.conversation.Tree.ActiveContext)
}

// GetModel returns the current model
func (oc *OrchestratorController) GetModel() string {
	return oc.conversation.Model
}

// SetModel sets the current model
func (oc *OrchestratorController) SetModel(model string) {
	oc.conversation.Model = model
}

// AddUserMessage adds a user message to the conversation
func (oc *OrchestratorController) AddUserMessage(content string) {
	userMsg := chat.NewUserMessage(content)
	oc.conversation.Tree.AddMessage(userMsg, oc.conversation.Tree.ActiveContext)
}

// AddErrorMessage adds an error message to the conversation
func (oc *OrchestratorController) AddErrorMessage(errorMsg string) {
	msg := chat.NewAssistantMessage(fmt.Sprintf("Error: %s", errorMsg))
	oc.conversation.Tree.AddMessage(msg, oc.conversation.Tree.ActiveContext)
}

// Reset resets the conversation
func (oc *OrchestratorController) Reset() {
	oc.conversation = chat.NewConversation(oc.conversation.Model)
}

// SetOllamaClient sets the Ollama client (for compatibility)
func (oc *OrchestratorController) SetOllamaClient(client any) {
	// Not used in orchestrator controller
}

// ValidateModel validates a model (for compatibility)
func (oc *OrchestratorController) ValidateModel(model string) error {
	// For now, accept all models
	return nil
}

// GetTokenUsage returns token usage
func (oc *OrchestratorController) GetTokenUsage() (promptTokens, responseTokens int) {
	return oc.lastPromptTokens, oc.lastResponseTokens
}

// CleanThinkingBlocks cleans thinking blocks from messages
func (oc *OrchestratorController) CleanThinkingBlocks() {
	// Not implemented yet
}

// Supporting types

// StreamManager manages streaming operations
type StreamManager struct {
	streams map[string]*AgentStream
	mu      sync.RWMutex
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		streams: make(map[string]*AgentStream),
	}
}

// ProgressDisplay handles progress visualization
type ProgressDisplay struct {
	activeJobs map[string]*JobProgress
	mu         sync.RWMutex
}

func NewProgressDisplay() *ProgressDisplay {
	return &ProgressDisplay{
		activeJobs: make(map[string]*JobProgress),
	}
}

// AgentStream represents an active agent execution stream
type AgentStream struct {
	ID      string
	Updates chan StreamingUpdate
	Context context.Context
	Started time.Time
}

// JobProgress tracks progress of a job
type JobProgress struct {
	ID         string
	Agent      string
	Status     string
	Progress   float64
	StartTime  time.Time
	UpdateTime time.Time
}

// Helper functions

func generateStreamID() string {
	return fmt.Sprintf("stream_%d", time.Now().UnixNano())
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
