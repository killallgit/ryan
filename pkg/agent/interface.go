package agent

import (
	"context"

	"github.com/killallgit/ryan/pkg/stream"
)

// Agent defines the interface for interacting with agents
// This interface is used by both TUI and headless modes
type Agent interface {
	// Execute handles a request and returns a response (blocking)
	Execute(ctx context.Context, prompt string) (string, error)

	// ExecuteStream handles a request with streaming response
	ExecuteStream(ctx context.Context, prompt string, handler stream.Handler) error

	// ClearMemory clears the conversation memory
	ClearMemory() error

	// Close cleans up resources
	Close() error
}

// Ensure ExecutorAgent implements Agent interface
var _ Agent = (*ExecutorAgent)(nil)
