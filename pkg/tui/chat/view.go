package chat

import (
	"github.com/charmbracelet/lipgloss"
)

func (m chatModel) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		m.statusBar.View(),
		m.textarea.View(),
	)
}
