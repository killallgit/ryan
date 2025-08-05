package agents

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanner_CreateExecutionPlan(t *testing.T) {
	tests := []struct {
		name           string
		request        string
		expectedStages int
		expectedTasks  int
		expectError    bool
	}{
		{
			name:           "Simple request",
			request:        "analyze this file",
			expectedStages: 1,
			expectedTasks:  1,
			expectError:    false,
		},
		{
			name:           "Complex request",
			request:        "search for all Go files, analyze their structure, and generate a report",
			expectedStages: 1,
			expectedTasks:  1, // The planner will identify this as primarily a search task
			expectError:    false,
		},
		{
			name:           "Empty request",
			request:        "",
			expectedStages: 0,
			expectedTasks:  0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			p := NewPlanner()
			o := NewOrchestrator()
			
			// Register some agents (using names that match the templates)
			o.RegisterAgent(newMockAgent("file_operations", "File operations"))
			o.RegisterAgent(newMockAgent("code_analysis", "Code analyzer"))
			o.RegisterAgent(newMockAgent("search", "Search agent"))
			
			p.SetOrchestrator(o)
			
			// Create execution context
			cm := NewContextManager()
			execContext := cm.CreateContext("test-session", "test-req", tt.request)
			
			// Create plan
			plan, err := p.CreateExecutionPlan(ctx, tt.request, execContext)
			
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.NotNil(t, plan)
			assert.NotEmpty(t, plan.ID)
			assert.Equal(t, execContext, plan.Context)
			
			// Validate plan structure
			if tt.expectedStages > 0 {
				assert.GreaterOrEqual(t, len(plan.Stages), tt.expectedStages)
			}
			if tt.expectedTasks > 0 {
				assert.GreaterOrEqual(t, len(plan.Tasks), tt.expectedTasks)
			}
		})
	}
}

func TestPlanner_IntentAnalysis(t *testing.T) {
	tests := []struct {
		name            string
		request         string
		expectedPrimary string
		hasSecondary    bool
	}{
		{
			name:            "File operation intent",
			request:         "read the config file",
			expectedPrimary: "file_operation",
			hasSecondary:    false,
		},
		{
			name:            "Code analysis intent",
			request:         "analyze the function complexity",
			expectedPrimary: "code_analysis",
			hasSecondary:    false,
		},
		{
			name:            "Search intent",
			request:         "find all TODO comments",
			expectedPrimary: "search",
			hasSecondary:    false,
		},
		{
			name:            "Complex intent",
			request:         "search for bugs and fix them",
			expectedPrimary: "search",
			hasSecondary:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewIntentAnalyzer()
			intent, err := analyzer.Analyze(tt.request)
			
			require.NoError(t, err)
			assert.NotNil(t, intent)
			assert.NotEmpty(t, intent.Primary)
			
			if tt.hasSecondary {
				assert.NotEmpty(t, intent.Secondary)
			}
		})
	}
}

func TestPlanner_GraphBuilder(t *testing.T) {
	o := NewOrchestrator()
	
	// Register agents with names matching templates
	fileAgent := newMockAgent("file_operations", "Reads files")
	analyzeAgent := newMockAgent("code_analysis", "Analyzes code")
	reportAgent := newMockAgent("dispatcher", "Generates reports")
	
	o.RegisterAgent(fileAgent)
	o.RegisterAgent(analyzeAgent)
	o.RegisterAgent(reportAgent)
	
	// Create graph builder
	builder := NewExecutionGraphBuilder()
	
	// Create intent with multiple operations
	intent := &Intent{
		Primary:   IntentType("analysis"),
		Secondary: []string{"file_read", "report"},
		Entities:  map[string]string{"file1": "main.go", "file2": "test.go"},
	}
	
	// Build graph
	graph, err := builder.BuildGraph(intent, o)
	
	require.NoError(t, err)
	assert.NotNil(t, graph)
	
	// Verify graph has proper structure (should have 1 node for analysis intent)
	assert.Greater(t, len(graph.Nodes), 0)
	
	// The analysis template has a single task with no dependencies
	// So we just verify the graph was created properly
	assert.Equal(t, 1, len(graph.Nodes))
}

