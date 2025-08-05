package agents

import (
	"context"
	"time"
)

// Agent represents a specialized agent that can handle specific types of tasks
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

// AgentRequest represents a request to an agent
type AgentRequest struct {
	// Prompt is the user's request
	Prompt string
	
	// Context provides additional context for the request
	Context map[string]interface{}
	
	// WorkingDirectory is the directory to operate in
	WorkingDirectory string
	
	// Options contains agent-specific options
	Options map[string]interface{}
}

// AgentResult represents the result of an agent's execution
type AgentResult struct {
	// Success indicates if the operation was successful
	Success bool
	
	// Summary provides a brief summary of what was done
	Summary string
	
	// Details contains the full details of the operation
	Details string
	
	// Artifacts contains any files or data produced
	Artifacts map[string]interface{}
	
	// Metadata contains execution metadata
	Metadata AgentMetadata
}

// AgentMetadata contains metadata about agent execution
type AgentMetadata struct {
	// AgentName is the name of the agent that executed
	AgentName string
	
	// StartTime is when execution began
	StartTime time.Time
	
	// EndTime is when execution completed
	EndTime time.Time
	
	// Duration is the total execution time
	Duration time.Duration
	
	// ToolsUsed lists all tools that were used
	ToolsUsed []string
	
	// FilesProcessed lists all files that were processed
	FilesProcessed []string
	
	// TokensUsed tracks token usage if applicable
	TokensUsed TokenUsage
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens    int
	ResponseTokens  int
	TotalTokens     int
}

// SubTask represents a sub-task that an agent delegates to another agent
type SubTask struct {
	// AgentName is the name of the agent to handle this task
	AgentName string
	
	// Request is the request for the sub-agent
	Request AgentRequest
	
	// Priority indicates the priority of this task
	Priority int
	
	// Dependencies lists other tasks this depends on
	Dependencies []string
}

// OrchestratorInterface defines the interface for agent orchestration
type OrchestratorInterface interface {
	// RegisterAgent registers a new agent
	RegisterAgent(agent Agent) error
	
	// Execute handles a user request by routing to appropriate agents
	Execute(ctx context.Context, request string, options map[string]interface{}) (AgentResult, error)
	
	// ExecuteWithAgent executes a request with a specific agent
	ExecuteWithAgent(ctx context.Context, agentName string, request AgentRequest) (AgentResult, error)
	
	// ListAgents returns all registered agents
	ListAgents() []Agent
}