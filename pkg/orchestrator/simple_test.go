package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

// Simple mock LLM for testing without external dependencies
type SimpleMockLLM struct {
	responses      []string
	index          int
	intentResponse *TaskIntent
	mu             sync.Mutex
}

func NewSimpleMockLLM() *SimpleMockLLM {
	return &SimpleMockLLM{
		responses: []string{"Mock response"},
		index:     0,
	}
}

func (m *SimpleMockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.index >= len(m.responses) {
		return "Default mock response", nil
	}

	response := m.responses[m.index]
	m.index++
	return response, nil
}

func (m *SimpleMockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	prompt := ""
	for _, msg := range messages {
		for _, part := range msg.Parts {
			if textPart, ok := part.(llms.TextContent); ok {
				prompt += textPart.Text + " "
			}
		}
	}

	response, err := m.Call(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{Content: response},
		},
	}, nil
}

func (m *SimpleMockLLM) CreateEmbedding(ctx context.Context, inputTexts []string) ([][]float32, error) {
	return nil, errors.New("embeddings not supported")
}

func (m *SimpleMockLLM) WithIntentResponse(intent *TaskIntent) *SimpleMockLLM {
	m.intentResponse = intent
	// Set appropriate JSON response
	m.responses = []string{`{
		"type": "` + intent.Type + `",
		"confidence": ` + fmt.Sprintf("%.2f", intent.Confidence) + `,
		"required_capabilities": ["general"],
		"reasoning": "Mock intent analysis"
	}`}
	return m
}

// Simple mock agent
type SimpleMockAgent struct {
	agentType AgentType
	responses []AgentResponse
	index     int
	mu        sync.Mutex
}

func NewSimpleMockAgent(agentType AgentType) *SimpleMockAgent {
	return &SimpleMockAgent{
		agentType: agentType,
		responses: []AgentResponse{
			{
				AgentType: agentType,
				Response:  "Mock agent response",
				Status:    "success",
				Timestamp: time.Now(),
			},
		},
		index: 0,
	}
}

func (a *SimpleMockAgent) Execute(ctx context.Context, decision *RouteDecision, state *TaskState) (*AgentResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.index >= len(a.responses) {
		return &AgentResponse{
			AgentType: a.agentType,
			Response:  "Default mock response",
			Status:    "success",
			Timestamp: time.Now(),
		}, nil
	}

	response := a.responses[a.index]
	a.index++
	response.Timestamp = time.Now()
	return &response, nil
}

func (a *SimpleMockAgent) GetCapabilities() []string {
	return []string{"mock"}
}

func (a *SimpleMockAgent) GetType() AgentType {
	return a.agentType
}

// Basic tests without external dependencies
func TestSimpleOrchestratorCreation(t *testing.T) {
	mockLLM := NewSimpleMockLLM()

	orchestrator, err := New(mockLLM)
	require.NoError(t, err)
	assert.NotNil(t, orchestrator)
}

func TestSimpleIntentAnalysis(t *testing.T) {
	intent := &TaskIntent{
		Type:       "tool_use",
		Confidence: 0.9,
	}

	mockLLM := NewSimpleMockLLM().WithIntentResponse(intent)
	orchestrator, err := New(mockLLM)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := orchestrator.AnalyzeIntent(ctx, "test query")

	require.NoError(t, err)
	assert.Equal(t, "tool_use", result.Type)
	assert.Greater(t, result.Confidence, 0.0)
}

func TestSimpleRouting(t *testing.T) {
	mockLLM := NewSimpleMockLLM()
	orchestrator, err := New(mockLLM)
	require.NoError(t, err)

	// Register a mock agent
	mockAgent := NewSimpleMockAgent(AgentToolCaller)
	err = orchestrator.RegisterAgent(AgentToolCaller, mockAgent)
	require.NoError(t, err)

	// Create test state and intent
	state := &TaskState{
		ID:    "test",
		Query: "test query",
	}

	intent := &TaskIntent{
		Type:       "tool_use",
		Confidence: 0.9,
	}

	ctx := context.Background()
	decision, err := orchestrator.Route(ctx, intent, state)

	require.NoError(t, err)
	assert.Equal(t, AgentToolCaller, decision.TargetAgent)
}

func TestSimpleExecution(t *testing.T) {
	intent := &TaskIntent{
		Type:       "tool_use",
		Confidence: 0.9,
	}

	mockLLM := NewSimpleMockLLM().WithIntentResponse(intent)
	orchestrator, err := New(mockLLM)
	require.NoError(t, err)

	// Register mock agent
	mockAgent := NewSimpleMockAgent(AgentToolCaller)
	err = orchestrator.RegisterAgent(AgentToolCaller, mockAgent)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := orchestrator.Execute(ctx, "test query")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test query", result.Query)
	assert.Equal(t, StatusCompleted, result.Status)
	assert.NotEmpty(t, result.History)
}

func TestStateManagerBasics(t *testing.T) {
	sm := NewStateManager()

	state := sm.CreateState("test query")
	assert.NotNil(t, state)
	assert.NotEmpty(t, state.ID)
	assert.Equal(t, "test query", state.Query)

	retrieved, err := sm.GetState(state.ID)
	require.NoError(t, err)
	assert.Equal(t, state.ID, retrieved.ID)
}

func TestRouterBasics(t *testing.T) {
	registry := NewAgentRegistry()
	router := NewRouter(registry)

	assert.NotNil(t, router)
	assert.NotEmpty(t, router.rules)

	// Register an agent
	mockAgent := NewSimpleMockAgent(AgentReasoner)
	registry.Register(AgentReasoner, mockAgent)

	ctx := context.Background()
	intent := &TaskIntent{
		Type:       "reasoning",
		Confidence: 0.8,
	}

	agentType, err := router.Route(ctx, intent)
	require.NoError(t, err)
	assert.Equal(t, AgentReasoner, agentType)
}

func TestAgentRegistryBasics(t *testing.T) {
	registry := NewAgentRegistry()

	mockAgent := NewSimpleMockAgent(AgentToolCaller)

	// Register agent
	err := registry.Register(AgentToolCaller, mockAgent)
	require.NoError(t, err)

	// Retrieve agent
	retrieved, err := registry.GetAgent(AgentToolCaller)
	require.NoError(t, err)
	assert.Equal(t, AgentToolCaller, retrieved.GetType())

	// Check if agent exists
	assert.True(t, registry.HasAgent(AgentToolCaller))
	assert.False(t, registry.HasAgent(AgentCodeGen))
}
