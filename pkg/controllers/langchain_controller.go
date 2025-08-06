package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/killallgit/ryan/pkg/agents"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/langchain"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
)

// LangChainController wraps the LangChain client to work with the existing controller interface
type LangChainController struct {
	client       *langchain.Client
	model        string
	toolRegistry *tools.Registry
	conversation chat.Conversation
	log          *logger.Logger
	historyFile  string
	orchestrator *agents.LangchainOrchestrator
}

// NewLangChainController creates a new controller using the LangChain client
func NewLangChainController(baseURL, model string, toolRegistry *tools.Registry) (*LangChainController, error) {
	client, err := langchain.NewClient(baseURL, model, toolRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create LangChain client: %w", err)
	}

	log := logger.WithComponent("langchain_controller")

	controller := &LangChainController{
		client:       client,
		model:        model,
		toolRegistry: toolRegistry,
		conversation: chat.NewConversation(model),
		log:          log,
		historyFile:  ".ryan/chat_history.json",
	}

	// Set up tool progress callback to show tool execution in chat
	client.SetProgressCallback(func(toolName, command string) {
		// Add tool progress message to conversation
		progressMsg := chat.NewToolProgressMessage(toolName, command)
		controller.conversation = chat.AddMessage(controller.conversation, progressMsg)

		log.Debug("Tool execution started", "tool", toolName, "command", command)
	})

	// Load existing chat history if available
	if err := controller.loadHistory(); err != nil {
		log.Debug("Could not load chat history", "error", err)
	}

	return controller, nil
}

// NewLangChainControllerWithSystem creates a new controller with system prompt
func NewLangChainControllerWithSystem(baseURL, model, systemPrompt string, toolRegistry *tools.Registry) (*LangChainController, error) {
	controller, err := NewLangChainController(baseURL, model, toolRegistry)
	if err != nil {
		return nil, err
	}

	// Add system message to conversation
	if systemPrompt != "" {
		controller.conversation = chat.AddMessage(controller.conversation, chat.NewSystemMessage(systemPrompt))
	}

	return controller, nil
}

// SendUserMessage sends a user message using the LangChain agent
func (lc *LangChainController) SendUserMessage(content string) (chat.Message, error) {
	return lc.SendUserMessageWithContext(context.Background(), content)
}

// SendUserMessageWithContext sends a user message with context using the LangChain agent
func (lc *LangChainController) SendUserMessageWithContext(ctx context.Context, content string) (chat.Message, error) {
	if strings.TrimSpace(content) == "" {
		return chat.Message{}, fmt.Errorf("message content cannot be empty")
	}

	lc.log.Debug("Sending user message with LangChain agent",
		"content_length", len(content),
		"has_tools", lc.toolRegistry != nil,
		"agent_enabled", lc.client != nil)

	// Add user message to conversation with deduplication
	userMsg := chat.NewUserMessage(content)
	lc.conversation = chat.AddMessageWithDeduplication(lc.conversation, userMsg)

	// Use orchestrator if available, otherwise use the client directly
	var response string
	var err error

	if lc.orchestrator != nil {
		// Use the orchestrator to select and execute with the best agent
		options := map[string]interface{}{
			"model": lc.model,
		}
		result, orchErr := lc.orchestrator.Execute(ctx, content, options)
		if orchErr != nil {
			err = orchErr
		} else {
			// Log the raw orchestrator result
			lc.log.Info("RAW ORCHESTRATOR RESPONSE",
				"result_type", fmt.Sprintf("%T", result),
				"success", result.Success,
				"summary", result.Summary,
				"details", result.Details,
				"artifacts", result.Artifacts,
				"metadata", result.Metadata)
			response = result.Details
		}
	} else {
		// Use the enhanced client to send the message
		response, err = lc.client.SendMessage(ctx, content)
		if err == nil {
			// Log the raw client response
			lc.log.Info("RAW LANGCHAIN CLIENT RESPONSE",
				"response_type", fmt.Sprintf("%T", response),
				"response_length", len(response),
				"response_content", response)
		}
	}
	if err != nil {
		errorMsg := fmt.Sprintf("LangChain agent failed: %v", err)
		lc.log.Error("Enhanced LangChain client failed", "error", err)

		// Add error message to conversation
		errMsg := chat.NewErrorMessage(errorMsg)
		lc.conversation = chat.AddMessage(lc.conversation, errMsg)

		return errMsg, fmt.Errorf("failed to send message: %w", err)
	}

	// Create assistant message from response (preserve thinking blocks for display)
	assistantMsg := chat.NewAssistantMessage(response)
	lc.conversation = chat.AddMessage(lc.conversation, assistantMsg)

	// Save history to disk after each interaction
	if err := lc.saveHistory(); err != nil {
		lc.log.Error("Failed to save chat history", "error", err)
	}

	lc.log.Debug("Enhanced LangChain agent response received",
		"response_length", len(response))

	return assistantMsg, nil
}

