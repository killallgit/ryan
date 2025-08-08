package status

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/killallgit/ryan/pkg/tui/theme"
)

// StatusModel represents the status bar component
type StatusModel struct {
	spinner    spinner.Model
	status     string        // "Streaming", "Thinking", "Sending"
	timer      time.Duration // Elapsed time
	icon       string        // "↑" sending, "↓" receiving, "🔨" tool
	tokensSent int
	tokensRecv int
	startTime  time.Time
	isActive   bool
	width      int
}

// NewStatusModel creates a new status bar model
func NewStatusModel() StatusModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorViolet)

	return StatusModel{
		spinner:  s,
		status:   "",
		icon:     "",
		isActive: false,
	}
}
