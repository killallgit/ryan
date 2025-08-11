package scenarios

import (
	"time"

	"github.com/killallgit/ryan/pkg/orchestrator"
	"github.com/killallgit/ryan/pkg/orchestrator/testing/helpers"
)

// CreateSimpleScenarios creates basic single-agent test scenarios
func CreateSimpleScenarios() []*helpers.TestScenario {
	return []*helpers.TestScenario{
		// Tool calling scenario
		helpers.NewScenario("simple_tool_call").
			WithDescription("Simple tool calling scenario").
			WithInput("list files in the current directory").
			ExpectsAgent(orchestrator.AgentToolCaller).
			ExpectsToolCall("bash").
			ShouldComplete().
			Build(),

		// Code generation scenario
		helpers.NewScenario("simple_code_gen").
			WithDescription("Simple code generation scenario").
			WithInput("write a function to reverse a string").
			ExpectsAgent(orchestrator.AgentCodeGen).
			ExpectsOutput(helpers.OutputMatcher{
				Contains: []string{"func", "string"},
			}).
			ShouldComplete().
			Build(),

		// Reasoning scenario
		helpers.NewScenario("simple_reasoning").
			WithDescription("Simple reasoning scenario").
			WithInput("explain how binary search works").
			ExpectsAgent(orchestrator.AgentReasoner).
			ShouldComplete().
			Build(),

		// Search scenario
		helpers.NewScenario("simple_search").
			WithDescription("Simple search scenario").
			WithInput("find all Go files in the project").
			ExpectsAgent(orchestrator.AgentSearcher).
			ShouldComplete().
			Build(),
	}
}

// CreateComplexScenarios creates multi-agent test scenarios
func CreateComplexScenarios() []*helpers.TestScenario {
	return []*helpers.TestScenario{
		// Multi-step planning and execution
		helpers.NewScenario("plan_and_execute").
			WithDescription("Planning followed by execution").
			WithInput("create a new Go module with a main function").
			ExpectsAgent(orchestrator.AgentPlanner).
			ExpectsAgent(orchestrator.AgentCodeGen).
			ExpectsAgent(orchestrator.AgentToolCaller).
			ShouldComplete().
			WithMaxIterations(5).
			Build(),

		// Code generation with file operations
		helpers.NewScenario("code_and_file_ops").
			WithDescription("Generate code and save to file").
			WithInput("write a HTTP server and save it to server.go").
			ExpectsAgent(orchestrator.AgentCodeGen).
			ExpectsAgent(orchestrator.AgentToolCaller).
			ExpectsToolCall("file_write").
			ShouldComplete().
			Build(),

		// Research and implement
		helpers.NewScenario("research_and_implement").
			WithDescription("Search for examples then implement").
			WithInput("find how to use channels in Go and implement an example").
			ExpectsAgent(orchestrator.AgentSearcher).
			ExpectsAgent(orchestrator.AgentCodeGen).
			ShouldComplete().
			WithMaxIterations(4).
			Build(),
	}
}

// CreateFailureScenarios creates scenarios that test error handling
func CreateFailureScenarios() []*helpers.TestScenario {
	return []*helpers.TestScenario{
		// Agent failure with retry
		helpers.NewScenario("agent_failure_retry").
			WithDescription("Agent fails but succeeds on retry").
			WithInput("execute failing command").
			ExpectsAgent(orchestrator.AgentToolCaller).
			ExpectsFailure().
			ExpectsAgent(orchestrator.AgentToolCaller). // Retry
			ShouldComplete().
			WithMaxIterations(3).
			Build(),

		// Max iterations reached
		helpers.NewScenario("max_iterations").
			WithDescription("Task exceeds maximum iterations").
			WithInput("complex recursive task").
			WithMaxIterations(2).
			ShouldFail().
			Build(),

		// Invalid intent
		helpers.NewScenario("unknown_intent").
			WithDescription("Unknown intent falls back to reasoner").
			WithInput("xyzabc123 unknown command").
			ExpectsAgent(orchestrator.AgentReasoner).
			ShouldComplete().
			Build(),
	}
}

// CreatePerformanceScenarios creates scenarios for performance testing
func CreatePerformanceScenarios() []*helpers.TestScenario {
	return []*helpers.TestScenario{
		// Fast routing decision
		helpers.NewScenario("fast_routing").
			WithDescription("Quick routing decision").
			WithInput("simple query").
			WithTimeout(1 * time.Second).
			ShouldComplete().
			Build(),

		// Concurrent execution
		helpers.NewScenario("concurrent_safe").
			WithDescription("Concurrent execution safety").
			WithInput("parallel processing test").
			ShouldComplete().
			Build(),
	}
}

// CreateRegressionScenarios creates critical scenarios that must always work
func CreateRegressionScenarios() []*helpers.TestScenario {
	return []*helpers.TestScenario{
		// Critical user journey - file operations
		helpers.NewScenario("critical_file_ops").
			WithDescription("Critical file operations workflow").
			WithInput("read package.json, modify version, write back").
			ExpectsAgent(orchestrator.AgentToolCaller).
			ExpectsToolCall("file_read").
			ExpectsToolCall("file_write").
			ShouldComplete().
			Build(),

		// Critical user journey - code assistance
		helpers.NewScenario("critical_code_help").
			WithDescription("Critical code assistance workflow").
			WithInput("help me debug this function").
			ExpectsAgent(orchestrator.AgentReasoner).
			ShouldComplete().
			Build(),

		// Critical user journey - project analysis
		helpers.NewScenario("critical_project_analysis").
			WithDescription("Critical project analysis workflow").
			WithInput("analyze the codebase structure").
			ExpectsAgent(orchestrator.AgentSearcher).
			ShouldComplete().
			Build(),
	}
}

// GetScenarioByName returns a scenario by name
func GetScenarioByName(name string) *helpers.TestScenario {
	allScenarios := append(
		append(
			append(CreateSimpleScenarios(), CreateComplexScenarios()...),
			CreateFailureScenarios()...),
		CreatePerformanceScenarios()...)

	for _, scenario := range allScenarios {
		if scenario.Name == name {
			return scenario
		}
	}
	return nil
}

// GetAllScenarios returns all test scenarios
func GetAllScenarios() []*helpers.TestScenario {
	return append(
		append(
			append(
				append(CreateSimpleScenarios(), CreateComplexScenarios()...),
				CreateFailureScenarios()...),
			CreatePerformanceScenarios()...),
		CreateRegressionScenarios()...)
}

// ScenariosByCategory returns scenarios grouped by category
func ScenariosByCategory() map[string][]*helpers.TestScenario {
	return map[string][]*helpers.TestScenario{
		"simple":      CreateSimpleScenarios(),
		"complex":     CreateComplexScenarios(),
		"failure":     CreateFailureScenarios(),
		"performance": CreatePerformanceScenarios(),
		"regression":  CreateRegressionScenarios(),
	}
}
