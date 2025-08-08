package chat

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func handleKeyMsg(m chatModel, msg tea.KeyMsg) (chatModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		fmt.Println(m.textarea.Value())
		return m, tea.Quit
	case tea.KeyEnter:
		m.messages = append(m.messages, m.createMessageNode(m.textarea.Value()))
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")))
		m.textarea.Reset()
		m.viewport.GotoBottom()
	}
	return m, nil
}
