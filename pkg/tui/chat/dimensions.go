package chat

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// calculateTextAreaHeight determines the visual height of the textarea
// based on its content and wrapping
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

// updateViewportHeight adjusts the viewport height based on textarea size
func (m *chatModel) updateViewportHeight() {
	if m.height > 0 {
		textAreaHeight := m.calculateTextAreaHeight()
		// Account for status bar (1 line) and spacing
		m.viewport.Height = m.height - textAreaHeight - 4
	}
}

// handleWindowResize updates all dimensions when window size changes
func (m *chatModel) handleWindowResize(width, height int) {
	m.width = width
	m.height = height

	// Set textarea width to use most of the available width
	// Account for padding/borders
	m.textarea.SetWidth(width - 4)

	// Calculate height after setting width
	textAreaHeight := m.calculateTextAreaHeight()
	m.textarea.SetHeight(textAreaHeight)

	// Update viewport dimensions
	m.viewport.Width = width
	// Account for textarea, status bar, and spacing
	m.viewport.Height = height - textAreaHeight - 4

	// Re-render content if we have messages
	if len(m.nodes) > 0 {
		m.updateViewportContent()
	}
}
