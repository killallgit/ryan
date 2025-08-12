package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/agent"
	chatpkg "github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/stream/tui"
	"github.com/killallgit/ryan/pkg/tui/chat"
)

// ChatView wraps the existing chat model to implement the View interface
type ChatView struct {
	model tea.Model
}

// NewChatView creates a new chat view
func NewChatView(streamManager *tui.Manager, chatManager *chatpkg.Manager, agent agent.Agent) ChatView {
	return ChatView{
		model: chat.NewChatModel(streamManager, chatManager, agent),
	}
}

// Init initializes the chat view
func (v ChatView) Init() tea.Cmd {
	return v.model.Init()
}

// Update handles messages for the chat view
func (v ChatView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := v.model.Update(msg)
	v.model = model
	return v, cmd
}

// View renders the chat view
func (v ChatView) View() string {
	return v.model.View()
}

// Name returns the display name for this view
func (v ChatView) Name() string {
	return "Chat"
}

// Description returns the description for this view
func (v ChatView) Description() string {
	return "Main chat interface"
}
