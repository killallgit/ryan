package agents

import (
	"github.com/killallgit/ryan/pkg/agents/interfaces"
)

// Type aliases for backward compatibility
// These maintain the existing API while delegating to the interfaces package

type (
	// Core interfaces
	Agent                   = interfaces.Agent
	OrchestratorInterface   = interfaces.OrchestratorInterface
	PlannerInterface        = interfaces.PlannerInterface
	ExecutorInterface       = interfaces.ExecutorInterface
	ContextManagerInterface = interfaces.ContextManagerInterface
	FeedbackLoopInterface   = interfaces.FeedbackLoopInterface
	AgentFactoryInterface   = interfaces.AgentFactoryInterface
	AgentCreator            = interfaces.AgentCreator

	// Request/Response types
	AgentRequest  = interfaces.AgentRequest
	AgentResult   = interfaces.AgentResult
	AgentMetadata = interfaces.AgentMetadata
	TokenUsage    = interfaces.TokenUsage
	SubTask       = interfaces.SubTask
)

// Additional agent-specific types that aren't in interfaces

// AgentCapabilityInfo describes what an agent can do
type AgentCapabilityInfo struct {
	Name        string
	Description string
	Examples    []string
}

// ExecutionStatus represents the status of an execution
type ExecutionStatus string

const (
	StatusPending    ExecutionStatus = "pending"
	StatusInProgress ExecutionStatus = "in_progress"
	StatusCompleted  ExecutionStatus = "completed"
	StatusFailed     ExecutionStatus = "failed"
	StatusCancelled  ExecutionStatus = "cancelled"
)

// Progress represents progress information
type Progress struct {
	Current int
	Total   int
	Message string
}

// ProgressHandler is a function that handles progress updates
type ProgressHandler func(progress Progress)
