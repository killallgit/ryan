package tui

import (
	"context"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/tui/chat"
	"github.com/spf13/viper"
)

func StartApp() error {
	ctx := context.Background()
	views := []tea.Model{chat.NewChatModel()}
	root := NewRootModel(ctx, views...)
	p := tea.NewProgram(root, tea.WithContext(ctx), tea.WithAltScreen())
	setupDebug()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

	return nil
}

func CmdHandler(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func setupDebug() {
	logLevel := viper.GetString("logging.level")
	if logLevel == "debug" {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}
}
