package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/agent"
	chatpkg "github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/killallgit/ryan/pkg/streaming"
	"github.com/killallgit/ryan/pkg/tui/chat"
	"github.com/spf13/viper"
)

func RunTUI(agent agent.Agent) error {
	ctx := context.Background()

	// Get configuration for chat history using config helper
	historyPath := config.BuildSettingsPath("chat_history.json")
	continueHistory := viper.GetBool("continue")

	// Create chat manager for history management
	chatManager, err := chatpkg.NewManager(historyPath)
	if err != nil {
		return fmt.Errorf("failed to create chat manager: %w", err)
	}

	// Clear history if not continuing
	if !continueHistory {
		if err := chatManager.ClearHistory(); err != nil {
			return fmt.Errorf("failed to clear history: %w", err)
		}
	}

	// Create streaming infrastructure
	registry := streaming.NewRegistry()
	manager := streaming.NewManager(registry)

	// Register Ollama provider
	ollamaClient := ollama.NewClient()
	registry.Register("ollama-main", "ollama", ollamaClient)

	// Create chat model with stream manager, chat manager, and injected agent
	chatModel := chat.NewChatModel(manager, chatManager, agent)

	// Store the chat model for later reference
	views := []tea.Model{chatModel}
	root := NewRootModel(ctx, views...)

	// Create the program
	p := tea.NewProgram(root, tea.WithContext(ctx), tea.WithAltScreen())

	// Store program reference in the stream manager for token updates
	manager.SetProgram(p)

	setupDebug()

	if _, err := p.Run(); err != nil {
		logger.Fatal("TUI program error: %v", err)
	}

	return nil
}

func setupDebug() {
	logLevel := viper.GetString("logging.level")
	if logLevel == "debug" {
		logPath := config.BuildSettingsPath("tui-debug.log")
		_, err := tea.LogToFile(logPath, "debug")
		if err != nil {
			logger.Fatal("TUI debug logging setup error: %v", err)
		}
		// Note: tea.LogToFile returns a file that Bubble Tea manages internally
		// We don't need to manually close it
	}
}
