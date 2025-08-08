package chat

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

func createViewport(width, height int) viewport.Model {
	vp := viewport.New(width, height)
	return vp
}

func (m *chatModel) setContent(messages []string) {
	m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(messages, "\n")))
}
