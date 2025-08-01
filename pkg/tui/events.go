package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
)

// Custom event types for non-blocking API communication

// MessageResponseEvent is sent when an API call succeeds
type MessageResponseEvent struct {
	tcell.EventTime
	Message chat.Message
}

// MessageErrorEvent is sent when an API call fails
type MessageErrorEvent struct {
	tcell.EventTime
	Error error
}

// NewMessageResponseEvent creates a new message response event
func NewMessageResponseEvent(message chat.Message) *MessageResponseEvent {
	return &MessageResponseEvent{
		EventTime: tcell.EventTime{},
		Message:   message,
	}
}

// NewMessageErrorEvent creates a new message error event
func NewMessageErrorEvent(err error) *MessageErrorEvent {
	return &MessageErrorEvent{
		EventTime: tcell.EventTime{},
		Error:     err,
	}
}