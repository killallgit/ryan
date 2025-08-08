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
	"github.com/spf13/viper"
)

// Runner runs the chat in headless mode
type Runner struct {
	chatManager *chat.Manager
	provider    llm.Provider
	output      *Output
	config      *Config
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

	// Log prompt if debug logging is enabled
	if r.config.DebugLogging {
		log.Printf("User prompt: %s", prompt)
	}

	// Add user message to history
	if err := r.chatManager.AddMessage(chat.RoleUser, prompt); err != nil {
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
	_, err := r.chatManager.StartStreaming(chat.RoleAssistant)
	if err != nil {
		return fmt.Errorf("failed to start streaming: %w", err)
	}

	// No spinner needed for headless mode

	// Generate streaming response with history
	history := r.chatManager.GetHistory()
	llmMessages := make([]llm.Message, 0, len(history))
	for _, msg := range history {
		llmMessages = append(llmMessages, llm.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
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
	if err := r.chatManager.AppendToStream(finalContent); err != nil {
		return fmt.Errorf("failed to append to stream: %w", err)
	}

	// End streaming
	if err := r.chatManager.EndStreaming(); err != nil {
		return fmt.Errorf("failed to end streaming: %w", err)
	}

	// Log completion if debug logging is enabled
	if r.config.DebugLogging {
		log.Printf("Response complete")
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
