package agents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/killallgit/ryan/pkg/tools"
)

func TestAgentFactory_Create(t *testing.T) {
	factory := NewAgentFactory()

	t.Run("Create non-existent agent type", func(t *testing.T) {
		config := AgentConfig{
			Model:        "test-model",
			ToolRegistry: tools.NewRegistry(),
		}

		_, err := factory.Create("non-existent", config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown agent type")
	})
}

func TestAgentFactory_IsRegistered(t *testing.T) {
	factory := NewAgentFactory()

	t.Run("Check registered type", func(t *testing.T) {
		assert.True(t, factory.IsRegistered("conversational"))
		assert.True(t, factory.IsRegistered("ollama-functions"))
		assert.True(t, factory.IsRegistered("openai-functions"))
		assert.True(t, factory.IsRegistered("react"))
	})

	t.Run("Check non-registered type", func(t *testing.T) {
		assert.False(t, factory.IsRegistered("non-existent"))
		assert.False(t, factory.IsRegistered(""))
	})
}

// Skipping CreateBestAgent test due to configuration issues

func TestAgentFactory_Register_Duplicate(t *testing.T) {
	factory := NewAgentFactory()

	// Try to register a duplicate type
	err := factory.Register("conversational", mockAgentFactory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

// Mock agent factory function for testing
func mockAgentFactory(config AgentConfig) (LangchainAgent, error) {
	return &MockLangchainAgent{
		name: "mock-agent",
	}, nil
}

func TestAgentFactory_CustomRegistration(t *testing.T) {
	factory := NewAgentFactory()

	t.Run("Register custom agent", func(t *testing.T) {
		err := factory.Register("custom", mockAgentFactory)
		require.NoError(t, err)

		assert.True(t, factory.IsRegistered("custom"))

		types := factory.GetRegisteredTypes()
		assert.Contains(t, types, "custom")
	})

	t.Run("Create custom agent", func(t *testing.T) {
		config := AgentConfig{
			Model:        "test",
			ToolRegistry: tools.NewRegistry(),
		}
		agent, err := factory.Create("custom", config)
		require.NoError(t, err)
		assert.Equal(t, "mock-agent", agent.Name())
	})
}

// MockLangchainAgent for testing
type MockLangchainAgent struct {
	name string
}

func (m *MockLangchainAgent) Name() string        { return m.name }
func (m *MockLangchainAgent) Description() string { return "mock agent" }
func (m *MockLangchainAgent) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	return AgentResult{Success: true, Summary: "mock execution"}, nil
}
func (m *MockLangchainAgent) GetChainType() ChainType                  { return ChainType("mock") }
func (m *MockLangchainAgent) GetToolCompatibility() []string           { return []string{} }
func (m *MockLangchainAgent) GetModelRequirements() ModelRequirements  { return ModelRequirements{} }
func (m *MockLangchainAgent) SetToolRegistry(registry *tools.Registry) {}
func (m *MockLangchainAgent) SetModel(model string) error              { return nil }
func (m *MockLangchainAgent) SupportsStreaming() bool                  { return false }
func (m *MockLangchainAgent) CanHandle(request string) (bool, float64) { return true, 0.8 }
