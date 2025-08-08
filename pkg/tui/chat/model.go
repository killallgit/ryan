package chat

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/killallgit/ryan/pkg/tui/theme"
)

type chatModel struct {
	viewport    viewport.Model
	messages    []string
	textarea    textarea.Model
	senderStyle lipgloss.Style
	err         error
	width       int
	height      int
	styles      *theme.Styles
}

func NewChatModel() chatModel {
	ta := textarea.New()
	ta.Focus()
	ta.Placeholder = "Type a message..."
	ta.CharLimit = 0
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	vp := viewport.New(80, 20)
	ta.KeyMap.InsertNewline.SetEnabled(true)

	styles := theme.DefaultStyles()
	return chatModel{
		textarea:    ta,
		messages:    []string{},
		viewport:    vp,
		senderStyle: styles.UserMessage,
		styles:      styles,
		err:         nil,
	}
}
