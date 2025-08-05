package mocks

import (
	"context"
	"sync"

	"github.com/killallgit/ryan/pkg/agents"
)

// MockAgent is a mock implementation of the Agent interface
type MockAgent struct {
	name        string
	description string
	canHandle   func(string) (bool, float64)
	execute     func(context.Context, agents.AgentRequest) (agents.AgentResult, error)
	mu          sync.Mutex
	calls       []string
}

// NewMockAgent creates a new mock agent
func NewMockAgent(name, description string) *MockAgent {
	return &MockAgent{
		name:        name,
		description: description,
		calls:       make([]string, 0),
		canHandle: func(request string) (bool, float64) {
			return true, 1.0
		},
		execute: func(ctx context.Context, req agents.AgentRequest) (agents.AgentResult, error) {
			return agents.AgentResult{
				Success: true,
				Summary: "Mock execution successful",
				Details: "Mock details",
			}, nil
		},
	}
}

// Name returns the agent's name
func (m *MockAgent) Name() string {
	return m.name
}

// Description returns the agent's description
func (m *MockAgent) Description() string {
	return m.description
}

// CanHandle determines if this agent can handle the request
func (m *MockAgent) CanHandle(request string) (bool, float64) {
	m.mu.Lock()
	m.calls = append(m.calls, "CanHandle:"+request)
	m.mu.Unlock()
	return m.canHandle(request)
}

// Execute performs the agent's task
func (m *MockAgent) Execute(ctx context.Context, request agents.AgentRequest) (agents.AgentResult, error) {
	m.mu.Lock()
	m.calls = append(m.calls, "Execute:"+request.Prompt)
	m.mu.Unlock()
	return m.execute(ctx, request)
}

// SetCanHandle sets the canHandle function
func (m *MockAgent) SetCanHandle(fn func(string) (bool, float64)) {
	m.canHandle = fn
}

// SetExecute sets the execute function
func (m *MockAgent) SetExecute(fn func(context.Context, agents.AgentRequest) (agents.AgentResult, error)) {
	m.execute = fn
}

// GetCalls returns the list of method calls made to this mock
func (m *MockAgent) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]string, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// ResetCalls clears the call history
func (m *MockAgent) ResetCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = make([]string, 0)
}

// MockExecutor is a mock implementation of the Executor
type MockExecutor struct {
	executeFunc func(context.Context, *agents.ExecutionPlan) (map[string]agents.TaskResult, error)
	calls       []string
	mu          sync.Mutex
}

// NewMockExecutor creates a new mock executor
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		calls: make([]string, 0),
		executeFunc: func(ctx context.Context, plan *agents.ExecutionPlan) (map[string]agents.TaskResult, error) {
			results := make(map[string]agents.TaskResult)
			for _, task := range plan.Tasks {
				results[task.ID] = agents.TaskResult{
					Task: task,
					Result: agents.AgentResult{
						Success: true,
						Summary: "Mock execution successful",
					},
				}
			}
			return results, nil
		},
	}
}

// Execute runs the execution plan
func (m *MockExecutor) Execute(ctx context.Context, plan *agents.ExecutionPlan) (map[string]agents.TaskResult, error) {
	m.mu.Lock()
	m.calls = append(m.calls, "Execute:"+plan.ID)
	m.mu.Unlock()
	return m.executeFunc(ctx, plan)
}

// SetExecuteFunc sets the execute function
func (m *MockExecutor) SetExecuteFunc(fn func(context.Context, *agents.ExecutionPlan) (map[string]agents.TaskResult, error)) {
	m.executeFunc = fn
}

// GetCalls returns the list of method calls
func (m *MockExecutor) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]string, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// MockPlanner is a mock implementation of the Planner
type MockPlanner struct {
	planFunc func(context.Context, string, *agents.ExecutionContext) (*agents.ExecutionPlan, error)
	calls    []string
	mu       sync.Mutex
}

// NewMockPlanner creates a new mock planner
func NewMockPlanner() *MockPlanner {
	return &MockPlanner{
		calls: make([]string, 0),
		planFunc: func(ctx context.Context, request string, execContext *agents.ExecutionContext) (*agents.ExecutionPlan, error) {
			return &agents.ExecutionPlan{
				ID:      "mock-plan",
				Context: execContext,
				Tasks: []agents.Task{
					{
						ID:    "task-1",
						Agent: "mock-agent",
					},
				},
				Stages: []agents.Stage{
					{
						ID:    "stage-1",
						Tasks: []string{"task-1"},
					},
				},
			}, nil
		},
	}
}

// CreateExecutionPlan creates an execution plan
func (m *MockPlanner) CreateExecutionPlan(ctx context.Context, request string, execContext *agents.ExecutionContext) (*agents.ExecutionPlan, error) {
	m.mu.Lock()
	m.calls = append(m.calls, "CreateExecutionPlan:"+request)
	m.mu.Unlock()
	return m.planFunc(ctx, request, execContext)
}

// SetPlanFunc sets the plan function
func (m *MockPlanner) SetPlanFunc(fn func(context.Context, string, *agents.ExecutionContext) (*agents.ExecutionPlan, error)) {
	m.planFunc = fn
}

// GetCalls returns the list of method calls
func (m *MockPlanner) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]string, len(m.calls))
	copy(calls, m.calls)
	return calls
}