// SendUserMessageWithStreamingContext sends a message with streaming support
func (lc *LangChainController) SendUserMessageWithStreamingContext(ctx context.Context, content string, outputChan chan<- chat.MessageChunk) (chat.Message, error) {
	if strings.TrimSpace(content) == "" {
		return chat.Message{}, fmt.Errorf("message content cannot be empty")
	}

	lc.log.Debug("Starting streaming message with LangChain agent")

	// Add user message to conversation with deduplication
	userMsg := chat.NewUserMessage(content)
	lc.conversation = chat.AddMessageWithDeduplication(lc.conversation, userMsg)

	// Create a channel to collect streamed content
	streamChan := make(chan string, 100)
	var fullResponse strings.Builder

	// Start streaming in a goroutine
	go func() {
		defer close(streamChan)
		if err := lc.client.StreamMessage(ctx, content, streamChan); err != nil {
			lc.log.Error("Streaming failed", "error", err)
			// Send error through output channel
			select {
			case outputChan <- chat.MessageChunk{Content: fmt.Sprintf("Error: %v", err), Done: true}:
			case <-ctx.Done():
			}
		}
	}()

	// Forward chunks to output channel and accumulate content
	for chunk := range streamChan {
		fullResponse.WriteString(chunk)
		select {
		case outputChan <- chat.MessageChunk{Content: chunk, Done: false}:
		case <-ctx.Done():
			return chat.Message{}, ctx.Err()
		}
	}

	// Send completion signal
	select {
	case outputChan <- chat.MessageChunk{Content: "", Done: true}:
	case <-ctx.Done():
		return chat.Message{}, ctx.Err()
	}

	// Create assistant message from full response
	response := fullResponse.String()
	assistantMsg := chat.NewAssistantMessage(response)
	lc.conversation = chat.AddMessage(lc.conversation, assistantMsg)

	// Save history to disk after each interaction
	if err := lc.saveHistory(); err != nil {
		lc.log.Error("Failed to save chat history", "error", err)
	}

	lc.log.Debug("Streaming response completed", "response_length", len(response))

	return assistantMsg, nil
}

// GetHistory returns the chat history
func (lc *LangChainController) GetHistory() []chat.Message {
	return chat.GetMessages(lc.conversation)
}

// GetConversation returns the current conversation
func (lc *LangChainController) GetConversation() chat.Conversation {
	return lc.conversation
}

// Reset clears the conversation and memory
func (lc *LangChainController) Reset() {
	lc.log.Debug("Resetting LangChain controller")

	// Get system prompt if exists
	systemPrompt := ""
	if chat.HasSystemMessage(lc.conversation) {
		messages := chat.GetMessagesByRole(lc.conversation, chat.RoleSystem)
		if len(messages) > 0 {
			systemPrompt = messages[0].Content
		}
	}

	// Clear conversation
	lc.conversation = chat.NewConversation(lc.model)

	// Clear LangChain memory
	ctx := context.Background()
	if err := lc.client.ClearMemory(ctx); err != nil {
		lc.log.Error("Failed to clear LangChain memory", "error", err)
	}

	// Delete history file if it exists
	if _, err := os.Stat(lc.historyFile); err == nil {
		if err := os.Remove(lc.historyFile); err != nil {
			lc.log.Error("Failed to delete chat history file", "file", lc.historyFile, "error", err)
		} else {
			lc.log.Debug("Deleted chat history file", "file", lc.historyFile)
		}
	}

	// Re-add system prompt if it existed
	if systemPrompt != "" {
		lc.conversation = chat.AddMessage(lc.conversation, chat.NewSystemMessage(systemPrompt))
	}
}

