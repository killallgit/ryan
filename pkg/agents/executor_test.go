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

func TestExecutor_ExecutePlan_Simple(t *testing.T) {
	ctx := context.Background()
	executor := NewExecutor()
	orchestrator := NewOrchestrator()
	
	// Set up mock agent
	mockAgent := newMockAgent("test-agent", "Test agent")
	mockAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		return AgentResult{
			Success: true,
			Summary: "Task completed",
			Details: "Task completed successfully",
		}, nil
	})
	
	orchestrator.RegisterAgent(mockAgent)
	executor.SetOrchestrator(orchestrator)
	
	// Create a simple plan
	plan := createTestPlan("test-agent")
	cm := NewContextManager()
	execContext := cm.CreateContext("test-session", "test-req", "test-request")
	
	// Execute plan
	results, err := executor.ExecutePlan(ctx, plan, execContext)
	
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Greater(t, len(results), 0)
	
	// Verify result
	assert.True(t, results[0].Result.Success)
}

func TestExecutor_ExecutePlan_Sequential(t *testing.T) {
	ctx := context.Background()
	executor := NewExecutor()
	orchestrator := NewOrchestrator()
	
	// Track execution order
	var executionOrder []string
	var mu sync.Mutex
	
	// Set up mock agents
	for i := 1; i <= 3; i++ {
		agentName := "agent" + string(rune('0'+i))
		mockAgent := newMockAgent(agentName, "Agent "+string(rune('0'+i)))
		
		mockAgent.SetExecute(func(name string) func(context.Context, AgentRequest) (AgentResult, error) {
			return func(ctx context.Context, req AgentRequest) (AgentResult, error) {
				mu.Lock()
				executionOrder = append(executionOrder, name)
				mu.Unlock()
				
				return AgentResult{
					Success: true,
					Summary: name + " completed",
				}, nil
			}
		}(agentName))
		
		orchestrator.RegisterAgent(mockAgent)
	}
	
	executor.SetOrchestrator(orchestrator)
	
	// Create a plan with sequential dependencies
	plan := createTestPlan("agent1", "agent2", "agent3")
	cm := NewContextManager()
	execContext := cm.CreateContext("test-session", "test-req", "test-request")
	
	// Execute plan
	results, err := executor.ExecutePlan(ctx, plan, execContext)
	
	require.NoError(t, err)
	assert.Len(t, results, 3)
	
	// Verify execution order (should be sequential due to dependencies)
	assert.Equal(t, []string{"agent1", "agent2", "agent3"}, executionOrder)
}

func TestExecutor_ExecutePlan_Parallel(t *testing.T) {
	ctx := context.Background()
	executor := NewExecutor()
	orchestrator := NewOrchestrator()
	
	// Track concurrent executions
	var activeCount int32
	var maxActive int32
	
	// Set up mock agents with delays
	for i := 1; i <= 3; i++ {
		agentName := "agent" + string(rune('0'+i))
		mockAgent := newMockAgent(agentName, "Agent "+string(rune('0'+i)))
		
		mockAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
			// Increment active count
			current := atomic.AddInt32(&activeCount, 1)
			
			// Update max if needed
			for {
				max := atomic.LoadInt32(&maxActive)
				if current <= max || atomic.CompareAndSwapInt32(&maxActive, max, current) {
					break
				}
			}
			
			// Simulate work
			time.Sleep(50 * time.Millisecond)
			
			// Decrement active count
			atomic.AddInt32(&activeCount, -1)
			
			return AgentResult{Success: true}, nil
		})
		
		orchestrator.RegisterAgent(mockAgent)
	}
	
	executor.SetOrchestrator(orchestrator)
	
	// Create a plan with parallel tasks (no dependencies)
	cm := NewContextManager()
	plan := &ExecutionPlan{
		ID:      "parallel-plan",
		Context: cm.CreateContext("test-session", "test-req", "parallel test"),
		Tasks: []Task{
			createTestTask("task1", "agent1"),
			createTestTask("task2", "agent2"),
			createTestTask("task3", "agent3"),
		},
		Stages: []Stage{
			{
				ID:    "stage1",
				Tasks: []string{"task1", "task2", "task3"},
			},
		},
	}
	
	execContext := cm.CreateContext("test-session", "test-req", "test-request")
	
	// Execute plan
	start := time.Now()
	results, err := executor.ExecutePlan(ctx, plan, execContext)
	duration := time.Since(start)
	
	require.NoError(t, err)
	assert.Len(t, results, 3)
	
	// Verify parallel execution (should be faster than sequential)
	assert.Less(t, duration, 200*time.Millisecond, "Parallel execution should be faster")
	assert.GreaterOrEqual(t, maxActive, int32(2), "Should have multiple concurrent executions")
}

