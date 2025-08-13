package agent

import (
	"context"

	"github.com/killallgit/ryan/pkg/stream/core"
)

// Agent defines the interface for the ReAct agent
// This interface is used by both TUI and headless modes
type Agent interface {
	// Execute handles a request and returns a response (blocking)
	Execute(ctx context.Context, prompt string) (string, error)

	// ExecuteStream handles a request with streaming response
	ExecuteStream(ctx context.Context, prompt string, handler core.Handler) error

	// ClearMemory clears the conversation memory
	ClearMemory() error

	// GetTokenStats returns the cumulative token usage statistics
	// Returns (tokensSent, tokensReceived)
	GetTokenStats() (int, int)

	// GetExecutionState returns a snapshot of the current execution state
	GetExecutionState() ExecutionStateSnapshot

	// Close cleans up resources
	Close() error
}

// Compile-time check that ReactAgent implements Agent interface
var _ Agent = (*ReactAgent)(nil)
