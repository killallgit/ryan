package agents

import (
	"sync"
	"time"

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

// OperationType represents different types of operations for progress tracking
type OperationType string

const (
	OperationTypeTool      OperationType = "tool"
	OperationTypeAgent     OperationType = "agent_spawn"
	OperationTypeAnalysis  OperationType = "analysis"
	OperationTypePlanning  OperationType = "planning"
	OperationTypeExecution OperationType = "execution"
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

// Hierarchical planning types

// Project represents a hierarchical project
type Project struct {
	ID          string
	Name        string
	Description string
	Context     *ProjectContext
	Status      ProjectStatus
	Epics       []*Epic
	Sprints     []*Sprint
	CreatedAt   time.Time
}

// ProjectContext provides context for project planning
type ProjectContext struct {
	ProjectPath   string
	Technologies  []string
	Constraints   map[string]interface{}
	TeamCapacity  int
	SprintLength  int
	Requirements  []string
}

// ProjectStatus represents project status
type ProjectStatus string

const (
	ProjectStatusPlanning    ProjectStatus = "planning"
	ProjectStatusInProgress  ProjectStatus = "in_progress"
	ProjectStatusCompleted   ProjectStatus = "completed"
	ProjectStatusOnHold      ProjectStatus = "on_hold"
	ProjectStatusCancelled   ProjectStatus = "cancelled"
)

// Epic represents a major feature or component
type Epic struct {
	ID          string
	Title       string
	Description string
	Priority    Priority
	Status      EpicStatus
	Stories     []*UserStory
}

// EpicStatus represents epic status
type EpicStatus string

const (
	EpicStatusTodo        EpicStatus = "todo"
	EpicStatusInProgress  EpicStatus = "in_progress"
	EpicStatusDone        EpicStatus = "done"
	EpicStatusBlocked     EpicStatus = "blocked"
)

// UserStory represents a user story
type UserStory struct {
	ID          string
	Title       string
	Description string
	Points      int
	Priority    Priority
	Status      StoryStatus
	Tasks       []Task
}

// StoryStatus represents story status
type StoryStatus string

const (
	StoryStatusTodo        StoryStatus = "todo"
	StoryStatusInProgress  StoryStatus = "in_progress"
	StoryStatusDone        StoryStatus = "done"
	StoryStatusBlocked     StoryStatus = "blocked"
)

// Sprint represents a development sprint
type Sprint struct {
	ID        string
	Number    int
	Name      string
	Goal      string
	StartDate time.Time
	EndDate   time.Time
	Status    SprintStatus
	Stories   []*UserStory
	Plans     []*ExecutionPlan
	Capacity  int
}

// SprintStatus represents sprint status
type SprintStatus string

const (
	SprintStatusPlanning    SprintStatus = "planning"
	SprintStatusActive      SprintStatus = "active"
	SprintStatusCompleted   SprintStatus = "completed"
	SprintStatusCancelled   SprintStatus = "cancelled"
)
