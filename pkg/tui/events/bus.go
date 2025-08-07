package events

import (
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// Event represents a generic event in the system
type Event struct {
	Type      string
	Payload   interface{}
	Source    string
	Timestamp time.Time
}

// Handler is a function that handles events
type Handler func(event Event)

// EventBus provides decoupled communication between components
type EventBus struct {
	handlers map[string][]Handler
	mutex    sync.RWMutex
	log      *logger.Logger
	buffer   chan Event
	done     chan struct{}
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	bus := &EventBus{
		handlers: make(map[string][]Handler),
		log:      logger.WithComponent("event_bus"),
		buffer:   make(chan Event, 100), // Buffered channel for async processing
		done:     make(chan struct{}),
	}

	// Start event processing goroutine
	go bus.processEvents()

	return bus
}

// Subscribe adds a handler for a specific event type
func (eb *EventBus) Subscribe(eventType string, handler Handler) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	eb.log.Debug("Handler subscribed", "eventType", eventType)
}

// Unsubscribe removes a handler for a specific event type
func (eb *EventBus) Unsubscribe(eventType string, handler Handler) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	handlers := eb.handlers[eventType]
	if len(handlers) > 0 {
		// Note: Function comparison in Go is limited, so we'll remove all handlers
		// for this event type. In practice, components should manage their subscriptions carefully.
		eb.handlers[eventType] = nil
		eb.log.Debug("Handler unsubscribed", "eventType", eventType)
	}
}

// Publish sends an event to all registered handlers
func (eb *EventBus) Publish(eventType string, payload interface{}, source string) {
	event := Event{
		Type:      eventType,
		Payload:   payload,
		Source:    source,
		Timestamp: time.Now(),
	}

	// Non-blocking send to buffer
	select {
	case eb.buffer <- event:
		eb.log.Debug("Event published", "type", eventType, "source", source)
	default:
		eb.log.Warn("Event buffer full, dropping event", "type", eventType, "source", source)
	}
}

// PublishSync sends an event synchronously to all handlers
func (eb *EventBus) PublishSync(eventType string, payload interface{}, source string) {
	event := Event{
		Type:      eventType,
		Payload:   payload,
		Source:    source,
		Timestamp: time.Now(),
	}

	eb.deliverEvent(event)
}

// processEvents runs in a goroutine to process events asynchronously
func (eb *EventBus) processEvents() {
	for {
		select {
		case event := <-eb.buffer:
			eb.deliverEvent(event)
		case <-eb.done:
			return
		}
	}
}

// deliverEvent delivers an event to all registered handlers
func (eb *EventBus) deliverEvent(event Event) {
	eb.mutex.RLock()
	handlers := eb.handlers[event.Type]
	globalHandlers := eb.handlers["*"] // Global handlers that receive all events
	eb.mutex.RUnlock()

	// Deliver to specific handlers
	for _, handler := range handlers {
		go func(h Handler) {
			defer func() {
				if r := recover(); r != nil {
					eb.log.Error("Event handler panicked", "type", event.Type, "error", r)
				}
			}()
			h(event)
		}(handler)
	}

	// Deliver to global handlers
	for _, handler := range globalHandlers {
		go func(h Handler) {
			defer func() {
				if r := recover(); r != nil {
					eb.log.Error("Global event handler panicked", "type", event.Type, "error", r)
				}
			}()
			h(event)
		}(handler)
	}
}

// Close shuts down the event bus
func (eb *EventBus) Close() {
	close(eb.done)
}

// Event type constants
const (
	// View events
	EventViewChanged   = "view_changed"
	EventViewSwitching = "view_switching"

	// Model events
	EventModelSelected   = "model_selected"
	EventModelValidating = "model_validating"
	EventModelValidated  = "model_validated"
	EventModelError      = "model_error"
	EventModelsRefresh   = "models_refresh"
	EventModelsRefreshed = "models_refreshed"

	// Chat events
	EventMessageSent     = "message_sent"
	EventMessageReceived = "message_received"
	EventStreamStarted   = "stream_started"
	EventStreamChunk     = "stream_chunk"
	EventStreamEnded     = "stream_ended"
	EventStreamError     = "stream_error"

	// UI state events
	EventSendingStarted   = "sending_started"
	EventSendingEnded     = "sending_ended"
	EventThinkingStarted  = "thinking_started"
	EventThinkingEnded    = "thinking_ended"
	EventExecutingStarted = "executing_started"
	EventExecutingEnded   = "executing_ended"

	// System events
	EventError            = "error"
	EventConnected        = "connected"
	EventDisconnected     = "disconnected"
	EventAsyncOpStarted   = "async_op_started"
	EventAsyncOpCompleted = "async_op_completed"
	EventAsyncOpFailed    = "async_op_failed"
)

// Event payload structures

type ViewChangedPayload struct {
	OldView string
	NewView string
}

type ModelSelectedPayload struct {
	ModelName string
}

type ModelValidatingPayload struct {
	ModelName string
}

type ModelValidatedPayload struct {
	ModelName string
	Valid     bool
	Error     string
}

type MessagePayload struct {
	Content string
	Role    string
}

type StreamPayload struct {
	StreamID string
	Content  string
}

type ErrorPayload struct {
	Error   string
	Source  string
	Context map[string]interface{}
}

type AsyncOpPayload struct {
	OperationID string
	Type        string
	Data        interface{}
}
