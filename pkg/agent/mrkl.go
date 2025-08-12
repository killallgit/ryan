package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/memory"
	"github.com/killallgit/ryan/pkg/prompt"
	"github.com/killallgit/ryan/pkg/stream/core"
	"github.com/killallgit/ryan/pkg/tokens"
	"github.com/killallgit/ryan/pkg/tools/registry"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
	lcmemory "github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/tools"
)

// MRKLAgent implements a simplified MRKL agent with unified behavior
type MRKLAgent struct {
	llm            llms.Model
	executor       *agents.Executor
	memory         *memory.Memory
	tools          []tools.Tool
	promptTemplate *prompt.ReactPromptTemplate
	customPrompt   string // For --system-prompt override
	appendPrompt   string // For --append-system-prompt
	tokenCounter   *tokens.TokenCounter
	tokensSent     int
	tokensRecv     int
	tokensMu       sync.RWMutex
}

// NewMRKLAgent creates a new MRKL agent with unified behavior
func NewMRKLAgent(llm llms.Model, continueHistory, skipPermissions bool) (*MRKLAgent, error) {
	logger.Info("Initializing unified MRKL Agent")

	// Load unified system prompt
	promptTemplate, err := prompt.LoadReactPrompt("unified")
	if err != nil {
		logger.Warn("Failed to load unified system prompt from file, using default: %v", err)
		promptTemplate = prompt.NewReactPromptTemplate(
			prompt.DefaultUnifiedPrompt(),
			"unified",
		)
	}

	// Generate session ID
	var sessionID string
	if continueHistory {
		sessionID = "default_project_session"
		logger.Debug("Using continued session ID: %s", sessionID)
	} else {
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
		logger.Debug("Created new session ID: %s", sessionID)
	}

	// Initialize memory
	mem, err := memory.New(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}

	// Get configuration
	settings := config.Get()

	// Get enabled tools
	agentTools := registry.Global().GetEnabled(settings, skipPermissions)
	logger.Info("Loaded %d tools", len(agentTools))

	// Create the conversational agent
	agent := agents.NewConversationalAgent(
		llm,
		agentTools,
	)

	// Create LangChain memory wrapper
	lcMem := lcmemory.NewConversationBuffer(
		lcmemory.WithChatHistory(mem.ChatMessageHistory()),
	)

	// Configure executor
	maxIterations := settings.LangChain.Tools.MaxIterations
	if maxIterations == 0 {
		maxIterations = 10 // Default max iterations
	}

	executor := agents.NewExecutor(
		agent,
		agents.WithMaxIterations(maxIterations),
		agents.WithMemory(lcMem),
	)

	// Initialize token counter
	modelName := settings.Ollama.DefaultModel
	tokenCounter, err := tokens.NewTokenCounter(modelName)
	if err != nil {
		logger.Warn("Could not initialize token counter: %v", err)
		tokenCounter = nil
	}

	return &MRKLAgent{
		llm:            llm,
		executor:       executor,
		memory:         mem,
		tools:          agentTools,
		promptTemplate: promptTemplate,
		tokenCounter:   tokenCounter,
	}, nil
}

// Execute handles a request and returns a response
func (m *MRKLAgent) Execute(ctx context.Context, prompt string) (string, error) {
	logger.Debug("MRKL Agent executing prompt: %s", prompt)

	// Track tokens
	if m.tokenCounter != nil {
		sentTokens := m.tokenCounter.CountTokens(prompt)
		m.updateTokensSent(sentTokens)
	}

	// Prepare the prompt with system context
	fullPrompt := m.preparePrompt(prompt)

	// Execute based on mode
	result, err := m.executor.Call(ctx, map[string]any{
		"input": fullPrompt,
	})
	if err != nil {
		logger.Error("MRKL execution failed: %v", err)
		return "", fmt.Errorf("execution failed: %w", err)
	}

	// Extract output
	output, ok := result["output"].(string)
	if !ok {
		return "", fmt.Errorf("invalid output type")
	}

	// Track response tokens
	if m.tokenCounter != nil {
		recvTokens := m.tokenCounter.CountTokens(output)
		m.updateTokensRecv(recvTokens)
	}

	// Clean output (unified approach)
	output = m.cleanOutput(output)

	logger.Debug("MRKL Agent completed execution")
	return output, nil
}

// ExecuteStream handles a request with streaming response
func (m *MRKLAgent) ExecuteStream(ctx context.Context, prompt string, handler core.Handler) error {
	logger.Debug("MRKL Agent executing streaming prompt: %s", prompt)

	// Track tokens
	if m.tokenCounter != nil {
		sentTokens := m.tokenCounter.CountTokens(prompt)
		m.updateTokensSent(sentTokens)
	}

	// Prepare the prompt
	fullPrompt := m.preparePrompt(prompt)

	// For now, execute without streaming and send result as single chunk
	// TODO: Implement proper streaming with ReAct pattern
	result, err := m.executor.Call(ctx, map[string]any{
		"input": fullPrompt,
	})
	if err != nil {
		logger.Error("MRKL streaming execution failed: %v", err)
		handler.OnError(err)
		return fmt.Errorf("streaming execution failed: %w", err)
	}

	output, ok := result["output"].(string)
	if !ok {
		handler.OnError(fmt.Errorf("invalid output type"))
		return fmt.Errorf("invalid output type")
	}

	// Track response tokens
	if m.tokenCounter != nil {
		recvTokens := m.tokenCounter.CountTokens(output)
		m.updateTokensRecv(recvTokens)
	}

	// Clean output (unified approach)
	output = m.cleanOutput(output)

	// Send as single chunk
	if err := handler.OnChunk([]byte(output)); err != nil {
		return err
	}

	// Notify completion
	if err := handler.OnComplete(output); err != nil {
		return err
	}

	logger.Debug("MRKL Agent completed streaming execution")
	return nil
}

