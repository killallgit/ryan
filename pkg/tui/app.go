package tui

import (
	"context"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/killallgit/ryan/pkg/streaming"
	"github.com/killallgit/ryan/pkg/tui/chat"
	"github.com/spf13/viper"
)

func StartApp() error {
	ctx := context.Background()

	// Create streaming infrastructure
	registry := streaming.NewRegistry()
	manager := streaming.NewManager(registry)

	// Register Ollama provider
	ollamaClient := ollama.NewClient()
	registry.Register("ollama-main", "ollama", ollamaClient)

	// Create chat model with stream manager
	chatModel := chat.NewChatModel(manager)

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
