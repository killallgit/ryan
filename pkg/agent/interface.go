package agent

import "context"

// Agent defines the interface for interacting with the orchestrator
// This interface is used by both TUI and headless modes
type Agent interface {
	// Execute handles a request and returns a response (blocking)
	Execute(ctx context.Context, prompt string) (string, error)

	// ExecuteStream handles a request with streaming response
	ExecuteStream(ctx context.Context, prompt string, handler StreamHandler) error

	// ClearMemory clears the conversation memory
	ClearMemory() error

	// Close cleans up resources
	Close() error
}

// Ensure Orchestrator implements Agent interface
var _ Agent = (*Orchestrator)(nil)
