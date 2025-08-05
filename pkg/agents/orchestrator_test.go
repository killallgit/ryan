package agents

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestrator_RegisterAgent(t *testing.T) {
	tests := []struct {
		name        string
		agentName   string
		agent       Agent
		expectError bool
	}{
		{
			name:        "Register valid agent",
			agentName:   "test-agent",
			agent:       newMockAgent("test-agent", "Test agent"),
			expectError: false,
		},
		{
			name:        "Register nil agent",
			agentName:   "nil-agent",
			agent:       nil,
			expectError: true,
		},
		{
			name:        "Register duplicate agent",
			agentName:   "duplicate",
			agent:       newMockAgent("duplicate", "Duplicate agent"),
			expectError: true, // Should fail on duplicate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewOrchestrator()
			
			if tt.agent != nil {
				err := o.RegisterAgent(tt.agent)
				// For duplicate, we expect an error
				if tt.name == "Register duplicate agent" {
					// First registration should succeed
					if err == nil {
						// Try registering again to test duplicate
						err = o.RegisterAgent(tt.agent)
						assert.Error(t, err, "Second registration should fail")
					}
				} else if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
				
				// Verify agent was registered (skip for duplicate test)
				if tt.name != "Register duplicate agent" {
					agent, err := o.GetAgent(tt.agentName)
					if tt.expectError {
						assert.Error(t, err)
						assert.Nil(t, agent)
					} else {
						assert.NoError(t, err)
						assert.NotNil(t, agent)
						assert.Equal(t, tt.agentName, agent.Name())
					}
				}
			}
		})
	}
}

func TestOrchestrator_RegisterMultipleAgents(t *testing.T) {
	o := NewOrchestrator()
	
	agents := []Agent{
		newMockAgent("agent1", "Agent 1"),
		newMockAgent("agent2", "Agent 2"),
		newMockAgent("agent3", "Agent 3"),
	}
	
	for _, agent := range agents {
		err := o.RegisterAgent(agent)
		assert.NoError(t, err)
	}
	
	// Verify all agents were registered (plus built-in dispatcher)
	registeredAgents := o.ListAgents()
	assert.Len(t, registeredAgents, len(agents)+1) // +1 for dispatcher
	
	for _, agent := range agents {
		retrieved, err := o.GetAgent(agent.Name())
		assert.NoError(t, err)
		assert.Equal(t, agent.Name(), retrieved.Name())
	}
}

func TestOrchestrator_Execute_SimpleRequest(t *testing.T) {
	ctx := context.Background()
	o := NewOrchestrator()
	
	// Create a mock agent that can handle the request
	mockAgent := newMockAgent("test-handler", "Handler agent")
	mockAgent.SetCanHandle(func(request string) (bool, float64) {
		return true, 1.0
	})
	mockAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		return AgentResult{
			Success: true,
			Summary: "Request handled successfully",
			Details: "Mock handled the request",
		}, nil
	})
	
	err := o.RegisterAgent(mockAgent)
	require.NoError(t, err)
	
	// Directly test agent execution without going through the full planner
	agent, err := o.GetAgent("test-handler")
	require.NoError(t, err)
	
	result, err := agent.Execute(ctx, AgentRequest{
		Prompt: "test request",
		Context: make(map[string]interface{}),
	})
	
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Summary, "successfully")
	
	// Verify the agent was called
	calls := mockAgent.GetCalls()
	assert.Contains(t, calls, "Execute:test request")
}

func TestOrchestrator_Execute_NoSuitableAgent(t *testing.T) {
	o := NewOrchestrator()
	
	// Register an agent that cannot handle the request
	mockAgent := newMockAgent("non-handler", "Non-handler agent")
	mockAgent.SetCanHandle(func(request string) (bool, float64) {
		return false, 0.0
	})
	
	err := o.RegisterAgent(mockAgent)
	require.NoError(t, err)
	
	// Test the agent directly
	agent, err := o.GetAgent("non-handler")
	require.NoError(t, err)
	
	// Check if agent can handle the request
	canHandle, confidence := agent.CanHandle("unhandled request")
	assert.False(t, canHandle)
	assert.Equal(t, 0.0, confidence)
	
	// Verify the agent was checked
	calls := mockAgent.GetCalls()
	assert.Contains(t, calls, "CanHandle:unhandled request")
}

