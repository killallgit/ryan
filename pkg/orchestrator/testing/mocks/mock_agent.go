package mocks

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/orchestrator"
)

// BehaviorConfig defines configurable behavior for mock agents
type BehaviorConfig struct {
	FailureRate    float64       // Probability of failure (0.0 to 1.0)
	ResponseDelay  time.Duration // Simulated processing time
	RequiresRetry  bool          // Whether agent should require retries
	MaxRetries     int           // Maximum number of retries before success
	PartialSuccess bool          // Return partial results
}

// MockAgent implements the Agent interface for testing
type MockAgent struct {
	Type         orchestrator.AgentType
	Capabilities []string
	Responses    []orchestrator.AgentResponse // Queue of responses
	Behavior     BehaviorConfig

	// Tracking
	ExecuteCount int
	LastDecision *orchestrator.RouteDecision
	LastState    *orchestrator.TaskState
	History      []ExecutionRecord

	// Callbacks for testing
	OnExecute func(decision *orchestrator.RouteDecision, state *orchestrator.TaskState)

	mu          sync.Mutex
	responseIdx int
	retryCount  map[string]int // Track retries per task
}

// ExecutionRecord tracks agent executions
type ExecutionRecord struct {
	Timestamp time.Time
	Decision  *orchestrator.RouteDecision
	State     *orchestrator.TaskState
	Response  *orchestrator.AgentResponse
	Error     error
}

// NewMockAgent creates a new mock agent
func NewMockAgent(agentType orchestrator.AgentType) *MockAgent {
	return &MockAgent{
		Type:         agentType,
		Capabilities: []string{"mock", "testing"},
		Responses:    make([]orchestrator.AgentResponse, 0),
		Behavior:     BehaviorConfig{},
		History:      make([]ExecutionRecord, 0),
		retryCount:   make(map[string]int),
	}
}

// WithCapabilities sets the agent's capabilities
func (m *MockAgent) WithCapabilities(capabilities ...string) *MockAgent {
	m.Capabilities = capabilities
	return m
}

// WithResponses sets the response queue
func (m *MockAgent) WithResponses(responses ...orchestrator.AgentResponse) *MockAgent {
	m.Responses = responses
	return m
}

// WithBehavior sets the behavior configuration
func (m *MockAgent) WithBehavior(config BehaviorConfig) *MockAgent {
	m.Behavior = config
	return m
}

// WithFailureRate sets the failure rate
func (m *MockAgent) WithFailureRate(rate float64) *MockAgent {
	m.Behavior.FailureRate = rate
	return m
}

// Execute implements the Agent interface
func (m *MockAgent) Execute(ctx context.Context, decision *orchestrator.RouteDecision, state *orchestrator.TaskState) (*orchestrator.AgentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Track execution
	m.ExecuteCount++
	m.LastDecision = decision
	m.LastState = state

	record := ExecutionRecord{
		Timestamp: time.Now(),
		Decision:  decision,
		State:     state,
	}

	// Call callback if set
	if m.OnExecute != nil {
		m.OnExecute(decision, state)
	}

	// Simulate processing delay
	if m.Behavior.ResponseDelay > 0 {
		time.Sleep(m.Behavior.ResponseDelay)
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		err := ctx.Err()
		record.Error = err
		m.History = append(m.History, record)
		return nil, err
	default:
	}

	// Handle retry logic
	if m.Behavior.RequiresRetry {
		retries := m.retryCount[state.ID]
		if retries < m.Behavior.MaxRetries {
			m.retryCount[state.ID]++
			err := fmt.Errorf("retry required (%d/%d)", retries+1, m.Behavior.MaxRetries)
			record.Error = err
			m.History = append(m.History, record)
			errorStr := err.Error()
			return &orchestrator.AgentResponse{
				AgentType: m.Type,
				Response:  "",
				Status:    "failed",
				Error:     &errorStr,
				Timestamp: time.Now(),
			}, nil
		}
		// Reset retry count after success
		delete(m.retryCount, state.ID)
	}

	// Simulate failures based on failure rate
	if m.Behavior.FailureRate > 0 && rand.Float64() < m.Behavior.FailureRate {
		err := errors.New("simulated agent failure")
		record.Error = err
		m.History = append(m.History, record)
		errorStr := err.Error()
		return &orchestrator.AgentResponse{
			AgentType: m.Type,
			Response:  "",
			Status:    "failed",
			Error:     &errorStr,
			Timestamp: time.Now(),
		}, nil
	}

	// Generate response
	response := m.generateResponse(decision, state)
	record.Response = response
	m.History = append(m.History, record)

	return response, nil
}

