package helpers

import (
	"time"

	"github.com/killallgit/ryan/pkg/orchestrator"
	"github.com/killallgit/ryan/pkg/orchestrator/testing/mocks"
)

// ScenarioBuilder builds test scenarios with a fluent API
type ScenarioBuilder struct {
	scenario *TestScenario
}

// TestScenario represents a complete test scenario
type TestScenario struct {
	Name          string
	Description   string
	Input         string
	ExpectedFlow  []ExpectedStep
	Assertions    []Assertion
	MaxIterations int
	Timeout       time.Duration
	MockResponses map[orchestrator.AgentType][]orchestrator.AgentResponse
}

// ExpectedStep represents an expected step in the flow
type ExpectedStep struct {
	AgentType      orchestrator.AgentType
	Action         string
	ExpectedOutput OutputMatcher
	ShouldFail     bool
}

// OutputMatcher defines how to match output
type OutputMatcher struct {
	Contains    []string
	NotContains []string
	Exact       string
	Pattern     string // Regex pattern
}

// Assertion represents a test assertion
type Assertion struct {
	Type        AssertionType
	Expected    interface{}
	Description string
}

// AssertionType defines types of assertions
type AssertionType string

const (
	AssertionRouting        AssertionType = "routing"
	AssertionCompletion     AssertionType = "completion"
	AssertionToolCall       AssertionType = "tool_call"
	AssertionIterationCount AssertionType = "iteration_count"
	AssertionStatus         AssertionType = "status"
)

// NewScenario creates a new scenario builder
func NewScenario(name string) *ScenarioBuilder {
	return &ScenarioBuilder{
		scenario: &TestScenario{
			Name:          name,
			ExpectedFlow:  make([]ExpectedStep, 0),
			Assertions:    make([]Assertion, 0),
			MaxIterations: 10,
			Timeout:       30 * time.Second,
			MockResponses: make(map[orchestrator.AgentType][]orchestrator.AgentResponse),
		},
	}
}

// WithDescription adds a description
func (b *ScenarioBuilder) WithDescription(desc string) *ScenarioBuilder {
	b.scenario.Description = desc
	return b
}

// WithInput sets the input query
func (b *ScenarioBuilder) WithInput(input string) *ScenarioBuilder {
	b.scenario.Input = input
	return b
}

// ExpectsAgent adds an expected agent interaction
func (b *ScenarioBuilder) ExpectsAgent(agentType orchestrator.AgentType) *ScenarioBuilder {
	step := ExpectedStep{
		AgentType: agentType,
		Action:    "execute",
	}
	b.scenario.ExpectedFlow = append(b.scenario.ExpectedFlow, step)
	return b
}

// ExpectsToolCall adds an expected tool call
func (b *ScenarioBuilder) ExpectsToolCall(tool string) *ScenarioBuilder {
	b.scenario.Assertions = append(b.scenario.Assertions, Assertion{
		Type:        AssertionToolCall,
		Expected:    tool,
		Description: "Should call tool: " + tool,
	})
	return b
}

// ExpectsOutput adds expected output matching
func (b *ScenarioBuilder) ExpectsOutput(matcher OutputMatcher) *ScenarioBuilder {
	if len(b.scenario.ExpectedFlow) > 0 {
		b.scenario.ExpectedFlow[len(b.scenario.ExpectedFlow)-1].ExpectedOutput = matcher
	}
	return b
}

// ExpectsFailure marks the last step as expected to fail
func (b *ScenarioBuilder) ExpectsFailure() *ScenarioBuilder {
	if len(b.scenario.ExpectedFlow) > 0 {
		b.scenario.ExpectedFlow[len(b.scenario.ExpectedFlow)-1].ShouldFail = true
	}
	return b
}

// ShouldComplete adds a completion assertion
func (b *ScenarioBuilder) ShouldComplete() *ScenarioBuilder {
	b.scenario.Assertions = append(b.scenario.Assertions, Assertion{
		Type:        AssertionCompletion,
		Expected:    true,
		Description: "Should complete successfully",
	})
	return b
}

// ShouldFail adds a failure assertion
func (b *ScenarioBuilder) ShouldFail() *ScenarioBuilder {
	b.scenario.Assertions = append(b.scenario.Assertions, Assertion{
		Type:        AssertionStatus,
		Expected:    orchestrator.StatusFailed,
		Description: "Should fail",
	})
	return b
}

// WithMaxIterations sets the maximum iterations
func (b *ScenarioBuilder) WithMaxIterations(max int) *ScenarioBuilder {
	b.scenario.MaxIterations = max
	return b
}

// WithTimeout sets the timeout
func (b *ScenarioBuilder) WithTimeout(timeout time.Duration) *ScenarioBuilder {
	b.scenario.Timeout = timeout
	return b
}