// AddUserMessage adds a user message to the conversation (for optimistic updates)
func (lc *LangChainController) AddUserMessage(content string) {
	if strings.TrimSpace(content) == "" {
		return
	}

	// Create optimistic user message for immediate UI feedback
	userMsg := chat.NewOptimisticUserMessage(content)
	lc.conversation = chat.AddMessage(lc.conversation, userMsg)

	// Save history to disk after adding user message
	if err := lc.saveHistory(); err != nil {
		lc.log.Error("Failed to save chat history", "error", err)
	}

	lc.log.Debug("Added optimistic user message", "content_length", len(content))
}

// AddErrorMessage adds an error message to the conversation
func (lc *LangChainController) AddErrorMessage(errorMsg string) {
	errMsg := chat.NewErrorMessage(errorMsg)
	lc.conversation = chat.AddMessage(lc.conversation, errMsg)

	// Save history to disk after adding error message
	if err := lc.saveHistory(); err != nil {
		lc.log.Error("Failed to save chat history", "error", err)
	}
}

// GetModel returns the model name
func (lc *LangChainController) GetModel() string {
	return lc.model
}

// GetToolRegistry returns the tool registry
func (lc *LangChainController) GetToolRegistry() *tools.Registry {
	return lc.toolRegistry
}

// SetModel sets the model for the conversation
func (lc *LangChainController) SetModel(model string) {
	lc.model = model
	lc.log.Debug("Model updated", "new_model", model)
}

// GetTokenUsage returns token usage (compatibility with ChatController interface)
func (lc *LangChainController) GetTokenUsage() (promptTokens, responseTokens int) {
	// LangChain doesn't provide the same token tracking as the basic client
	// Return 0,0 for now - this could be enhanced with LangChain usage tracking
	return 0, 0
}

// GetClient returns the underlying LangChain client
func (lc *LangChainController) GetClient() *langchain.Client {
	return lc.client
}

// SetOllamaClient is a no-op for LangChain controller (compatibility with ChatController interface)
func (lc *LangChainController) SetOllamaClient(client any) {
	// LangChain controller doesn't need separate Ollama client
	lc.log.Debug("SetOllamaClient called on LangChain controller (no-op)")
}

// ValidateModel validates that the model is available (compatibility with ChatController interface)
func (lc *LangChainController) ValidateModel(model string) error {
	// For now, assume the model is valid since LangChain handles this internally
	// This could be enhanced to actually validate against available models
	lc.log.Debug("ValidateModel called", "model", model)
	return nil
}