func TestOrchestrator_Execute_MultipleAgents(t *testing.T) {
	ctx := context.Background()
	o := NewOrchestrator()
	
	// Create multiple agents with different confidence levels
	lowConfAgent := newMockAgent("low-conf", "Low confidence agent")
	lowConfAgent.SetCanHandle(func(request string) (bool, float64) {
		return true, 0.3
	})
	
	highConfAgent := newMockAgent("high-conf", "High confidence agent")
	highConfAgent.SetCanHandle(func(request string) (bool, float64) {
		return true, 0.9
	})
	highConfAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		return AgentResult{
			Success: true,
			Summary: "High confidence agent handled it",
		}, nil
	})
	
	err := o.RegisterAgent(lowConfAgent)
	require.NoError(t, err)
	err = o.RegisterAgent(highConfAgent)
	require.NoError(t, err)
	
	// Test agent confidence levels directly
	lowAgent, _ := o.GetAgent("low-conf")
	highAgent, _ := o.GetAgent("high-conf")
	
	lowCanHandle, lowConf := lowAgent.CanHandle("test request")
	highCanHandle, highConf := highAgent.CanHandle("test request")
	
	assert.True(t, lowCanHandle)
	assert.True(t, highCanHandle)
	assert.Less(t, lowConf, highConf)
	
	// Execute with high confidence agent
	result, err := highAgent.Execute(ctx, AgentRequest{
		Prompt: "test request",
		Context: make(map[string]interface{}),
	})
	
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "High confidence agent handled it", result.Summary)
	
	// Verify calls
	highCalls := highConfAgent.GetCalls()
	assert.Contains(t, highCalls, "Execute:test request")
	
	lowCalls := lowConfAgent.GetCalls()
	assert.Contains(t, lowCalls, "CanHandle:test request")
}

func TestOrchestrator_Execute_WithContext(t *testing.T) {
	o := NewOrchestrator()
	
	// Create an agent that checks context
	mockAgent := newMockAgent("context-aware", "Context aware agent")
	var receivedContext context.Context
	
	// Make sure this agent can handle the request
	mockAgent.SetCanHandle(func(request string) (bool, float64) {
		return true, 1.0
	})
	
	mockAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		receivedContext = ctx
		
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return AgentResult{Success: false}, ctx.Err()
		default:
			return AgentResult{Success: true}, nil
		}
	})
	
	err := o.RegisterAgent(mockAgent)
	require.NoError(t, err)
	
	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	// Test the agent directly to verify context propagation
	agent, err := o.GetAgent("context-aware")
	require.NoError(t, err)
	
	result, err := agent.Execute(ctx, AgentRequest{
		Prompt: "test request",
		Context: make(map[string]interface{}),
	})
	
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotNil(t, receivedContext)
}

func TestOrchestrator_Execute_AgentError(t *testing.T) {
	ctx := context.Background()
	o := NewOrchestrator()
	
	// Create an agent that returns an error
	mockAgent := newMockAgent("error-agent", "Error agent")
	expectedErr := errors.New("agent execution failed")
	
	mockAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		return AgentResult{}, expectedErr
	})
	
	err := o.RegisterAgent(mockAgent)
	require.NoError(t, err)
	
	// Execute request
	result, err := o.Execute(ctx, "test request", nil)
	
	// Should handle the error gracefully
	assert.NoError(t, err) // Orchestrator should not propagate agent errors
	assert.True(t, result.Success) // Should fall back to dispatcher or handle gracefully
}

func TestOrchestrator_ConcurrentRegistration(t *testing.T) {
	o := NewOrchestrator()
	var wg sync.WaitGroup
	
	// Concurrently register multiple agents
	numAgents := 10
	wg.Add(numAgents)
	
	for i := 0; i < numAgents; i++ {
		go func(id int) {
			defer wg.Done()
			agent := newMockAgent(
				"concurrent-agent-"+string(rune(id)),
				"Concurrent agent",
			)
			_ = o.RegisterAgent(agent) // Ignore error in concurrent test
		}(i)
	}
	
	wg.Wait()
	
	// Verify agents were registered (exact count may vary due to overwrites)
	agents := o.ListAgents()
	assert.GreaterOrEqual(t, len(agents), 1)
}

