package headless

import (
	"context"
	"fmt"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tokens"
)

// runner runs the chat in headless mode
type runner struct {
	chatManager *chat.Manager
	agent       agent.Agent
	output      *Output
	config      *runConfig
	tokensSent  int
	tokensRecv  int
}

// runConfig contains headless runner configuration
type runConfig struct {
	historyPath     string
	showThinking    bool
	continueHistory bool
}

// newRunner creates a new headless runner with injected agent
func newRunner(agent agent.Agent) (*runner, error) {
	return newRunnerWithOptions(agent, false)
}

// newRunnerWithOptions creates a new headless runner with injected agent and options
func newRunnerWithOptions(agent agent.Agent, continueHistory bool) (*runner, error) {
	// Setup configuration using config helper
	settings := config.Get()
	cfg := &runConfig{
		historyPath:     config.BuildSettingsPath("chat_history.json"),
		showThinking:    settings.ShowThinking,
		continueHistory: continueHistory,
	}

	// Create chat manager
	chatManager, err := chat.NewManager(cfg.historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat manager: %w", err)
	}

	// Clear history if not continuing
	if !cfg.continueHistory {
		if err := chatManager.ClearHistory(); err != nil {
			return nil, fmt.Errorf("failed to clear history: %w", err)
		}
	}

	// Create output handler
	output := NewOutput()

	return &runner{
		chatManager: chatManager,
		agent:       agent,
		output:      output,
		config:      cfg,
	}, nil
}

// run executes a single prompt in headless mode
func (r *runner) run(ctx context.Context, prompt string) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty in headless mode")
	}

	// Initialize token counter (declare at function scope)
	settings := config.Get()
	tokenCounter, err := tokens.NewTokenCounter(settings.Ollama.DefaultModel)
	if err != nil {
		// Log warning but continue
		logger.Warn("Could not initialize token counter: %v", err)
		tokenCounter = nil
	}

	// Count tokens for the prompt
	promptTokens := 0
	if tokenCounter != nil {
		promptTokens = tokenCounter.CountTokens(prompt)
		r.tokensSent += promptTokens
	}

	// Log prompt for debugging
	logger.Debug("User prompt: %s (tokens: %d)", prompt, promptTokens)

	// Add user message to history with token count
	userMsg := chat.NewMessage(chat.RoleUser, prompt)
	userMsg.Metadata.TokensUsed = promptTokens
	if err := r.chatManager.AddMessageWithMetadata(userMsg); err != nil {
		return fmt.Errorf("failed to add user message: %w", err)
	}

	// Setup chat manager streaming callback
	r.chatManager.SetStreamCallback(func(content string) error {
		// Content is already printed by stream handler
		return nil
	})

	// Start streaming response
	_, err = r.chatManager.StartStreaming(chat.RoleAssistant)
	if err != nil {
		return fmt.Errorf("failed to start streaming: %w", err)
	}

	// Create a stream handler that prints to console and collects content
	streamHandler := newHeadlessStreamHandler()

	// Use agent to generate streaming response
	generateErr := r.agent.ExecuteStream(ctx, prompt, streamHandler)
	if generateErr != nil {
		r.output.Error(fmt.Sprintf("Generation error: %v", generateErr))
		return generateErr
	}

	// Get final content from handler
	finalContent := streamHandler.GetContent()

	// Count response tokens
	responseTokens := 0
	if tokenCounter != nil {
		responseTokens = tokenCounter.CountTokens(finalContent)
		r.tokensRecv += responseTokens
	}

	// Append to stream with token metadata
	if err := r.chatManager.AppendToStream(finalContent); err != nil {
		return fmt.Errorf("failed to append to stream: %w", err)
	}

	// End streaming with token count
	if err := r.chatManager.EndStreamingWithTokens(responseTokens); err != nil {
		return fmt.Errorf("failed to end streaming: %w", err)
	}

	// Print token summary
	fmt.Printf("\n[Tokens - Sent: %d, Received: %d, Total: %d]\n",
		r.tokensSent, r.tokensRecv, r.tokensSent+r.tokensRecv)

	// Log completion for debugging
	logger.Debug("Response complete (tokens: %d)", responseTokens)
	logger.Debug("Total tokens - Sent: %d, Received: %d", r.tokensSent, r.tokensRecv)

	return nil
}

// cleanup performs cleanup operations
func (r *runner) cleanup() error {
	// Note: agent cleanup is handled by the caller (cmd/root.go)
	// since it owns the agent lifecycle
	return nil
}
