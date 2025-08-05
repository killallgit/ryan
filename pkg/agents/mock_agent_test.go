package agents

import (
	"context"
	"sync"
)

// mockAgent is a mock implementation of the Agent interface for testing
type mockAgent struct {
	name        string
	description string
	canHandle   func(string) (bool, float64)
	execute     func(context.Context, AgentRequest) (AgentResult, error)
	mu          sync.Mutex
	calls       []string
}

// newMockAgent creates a new mock agent
func newMockAgent(name, description string) *mockAgent {
	return &mockAgent{
		name:        name,
		description: description,
		calls:       make([]string, 0),
		canHandle: func(request string) (bool, float64) {
			return true, 1.0
		},
		execute: func(ctx context.Context, req AgentRequest) (AgentResult, error) {
			return AgentResult{
				Success: true,
				Summary: "Mock execution successful",
				Details: "Mock agent executed successfully",
			}, nil
		},
	}
}

// Name returns the agent's name
func (m *mockAgent) Name() string {
	return m.name
}

// Description returns the agent's description
func (m *mockAgent) Description() string {
	return m.description
}

// CanHandle determines if this agent can handle the request
func (m *mockAgent) CanHandle(request string) (bool, float64) {
	m.mu.Lock()
	m.calls = append(m.calls, "CanHandle:"+request)
	m.mu.Unlock()
	return m.canHandle(request)
}

// Execute performs the agent's task
func (m *mockAgent) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	m.mu.Lock()
	m.calls = append(m.calls, "Execute:"+request.Prompt)
	m.mu.Unlock()
	return m.execute(ctx, request)
}

// SetCanHandle sets the canHandle function
func (m *mockAgent) SetCanHandle(fn func(string) (bool, float64)) {
	m.canHandle = fn
}

// SetExecute sets the execute function
func (m *mockAgent) SetExecute(fn func(context.Context, AgentRequest) (AgentResult, error)) {
	m.execute = fn
}

// GetCalls returns the list of method calls made to this mock
func (m *mockAgent) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]string, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// ResetCalls clears the call history
func (m *mockAgent) ResetCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = make([]string, 0)
}
