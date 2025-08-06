// Package interfaces defines the core interfaces for the agent system.
// This package helps break circular dependencies by providing interface definitions
// that can be imported by all other agent packages.
package interfaces

import (
	"context"
	"sync"

	"github.com/killallgit/ryan/pkg/tools"
)

// Agent defines the interface for all agents in the system
type Agent interface {
	// Name returns the unique name of this agent
	Name() string

	// Description returns a human-readable description of what this agent does
	Description() string

	// CanHandle determines if this agent can handle the given request
	CanHandle(request string) (bool, float64) // returns can handle and confidence score (0-1)

	// Execute performs the agent's task
	Execute(ctx context.Context, request AgentRequest) (AgentResult, error)
}

// Orchestrator defines the interface for the orchestrator
// This allows components to depend on the interface rather than the concrete type
type Orchestrator interface {
	// RegisterAgent registers a new agent with the orchestrator
	RegisterAgent(agent Agent) error

	// GetAgent retrieves an agent by name
	GetAgent(name string) (Agent, error)

	// ListAgents returns all registered agents
	ListAgents() []Agent

	// Execute runs a request through the orchestrator
	Execute(ctx context.Context, prompt string, options map[string]interface{}) (AgentResult, error)

	// GetToolRegistry returns the tool registry
	GetToolRegistry() *tools.Registry

	// ExecuteWithPlan executes a pre-built execution plan
	ExecuteWithPlan(ctx context.Context, plan *ExecutionPlan, execContext *ExecutionContext) ([]TaskResult, error)
}

// Planner defines the interface for the planner
type Planner interface {
	// CreateExecutionPlan creates an execution plan for the given prompt
	CreateExecutionPlan(ctx context.Context, prompt string, execContext *ExecutionContext) (*ExecutionPlan, error)

	// OptimizePlan optimizes an execution plan
	OptimizePlan(plan *ExecutionPlan) (*ExecutionPlan, error)
}

// Executor defines the interface for the executor
type Executor interface {
	// ExecutePlan executes a plan
	ExecutePlan(ctx context.Context, plan *ExecutionPlan, execContext *ExecutionContext) (*TaskResult, error)

	// ExecuteTask executes a single task
	ExecuteTask(ctx context.Context, task *Task, execContext *ExecutionContext) (*TaskResult, error)
}

// ContextManager defines the interface for context management
type ContextManager interface {
	// CreateContext creates a new execution context
	CreateContext(sessionID, requestID string) *ExecutionContext

	// GetContext retrieves a context by ID
	GetContext(requestID string) (*ExecutionContext, bool)

	// UpdateContext updates an existing context
	UpdateContext(requestID string, update func(*ExecutionContext))
}

// FeedbackLoop defines the interface for feedback handling
type FeedbackLoop interface {
	// ProcessFeedback processes feedback for a task
	ProcessFeedback(feedback *FeedbackRequest) error

	// GetFeedbackHistory retrieves feedback history
	GetFeedbackHistory(taskID string) []FeedbackRequest
}

// AgentFactory defines the interface for agent creation
type AgentFactory interface {
	// CreateAgent creates an agent of the specified type
	CreateAgent(agentType string, config map[string]interface{}) (Agent, error)

	// RegisterCreator registers a new agent creator
	RegisterCreator(agentType string, creator AgentCreator)
}

// AgentCreator is a function that creates an agent
type AgentCreator func(config map[string]interface{}) (Agent, error)

// Copy shared types here to avoid circular dependencies
// These will be the canonical definitions

// AgentRequest represents a request to an agent
type AgentRequest struct {
	// Prompt is the user's request or task description
	Prompt string

	// Context provides additional context for the request
	Context map[string]interface{}

	// Options provides execution options
	Options map[string]interface{}

	// Files lists any files that should be considered
	Files []string

	// History provides conversation history if relevant
	History []Message
}

// AgentResult represents the result of an agent's execution
type AgentResult struct {
	// Success indicates whether the task completed successfully
	Success bool

	// Summary provides a brief summary of what was done
	Summary string

	// Details provides detailed information about the execution
	Details string

	// Data contains any structured data produced by the agent
	Data map[string]interface{}

	// SubTasks lists any subtasks that were executed
	SubTasks []SubTask

	// Artifacts contains any files or artifacts created
	Artifacts map[string]interface{}

	// Metadata provides execution metadata
	Metadata AgentMetadata
}