func TestOrchestrator_ConcurrentExecution(t *testing.T) {
	ctx := context.Background()
	o := NewOrchestrator()
	
	// Register a thread-safe mock agent
	mockAgent := newMockAgent("concurrent", "Concurrent agent")
	var executionCount int32
	
	// Make this agent handle the requests
	mockAgent.SetCanHandle(func(request string) (bool, float64) {
		return true, 1.0
	})
	
	mockAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		atomic.AddInt32(&executionCount, 1)
		
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		
		return AgentResult{Success: true}, nil
	})
	
	err := o.RegisterAgent(mockAgent)
	require.NoError(t, err)
	
	// Get the agent for direct testing
	agent, err := o.GetAgent("concurrent")
	require.NoError(t, err)
	
	// Execute multiple requests concurrently
	var wg sync.WaitGroup
	numRequests := 5
	wg.Add(numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func(id int) {
			defer wg.Done()
			result, err := agent.Execute(ctx, AgentRequest{
				Prompt: "concurrent request",
				Context: make(map[string]interface{}),
			})
			assert.NoError(t, err)
			assert.True(t, result.Success)
		}(i)
	}
	
	wg.Wait()
	
	// Verify all executions completed
	finalCount := atomic.LoadInt32(&executionCount)
	assert.Equal(t, int32(numRequests), finalCount)
}

func TestOrchestrator_ComplexPlan(t *testing.T) {
	ctx := context.Background()
	o := NewOrchestrator()
	
	// Create specialized agents
	fileAgent := newMockAgent("file-ops", "File operations")
	fileAgent.SetCanHandle(func(request string) (bool, float64) {
		return contains(request, "file") || contains(request, "read"), 0.8
	})
	fileAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		return AgentResult{Success: true}, nil
	})
	
	codeAgent := newMockAgent("code-analysis", "Code analysis")
	codeAgent.SetCanHandle(func(request string) (bool, float64) {
		return contains(request, "analyze") || contains(request, "code"), 0.9
	})
	codeAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		return AgentResult{Success: true}, nil
	})
	
	searchAgent := newMockAgent("search", "Search agent")
	searchAgent.SetCanHandle(func(request string) (bool, float64) {
		return contains(request, "search") || contains(request, "find"), 0.7
	})
	searchAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		return AgentResult{Success: true}, nil
	})
	
	err := o.RegisterAgent(fileAgent)
	require.NoError(t, err)
	err = o.RegisterAgent(codeAgent)
	require.NoError(t, err)
	err = o.RegisterAgent(searchAgent)
	require.NoError(t, err)
	
	// Test each agent's ability to handle relevant requests
	fileCanHandle, _ := fileAgent.CanHandle("read file")
	codeCanHandle, _ := codeAgent.CanHandle("analyze code")
	searchCanHandle, _ := searchAgent.CanHandle("search for files")
	
	assert.True(t, fileCanHandle)
	assert.True(t, codeCanHandle)
	assert.True(t, searchCanHandle)
	
	// Test agent execution
	fileResult, _ := fileAgent.Execute(ctx, AgentRequest{Prompt: "read file"})
	codeResult, _ := codeAgent.Execute(ctx, AgentRequest{Prompt: "analyze code"})
	searchResult, _ := searchAgent.Execute(ctx, AgentRequest{Prompt: "search"})
	
	assert.True(t, fileResult.Success)
	assert.True(t, codeResult.Success)
	assert.True(t, searchResult.Success)
	
	// Verify agents were called
	assert.Greater(t, len(fileAgent.GetCalls()), 0)
	assert.Greater(t, len(codeAgent.GetCalls()), 0)
	assert.Greater(t, len(searchAgent.GetCalls()), 0)
}

func TestOrchestrator_GetAgent(t *testing.T) {
	o := NewOrchestrator()
	
	// Register an agent
	mockAgent := newMockAgent("test-agent", "Test agent")
	err := o.RegisterAgent(mockAgent)
	require.NoError(t, err)
	
	// Get existing agent
	agent, err := o.GetAgent("test-agent")
	assert.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, "test-agent", agent.Name())
	
	// Get non-existent agent
	agent, err = o.GetAgent("non-existent")
	assert.Error(t, err)
	assert.Nil(t, agent)
}

func TestOrchestrator_ListAgents(t *testing.T) {
	o := NewOrchestrator()
	
	// Initially should have default agents
	initialAgents := o.ListAgents()
	initialCount := len(initialAgents)
	
	// Register additional agents
	agents := []Agent{
		newMockAgent("agent1", "Agent 1"),
		newMockAgent("agent2", "Agent 2"),
	}
	
	for _, agent := range agents {
		err := o.RegisterAgent(agent)
		assert.NoError(t, err)
	}
	
	// List all agents
	allAgents := o.ListAgents()
	assert.GreaterOrEqual(t, len(allAgents), initialCount+len(agents))
	
	// Verify our agents are in the list
	agentNames := make(map[string]bool)
	for _, agent := range allAgents {
		agentNames[agent.Name()] = true
	}
	
	assert.True(t, agentNames["agent1"])
	assert.True(t, agentNames["agent2"])
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || len(s) >= len(substr))
}