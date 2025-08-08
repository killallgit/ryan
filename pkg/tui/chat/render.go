package chat

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m chatModel) renderNodes() string {
	var rendered []string

	// Calculate available width for wrapping
	availableWidth := m.viewport.Width
	if availableWidth <= 0 {
		availableWidth = 80 // Default fallback
	}

	for _, node := range m.nodes {
		var nodeContent string
		var style lipgloss.Style

		// Select appropriate style based on node type
		switch node.Type {
		case "system":
			style = m.styles.SystemMessage
		case "user":
			style = m.styles.UserMessage
		case "assistant":
			style = m.styles.AssistantMessage
		case "agent":
			// Add agent style if not exists, use assistant style for now
			style = m.styles.AssistantMessage
		case "tool":
			// Add tool style if not exists, use info style for now
			style = m.styles.InfoMessage
		case "error":
			style = m.styles.ErrorMessage
		default:
			style = m.styles.DefaultMessage
		}

		// Apply width constraint for word wrapping and add top padding
		style = style.Width(availableWidth).PaddingTop(1)
		nodeContent = style.Render(node.Content)

		rendered = append(rendered, nodeContent)
	}

	return strings.Join(rendered, "\n\n")
}

func (m *chatModel) updateViewportContent() {
	content := m.renderNodes()
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}
