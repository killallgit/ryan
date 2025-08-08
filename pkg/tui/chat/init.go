package chat

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m chatModel) Init() tea.Cmd {
	return textarea.Blink
}
