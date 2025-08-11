package orchestrator

import (
	"encoding/json"
	"time"
)

// AgentType represents the type of specialized agent
type AgentType string

const (
	AgentOrchestrator AgentType = "orchestrator" // Main coordinator
	AgentToolCaller   AgentType = "tool_caller"  // Tool/function calling specialist
	AgentCodeGen      AgentType = "code_gen"     // Code generation specialist
	AgentReasoner     AgentType = "reasoner"     // Complex reasoning chains
	AgentSearcher     AgentType = "searcher"     // Code search and analysis
	AgentPlanner      AgentType = "planner"      // Task planning and decomposition
)

// Status represents the current status of a task
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

// Phase represents the current phase of task execution
type Phase string

const (
	PhaseAnalysis  Phase = "analysis"
	PhaseRouting   Phase = "routing"
	PhaseExecution Phase = "execution"
	PhaseFeedback  Phase = "feedback"
	PhaseComplete  Phase = "complete"
)

// OutputFormat specifies the expected output format
type OutputFormat string

const (
	OutputFormatText OutputFormat = "text"
	OutputFormatJSON OutputFormat = "json"
	OutputFormatCode OutputFormat = "code"
	OutputFormatList OutputFormat = "list"
)

// Action represents the next action to take
type Action string

const (
	ActionContinue Action = "continue"
	ActionComplete Action = "complete"
	ActionRetry    Action = "retry"
	ActionFail     Action = "fail"
)

// TaskState maintains the state of a task throughout execution
type TaskState struct {
	ID           string                 `json:"id"`
	Query        string                 `json:"query"`
	Intent       *TaskIntent            `json:"intent"`
	CurrentPhase Phase                  `json:"current_phase"`
	History      []AgentResponse        `json:"history"`
	Status       Status                 `json:"status"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// TaskIntent represents the analyzed intent of a task
type TaskIntent struct {
	Type                 string   `json:"type"`
	Confidence           float64  `json:"confidence"`
	RequiredCapabilities []string `json:"required_capabilities"`
}

// AgentResponse represents a response from an agent
type AgentResponse struct {
	AgentType   AgentType      `json:"agent_type"`
	Response    string         `json:"response"`
	ToolsCalled []ToolCall     `json:"tools_called,omitempty"`
	Status      string         `json:"status"`
	NextAction  *RouteDecision `json:"next_action,omitempty"`
	Error       *string        `json:"error,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    string                 `json:"result"`
}

// RouteDecision represents a routing decision made by the orchestrator
type RouteDecision struct {
	TargetAgent    AgentType              `json:"target_agent"`
	Instruction    string                 `json:"instruction"`
	Tools          []string               `json:"tools,omitempty"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
	ExpectedOutput OutputFormat           `json:"expected_output"`
}

// NextStep represents the next step in task execution
type NextStep struct {
	Action   Action         `json:"action"`
	Decision *RouteDecision `json:"decision,omitempty"`
}

// TaskResult represents the final result of task execution
type TaskResult struct {
	ID        string                 `json:"id"`
	Query     string                 `json:"query"`
	Result    string                 `json:"result"`
	Status    Status                 `json:"status"`
	History   []AgentResponse        `json:"history"`
	Metadata  map[string]interface{} `json:"metadata"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration"`
}

// OrchestratorCommand represents a command to the orchestrator
type OrchestratorCommand struct {
	Type    string          `json:"type"` // "route", "execute", "feedback"
	Payload json.RawMessage `json:"payload"`
	Context *TaskContext    `json:"context,omitempty"`
}

// TaskContext provides context for task execution
type TaskContext struct {
	SessionID   string                 `json:"session_id"`
	UserID      string                 `json:"user_id,omitempty"`
	Environment map[string]string      `json:"environment,omitempty"`
	Preferences map[string]interface{} `json:"preferences,omitempty"`
}

// FeedbackMessage represents feedback from an agent
type FeedbackMessage struct {
	FromAgent    AgentType   `json:"from_agent"`
	Status       string      `json:"status"` // "success", "partial", "failed", "needs_input"
	Result       interface{} `json:"result"`
	NextRequired *string     `json:"next_required,omitempty"`
	Error        *string     `json:"error,omitempty"`
}