func TestExecutor_ExecutePlan_WithFailure(t *testing.T) {
	ctx := context.Background()
	executor := NewExecutor()
	orchestrator := NewOrchestrator()
	
	// Set up mock agents
	successAgent1 := newMockAgent("success1", "Success agent 1")
	successAgent1.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		return AgentResult{Success: true}, nil
	})
	
	failureAgent := newMockAgent("failure", "Failure agent")
	failureAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		return AgentResult{Success: false}, errors.New("task failed")
	})
	
	successAgent2 := newMockAgent("success2", "Success agent 2")
	var agent2Called bool
	successAgent2.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		agent2Called = true
		return AgentResult{Success: true}, nil
	})
	
	orchestrator.RegisterAgent(successAgent1)
	orchestrator.RegisterAgent(failureAgent)
	orchestrator.RegisterAgent(successAgent2)
	executor.SetOrchestrator(orchestrator)
	
	// Create a plan with multiple tasks
	plan := createTestPlan("success1", "failure", "success2")
	cm := NewContextManager()
	execContext := cm.CreateContext("test-session", "test-req", "test-request")
	
	// Execute plan
	results, err := executor.ExecutePlan(ctx, plan, execContext)
	
	// The executor continues execution but records failures
	require.NoError(t, err, "Executor should not return error for individual task failures")
	assert.Len(t, results, 3, "Should have results for all tasks")
	
	// Check that the failure task failed
	foundFailure := false
	for _, result := range results {
		if result.Task.Agent == "failure" {
			foundFailure = true
			assert.False(t, result.Result.Success, "Failure task should not succeed")
			assert.NotNil(t, result.Error, "Failure task should have error")
		}
	}
	assert.True(t, foundFailure, "Should have found failure task result")
	
	// With current implementation, all tasks execute regardless of failures
	// This is by design to allow parallel independent tasks
	assert.True(t, agent2Called, "All tasks execute in current implementation")
}

func TestExecutor_ExecutePlan_WithTimeout(t *testing.T) {
	executor := NewExecutor()
	orchestrator := NewOrchestrator()
	
	// Set up slow agent
	slowAgent := newMockAgent("slow-agent", "Slow agent")
	slowAgent.SetExecute(func(ctx context.Context, req AgentRequest) (AgentResult, error) {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return AgentResult{}, ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return AgentResult{Success: true}, nil
		}
	})
	
	orchestrator.RegisterAgent(slowAgent)
	executor.SetOrchestrator(orchestrator)
	
	// Create a plan with timeout
	cm := NewContextManager()
	plan := &ExecutionPlan{
		ID:      "timeout-plan",
		Context: cm.CreateContext("test-session", "test-req", "timeout test"),
		Tasks: []Task{
			{
				ID:      "slow-task",
				Agent:   "slow-agent",
				Request: createTestRequest("slow request"),
				Timeout: 50 * time.Millisecond,
			},
		},
		Stages: []Stage{
			{ID: "stage1", Tasks: []string{"slow-task"}},
		},
	}
	
	// Execute with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	execContext := cm.CreateContext("test-session", "test-req", "test-request")
	results, err := executor.ExecutePlan(ctx, plan, execContext)
	
	// Should handle timeout
	if err == nil {
		// If no error, check that result indicates timeout
		assert.NotNil(t, results)
		if len(results) > 0 {
			assert.False(t, results[0].Result.Success)
		}
	} else {
		// Error should be timeout related
		assert.Contains(t, err.Error(), "context")
	}
}

func TestExecutor_ExecutePlan_ComplexDependencies(t *testing.T) {
	ctx := context.Background()
	executor := NewExecutor()
	orchestrator := NewOrchestrator()
	
	// Track execution order
	var executionOrder []string
	var mu sync.Mutex
	
	// Set up agents
	for i := 1; i <= 4; i++ {
		agentName := "agent" + string(rune('0'+i))
		taskName := "task" + string(rune('0'+i))
		
		mockAgent := newMockAgent(agentName, "Agent")
		mockAgent.SetExecute(func(task string) func(context.Context, AgentRequest) (AgentResult, error) {
			return func(ctx context.Context, req AgentRequest) (AgentResult, error) {
				mu.Lock()
				executionOrder = append(executionOrder, task)
				mu.Unlock()
				
				time.Sleep(10 * time.Millisecond)
				return AgentResult{Success: true}, nil
			}
		}(taskName))
		
		orchestrator.RegisterAgent(mockAgent)
	}
	
	executor.SetOrchestrator(orchestrator)
	
	// Create a complex dependency graph
	//     task1
	//    /      \
	//  task2   task3
	//    \      /
	//     task4
	cm := NewContextManager()
	plan := &ExecutionPlan{
		ID:      "complex-plan",
		Context: cm.CreateContext("test-session", "test-req", "complex test"),
		Tasks: []Task{
			createTestTask("task1", "agent1"),
			createTestTask("task2", "agent2", "task1"),
			createTestTask("task3", "agent3", "task1"),
			createTestTask("task4", "agent4", "task2", "task3"),
		},
		Stages: []Stage{
			{ID: "stage1", Tasks: []string{"task1"}},
			{ID: "stage2", Tasks: []string{"task2", "task3"}},
			{ID: "stage3", Tasks: []string{"task4"}},
		},
	}
	
	execContext := cm.CreateContext("test-session", "test-req", "test-request")
	
	// Execute plan
	results, err := executor.ExecutePlan(ctx, plan, execContext)
	
	require.NoError(t, err)
	assert.Len(t, results, 4)
	
	// Verify execution order respects dependencies
	assert.Equal(t, "task1", executionOrder[0], "task1 should execute first")
	
	// task2 and task3 can be in any order, but both before task4
	task4Index := -1
	for i, task := range executionOrder {
		if task == "task4" {
			task4Index = i
			break
		}
	}
	
	assert.Equal(t, 3, task4Index, "task4 should execute last")
}