// AgentMetadata contains metadata about an agent's execution
type AgentMetadata struct {
	// AgentName is the name of the agent that executed
	AgentName string

	// AgentVersion is the version of the agent
	AgentVersion string

	// StartTime is when execution started
	StartTime interface{} // time.Time

	// EndTime is when execution ended
	EndTime interface{} // time.Time

	// Duration is how long execution took
	Duration interface{} // time.Duration

	// TokenUsage tracks token usage if applicable
	TokenUsage *TokenUsage

	// Error contains any error message
	Error string

	// Confidence is the agent's confidence in the result (0-1)
	Confidence float64

	// Tags provides additional categorization
	Tags []string
}

// TokenUsage tracks token usage for LLM-based agents
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// SubTask represents a subtask executed as part of a larger task
type SubTask struct {
	ID          string
	Name        string
	Description string
	AgentName   string
	Status      string
	Result      *AgentResult
	StartTime   interface{} // time.Time
	EndTime     interface{} // time.Time
	Duration    interface{} // time.Duration
}

// Message represents a conversation message
type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp interface{}            `json:"timestamp"` // time.Time
}

// ExecutionContext holds shared context for an execution
type ExecutionContext struct {
	SessionID    string
	RequestID    string
	UserPrompt   string
	SharedData   map[string]interface{}
	FileContext  []FileInfo
	Artifacts    map[string]interface{}
	Progress     chan ProgressUpdate
	Options      map[string]interface{}
	ParentTaskID string
	Depth        int

	// Mutex to protect concurrent access to SharedData and FileContext
	Mu sync.RWMutex
}

// FileInfo contains information about a file in the context
type FileInfo struct {
	Path         string
	Content      string
	Size         int64
	LastModified interface{} // time.Time
	Type         string
}

// ProgressUpdate represents a progress update
type ProgressUpdate struct {
	TaskID      string
	Stage       string
	Progress    float64 // 0-1
	Message     string
	Details     map[string]interface{}
	Timestamp   interface{} // time.Time
	IsCompleted bool
	Error       error
}

// Task represents a task to be executed
type Task struct {
	ID           string
	Name         string
	Description  string
	Agent        string // The agent to execute this task
	AgentName    string // Alias for Agent for backward compatibility
	Request      AgentRequest
	Priority     int
	Dependencies []string
	Input        map[string]interface{}
	Output       map[string]interface{}
}

// Stage represents a stage in an execution plan
type Stage struct {
	ID    string
	Name  string
	Tasks []Task
}

// ExecutionPlan represents a plan for executing a request
type ExecutionPlan struct {
	ID                string
	RequestID         string
	Tasks             []Task
	Stages            []Stage
	Graph             *ExecutionGraph
	Metadata          map[string]interface{}
	CreatedAt         interface{} // time.Time
	EstimatedDuration interface{} // time.Duration
	Confidence        float64
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID   string
	Task     Task
	Success  bool
	Result   interface{}
	Error    error
	Duration interface{} // time.Duration
}

// ExecutionGraph represents a dependency graph for execution
type ExecutionGraph struct {
	Nodes map[string]*GraphNode
	Edges map[string][]string // node ID -> dependent node IDs
}

// GraphNode represents a node in the execution graph
type GraphNode struct {
	ID           string
	Task         *Task
	Status       string
	Result       *TaskResult
	Dependencies []string
	Dependents   []string
}

// FeedbackRequest represents a request for feedback
type FeedbackRequest struct {
	TaskID      string
	RequestID   string
	Type        string // "approval", "correction", "clarification"
	Message     string
	Options     []string
	Context     map[string]interface{}
	Response    *FeedbackResponse
	RespondedAt interface{} // time.Time
}

// FeedbackResponse represents a response to feedback
type FeedbackResponse struct {
	Choice      string
	Message     string
	Corrections map[string]interface{}
}

// Intent represents the analyzed intent of a request
type Intent struct {
	Primary      string
	Secondary    []string
	Entities     map[string]interface{}
	Confidence   float64
	RequiresCode bool
	RequiresFile bool
	RequiresWeb  bool
}
