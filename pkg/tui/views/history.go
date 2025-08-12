package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/chat"
)

// HistoryView displays chat history
type HistoryView struct {
	width       int
	height      int
	chatManager *chat.Manager
	history     *chat.History
	err         error
}

// NewHistoryView creates a new history view
func NewHistoryView(chatManager *chat.Manager) HistoryView {
	v := HistoryView{
		chatManager: chatManager,
	}
	// Load history on creation
	if chatManager != nil {
		messages := chatManager.GetHistory()
		v.history = &chat.History{Messages: messages}
	}
	return v
}

// Init initializes the history view
func (v HistoryView) Init() tea.Cmd {
	return nil
}

// Update handles messages for the history view
func (v HistoryView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			// Refresh history
			if v.chatManager != nil {
				messages := v.chatManager.GetHistory()
				v.history = &chat.History{Messages: messages}
				v.err = nil
			}
		}
	}
	return v, nil
}

// View renders the history view
func (v HistoryView) View() string {
	var b strings.Builder

	b.WriteString("=== Chat History ===\n\n")

	if v.err != nil {
		b.WriteString(fmt.Sprintf("Error loading history: %v\n", v.err))
		return b.String()
	}

	if v.history == nil || len(v.history.Messages) == 0 {
		b.WriteString("No chat history available.\n")
	} else {
		b.WriteString(fmt.Sprintf("Total messages: %d\n\n", len(v.history.Messages)))

		// Show last 10 messages
		start := len(v.history.Messages) - 10
		if start < 0 {
			start = 0
		}

		for i := start; i < len(v.history.Messages); i++ {
			msg := v.history.Messages[i]
			role := "Unknown"
			switch msg.Role {
			case chat.RoleUser:
				role = "User"
			case chat.RoleAssistant:
				role = "Assistant"
			case chat.RoleSystem:
				role = "System"
			}

			// Truncate long messages
			content := msg.Content
			if len(content) > 80 {
				content = content[:77] + "..."
			}

			b.WriteString(fmt.Sprintf("[%s] %s\n", role, content))
		}
	}

	b.WriteString("\n")
	b.WriteString("Press 'r' to refresh\n")
	b.WriteString("Press Ctrl+P to switch views\n")
	b.WriteString("Press q to quit\n")

	return b.String()
}

// Name returns the display name for this view
func (v HistoryView) Name() string {
	return "History"
}

// Description returns the description for this view
func (v HistoryView) Description() string {
	return "Browse chat history"
}
