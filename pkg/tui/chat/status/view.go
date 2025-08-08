package status

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/killallgit/ryan/pkg/tui/theme"
)

func (m StatusModel) View() string {
	// Hide the entire status bar when not active
	if !m.isActive || m.width == 0 {
		return ""
	}

	// Build status components
	var components []string

	// Spinner (always show when active)
	components = append(components, m.spinner.View())

	// Status text (only show if set)
	if m.status != "" {
		statusStyle := lipgloss.NewStyle().Foreground(theme.ColorBase05)
		components = append(components, statusStyle.Render(m.status))
	}

	// Timer (if active)
	if m.isActive && m.timer > 0 {
		minutes := int(m.timer.Minutes())
		seconds := int(m.timer.Seconds()) % 60
		timerText := fmt.Sprintf("%02d:%02d", minutes, seconds)
		timerStyle := lipgloss.NewStyle().Foreground(theme.ColorBase04)
		components = append(components, timerStyle.Render(timerText))
	}

	// Icon (if set)
	if m.icon != "" {
		iconStyle := lipgloss.NewStyle().Foreground(theme.ColorOrange)
		components = append(components, iconStyle.Render(m.icon))
	}

	// Token counter
	totalTokens := m.tokensSent + m.tokensRecv
	if totalTokens > 0 {
		tokenText := fmt.Sprintf("%d tokens", totalTokens)
		tokenStyle := lipgloss.NewStyle().Foreground(theme.ColorBase04)
		components = append(components, tokenStyle.Render(tokenText))
	}

	// Join components with separator
	separator := lipgloss.NewStyle().Foreground(theme.ColorBase03).Render(" | ")
	statusLine := ""
	for i, component := range components {
		if i > 0 {
			statusLine += separator
		}
		statusLine += component
	}

	// Create full-width status bar with background
	statusBar := lipgloss.NewStyle().
		Width(m.width).
		Background(theme.ColorBase01).
		Padding(0, 1).
		Render(statusLine)

	return statusBar
}
