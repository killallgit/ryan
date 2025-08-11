package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/orchestrator"
	"github.com/killallgit/ryan/pkg/orchestrator/testing/helpers"
	"github.com/killallgit/ryan/pkg/orchestrator/testing/scenarios"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestratorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get all simple scenarios for testing
	simpleScenarios := scenarios.CreateSimpleScenarios()

	for _, scenario := range simpleScenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Create orchestrator with mock agents
			orch, err := helpers.NewOrchestrator().
				WithDefaultAgents().
				WithMaxIterations(scenario.MaxIterations).
				Build()
			require.NoError(t, err)

			// Execute the scenario
			ctx, cancel := context.WithTimeout(context.Background(), scenario.Timeout)
			defer cancel()

			result, err := orch.Execute(ctx, scenario.Input)
			require.NoError(t, err, "Scenario %s should not error", scenario.Name)

			// Run scenario assertions
			helpers.AssertScenario(t, result, scenario)

			// Basic sanity checks
			assert.NotEmpty(t, result.Result, "Result should have content")
			assert.NotEmpty(t, result.History, "Should have execution history")
			assert.Greater(t, result.Duration, time.Duration(0), "Should have measurable duration")
		})
	}
}

func TestOrchestratorComplexScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	complexScenarios := scenarios.CreateComplexScenarios()

	for _, scenario := range complexScenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Configure agents with specific responses for multi-agent scenarios
			orch, err := helpers.NewOrchestrator().
				WithDefaultAgents().
				WithScenarioAgents(scenario).
				WithMaxIterations(scenario.MaxIterations).
				Build()
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), scenario.Timeout)
			defer cancel()

			result, err := orch.Execute(ctx, scenario.Input)
			require.NoError(t, err)

			helpers.AssertScenario(t, result, scenario)

			// Complex scenarios should involve multiple agents
			if len(scenario.ExpectedFlow) > 1 {
				assert.GreaterOrEqual(t, len(result.History), 2, "Complex scenarios should use multiple agents")
			}
		})
	}
}

func TestOrchestratorFailureRecovery(t *testing.T) {
	failureScenarios := scenarios.CreateFailureScenarios()

	for _, scenario := range failureScenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			orch, err := helpers.NewOrchestrator().
				WithDefaultAgents().
				WithScenarioAgents(scenario).
				WithMaxIterations(scenario.MaxIterations).
				Build()
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), scenario.Timeout)
			defer cancel()

			result, err := orch.Execute(ctx, scenario.Input)

			// Some failure scenarios expect errors, others expect graceful handling
			if scenario.Name == "max_iterations" {
				// Max iterations should return an error and the result should indicate failure
				if assert.Error(t, err, "Max iterations should return an error") {
					assert.Contains(t, err.Error(), "max iterations", "Error should mention max iterations")
				}
				if result != nil {
					assert.Equal(t, orchestrator.StatusFailed, result.Status)
				}
			} else {
				// Most scenarios should complete even with retries
				require.NoError(t, err)
				helpers.AssertScenario(t, result, scenario)
			}
		})
	}
}

func TestOrchestratorPerformance(t *testing.T) {
	// Test basic performance characteristics
	orch, err := helpers.NewOrchestrator().
		WithDefaultAgents().
		Build()
	require.NoError(t, err)

	// Test quick routing decisions
	start := time.Now()
	ctx := context.Background()

	result, err := orch.Execute(ctx, "simple test query")
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, duration, 1*time.Second, "Simple queries should complete quickly")
	helpers.AssertCompleted(t, result)
}

func TestOrchestratorConcurrency(t *testing.T) {
	orch, err := helpers.NewOrchestrator().
		WithDefaultAgents().
		Build()
	require.NoError(t, err)

	const numConcurrent = 5
	results := make([]*orchestrator.TaskResult, numConcurrent)
	errors := make([]error, numConcurrent)

	// Run concurrent executions
	for i := 0; i < numConcurrent; i++ {
		go func(index int) {
			ctx := context.Background()
			query := fmt.Sprintf("concurrent test %d", index)
			results[index], errors[index] = orch.Execute(ctx, query)
		}(i)
	}

	// Wait for all to complete
	time.Sleep(200 * time.Millisecond)

	// Verify all completed successfully
	for i := 0; i < numConcurrent; i++ {
		assert.NoError(t, errors[i], "Concurrent execution %d should not error", i)
		if results[i] != nil {
			helpers.AssertCompleted(t, results[i])
		}
	}
}

func TestOrchestratorStateManagement(t *testing.T) {
	orch, err := helpers.NewOrchestrator().
		WithDefaultAgents().
		Build()
	require.NoError(t, err)

	ctx := context.Background()

	// Execute multiple tasks
	result1, err := orch.Execute(ctx, "first task")
	require.NoError(t, err)

	result2, err := orch.Execute(ctx, "second task")
	require.NoError(t, err)

	result3, err := orch.Execute(ctx, "third task")
	require.NoError(t, err)

	// Each task should have unique ID
	assert.NotEqual(t, result1.ID, result2.ID)
	assert.NotEqual(t, result2.ID, result3.ID)
	assert.NotEqual(t, result1.ID, result3.ID)

	// All should be completed
	helpers.AssertCompleted(t, result1)
	helpers.AssertCompleted(t, result2)
	helpers.AssertCompleted(t, result3)
}

func TestOrchestratorRegressionSuite(t *testing.T) {
	// Run all regression scenarios to ensure no breaking changes
	regressionScenarios := scenarios.CreateRegressionScenarios()

	for _, scenario := range regressionScenarios {
		t.Run("regression_"+scenario.Name, func(t *testing.T) {
			orch, err := helpers.NewOrchestrator().
				WithDefaultAgents().
				WithScenarioAgents(scenario).
				Build()
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			result, err := orch.Execute(ctx, scenario.Input)
			require.NoError(t, err, "Regression test %s must not fail", scenario.Name)

			helpers.AssertScenario(t, result, scenario)
			helpers.AssertCompleted(t, result)
		})
	}
}

func BenchmarkOrchestratorExecution(b *testing.B) {
	orch, err := helpers.NewOrchestrator().
		WithDefaultAgents().
		Build()
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, err := orch.Execute(ctx, "benchmark test query")
		require.NoError(b, err)
		require.NotNil(b, result)
	}
}

func BenchmarkOrchestratorRouting(b *testing.B) {
	orch, err := helpers.NewOrchestrator().
		WithDefaultAgents().
		Build()
	require.NoError(b, err)

	ctx := context.Background()
	intent := &orchestrator.TaskIntent{
		Type:       "tool_use",
		Confidence: 0.9,
	}

	state := helpers.NewState("benchmark").Build()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		decision, err := orch.Route(ctx, intent, state)
		require.NoError(b, err)
		require.NotNil(b, decision)
	}
}

func BenchmarkIntentAnalysis(b *testing.B) {
	orch, err := helpers.NewOrchestrator().
		WithDefaultAgents().
		Build()
	require.NoError(b, err)

	ctx := context.Background()
	query := "analyze this benchmark query"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		intent, err := orch.AnalyzeIntent(ctx, query)
		require.NoError(b, err)
		require.NotNil(b, intent)
	}
}
