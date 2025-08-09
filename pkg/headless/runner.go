package headless

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/tokens"
	"github.com/spf13/viper"
)

// runner runs the chat in headless mode
type runner struct {
	chatManager  *chat.Manager
	orchestrator agent.Agent
	output       *Output
	config       *config
	tokensSent   int
	tokensRecv   int
}

// config contains headless runner configuration
type config struct {
	historyPath     string
	showThinking    bool
	continueHistory bool
	debugLogging    bool
	logFile         string
}

// newRunner creates a new headless runner with injected orchestrator
func newRunner(orchestrator agent.Agent) (*runner, error) {
	// Get the base directory from the config file location
	configFile := viper.ConfigFileUsed()
	baseDir := filepath.Dir(configFile)
	if configFile == "" {
		baseDir = ".ryan"
	}

	// Setup configuration
	cfg := &config{
		historyPath:     filepath.Join(baseDir, "chat_history.json"),
		showThinking:    viper.GetBool("show_thinking"),
		continueHistory: viper.GetBool("continue"),
		debugLogging:    viper.GetString("logging.level") == "debug",
		logFile:         viper.GetString("logging.log_file"),
	}

	// Setup logging if debug is enabled
	if cfg.debugLogging && cfg.logFile != "" {
		// Handle log file path resolution
		logPath := cfg.logFile
		if !filepath.IsAbs(logPath) {
			// Clean the path and handle relative paths properly
			logPath = filepath.Clean(logPath)
			// If it starts with ./, remove it and treat as relative to current directory
			logPath = strings.TrimPrefix(logPath, "./")

			// Now join with the current working directory
			logPath = filepath.Join(".", logPath)
		}

		// Ensure log directory exists
		logDir := filepath.Dir(logPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// Create log file
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		log.SetOutput(logFile)
		log.SetPrefix("[HEADLESS] ")
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
		chatManager:  chatManager,
		orchestrator: orchestrator,
		output:       output,
		config:       cfg,
	}, nil
}

// run executes a single prompt in headless mode
func (r *runner) run(ctx context.Context, prompt string) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty in headless mode")
	}

	// Initialize token counter (declare at function scope)
	modelName := viper.GetString("ollama.default_model")
	tokenCounter, err := tokens.NewTokenCounter(modelName)
	if err != nil {
		// Log warning but continue
		if r.config.debugLogging {
			log.Printf("Warning: Could not initialize token counter: %v", err)
		}
		tokenCounter = nil
	}

	// Count tokens for the prompt
	promptTokens := 0
	if tokenCounter != nil {
		promptTokens = tokenCounter.CountTokens(prompt)
		r.tokensSent += promptTokens
	}

	// Log prompt if debug logging is enabled
	if r.config.debugLogging {
		log.Printf("User prompt: %s (tokens: %d)", prompt, promptTokens)
	}

	// Add user message to history with token count
	userMsg := chat.NewMessage(chat.RoleUser, prompt)
	userMsg.Metadata.TokensUsed = promptTokens
	if err := r.chatManager.AddMessageWithMetadata(userMsg); err != nil {
		return fmt.Errorf("failed to add user message: %w", err)
	}

	// Log prompt for debugging

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
	var finalContent string
	streamHandler := &consoleStreamHandler{
		content: "",
	}

	// Use orchestrator to generate streaming response
	generateErr := r.orchestrator.ExecuteStream(ctx, prompt, streamHandler)
	if generateErr != nil {
		r.output.Error(fmt.Sprintf("Generation error: %v", generateErr))
		return generateErr
	}

	// Get final content from handler
	finalContent = streamHandler.content

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

	// Log completion if debug logging is enabled
	if r.config.debugLogging {
		log.Printf("Response complete (tokens: %d)", responseTokens)
		log.Printf("Total tokens - Sent: %d, Received: %d", r.tokensSent, r.tokensRecv)
	}

	return nil
}

// runInteractive runs an interactive session (for future implementation)
func (r *runner) runInteractive(ctx context.Context) error {
	return fmt.Errorf("interactive headless mode not yet implemented")
}

// cleanup performs cleanup operations
func (r *runner) cleanup() error {
	// Note: orchestrator cleanup is handled by the caller (cmd/root.go)
	// since it owns the orchestrator lifecycle
	return nil
}

// consoleStreamHandler implements the agent.StreamHandler interface
type consoleStreamHandler struct {
	content string
}

func (h *consoleStreamHandler) OnChunk(chunk string) error {
	fmt.Print(chunk)
	h.content += chunk
	return nil
}

func (h *consoleStreamHandler) OnComplete(finalContent string) error {
	if finalContent != "" {
		h.content = finalContent
	}
	return nil
}

func (h *consoleStreamHandler) OnError(err error) {
	fmt.Fprintf(os.Stderr, "\nStreaming error: %v\n", err)
}