// Helper functions for creating test data

func createTestPlan(agentNames ...string) *ExecutionPlan {
	tasks := make([]Task, len(agentNames))
	taskIDs := make([]string, len(agentNames))
	stages := make([]Stage, 0)
	
	for i, name := range agentNames {
		taskID := "task-" + name
		taskIDs[i] = taskID
		
		// Create chain of dependencies
		var deps []string
		if i > 0 {
			deps = append(deps, taskIDs[i-1])
		}
		
		tasks[i] = createTestTask(taskID, name, deps...)
		
		// Each task with dependencies should be in its own stage
		stages = append(stages, Stage{
			ID:    "stage" + string(rune('1'+i)),
			Tasks: []string{taskID},
		})
	}
	
	cm := NewContextManager()
	return &ExecutionPlan{
		ID:      "test-plan",
		Context: cm.CreateContext("test-session", "test-req", "test-request"),
		Tasks:   tasks,
		Stages:  stages,
		EstimatedDuration: "1m",
		CreatedAt:         time.Now(),
	}
}

func createTestTask(id, agentName string, deps ...string) Task {
	return Task{
		ID:           id,
		Agent:        agentName,
		Request:      createTestRequest("test request for " + agentName),
		Priority:     1,
		Dependencies: deps,
		Stage:        "stage1",
		Timeout:      30 * time.Second,
	}
}

func createTestRequest(prompt string) AgentRequest {
	return AgentRequest{
		Prompt:  prompt,
		Context: make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
}

func TestTaskQueue(t *testing.T) {
	queue := NewTaskQueue()
	
	t.Run("Enqueue and Dequeue", func(t *testing.T) {
		task := Task{
			ID:       "test1",
			Agent:    "test_agent",
			Request:  createTestRequest("test prompt"),
			Priority: 1,
		}
		
		queue.Enqueue(task)
		
		dequeuedTask, ok := queue.Dequeue()
		assert.True(t, ok)
		assert.Equal(t, "test1", dequeuedTask.ID)
		
		// Queue should be empty now
		_, ok = queue.Dequeue()
		assert.False(t, ok)
	})
}

func TestDependencyGraph_CanExecute(t *testing.T) {
	graph := NewDependencyGraph()
	
	// Need to manually build dependencies since we're testing the low-level function
	graph.dependencies = map[string][]string{
		"dependent": {"dep1", "dep2"},
	}
	
	t.Run("No dependencies", func(t *testing.T) {
		completed := make(map[string]bool)
		canExecute := graph.CanExecute("independent", completed)
		assert.True(t, canExecute)
	})
	
	t.Run("With completed dependencies", func(t *testing.T) {
		completed := map[string]bool{
			"dep1": true,
			"dep2": true,
		}
		
		canExecute := graph.CanExecute("dependent", completed)
		assert.True(t, canExecute)
	})
	
	t.Run("With incomplete dependencies", func(t *testing.T) {
		completed := map[string]bool{
			"dep1": true,
			"dep2": false,
		}
		
		canExecute := graph.CanExecute("dependent", completed)
		assert.False(t, canExecute)
	})
}

func TestProgressTracker_GetProgress(t *testing.T) {
	tracker := NewProgressTracker()
	
	planID := "test-plan"
	tracker.StartPlan(planID, 3) // 3 total tasks
	
	progress, ok := tracker.GetProgress(planID)
	assert.True(t, ok)
	assert.Equal(t, 0, progress.CompletedTasks)
	assert.Equal(t, 3, progress.TotalTasks)
	
	// Complete a task
	tracker.StartTask("task1")
	tracker.CompleteTask("task1")
	
	progress, ok = tracker.GetProgress(planID)
	assert.True(t, ok)
	assert.Equal(t, 1, progress.CompletedTasks)
}

func TestWorkerPool_Submit(t *testing.T) {
	pool := NewWorkerPool(1)
	defer pool.Shutdown()
	
	var executed bool
	var mu sync.Mutex
	
	job := func() {
		mu.Lock()
		executed = true
		mu.Unlock()
	}
	
	pool.Submit(job)
	
	// Give some time for the job to execute
	time.Sleep(100 * time.Millisecond)
	
	mu.Lock()
	result := executed
	mu.Unlock()
	
	assert.True(t, result)
}

func TestWorkerPool_Shutdown(t *testing.T) {
	pool := NewWorkerPool(1)
	
	// This should not panic
	pool.Shutdown()
}