// StartStreaming initiates streaming for a user message (compatibility with ChatController interface)
func (lc *LangChainController) StartStreaming(ctx context.Context, content string) (<-chan StreamingUpdate, error) {
	lc.log.Debug("StartStreaming called", "content_length", len(content))

	// Create update channel
	updates := make(chan StreamingUpdate, 100)

	// Start streaming in goroutine
	go func() {
		defer close(updates)

		// Signal stream started
		updates <- StreamingUpdate{
			Type:     StreamStarted,
			StreamID: "langchain-stream",
			Content:  "",
		}

		// Replace any optimistic user message with final one
		finalUserMsg := chat.NewUserMessage(content)
		lc.conversation = chat.AddMessageWithDeduplication(lc.conversation, finalUserMsg)

		// Use real LangChain streaming instead of simulated chunks
		streamChan := make(chan string, 100)
		var fullResponse strings.Builder

		// Start real streaming in background
		go func() {
			defer close(streamChan)
			if err := lc.client.StreamMessage(ctx, content, streamChan); err != nil {
				lc.log.Error("LangChain streaming failed", "error", err)
				select {
				case updates <- StreamingUpdate{Type: StreamError, Error: err}:
				case <-ctx.Done():
				}
			}
		}()

		// Forward real stream chunks to TUI
		for chunk := range streamChan {
			fullResponse.WriteString(chunk)
			select {
			case updates <- StreamingUpdate{
				Type:     ChunkReceived,
				StreamID: "langchain-stream",
				Content:  chunk,
			}:
			case <-ctx.Done():
				return
			}
		}

		response := fullResponse.String()

		// Note: LangChain agent handles tool execution internally
		// We don't need to parse or simulate tool executions from the response

		// Add final assistant message to conversation
		assistantMsg := chat.NewAssistantMessage(response)
		lc.conversation = chat.AddMessage(lc.conversation, assistantMsg)

		// Signal completion
		updates <- StreamingUpdate{
			Type:     MessageComplete,
			StreamID: "langchain-stream",
		}
	}()

	return updates, nil
}

// saveHistory saves the current conversation to disk
func (lc *LangChainController) saveHistory() error {
	// Ensure the directory exists
	dir := filepath.Dir(lc.historyFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	// Get all messages from the conversation
	messages := chat.GetMessages(lc.conversation)

	// Save to JSON file (overwrite each time)
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal chat history: %w", err)
	}

	if err := os.WriteFile(lc.historyFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write chat history: %w", err)
	}

	lc.log.Debug("Chat history saved", "file", lc.historyFile, "messages", len(messages))
	return nil
}

// loadHistory loads chat history from disk
func (lc *LangChainController) loadHistory() error {
	// Check if history file exists
	if _, err := os.Stat(lc.historyFile); os.IsNotExist(err) {
		lc.log.Debug("No existing chat history found", "file", lc.historyFile)
		return nil
	}

	// Read the history file
	data, err := os.ReadFile(lc.historyFile)
	if err != nil {
		return fmt.Errorf("failed to read chat history: %w", err)
	}

	// Parse JSON
	var messages []chat.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("failed to unmarshal chat history: %w", err)
	}

	// Reconstruct conversation
	lc.conversation = chat.NewConversation(lc.model)
	for _, msg := range messages {
		lc.conversation = chat.AddMessage(lc.conversation, msg)
	}

	lc.log.Debug("Chat history loaded", "file", lc.historyFile, "messages", len(messages))
	return nil
}

// SaveHistoryToDisk saves the current conversation state to disk
func (lc *LangChainController) SaveHistoryToDisk() error {
	return lc.saveHistory()
}

// CleanThinkingBlocks removes thinking blocks from all assistant messages in the conversation
func (lc *LangChainController) CleanThinkingBlocks() {
	// Get current messages using the new API
	currentMessages := chat.GetMessages(lc.conversation)
	cleanedMessages := make([]chat.Message, 0, len(currentMessages))

	for _, msg := range currentMessages {
		if msg.Role == chat.RoleAssistant && msg.Content != "" {
			// Remove thinking blocks from assistant messages
			cleanedContent := chat.RemoveThinkingBlocks(msg.Content)
			cleanedMsg := msg
			cleanedMsg.Content = cleanedContent
			cleanedMessages = append(cleanedMessages, cleanedMsg)
		} else {
			// Keep other messages as-is
			cleanedMessages = append(cleanedMessages, msg)
		}
	}

	// Create a new conversation with cleaned messages
	lc.conversation = chat.NewConversation(lc.conversation.Model)
	for _, msg := range cleanedMessages {
		lc.conversation = chat.AddMessage(lc.conversation, msg)
	}
}

// SetAgentOrchestrator sets the agent orchestrator for dynamic agent selection
func (lc *LangChainController) SetAgentOrchestrator(orchestrator *agents.LangchainOrchestrator) {
	lc.orchestrator = orchestrator
	lc.log.Debug("Agent orchestrator set")
}