// generateResponse creates an appropriate response
func (m *MockAgent) generateResponse(decision *orchestrator.RouteDecision, state *orchestrator.TaskState) *orchestrator.AgentResponse {
	// Use queued response if available
	if len(m.Responses) > 0 {
		if m.responseIdx < len(m.Responses) {
			resp := m.Responses[m.responseIdx]
			m.responseIdx++
			// Ensure correct agent type
			resp.AgentType = m.Type
			resp.Timestamp = time.Now()
			return &resp
		}
	}

	// Generate default response based on agent type
	response := &orchestrator.AgentResponse{
		AgentType: m.Type,
		Timestamp: time.Now(),
	}

	switch m.Type {
	case orchestrator.AgentToolCaller:
		response.Response = fmt.Sprintf("Executed tools for: %s", decision.Instruction)
		response.Status = "success"

		// Simulate specific tool calls based on the instruction content
		toolCalls := m.getToolCallsForInstruction(decision.Instruction, decision.Tools)
		response.ToolsCalled = toolCalls

	case orchestrator.AgentCodeGen:
		response.Response = fmt.Sprintf("```go\n// Generated code for: %s\nfunc mockFunction() {\n\t// Implementation\n}\n```", decision.Instruction)
		response.Status = "success"

	case orchestrator.AgentReasoner:
		response.Response = fmt.Sprintf("Reasoning about: %s\n\nAnalysis: This is a mock reasoning response.", decision.Instruction)
		response.Status = "success"

	case orchestrator.AgentSearcher:
		response.Response = fmt.Sprintf("Search results for: %s\n- Result 1\n- Result 2\n- Result 3", decision.Instruction)
		response.Status = "success"

	case orchestrator.AgentPlanner:
		response.Response = fmt.Sprintf("Plan for: %s\n1. Step one\n2. Step two\n3. Step three", decision.Instruction)
		response.Status = "success"
		// Planner might suggest next action
		if m.Behavior.PartialSuccess {
			response.NextAction = &orchestrator.RouteDecision{
				TargetAgent: orchestrator.AgentToolCaller,
				Instruction: "Execute the plan",
			}
		}

	default:
		response.Response = fmt.Sprintf("Mock response from %s", m.Type)
		response.Status = "success"
	}

	// Handle partial success - suggest continuing with another agent to create infinite loop
	if m.Behavior.PartialSuccess {
		response.Status = "partial"
		// Create a cycle between different agent types to exceed max iterations
		var nextAgent orchestrator.AgentType
		switch m.Type {
		case orchestrator.AgentReasoner:
			nextAgent = orchestrator.AgentPlanner
		case orchestrator.AgentPlanner:
			nextAgent = orchestrator.AgentToolCaller
		case orchestrator.AgentToolCaller:
			nextAgent = orchestrator.AgentCodeGen
		case orchestrator.AgentCodeGen:
			nextAgent = orchestrator.AgentReasoner
		default:
			nextAgent = orchestrator.AgentReasoner
		}

		response.NextAction = &orchestrator.RouteDecision{
			TargetAgent: nextAgent,
			Instruction: fmt.Sprintf("Continue processing: %s", decision.Instruction),
		}
	}

	return response
}

