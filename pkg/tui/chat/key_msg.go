package chat

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func handleKeyMsg(m chatModel, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.numEscPress++
		if m.numEscPress == 2 {
			m.textarea.Reset()
			m.numEscPress = 0
			return m, nil
		}
	case tea.KeyEnter:
		if msg.Alt {
			// Alt+Enter adds a newline
			break
		}
		if m.textarea.Value() != "" {
			m.messages = append(m.messages, m.createMessageNode("user", m.textarea.Value()))
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")))
			m.textarea.Reset()
			m.textarea.SetHeight(1)
			m.updateViewportHeight()
			m.viewport.GotoBottom()
			return m, nil
		}
	}

	// Let the textarea handle the key
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)

	// Recalculate and update height after any key input
	newHeight := m.calculateTextAreaHeight()
	if m.textarea.Height() != newHeight {
		m.textarea.SetHeight(newHeight)
		m.updateViewportHeight()
	}

	return m, cmd
}
