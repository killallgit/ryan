package tui

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/agent"
	chatpkg "github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/killallgit/ryan/pkg/streaming"
	"github.com/killallgit/ryan/pkg/tui/chat"
	"github.com/spf13/viper"
)

func RunTUI(orchestrator agent.Agent) error {
	ctx := context.Background()

	// Get configuration for chat history
	configFile := viper.ConfigFileUsed()
	baseDir := filepath.Dir(configFile)
	if configFile == "" {
		baseDir = ".ryan"
	}
	historyPath := filepath.Join(baseDir, "chat_history.json")
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

	// Create chat model with stream manager, chat manager, and injected orchestrator
	chatModel := chat.NewChatModel(manager, chatManager, orchestrator)

	// Store the chat model for later reference
	views := []tea.Model{chatModel}
	root := NewRootModel(ctx, views...)

	// Create the program
	p := tea.NewProgram(root, tea.WithContext(ctx), tea.WithAltScreen())

	// Store program reference in the stream manager for token updates
	manager.SetProgram(p)

	setupDebug()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

	return nil
}

func setupDebug() {
	logLevel := viper.GetString("logging.level")
	if logLevel == "debug" {
		f, err := tea.LogToFile("system.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}
}
