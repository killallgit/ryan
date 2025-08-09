package stream

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// StateChangeMsg is a Bubble Tea message for stream state changes
type StateChangeMsg struct {
	StreamID  string
	State     State
	Timestamp time.Time
	Error     error
}

// ChunkMsg is a Bubble Tea message for stream chunks
type ChunkMsg struct {
	StreamID string
	Content  string
}

// CompleteMsg is a Bubble Tea message for stream completion
type CompleteMsg struct {
	StreamID     string
	FinalContent string
	Timestamp    time.Time
}

// ErrorMsg is a Bubble Tea message for stream errors
type ErrorMsg struct {
	StreamID  string
	Error     error
	Timestamp time.Time
}

// MessageHandler converts stream events to Bubble Tea messages
type MessageHandler struct {
	streamID string
	program  *tea.Program
}

// NewMessageHandler creates a handler that sends Bubble Tea messages
func NewMessageHandler(streamID string, program *tea.Program) *MessageHandler {
	return &MessageHandler{
		streamID: streamID,
		program:  program,
	}
}

// OnChunk sends a chunk message
func (m *MessageHandler) OnChunk(chunk string) error {
	if m.program != nil {
		m.program.Send(ChunkMsg{
			StreamID: m.streamID,
			Content:  chunk,
		})
	}
	return nil
}

// OnComplete sends a completion message
func (m *MessageHandler) OnComplete(finalContent string) error {
	if m.program != nil {
		m.program.Send(CompleteMsg{
			StreamID:     m.streamID,
			FinalContent: finalContent,
			Timestamp:    time.Now(),
		})
		m.program.Send(StateChangeMsg{
			StreamID:  m.streamID,
			State:     StateComplete,
			Timestamp: time.Now(),
		})
	}
	return nil
}

// OnError sends an error message
func (m *MessageHandler) OnError(err error) {
	if m.program != nil {
		m.program.Send(ErrorMsg{
			StreamID:  m.streamID,
			Error:     err,
			Timestamp: time.Now(),
		})
		m.program.Send(StateChangeMsg{
			StreamID:  m.streamID,
			State:     StateError,
			Timestamp: time.Now(),
			Error:     err,
		})
	}
}

// Ensure MessageHandler implements Handler
var _ Handler = (*MessageHandler)(nil)