// ClearMemory clears the conversation memory
func (m *MRKLAgent) ClearMemory() error {
	logger.Debug("Clearing MRKL Agent memory")
	return m.memory.Clear()
}

// GetTokenStats returns the cumulative token usage statistics
func (m *MRKLAgent) GetTokenStats() (int, int) {
	m.tokensMu.RLock()
	defer m.tokensMu.RUnlock()
	return m.tokensSent, m.tokensRecv
}

// Close cleans up resources
func (m *MRKLAgent) Close() error {
	logger.Debug("Closing MRKL Agent")
	if m.memory != nil {
		return m.memory.Close()
	}
	return nil
}

// SetCustomPrompt sets a custom system prompt override
func (m *MRKLAgent) SetCustomPrompt(systemPrompt string) {
	m.customPrompt = systemPrompt
	logger.Info("Set custom system prompt")
}

// SetAppendPrompt sets additional prompt instructions
func (m *MRKLAgent) SetAppendPrompt(appendPrompt string) {
	m.appendPrompt = appendPrompt
	logger.Info("Set append prompt instructions")
}

// preparePrompt prepares the full prompt with system context
func (m *MRKLAgent) preparePrompt(userInput string) string {
	// If custom prompt is set, use it instead
	if m.customPrompt != "" {
		basePrompt := m.customPrompt
		if m.appendPrompt != "" {
			basePrompt += "\n\n" + m.appendPrompt
		}
		return basePrompt + "\n\nUser input: " + userInput
	}

	// Build tool descriptions
	var toolDescs []string
	for _, tool := range m.tools {
		toolDescs = append(toolDescs, fmt.Sprintf("- %s: %s", tool.Name(), tool.Description()))
	}
	toolDescriptions := strings.Join(toolDescs, "\n")

	// Get conversation history from memory if available
	history := ""
	if m.memory != nil && m.memory.ChatMessageHistory() != nil {
		// Get recent messages for context
		msgs, err := m.memory.ChatMessageHistory().Messages(context.Background())
		if err == nil && len(msgs) > 0 {
			var historyLines []string
			// Take last few messages for context
			start := 0
			if len(msgs) > 6 { // Keep last 3 exchanges
				start = len(msgs) - 6
			}
			for _, msg := range msgs[start:] {
				switch msg.GetType() {
				case "human":
					historyLines = append(historyLines, fmt.Sprintf("Human: %s", msg.GetContent()))
				case "ai":
					historyLines = append(historyLines, fmt.Sprintf("Assistant: %s", msg.GetContent()))
				}
			}
			if len(historyLines) > 0 {
				history = strings.Join(historyLines, "\n")
			}
		}
	}

	// Use the loaded prompt template to format the prompt
	formatted, err := m.promptTemplate.Format(toolDescriptions, history, userInput)
	if err != nil {
		logger.Error("Failed to format prompt: %v", err)
		return userInput
	}

	// Apply append prompt if set
	if m.appendPrompt != "" {
		formatted += "\n\nAdditional Instructions: " + m.appendPrompt
	}

	return formatted
}

// cleanOutput cleans the output using unified approach
func (m *MRKLAgent) cleanOutput(output string) string {
	// Look for Final Answer first
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Final Answer:") {
			answer := strings.TrimPrefix(trimmed, "Final Answer:")
			return strings.TrimSpace(answer)
		}
	}

	// If no Final Answer found, clean up ReAct artifacts but preserve content
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip internal ReAct pattern lines that aren't useful to user
		if strings.HasPrefix(trimmed, "Observation:") ||
			(strings.HasPrefix(trimmed, "Thought:") && strings.Contains(trimmed, "I need to")) ||
			(strings.HasPrefix(trimmed, "Action:") && trimmed != "Action:") {
			continue
		}

		if trimmed != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// updateTokensSent atomically updates sent tokens count
func (m *MRKLAgent) updateTokensSent(count int) {
	m.tokensMu.Lock()
	m.tokensSent += count
	m.tokensMu.Unlock()
}

// updateTokensRecv atomically updates received tokens count
func (m *MRKLAgent) updateTokensRecv(count int) {
	m.tokensMu.Lock()
	m.tokensRecv += count
	m.tokensMu.Unlock()
}

// Ensure MRKLAgent implements Agent interface
var _ Agent = (*MRKLAgent)(nil)