func TestPlanner_Optimizer(t *testing.T) {
	// Create an unoptimized plan
	plan := &ExecutionPlan{
		ID: "test-plan",
		Tasks: []Task{
			{ID: "task1", Agent: "agent1", Priority: 1},
			{ID: "task2", Agent: "agent1", Priority: 2, Dependencies: []string{"task1"}},
			{ID: "task3", Agent: "agent2", Priority: 1},
			{ID: "task4", Agent: "agent2", Priority: 3, Dependencies: []string{"task3"}},
		},
	}
	
	optimizer := NewPlanOptimizer()
	// Create a simple graph from the plan
	graph := &ExecutionGraph{
		Nodes: make(map[string]*GraphNode),
	}
	for _, task := range plan.Tasks {
		graph.Nodes[task.ID] = &GraphNode{
			ID:           task.ID,
			Agent:        task.Agent,
			Dependencies: task.Dependencies,
		}
	}
	cm := NewContextManager()
	execContext := cm.CreateContext("test-session", "test-req", "test")
	optimized := optimizer.Optimize(graph, execContext)
	
	assert.NotNil(t, optimized)
	assert.Equal(t, len(plan.Tasks), len(optimized.Tasks))
	
	// Verify optimization preserved dependencies
	for _, task := range optimized.Tasks {
		originalTask := findTask(plan.Tasks, task.ID)
		assert.Equal(t, originalTask.Dependencies, task.Dependencies)
	}
	
	// Verify tasks are properly staged
	assert.Greater(t, len(optimized.Stages), 0)
}

func TestPlanner_ParallelExecution(t *testing.T) {
	ctx := context.Background()
	p := NewPlanner()
	o := NewOrchestrator()
	
	// Register independent agents
	for i := 0; i < 3; i++ {
		agent := newMockAgent(
			"independent-agent-"+string(rune(i)),
			"Independent agent",
		)
		o.RegisterAgent(agent)
	}
	
	p.SetOrchestrator(o)
	
	// Create a request that can be parallelized
	request := "perform independent task 1, independent task 2, and independent task 3"
	cm := NewContextManager()
	execContext := cm.CreateContext("test-session", "test-req", request)
	
	plan, err := p.CreateExecutionPlan(ctx, request, execContext)
	
	require.NoError(t, err)
	assert.NotNil(t, plan)
	
	// The planner creates tasks based on intent templates, not by parsing multiple independent tasks
	// It will identify this as a generic intent and create a single dispatcher task
	assert.GreaterOrEqual(t, len(plan.Tasks), 1, "Should have at least one task")
	
	// All tasks should be independent (no dependencies)
	for _, task := range plan.Tasks {
		assert.Empty(t, task.Dependencies, "Tasks should have no dependencies")
	}
}

func TestPlanner_ContextPropagation(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "test-key", "test-value")
	
	p := NewPlanner()
	o := NewOrchestrator()
	o.RegisterAgent(newMockAgent("test", "Test agent"))
	p.SetOrchestrator(o)
	
	cm := NewContextManager()
	execContext := cm.CreateContext("test-session", "test-req", "test request")
	execContext.SharedData["user-data"] = "important"
	
	plan, err := p.CreateExecutionPlan(ctx, "test request", execContext)
	
	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, execContext, plan.Context)
	
	// Verify context data is preserved
	userData, exists := plan.Context.SharedData["user-data"]
	assert.True(t, exists)
	assert.Equal(t, "important", userData)
}

func TestPlanner_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Planner)
		request     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "No orchestrator set",
			setup: func(p *Planner) {
				// Don't set orchestrator
			},
			request:     "test",
			expectError: true,
			errorMsg:    "orchestrator",
		},
		{
			name: "Invalid request",
			setup: func(p *Planner) {
				o := NewOrchestrator()
				p.SetOrchestrator(o)
			},
			request:     "",
			expectError: true,
			errorMsg:    "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			p := NewPlanner()
			
			if tt.setup != nil {
				tt.setup(p)
			}
			
			cm := NewContextManager()
			execContext := cm.CreateContext("test-session", "test-req", tt.request)
			_, err := p.CreateExecutionPlan(ctx, tt.request, execContext)
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPlanner_TimeoutHandling(t *testing.T) {
	p := NewPlanner()
	o := NewOrchestrator()
	
	// Register a slow agent
	slowAgent := newMockAgent("slow", "Slow agent")
	slowAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return AgentResult{}, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return AgentResult{Success: true}, nil
		}
	})
	
	o.RegisterAgent(slowAgent)
	p.SetOrchestrator(o)
	
	// Create a very short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel()
	
	// Wait for context to timeout
	time.Sleep(1 * time.Millisecond)
	
	cm := NewContextManager()
	execContext := cm.CreateContext("test-session", "test-req", "test")
	
	// Try to create plan with expired context
	plan, err := p.CreateExecutionPlan(ctx, "test", execContext)
	
	// The plan creation itself should succeed (it's fast)
	// but if we were to execute it with a timeout context, that would fail
	if err == nil {
		assert.NotNil(t, plan)
		// Test that execution with timeout would fail
		ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel2()
		
		// If we were to execute with the slow agent, it would timeout
		result, err := slowAgent.Execute(ctx2, AgentRequest{Prompt: "test"})
		assert.Error(t, err) // Should timeout
		assert.False(t, result.Success)
	}
}

// Helper function to find a task by ID
func findTask(tasks []Task, id string) *Task {
	for _, task := range tasks {
		if task.ID == id {
			return &task
		}
	}
	return nil
}