// WithMockResponse adds a mock response for an agent
func (b *ScenarioBuilder) WithMockResponse(agentType orchestrator.AgentType, response orchestrator.AgentResponse) *ScenarioBuilder {
	if b.scenario.MockResponses[agentType] == nil {
		b.scenario.MockResponses[agentType] = make([]orchestrator.AgentResponse, 0)
	}
	b.scenario.MockResponses[agentType] = append(b.scenario.MockResponses[agentType], response)
	return b
}

// Build returns the completed scenario
func (b *ScenarioBuilder) Build() *TestScenario {
	return b.scenario
}

// StateBuilder builds test states
type StateBuilder struct {
	state *orchestrator.TaskState
}

// NewState creates a new state builder
func NewState(query string) *StateBuilder {
	return &StateBuilder{
		state: &orchestrator.TaskState{
			ID:           "test-" + time.Now().Format("20060102-150405"),
			Query:        query,
			CurrentPhase: orchestrator.PhaseAnalysis,
			History:      make([]orchestrator.AgentResponse, 0),
			Status:       orchestrator.StatusPending,
			Metadata:     make(map[string]interface{}),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}
}

// WithIntent sets the intent
func (b *StateBuilder) WithIntent(intentType string, confidence float64) *StateBuilder {
	b.state.Intent = &orchestrator.TaskIntent{
		Type:       intentType,
		Confidence: confidence,
	}
	return b
}

// WithPhase sets the current phase
func (b *StateBuilder) WithPhase(phase orchestrator.Phase) *StateBuilder {
	b.state.CurrentPhase = phase
	return b
}

// WithStatus sets the status
func (b *StateBuilder) WithStatus(status orchestrator.Status) *StateBuilder {
	b.state.Status = status
	return b
}

// WithHistory adds to the history
func (b *StateBuilder) WithHistory(responses ...orchestrator.AgentResponse) *StateBuilder {
	b.state.History = append(b.state.History, responses...)
	return b
}

// WithMetadata adds metadata
func (b *StateBuilder) WithMetadata(key string, value interface{}) *StateBuilder {
	b.state.Metadata[key] = value
	return b
}

// Build returns the completed state
func (b *StateBuilder) Build() *orchestrator.TaskState {
	return b.state
}

// OrchestratorBuilder builds test orchestrators
type OrchestratorBuilder struct {
	llm     *mocks.MockLLM
	agents  map[orchestrator.AgentType]*mocks.MockAgent
	options []orchestrator.Option
}

// NewOrchestrator creates a new orchestrator builder
func NewOrchestrator() *OrchestratorBuilder {
	return &OrchestratorBuilder{
		agents:  make(map[orchestrator.AgentType]*mocks.MockAgent),
		options: make([]orchestrator.Option, 0),
	}
}

// WithMockLLM sets the mock LLM
func (b *OrchestratorBuilder) WithMockLLM(llm *mocks.MockLLM) *OrchestratorBuilder {
	b.llm = llm
	return b
}

// WithAgent adds a mock agent
func (b *OrchestratorBuilder) WithAgent(agent *mocks.MockAgent) *OrchestratorBuilder {
	b.agents[agent.Type] = agent
	return b
}

// WithDefaultAgents adds default mock agents
func (b *OrchestratorBuilder) WithDefaultAgents() *OrchestratorBuilder {
	factory := mocks.NewMockAgentFactory()
	agents := factory.CreateDefaultAgents()
	for _, agent := range agents {
		b.agents[agent.Type] = agent
	}
	return b
}

// WithScenarioAgents configures agents for specific scenarios
func (b *OrchestratorBuilder) WithScenarioAgents(scenario *TestScenario) *OrchestratorBuilder {
	// Configure agents based on scenario requirements
	if scenario.Name == "max_iterations" {
		// Create agents that always require more iterations by suggesting next actions
		for agentType, agent := range b.agents {
			// Configure agents to always suggest continuing to next agent
			agent.WithBehavior(mocks.BehaviorConfig{
				PartialSuccess: true, // Always return partial results requiring more work
			})
			b.agents[agentType] = agent
		}
	}

	return b
}

// WithMaxIterations sets max iterations
func (b *OrchestratorBuilder) WithMaxIterations(max int) *OrchestratorBuilder {
	b.options = append(b.options, orchestrator.WithMaxIterations(max))
	return b
}

// Build creates the orchestrator
func (b *OrchestratorBuilder) Build() (*orchestrator.Orchestrator, error) {
	if b.llm == nil {
		b.llm = mocks.NewMockLLM()
	}

	o, err := orchestrator.New(b.llm, b.options...)
	if err != nil {
		return nil, err
	}

	// Register agents
	for _, agent := range b.agents {
		if err := o.RegisterAgent(agent.Type, agent); err != nil {
			return nil, err
		}
	}

	return o, nil
}
