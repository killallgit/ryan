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

	// Create views with stream manager
	views := []tea.Model{chat.NewChatModel(manager)}
	root := NewRootModel(ctx, views...)
	p := tea.NewProgram(root, tea.WithContext(ctx), tea.WithAltScreen())
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
