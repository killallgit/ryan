package fixtures

import (
	"context"
	"time"

	"github.com/killallgit/ryan/pkg/agents"
)

// CreateTestAgentRequest creates a test agent request
func CreateTestAgentRequest(prompt string) agents.AgentRequest {
	return agents.AgentRequest{
		Prompt:  prompt,
		Context: make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
}

// CreateTestAgentResult creates a test agent result
func CreateTestAgentResult(success bool, summary string) agents.AgentResult {
	return agents.AgentResult{
		Success: success,
		Summary: summary,
		Details: "Test result details",
	}
}

// CreateTestTask creates a test task
func CreateTestTask(id, agentName string, deps ...string) agents.Task {
	return agents.Task{
		ID:           id,
		Agent:        agentName,
		Request:      CreateTestAgentRequest("test request for " + agentName),
		Priority:     1,
		Dependencies: deps,
		Stage:        "stage1",
		Timeout:      30 * time.Second,
	}
}

// CreateTestExecutionPlan creates a test execution plan
func CreateTestExecutionPlan(taskNames ...string) *agents.ExecutionPlan {
	tasks := make([]agents.Task, len(taskNames))
	taskIDs := make([]string, len(taskNames))

	for i, name := range taskNames {
		taskID := "task-" + name
		taskIDs[i] = taskID

		// Create chain of dependencies
		var deps []string
		if i > 0 {
			deps = append(deps, taskIDs[i-1])
		}

		tasks[i] = CreateTestTask(taskID, name, deps...)
	}

	cm := agents.NewContextManager()
	return &agents.ExecutionPlan{
		ID:      "test-plan",
		Context: cm.CreateContext("test-session", "test-req", "test-request"),
		Tasks:   tasks,
		Stages: []agents.Stage{
			{
				ID:    "stage1",
				Tasks: taskIDs,
			},
		},
		EstimatedDuration: "1m",
		CreatedAt:         time.Now(),
	}
}

// CreateTestTaskResult creates a test task result
func CreateTestTaskResult(taskID string, success bool) agents.TaskResult {
	return agents.TaskResult{
		Task:      CreateTestTask(taskID, "test-agent"),
		Result:    CreateTestAgentResult(success, "Task completed"),
		Error:     nil,
		StartTime: time.Now().Add(-1 * time.Minute),
		EndTime:   time.Now(),
	}
}

// CreateTestContext creates a test context with common values
func CreateTestContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "test", true)
	return ctx
}

// AgentTestCase represents a test case for agent testing
type AgentTestCase struct {
	Name           string
	Request        string
	ExpectedHandle bool
	Confidence     float64
	ExecuteError   error
	ExpectedResult agents.AgentResult
}

// GetStandardAgentTestCases returns standard test cases for agents
func GetStandardAgentTestCases() []AgentTestCase {
	return []AgentTestCase{
		{
			Name:           "Simple request",
			Request:        "test request",
			ExpectedHandle: true,
			Confidence:     1.0,
			ExecuteError:   nil,
			ExpectedResult: CreateTestAgentResult(true, "Success"),
		},
		{
			Name:           "Complex request",
			Request:        "analyze code and find bugs",
			ExpectedHandle: true,
			Confidence:     0.8,
			ExecuteError:   nil,
			ExpectedResult: CreateTestAgentResult(true, "Analysis complete"),
		},
		{
			Name:           "Unsupported request",
			Request:        "unsupported operation",
			ExpectedHandle: false,
			Confidence:     0.0,
			ExecuteError:   nil,
			ExpectedResult: CreateTestAgentResult(false, "Cannot handle"),
		},
	}
}
