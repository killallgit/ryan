package chat

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const gap = "\n\n"

type (
	errMsg error
)

func (m chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width - 4)

		textAreaHeight := m.calculateTextAreaHeight()
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - textAreaHeight - 3

		if len(m.messages) > 0 {
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")))
		}
		m.viewport.GotoBottom()
	case tea.KeyMsg:
		return handleKeyMsg(m, msg)

	case errMsg:
		m.err = msg
		return m, nil
	}

	prevHeight := m.textarea.Height()
	newHeight := m.calculateTextAreaHeight()
	if prevHeight != newHeight {
		m.textarea.SetHeight(newHeight)
		m.updateViewportHeight()
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m *chatModel) calculateTextAreaHeight() int {
	lines := strings.Count(m.textarea.Value(), "\n") + 1
	maxHeight := 10
	if lines > maxHeight {
		return maxHeight
	}
	if lines < 1 {
		return 1
	}
	return lines
}

func (m *chatModel) updateViewportHeight() {
	if m.height > 0 {
		textAreaHeight := m.calculateTextAreaHeight()
		m.viewport.Height = m.height - textAreaHeight - 3
	}
}
