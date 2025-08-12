package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/memory"
	"github.com/killallgit/ryan/pkg/stream/core"
	"github.com/killallgit/ryan/pkg/tokens"
	"github.com/killallgit/ryan/pkg/tools/registry"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
	lcmemory "github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/tools"
)

// OperatingMode defines the agent's operating mode
type OperatingMode string

const (
	// ExecuteMode - Direct execution with ReAct pattern
	ExecuteMode OperatingMode = "execute-mode"
	// PlanMode - Planning and strategizing before execution
	PlanMode OperatingMode = "plan-mode"
)

// MRKLAgent implements a simplified MRKL agent with configurable operating modes
type MRKLAgent struct {
	llm          llms.Model
	executor     *agents.Executor
	memory       *memory.Memory
	tools        []tools.Tool
	mode         OperatingMode
	systemPrompt string
	tokenCounter *tokens.TokenCounter
	tokensSent   int
	tokensRecv   int
	tokensMu     sync.RWMutex
}

// NewMRKLAgent creates a new MRKL agent with the specified operating mode
func NewMRKLAgent(llm llms.Model, mode OperatingMode, continueHistory, skipPermissions bool) (*MRKLAgent, error) {
	logger.Info("Initializing MRKL Agent in %s", mode)

	// Load system prompt for the operating mode
	systemPrompt, err := loadSystemPrompt(mode)
	if err != nil {
		logger.Warn("Failed to load system prompt from file, using default: %v", err)
		systemPrompt = getDefaultPrompt(mode)
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

	// Create the appropriate agent based on mode
	var agent agents.Agent
	if mode == ExecuteMode {
		// Use conversational agent for execution mode
		agent = agents.NewConversationalAgent(
			llm,
			agentTools,
		)
	} else {
		// For plan mode, we can use a simpler agent or custom implementation
		// For now, using conversational agent with different prompting
		agent = agents.NewConversationalAgent(
			llm,
			agentTools,
		)
	}

	// Create LangChain memory wrapper
	lcMem := lcmemory.NewConversationBuffer(
		lcmemory.WithChatHistory(mem.ChatMessageHistory()),
	)

	// Configure executor
	maxIterations := settings.LangChain.Tools.MaxIterations
	if maxIterations == 0 {
		if mode == ExecuteMode {
			maxIterations = 10
		} else {
			maxIterations = 3 // Fewer iterations for planning
		}
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
		llm:          llm,
		executor:     executor,
		memory:       mem,
		tools:        agentTools,
		mode:         mode,
		systemPrompt: systemPrompt,
		tokenCounter: tokenCounter,
	}, nil
}

// Execute handles a request and returns a response
func (m *MRKLAgent) Execute(ctx context.Context, prompt string) (string, error) {
	logger.Debug("MRKL Agent (%s) executing prompt: %s", m.mode, prompt)

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

	// Clean output based on mode
	if m.mode == ExecuteMode {
		output = m.cleanExecuteModeOutput(output)
	} else {
		output = m.cleanPlanModeOutput(output)
	}

	logger.Debug("MRKL Agent completed execution")
	return output, nil
}

// ExecuteStream handles a request with streaming response
func (m *MRKLAgent) ExecuteStream(ctx context.Context, prompt string, handler core.Handler) error {
	logger.Debug("MRKL Agent (%s) executing streaming prompt: %s", m.mode, prompt)

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

	// Clean output based on mode
	if m.mode == ExecuteMode {
		output = m.cleanExecuteModeOutput(output)
	} else {
		output = m.cleanPlanModeOutput(output)
	}

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

// GetMode returns the current operating mode
func (m *MRKLAgent) GetMode() OperatingMode {
	return m.mode
}

// SetMode changes the operating mode
func (m *MRKLAgent) SetMode(mode OperatingMode) error {
	systemPrompt, err := loadSystemPrompt(mode)
	if err != nil {
		logger.Warn("Failed to load system prompt for mode %s, using default", mode)
		systemPrompt = getDefaultPrompt(mode)
	}
	m.mode = mode
	m.systemPrompt = systemPrompt
	logger.Info("Switched to %s", mode)
	return nil
}

// preparePrompt prepares the full prompt with system context
func (m *MRKLAgent) preparePrompt(userInput string) string {
	// Build tool descriptions
	var toolDescs []string
	for _, tool := range m.tools {
		toolDescs = append(toolDescs, fmt.Sprintf("- %s: %s", tool.Name(), tool.Description()))
	}
	toolDescriptions := strings.Join(toolDescs, "\n")

	// Get conversation history
	history := "" // Could be populated from memory if needed

	// Create prompt template
	tmpl := prompts.NewPromptTemplate(
		m.systemPrompt,
		[]string{"tool_descriptions", "history", "input"},
	)

	// Format the prompt
	formatted, err := tmpl.Format(map[string]any{
		"tool_descriptions": toolDescriptions,
		"history":           history,
		"input":             userInput,
	})
	if err != nil {
		logger.Error("Failed to format prompt: %v", err)
		return userInput
	}

	return formatted
}

// cleanExecuteModeOutput cleans the output for execute mode
func (m *MRKLAgent) cleanExecuteModeOutput(output string) string {
	// Remove ReAct pattern artifacts
	lines := strings.Split(output, "\n")
	var cleaned []string
	skipNext := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip ReAct pattern lines
		if strings.HasPrefix(trimmed, "Thought:") ||
			strings.HasPrefix(trimmed, "Action:") ||
			strings.HasPrefix(trimmed, "Action Input:") ||
			strings.HasPrefix(trimmed, "Observation:") {
			skipNext = true
			continue
		}

		// Extract final answer
		if strings.HasPrefix(trimmed, "Final Answer:") {
			answer := strings.TrimPrefix(trimmed, "Final Answer:")
			return strings.TrimSpace(answer)
		}

		if !skipNext && trimmed != "" {
			cleaned = append(cleaned, line)
		}
		skipNext = false
	}

	result := strings.Join(cleaned, "\n")
	return strings.TrimSpace(result)
}

// cleanPlanModeOutput cleans the output for plan mode
func (m *MRKLAgent) cleanPlanModeOutput(output string) string {
	// For plan mode, we want to keep the structured output
	// but remove any ReAct artifacts that might leak through
	lines := strings.Split(output, "\n")
	var cleaned []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip pure ReAct pattern lines that aren't part of the plan
		if strings.HasPrefix(trimmed, "Observation:") {
			continue
		}

		cleaned = append(cleaned, line)
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

// loadSystemPrompt loads the system prompt from a markdown file
func loadSystemPrompt(mode OperatingMode) (string, error) {
	promptPath := filepath.Join("prompts", string(mode), "SYSTEM_PROMPT.md")
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file: %w", err)
	}
	return string(content), nil
}

// getDefaultPrompt returns a default prompt if file loading fails
func getDefaultPrompt(mode OperatingMode) string {
	if mode == ExecuteMode {
		return `You are a helpful AI assistant. Use the available tools to help answer questions and complete tasks.

Available tools:
{{.tool_descriptions}}

Previous conversation:
{{.history}}

User input: {{.input}}

Respond using the ReAct pattern (Thought, Action, Action Input, Observation) when using tools.`
	}

	// Plan mode default
	return `You are a helpful AI assistant in planning mode. Analyze tasks and create detailed plans.

Available tools for planning:
{{.tool_descriptions}}

Previous conversation:
{{.history}}

Task to plan: {{.input}}

Create a detailed, step-by-step plan for accomplishing this task.`
}

// Ensure MRKLAgent implements Agent interface
var _ Agent = (*MRKLAgent)(nil)
