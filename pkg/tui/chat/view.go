package chat

import (
	"fmt"
	// "github.com/charmbracelet/lipgloss"
)

// var gap = lipgloss.NewStyle().Height(1).Render(" ")

func (m chatModel) View() string {
	return fmt.Sprintf(
		"%s%s%s",
		m.viewport.View(),
		"\n\n",
		m.textarea.View(),
	)
}
