package agents

import (
	"sync"

	"github.com/killallgit/ryan/pkg/agents/interfaces"
)

// Type aliases for protocol types
type (
	Message          = interfaces.Message
	ExecutionContext = interfaces.ExecutionContext
	FileInfo         = interfaces.FileInfo
	ProgressUpdate   = interfaces.ProgressUpdate
	Task             = interfaces.Task
	Stage            = interfaces.Stage
	ExecutionPlan    = interfaces.ExecutionPlan
	TaskResult       = interfaces.TaskResult
	ExecutionGraph   = interfaces.ExecutionGraph
	GraphNode        = interfaces.GraphNode
	FeedbackRequest  = interfaces.FeedbackRequest
	FeedbackResponse = interfaces.FeedbackResponse
)

// Priority defines message priority
type Priority int

const (
	PriorityLow    Priority = 0
	PriorityMedium Priority = 5
	PriorityHigh   Priority = 10
)

// Additional protocol types specific to implementation

// MessageBus handles inter-agent communication
type MessageBus struct {
	subscribers map[string][]chan Message
	mu          sync.RWMutex
}

// NewMessageBus creates a new message bus
func NewMessageBus() *MessageBus {
	return &MessageBus{
		subscribers: make(map[string][]chan Message),
	}
}

// Subscribe subscribes to messages for a topic
func (mb *MessageBus) Subscribe(topic string) <-chan Message {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	ch := make(chan Message, 100)
	mb.subscribers[topic] = append(mb.subscribers[topic], ch)
	return ch
}

// Publish publishes a message to a topic
func (mb *MessageBus) Publish(topic string, msg Message) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	for _, ch := range mb.subscribers[topic] {
		select {
		case ch <- msg:
		default:
			// Channel full, skip
		}
	}
}

// Protocol-specific constants
const (
	MessageTypeRequest  = "request"
	MessageTypeResponse = "response"
	MessageTypeStatus   = "status"
	MessageTypeError    = "error"
)

// Feedback type constants
const (
	FeedbackTypeApproval        = "approval"
	FeedbackTypeCorrection      = "correction"
	FeedbackTypeClarification   = "clarification"
	FeedbackTypeNeedMoreContext = "need_more_context"
	FeedbackTypeError           = "error"
	FeedbackTypeSuccess         = "success"
	FeedbackTypeValidationError = "validation_error"
	FeedbackTypeRetry           = "retry"
	FeedbackTypeRefine          = "refine"
)