// getToolCallsForInstruction determines which tools to simulate based on the instruction
func (m *MockAgent) getToolCallsForInstruction(instruction string, availableTools []string) []orchestrator.ToolCall {
	var toolCalls []orchestrator.ToolCall
	instLower := strings.ToLower(instruction)

	// Special case for package.json modify scenario
	if strings.Contains(instLower, "package.json") && strings.Contains(instLower, "modify") {
		toolCalls = append(toolCalls, orchestrator.ToolCall{
			Name:      "file_read",
			Arguments: map[string]interface{}{"path": "package.json"},
			Result:    `{"version": "1.0.0"}`,
		})
		toolCalls = append(toolCalls, orchestrator.ToolCall{
			Name:      "file_write",
			Arguments: map[string]interface{}{"path": "package.json", "content": `{"version": "1.0.1"}`},
			Result:    "File written successfully",
		})
		return toolCalls // Return early for this specific case
	}

	// General file operations
	if strings.Contains(instLower, "read") && strings.Contains(instLower, "file") {
		toolCalls = append(toolCalls, orchestrator.ToolCall{
			Name:      "file_read",
			Arguments: map[string]interface{}{"path": "example.txt"},
			Result:    "Mock file content",
		})
	}

	if strings.Contains(instLower, "write") && (strings.Contains(instLower, "file") || strings.Contains(instLower, "save")) {
		toolCalls = append(toolCalls, orchestrator.ToolCall{
			Name:      "file_write",
			Arguments: map[string]interface{}{"path": "server.go", "content": "package main..."},
			Result:    "File written successfully",
		})
	}

	// Bash/shell operations
	if strings.Contains(instLower, "list") && strings.Contains(instLower, "files") {
		toolCalls = append(toolCalls, orchestrator.ToolCall{
			Name:      "bash",
			Arguments: map[string]interface{}{"command": "ls -la"},
			Result:    "file1.txt\nfile2.go\nfile3.md",
		})
	}

	// Git operations
	if strings.Contains(instLower, "git") {
		toolCalls = append(toolCalls, orchestrator.ToolCall{
			Name:      "git",
			Arguments: map[string]interface{}{"command": "status"},
			Result:    "On branch main\nnothing to commit, working tree clean",
		})
	}

	// If no specific tool calls were generated, use the first available tool as fallback
	if len(toolCalls) == 0 && len(availableTools) > 0 {
		toolCalls = append(toolCalls, orchestrator.ToolCall{
			Name:      availableTools[0],
			Arguments: map[string]interface{}{"query": instruction},
			Result:    "Mock tool result",
		})
	}

	return toolCalls
}

// GetCapabilities implements the Agent interface
func (m *MockAgent) GetCapabilities() []string {
	return m.Capabilities
}

// GetType implements the Agent interface
func (m *MockAgent) GetType() orchestrator.AgentType {
	return m.Type
}

// GetExecutionHistory returns the execution history
func (m *MockAgent) GetExecutionHistory() []ExecutionRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ExecutionRecord{}, m.History...)
}

// Reset clears the agent's state
func (m *MockAgent) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExecuteCount = 0
	m.LastDecision = nil
	m.LastState = nil
	m.History = make([]ExecutionRecord, 0)
	m.responseIdx = 0
	m.retryCount = make(map[string]int)
}

// MockAgentFactory creates mock agents with predefined behaviors
type MockAgentFactory struct {
	agents map[orchestrator.AgentType]*MockAgent
}

// NewMockAgentFactory creates a new mock agent factory
func NewMockAgentFactory() *MockAgentFactory {
	return &MockAgentFactory{
		agents: make(map[orchestrator.AgentType]*MockAgent),
	}
}

// CreateDefaultAgents creates a set of default mock agents
func (f *MockAgentFactory) CreateDefaultAgents() map[orchestrator.AgentType]*MockAgent {
	agents := map[orchestrator.AgentType]*MockAgent{
		orchestrator.AgentToolCaller: NewMockAgent(orchestrator.AgentToolCaller).
			WithCapabilities("bash", "file", "git", "web"),

		orchestrator.AgentCodeGen: NewMockAgent(orchestrator.AgentCodeGen).
			WithCapabilities("coding", "refactoring"),

		orchestrator.AgentReasoner: NewMockAgent(orchestrator.AgentReasoner).
			WithCapabilities("reasoning", "analysis"),

		orchestrator.AgentSearcher: NewMockAgent(orchestrator.AgentSearcher).
			WithCapabilities("search", "find"),

		orchestrator.AgentPlanner: NewMockAgent(orchestrator.AgentPlanner).
			WithCapabilities("planning", "decomposition"),
	}

	f.agents = agents
	return agents
}

// GetAgent returns a mock agent by type
func (f *MockAgentFactory) GetAgent(agentType orchestrator.AgentType) *MockAgent {
	return f.agents[agentType]
}

// RegisterAll registers all agents with an orchestrator
func (f *MockAgentFactory) RegisterAll(o *orchestrator.Orchestrator) error {
	for agentType, agent := range f.agents {
		if err := o.RegisterAgent(agentType, agent); err != nil {
			return err
		}
	}
	return nil
}
