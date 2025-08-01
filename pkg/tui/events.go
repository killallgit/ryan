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

// ViewChangeEvent is sent when a view change is requested
type ViewChangeEvent struct {
	tcell.EventTime
	ViewName string
}

// MenuToggleEvent is sent when the menu should be toggled
type MenuToggleEvent struct {
	tcell.EventTime
	Show bool
}

// ModelListUpdateEvent is sent when model list data is updated
type ModelListUpdateEvent struct {
	tcell.EventTime
	Models []ModelInfo
}

// ModelStatsUpdateEvent is sent when model statistics are updated
type ModelStatsUpdateEvent struct {
	tcell.EventTime
	Stats ModelStats
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

// NewViewChangeEvent creates a new view change event
func NewViewChangeEvent(viewName string) *ViewChangeEvent {
	return &ViewChangeEvent{
		EventTime: tcell.EventTime{},
		ViewName:  viewName,
	}
}

// NewMenuToggleEvent creates a new menu toggle event
func NewMenuToggleEvent(show bool) *MenuToggleEvent {
	return &MenuToggleEvent{
		EventTime: tcell.EventTime{},
		Show:      show,
	}
}

// NewModelListUpdateEvent creates a new model list update event
func NewModelListUpdateEvent(models []ModelInfo) *ModelListUpdateEvent {
	return &ModelListUpdateEvent{
		EventTime: tcell.EventTime{},
		Models:    models,
	}
}

// ModelErrorEvent is sent when model operations fail
type ModelErrorEvent struct {
	tcell.EventTime
	Error error
}

// NewModelStatsUpdateEvent creates a new model stats update event
func NewModelStatsUpdateEvent(stats ModelStats) *ModelStatsUpdateEvent {
	return &ModelStatsUpdateEvent{
		EventTime: tcell.EventTime{},
		Stats:     stats,
	}
}

// NewModelErrorEvent creates a new model error event
func NewModelErrorEvent(err error) *ModelErrorEvent {
	return &ModelErrorEvent{
		EventTime: tcell.EventTime{},
		Error:     err,
	}
}

// ChatMessageSendEvent is sent when a chat message should be sent
type ChatMessageSendEvent struct {
	tcell.EventTime
	Content string
}

// NewChatMessageSendEvent creates a new chat message send event
func NewChatMessageSendEvent(content string) *ChatMessageSendEvent {
	return &ChatMessageSendEvent{
		EventTime: tcell.EventTime{},
		Content:   content,
	}
}

// SpinnerAnimationEvent is sent to update spinner animation frames
type SpinnerAnimationEvent struct {
	tcell.EventTime
}

// NewSpinnerAnimationEvent creates a new spinner animation event
func NewSpinnerAnimationEvent() *SpinnerAnimationEvent {
	return &SpinnerAnimationEvent{
		EventTime: tcell.EventTime{},
	}
}