package chat

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/killallgit/ryan/pkg/streaming"
	"github.com/killallgit/ryan/pkg/tui/chat/status"
	"github.com/killallgit/ryan/pkg/tui/theme"
)

type chatModel struct {
	viewport      viewport.Model
	messages      []string
	textarea      textarea.Model
	err           error
	width         int
	height        int
	styles        *theme.Styles
	messageIndex  int
	numEscPress   int
	streamManager *streaming.Manager
	nodes         []MessageNode
	statusBar     status.StatusModel
	isStreaming   bool
	currentStream string
}

func NewChatModel(streamManager *streaming.Manager) chatModel {
	ta := textarea.New()
	ta.Focus()
	ta.Placeholder = "Type a message..."
	ta.SetHeight(1)
	ta.SetWidth(30)
	ta.MaxHeight = 10 // Set maximum height for auto-expansion
	ta.ShowLineNumbers = false
	ta.CharLimit = 0 // No character limit

	// Style the textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.ColorOrange))

	// Set prompt style
	ta.Prompt = "> "
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorOrange))

	vp := viewport.New(80, 30)

	ta.KeyMap.InsertNewline.SetEnabled(true)

	// Initialize status bar
	statusBar := status.NewStatusModel()

	return chatModel{
		textarea:      ta,
		messages:      []string{},
		messageIndex:  -1,
		viewport:      vp,
		numEscPress:   0,
		err:           nil,
		styles:        theme.DefaultStyles(),
		streamManager: streamManager,
		nodes:         []MessageNode{},
		statusBar:     statusBar,
		isStreaming:   false,
		currentStream: "",
	}
}
