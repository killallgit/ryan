package chat

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/killallgit/ryan/pkg/tui/theme"
)

type chatModel struct {
	viewport     viewport.Model
	messages     []string
	textarea     textarea.Model
	err          error
	width        int
	height       int
	styles       *theme.Styles
	messageIndex int
	numEscPress  int
}

func NewChatModel() chatModel {
	ta := textarea.New()
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorOrange))
	ta.FocusedStyle.Prompt.Render("> ")
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorOrange))
	ta.Focus()
	ta.Placeholder = "Type a message..."
	ta.SetHeight(1)
	ta.SetWidth(30)
	ta.MaxHeight = 10 // Set maximum height for auto-expansion
	ta.ShowLineNumbers = false
	ta.CharLimit = 0 // No character limit

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	vp := createViewport(80, 20)

	ta.KeyMap.InsertNewline.SetEnabled(true)

	return chatModel{
		textarea:     ta,
		messages:     []string{},
		messageIndex: -1,
		viewport:     vp,
		numEscPress:  0,
		err:          nil,
		styles:       theme.DefaultStyles(),
	}
}
