package helpers

import (
	"regexp"
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/orchestrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertRoutedTo verifies that the task was routed to the expected agent
func AssertRoutedTo(t *testing.T, result *orchestrator.TaskResult, expectedAgent orchestrator.AgentType) {
	require.NotEmpty(t, result.History, "Task should have execution history")

	found := false
	for _, response := range result.History {
		if response.AgentType == expectedAgent {
			found = true
			break
		}
	}

	assert.True(t, found, "Task should have been routed to agent %s", expectedAgent)
}

// AssertFirstRoutedTo verifies the first agent in the execution history
func AssertFirstRoutedTo(t *testing.T, result *orchestrator.TaskResult, expectedAgent orchestrator.AgentType) {
	require.NotEmpty(t, result.History, "Task should have execution history")

	firstResponse := result.History[0]
	assert.Equal(t, expectedAgent, firstResponse.AgentType,
		"First agent should be %s, got %s", expectedAgent, firstResponse.AgentType)
}

// AssertToolsCalled verifies that specific tools were called
func AssertToolsCalled(t *testing.T, result *orchestrator.TaskResult, expectedTools []string) {
	var calledTools []string
	for _, response := range result.History {
		for _, toolCall := range response.ToolsCalled {
			calledTools = append(calledTools, toolCall.Name)
		}
	}

	for _, expectedTool := range expectedTools {
		assert.Contains(t, calledTools, expectedTool,
			"Tool %s should have been called. Called tools: %v", expectedTool, calledTools)
	}
}

// AssertToolCalled verifies that a specific tool was called
func AssertToolCalled(t *testing.T, result *orchestrator.TaskResult, expectedTool string) {
	AssertToolsCalled(t, result, []string{expectedTool})
}

// AssertCompletedWithin verifies task completed within expected iterations
func AssertCompletedWithin(t *testing.T, result *orchestrator.TaskResult, maxIterations int) {
	assert.LessOrEqual(t, len(result.History), maxIterations,
		"Task should complete within %d iterations, took %d", maxIterations, len(result.History))
}

// AssertFeedbackLoopCount verifies the number of feedback loop iterations
func AssertFeedbackLoopCount(t *testing.T, result *orchestrator.TaskResult, expected int) {
	assert.Equal(t, expected, len(result.History),
		"Expected %d feedback loop iterations, got %d", expected, len(result.History))
}

// AssertStatus verifies the final status
func AssertStatus(t *testing.T, result *orchestrator.TaskResult, expectedStatus orchestrator.Status) {
	assert.Equal(t, expectedStatus, result.Status,
		"Expected status %s, got %s", expectedStatus, result.Status)
}

// AssertCompleted verifies the task completed successfully
func AssertCompleted(t *testing.T, result *orchestrator.TaskResult) {
	AssertStatus(t, result, orchestrator.StatusCompleted)
}

// AssertFailed verifies the task failed
func AssertFailed(t *testing.T, result *orchestrator.TaskResult) {
	AssertStatus(t, result, orchestrator.StatusFailed)
}

// AssertContains verifies the result contains specific text
func AssertContains(t *testing.T, result *orchestrator.TaskResult, expectedText string) {
	assert.Contains(t, result.Result, expectedText,
		"Result should contain '%s'", expectedText)
}

// AssertMatches verifies the result matches a regex pattern
func AssertMatches(t *testing.T, result *orchestrator.TaskResult, pattern string) {
	matched, err := regexp.MatchString(pattern, result.Result)
	require.NoError(t, err, "Invalid regex pattern: %s", pattern)
	assert.True(t, matched, "Result should match pattern '%s'. Result: %s", pattern, result.Result)
}

// AssertResponseCount verifies the number of agent responses
func AssertResponseCount(t *testing.T, result *orchestrator.TaskResult, expected int) {
	assert.Len(t, result.History, expected,
		"Expected %d agent responses, got %d", expected, len(result.History))
}

// AssertAgentSequence verifies the sequence of agents called
func AssertAgentSequence(t *testing.T, result *orchestrator.TaskResult, expectedSequence []orchestrator.AgentType) {
	require.Len(t, result.History, len(expectedSequence),
		"Expected %d responses for sequence verification", len(expectedSequence))

	for i, expectedAgent := range expectedSequence {
		actualAgent := result.History[i].AgentType
		assert.Equal(t, expectedAgent, actualAgent,
			"Step %d: expected agent %s, got %s", i+1, expectedAgent, actualAgent)
	}
}

// AssertNoErrors verifies no errors occurred during execution
func AssertNoErrors(t *testing.T, result *orchestrator.TaskResult) {
	for i, response := range result.History {
		assert.Nil(t, response.Error,
			"Response %d from %s should not have error: %v", i+1, response.AgentType, response.Error)
	}
}

// AssertHasError verifies that at least one error occurred
func AssertHasError(t *testing.T, result *orchestrator.TaskResult) {
	hasError := false
	for _, response := range result.History {
		if response.Error != nil {
			hasError = true
			break
		}
	}
	assert.True(t, hasError, "Expected at least one error in execution history")
}

// AssertExecutionTime verifies execution time is within bounds
func AssertExecutionTime(t *testing.T, result *orchestrator.TaskResult, maxDuration, minDuration *time.Duration) {
	if maxDuration != nil {
		assert.LessOrEqual(t, result.Duration.Nanoseconds(), maxDuration.Nanoseconds(),
			"Execution should complete within %v, took %v", *maxDuration, result.Duration)
	}

	if minDuration != nil {
		assert.GreaterOrEqual(t, result.Duration.Nanoseconds(), minDuration.Nanoseconds(),
			"Execution should take at least %v, took %v", *minDuration, result.Duration)
	}
}

// AssertMetadata verifies metadata exists
func AssertMetadata(t *testing.T, result *orchestrator.TaskResult, key string, expectedValue interface{}) {
	require.Contains(t, result.Metadata, key, "Metadata should contain key '%s'", key)
	assert.Equal(t, expectedValue, result.Metadata[key],
		"Metadata[%s] should equal %v, got %v", key, expectedValue, result.Metadata[key])
}

// AssertScenario runs assertions defined in a test scenario
func AssertScenario(t *testing.T, result *orchestrator.TaskResult, scenario *TestScenario) {
	for _, assertion := range scenario.Assertions {
		switch assertion.Type {
		case AssertionRouting:
			if agentType, ok := assertion.Expected.(orchestrator.AgentType); ok {
				AssertRoutedTo(t, result, agentType)
			}

		case AssertionCompletion:
			if shouldComplete, ok := assertion.Expected.(bool); ok {
				if shouldComplete {
					AssertCompleted(t, result)
				} else {
					AssertFailed(t, result)
				}
			}

		case AssertionToolCall:
			if toolName, ok := assertion.Expected.(string); ok {
				AssertToolCalled(t, result, toolName)
			}

		case AssertionIterationCount:
			if count, ok := assertion.Expected.(int); ok {
				AssertFeedbackLoopCount(t, result, count)
			}

		case AssertionStatus:
			if status, ok := assertion.Expected.(orchestrator.Status); ok {
				AssertStatus(t, result, status)
			}
		}
	}
}

// AssertOutputMatcher verifies output matches the given matcher
func AssertOutputMatcher(t *testing.T, output string, matcher OutputMatcher) {
	if matcher.Exact != "" {
		assert.Equal(t, matcher.Exact, output, "Output should exactly match")
		return
	}

	if matcher.Pattern != "" {
		AssertMatches(t, &orchestrator.TaskResult{Result: output}, matcher.Pattern)
	}

	for _, expected := range matcher.Contains {
		assert.Contains(t, output, expected, "Output should contain '%s'", expected)
	}

	for _, notExpected := range matcher.NotContains {
		assert.NotContains(t, output, notExpected, "Output should not contain '%s'", notExpected)
	}
}

// AssertFlow verifies the expected flow matches the actual execution
func AssertFlow(t *testing.T, result *orchestrator.TaskResult, expectedFlow []ExpectedStep) {
	for i, step := range expectedFlow {
		if i >= len(result.History) {
			t.Errorf("Expected step %d (agent %s), but execution stopped at %d steps",
				i+1, step.AgentType, len(result.History))
			continue
		}

		response := result.History[i]

		// Check agent type
		assert.Equal(t, step.AgentType, response.AgentType,
			"Step %d: expected agent %s, got %s", i+1, step.AgentType, response.AgentType)

		// Check if failure was expected
		if step.ShouldFail {
			assert.Equal(t, "failed", response.Status,
				"Step %d should have failed", i+1)
		} else {
			assert.NotEqual(t, "failed", response.Status,
				"Step %d should not have failed: %v", i+1, response.Error)
		}

		// Check output if specified
		if step.ExpectedOutput.Contains != nil || step.ExpectedOutput.Pattern != "" || step.ExpectedOutput.Exact != "" {
			AssertOutputMatcher(t, response.Response, step.ExpectedOutput)
		}
	}
}
