package chat

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type (
	errMsg error
)

func (m chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Set textarea width to use most of the available width
		// Account for padding/borders
		m.textarea.SetWidth(msg.Width - 4)

		// Calculate height after setting width
		textAreaHeight := m.calculateTextAreaHeight()
		m.textarea.SetHeight(textAreaHeight)

		// Update viewport dimensions
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - textAreaHeight - 3

		if len(m.messages) > 0 {
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, "\n")))
		}
		m.viewport.GotoBottom()

	case tea.KeyMsg:
		// All key handling happens in handleKeyMsg
		return handleKeyMsg(m, msg)

	case errMsg:
		m.err = msg
		return m, nil

	default:
		// Update textarea for other messages (like blink cursor)
		var tiCmd tea.Cmd
		m.textarea, tiCmd = m.textarea.Update(msg)
		cmds = append(cmds, tiCmd)

		// Update viewport for other messages
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *chatModel) calculateTextAreaHeight() int {
	content := m.textarea.Value()
	if content == "" {
		return 1
	}

	// Split content into actual lines
	lines := strings.Split(content, "\n")
	totalVisualLines := 0

	// Get the textarea width for calculating wrapped lines
	textWidth := m.textarea.Width()
	if textWidth <= 0 {
		textWidth = m.width - 4
		if textWidth <= 0 {
			textWidth = 80 // fallback
		}
	}

	// Calculate visual lines for each actual line
	for _, line := range lines {
		if line == "" {
			totalVisualLines++
		} else {
			// Calculate display width using runewidth for proper Unicode handling
			lineWidth := runewidth.StringWidth(line)
			// Calculate how many visual lines this takes
			visualLines := (lineWidth + textWidth - 1) / textWidth
			if visualLines < 1 {
				visualLines = 1
			}
			totalVisualLines += visualLines
		}
	}

	// Apply max height constraint
	maxHeight := 10
	if totalVisualLines > maxHeight {
		return maxHeight
	}
	if totalVisualLines < 1 {
		return 1
	}

	return totalVisualLines
}

func (m *chatModel) updateViewportHeight() {
	if m.height > 0 {
		textAreaHeight := m.calculateTextAreaHeight()
		m.viewport.Height = m.height - textAreaHeight - 3
	}
}
