package chat

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/stream/tui"
	"github.com/killallgit/ryan/pkg/tui/chat/status"
	"github.com/killallgit/ryan/pkg/tui/theme"
)

// StreamChunk represents a chunk of streaming data
type StreamChunk struct {
	StreamID string
	Content  string
	IsEnd    bool
	Error    error
}

// chunkMsg wraps StreamChunk for Bubble Tea messaging
type chunkMsg StreamChunk

// waitForChunk waits for the next chunk from the streaming channel
func waitForChunk(chunkChan <-chan StreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk := <-chunkChan
		return chunkMsg(chunk)
	}
}

type chatModel struct {
	viewport      viewport.Model
	messages      []string
	textarea      textarea.Model
	err           error
	width         int
	height        int
	styles        *theme.StyleSet
	messageIndex  int
	numEscPress   int
	streamManager *tui.Manager
	chatManager   *chat.Manager
	agent         agent.Agent
	nodes         []MessageNode
	statusBar     status.StatusModel
	isStreaming   bool
	currentStream string

	// Channel-based streaming
	chunkChan chan StreamChunk
	stopChan  chan struct{}

	// Token tracking
	lastTokensSent int
	lastTokensRecv int
}

func NewChatModel(streamManager *tui.Manager, chatManager *chat.Manager, agent agent.Agent) chatModel {
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
		BorderForeground(lipgloss.Color(theme.ColorGrey))

	// Set prompt style
	ta.Prompt = "> "
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorGrey))

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
		chatManager:   chatManager,
		agent:         agent,
		nodes:         []MessageNode{},
		statusBar:     statusBar,
		isStreaming:   false,
		currentStream: "",

		// Initialize channels for streaming
		chunkChan: make(chan StreamChunk, 100), // Buffered channel for performance
		stopChan:  make(chan struct{}),

		// Initialize token tracking
		lastTokensSent: 0,
		lastTokensRecv: 0,
	}
}
