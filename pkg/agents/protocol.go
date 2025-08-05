package agents

import (
	"sync"
	"time"
)

// Message represents a message between agents
type Message struct {
	ID        string
	Type      MessageType
	Source    string
	Target    string
	Payload   interface{}
	Context   *ExecutionContext
	Priority  Priority
	Timestamp time.Time
}

// MessageType defines the type of message
type MessageType string

const (
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
	MessageTypeFeedback MessageType = "feedback"
	MessageTypeError    MessageType = "error"
	MessageTypeProgress MessageType = "progress"
)

// Priority defines message priority
type Priority int

const (
	PriorityLow    Priority = 0
	PriorityMedium Priority = 5
	PriorityHigh   Priority = 10
)

// ExecutionContext represents the shared execution context
type ExecutionContext struct {
	SessionID   string
	RequestID   string
	UserPrompt  string
	SharedData  map[string]interface{}
	FileContext []FileInfo
	Artifacts   map[string]interface{}
	Progress    chan<- ProgressUpdate
	Options     map[string]interface{}
	mu          sync.RWMutex
}

// FileInfo represents information about a file
type FileInfo struct {
	Path         string
	LastModified time.Time
	Size         int64
	Hash         string
	Content      string
}

// ProgressUpdate represents a progress update
type ProgressUpdate struct {
	TaskID        string
	Agent         string
	Status        string
	Operation     string        // e.g., "Bash(ls -al)", "Read(file.go)", "SpawnAgent(FileAgent)"
	OperationType OperationType // tool, agent_spawn, analysis, planning, execution
	ParentTaskID  string        // For nested operations
	Progress      float64
	Message       string
	Timestamp     time.Time
}

// Task represents a single task in an execution plan
type Task struct {
	ID           string
	Agent        string
	Request      AgentRequest
	Priority     int
	Dependencies []string
	Stage        string
	Timeout      time.Duration
}

// Stage represents a stage in the execution plan
type Stage struct {
	ID    string
	Tasks []string
}

// ExecutionPlan represents a complete execution plan
type ExecutionPlan struct {
	ID                string
	Context           *ExecutionContext
	Tasks             []Task
	Stages            []Stage
	EstimatedDuration string
	CreatedAt         time.Time
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	Task      Task
	Result    AgentResult
	Error     error
	StartTime time.Time
	EndTime   time.Time
}

// ExecutionGraph represents a graph of execution nodes
type ExecutionGraph struct {
	Nodes map[string]*GraphNode
	Edges map[string][]string
}

// GraphNode represents a node in the execution graph
type GraphNode struct {
	ID           string
	Agent        string
	AgentRef     Agent
	Request      AgentRequest
	Priority     int
	Dependencies []string
}

// FeedbackRequest represents a feedback request from an agent
type FeedbackRequest struct {
	ID         string
	SourceTask string
	TargetTask string
	Type       FeedbackType
	Content    interface{}
	Context    *ExecutionContext
}

// FeedbackType defines types of feedback
type FeedbackType string

const (
	FeedbackTypeNeedMoreContext FeedbackType = "need_more_context"
	FeedbackTypeValidationError FeedbackType = "validation_error"
	FeedbackTypeRetry           FeedbackType = "retry"
	FeedbackTypeRefine          FeedbackType = "refine"
)

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

// Subscribe subscribes to messages for an agent
func (mb *MessageBus) Subscribe(agentID string) <-chan Message {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	ch := make(chan Message, 100)
	mb.subscribers[agentID] = append(mb.subscribers[agentID], ch)
	return ch
}

// Unsubscribe removes a subscription
func (mb *MessageBus) Unsubscribe(agentID string, ch <-chan Message) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	channels := mb.subscribers[agentID]
	for i, c := range channels {
		if c == ch {
			mb.subscribers[agentID] = append(channels[:i], channels[i+1:]...)
			close(c)
			break
		}
	}
}

// Publish publishes a message to the bus
func (mb *MessageBus) Publish(msg Message) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	msg.Timestamp = time.Now()

	// Send to target agent's subscribers
	if channels, exists := mb.subscribers[msg.Target]; exists {
		for _, ch := range channels {
			select {
			case ch <- msg:
			default:
				// Channel full, skip
			}
		}
	}

	// Also send to wildcard subscribers
	if channels, exists := mb.subscribers["*"]; exists {
		for _, ch := range channels {
			select {
			case ch <- msg:
			default:
				// Channel full, skip
			}
		}
	}
}
