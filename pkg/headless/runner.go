package headless

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/llm"
	"github.com/killallgit/ryan/pkg/tokens"
	"github.com/spf13/viper"
)

// Runner runs the chat in headless mode
type Runner struct {
	chatManager *chat.Manager
	provider    llm.Provider
	output      *Output
	config      *Config
	tokensSent  int
	tokensRecv  int
}

// Config contains headless runner configuration
type Config struct {
	HistoryPath     string
	ShowThinking    bool
	ContinueHistory bool
	DebugLogging    bool
	LogFile         string
}

// NewRunner creates a new headless runner
func NewRunner() (*Runner, error) {
	// Get the base directory from the config file location
	configFile := viper.ConfigFileUsed()
	baseDir := filepath.Dir(configFile)
	if configFile == "" {
		baseDir = ".ryan"
	}

	// Setup configuration
	config := &Config{
		HistoryPath:     filepath.Join(baseDir, "chat_history.json"),
		ShowThinking:    viper.GetBool("show_thinking"),
		ContinueHistory: viper.GetBool("continue"),
		DebugLogging:    viper.GetString("logging.level") == "debug",
		LogFile:         viper.GetString("logging.log_file"),
	}

	// Setup logging if debug is enabled
	if config.DebugLogging && config.LogFile != "" {
		// Handle log file path resolution
		logPath := config.LogFile
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
	chatManager, err := chat.NewManager(config.HistoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat manager: %w", err)
	}

	// Clear history if not continuing
	if !config.ContinueHistory {
		if err := chatManager.ClearHistory(); err != nil {
			return nil, fmt.Errorf("failed to clear history: %w", err)
		}
	}

	// Create LLM provider
	provider, err := llm.NewOllamaAdapter()
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama adapter: %w", err)
	}

	// Create output handler
	output := NewOutput()

	return &Runner{
		chatManager: chatManager,
		provider:    provider,
		output:      output,
		config:      config,
	}, nil
}

// Run executes a single prompt in headless mode
func (r *Runner) Run(ctx context.Context, prompt string) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty in headless mode")
	}

	// Initialize token counter (declare at function scope)
	modelName := viper.GetString("ollama.default_model")
	tokenCounter, err := tokens.NewTokenCounter(modelName)
	if err != nil {
		// Log warning but continue
		if r.config.DebugLogging {
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
	if r.config.DebugLogging {
		log.Printf("User prompt: %s (tokens: %d)", prompt, promptTokens)
	}

	// Add user message to history with token count
	userMsg := chat.NewMessage(chat.RoleUser, prompt)
	userMsg.Metadata.TokensUsed = promptTokens
	if err := r.chatManager.AddMessageWithMetadata(userMsg); err != nil {
		return fmt.Errorf("failed to add user message: %w", err)
	}

	// Create streaming handler
	handler := llm.NewConsoleStreamHandler()

	// Setup chat manager streaming callback
	r.chatManager.SetStreamCallback(func(content string) error {
		// Content is already printed by ConsoleStreamHandler
		return nil
	})

	// Start streaming response
	_, err = r.chatManager.StartStreaming(chat.RoleAssistant)
	if err != nil {
		return fmt.Errorf("failed to start streaming: %w", err)
	}

	// No spinner needed for headless mode

	// Generate streaming response with history
	// Try to use memory if enabled, otherwise fall back to regular history
	var llmMessages []llm.Message
	memoryMessages, err := r.chatManager.GetMemoryMessages()
	if err != nil {
		return fmt.Errorf("failed to get memory messages: %w", err)
	}

	if len(memoryMessages) > 0 {
		// Use memory messages if available
		llmMessages = memoryMessages
	} else {
		// Fall back to regular history
		history := r.chatManager.GetHistory()
		llmMessages = make([]llm.Message, 0, len(history))
		for _, msg := range history {
			llmMessages = append(llmMessages, llm.Message{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
		}
	}

	// Use provider to generate response
	var generateErr error
	if convProvider, ok := r.provider.(llm.ConversationalProvider); ok {
		generateErr = convProvider.GenerateStreamWithHistory(ctx, llmMessages, handler)
	} else {
		generateErr = r.provider.GenerateStream(ctx, prompt, handler)
	}

	// No spinner to stop

	if generateErr != nil {
		r.output.Error(fmt.Sprintf("Generation error: %v", generateErr))
		return generateErr
	}

	// Update stream with final content
	finalContent := handler.GetContent()

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
	if r.config.DebugLogging {
		log.Printf("Response complete (tokens: %d)", responseTokens)
		log.Printf("Total tokens - Sent: %d, Received: %d", r.tokensSent, r.tokensRecv)
	}

	return nil
}

// RunInteractive runs an interactive session (for future implementation)
func (r *Runner) RunInteractive(ctx context.Context) error {
	return fmt.Errorf("interactive headless mode not yet implemented")
}

// Cleanup performs cleanup operations
func (r *Runner) Cleanup() error {
	// Save history one final time
	if r.chatManager != nil {
		// History is already saved after each message
		return nil
	}
	return nil
}
