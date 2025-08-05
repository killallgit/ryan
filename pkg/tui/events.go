package tui

import (
	"sync"
)

// EventType represents the type of event
type EventType string

const (
	// View events
	EventViewSwitched        EventType = "view.switched"
	EventViewSwitchRequested EventType = "view.switch_requested"

	// Message events
	EventMessageSent     EventType = "message.sent"
	EventMessageReceived EventType = "message.received"
	EventMessageError    EventType = "message.error"

	// Streaming events
	EventStreamStarted       EventType = "stream.started"
	EventStreamChunkReceived EventType = "stream.chunk_received"
	EventStreamCompleted     EventType = "stream.completed"
	EventStreamError         EventType = "stream.error"

	// Model events
	EventModelChanged         EventType = "model.changed"
	EventModelValidationError EventType = "model.validation_error"

	// UI events
	EventUIUpdateRequested  EventType = "ui.update_requested"
	EventUIRefreshRequested EventType = "ui.refresh_requested"

	// Application events
	EventAppExit  EventType = "app.exit"
	EventAppError EventType = "app.error"
)

// Event represents an event in the system
type Event struct {
	Type      EventType
	Payload   interface{}
	Source    string // Component that generated the event
	Timestamp int64  // Unix timestamp
}

// EventHandler is a function that handles events
type EventHandler func(event Event)

// EventBus manages event publishing and subscription
type EventBus struct {
	subscribers map[EventType][]EventHandler
	mu          sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]EventHandler),
	}
}

// Subscribe adds an event handler for a specific event type
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.subscribers[eventType] == nil {
		eb.subscribers[eventType] = make([]EventHandler, 0)
	}

	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

// Unsubscribe removes an event handler for a specific event type
// Note: This is a simplified implementation. In practice, you'd want
// to use handler IDs or other identification methods
func (eb *EventBus) Unsubscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers := eb.subscribers[eventType]
	if handlers == nil {
		return
	}

	// This is a placeholder - Go doesn't allow function comparison
	// In practice, you'd use handler IDs or other identification
	_ = handler
	// eb.subscribers[eventType] = removeHandler(handlers, handler)
}

// Publish publishes an event to all subscribers
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	handlers := make([]EventHandler, 0)
	if eb.subscribers[event.Type] != nil {
		handlers = make([]EventHandler, len(eb.subscribers[event.Type]))
		copy(handlers, eb.subscribers[event.Type])
	}
	eb.mu.RUnlock()

	// Execute handlers in separate goroutines to avoid blocking
	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but don't crash the application
					// In practice, you'd use proper logging here
					_ = r
				}
			}()
			h(event)
		}(handler)
	}
}

// PublishSync publishes an event synchronously to all subscribers
func (eb *EventBus) PublishSync(event Event) {
	eb.mu.RLock()
	handlers := make([]EventHandler, 0)
	if eb.subscribers[event.Type] != nil {
		handlers = make([]EventHandler, len(eb.subscribers[event.Type]))
		copy(handlers, eb.subscribers[event.Type])
	}
	eb.mu.RUnlock()

	// Execute handlers synchronously
	for _, handler := range handlers {
		func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but don't crash the application
					_ = r
				}
			}()
			h(event)
		}(handler)
	}
}

// GetSubscriberCount returns the number of subscribers for an event type
func (eb *EventBus) GetSubscriberCount(eventType EventType) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if eb.subscribers[eventType] == nil {
		return 0
	}

	return len(eb.subscribers[eventType])
}

// Clear removes all subscribers
func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers = make(map[EventType][]EventHandler)
}

// Event payload types for different event types

// ViewSwitchPayload contains data for view switch events
type ViewSwitchPayload struct {
	FromView string
	ToView   string
}

// MessagePayload contains data for message events
type MessagePayload struct {
	Content   string
	Role      string
	MessageID string
	Error     error
}

// StreamPayload contains data for streaming events
type StreamPayload struct {
	StreamID string
	Content  string
	Complete bool
	Error    error
}

// ModelPayload contains data for model events
type ModelPayload struct {
	ModelName string
	OldModel  string
	Error     error
}

// UIUpdatePayload contains data for UI update events
type UIUpdatePayload struct {
	Component string
	Action    string
	Data      interface{}
}

// ErrorPayload contains error information
type ErrorPayload struct {
	Error   error
	Message string
	Code    string
}

// Helper functions to create common events

// NewViewSwitchEvent creates a new view switch event
func NewViewSwitchEvent(source, fromView, toView string) Event {
	return Event{
		Type:   EventViewSwitched,
		Source: source,
		Payload: ViewSwitchPayload{
			FromView: fromView,
			ToView:   toView,
		},
	}
}

// NewMessageEvent creates a new message event
func NewMessageEvent(eventType EventType, source, content, role, messageID string, err error) Event {
	return Event{
		Type:   eventType,
		Source: source,
		Payload: MessagePayload{
			Content:   content,
			Role:      role,
			MessageID: messageID,
			Error:     err,
		},
	}
}

// NewStreamEvent creates a new stream event
func NewStreamEvent(eventType EventType, source, streamID, content string, complete bool, err error) Event {
	return Event{
		Type:   eventType,
		Source: source,
		Payload: StreamPayload{
			StreamID: streamID,
			Content:  content,
			Complete: complete,
			Error:    err,
		},
	}
}

// NewModelEvent creates a new model event
func NewModelEvent(source, modelName, oldModel string, err error) Event {
	eventType := EventModelChanged
	if err != nil {
		eventType = EventModelValidationError
	}

	return Event{
		Type:   eventType,
		Source: source,
		Payload: ModelPayload{
			ModelName: modelName,
			OldModel:  oldModel,
			Error:     err,
		},
	}
}

// NewUIUpdateEvent creates a new UI update event
func NewUIUpdateEvent(source, component, action string, data interface{}) Event {
	return Event{
		Type:   EventUIUpdateRequested,
		Source: source,
		Payload: UIUpdatePayload{
			Component: component,
			Action:    action,
			Data:      data,
		},
	}
}

// NewErrorEvent creates a new error event
func NewErrorEvent(source, message, code string, err error) Event {
	return Event{
		Type:   EventAppError,
		Source: source,
		Payload: ErrorPayload{
			Error:   err,
			Message: message,
			Code:    code,
		},
	}
}
