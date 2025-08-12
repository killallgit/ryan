package agent

import (
	"context"

	"github.com/killallgit/ryan/pkg/stream/core"
)

// Agent defines the interface for interacting with agents
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

	// Close cleans up resources
	Close() error
}

// ModeAgent extends Agent with operating mode support
type ModeAgent interface {
	Agent

	// GetMode returns the current operating mode
	GetMode() OperatingMode

	// SetMode changes the operating mode
	SetMode(mode OperatingMode) error
}

// Ensure implementations satisfy interfaces
var _ Agent = (*ExecutorAgent)(nil)
var _ ModeAgent = (*MRKLAgent)(nil)
