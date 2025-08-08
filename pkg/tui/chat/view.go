package chat

import (
	"github.com/charmbracelet/lipgloss"
)

func (m chatModel) View() string {
	statusLine := ""
	if m.isStreaming {
		statusLine = m.spinner.View() + " Generating response..."
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		statusLine,
		m.textarea.View(),
	)
}
