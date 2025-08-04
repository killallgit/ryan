package langchain

import (
	"context"
	"testing"

	"github.com/killallgit/ryan/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestOllamaToolCaller(t *testing.T) {
	// Create a mock LLM
	mockLLM := &MockLLM{}

	// Create a tool registry with basic tools
	registry := tools.NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)

	// Create the tool caller
	toolCaller := NewOllamaToolCaller(mockLLM, registry)
	assert.NotNil(t, toolCaller)

	t.Run("formatToolsForOllama", func(t *testing.T) {
		toolDefs := toolCaller.formatToolsForOllama()
		assert.NotEmpty(t, toolDefs)

		// Should have at least the builtin tools
		assert.GreaterOrEqual(t, len(toolDefs), 2)

		// Check format
		for _, toolDef := range toolDefs {
			assert.Equal(t, "function", toolDef["type"])
			assert.Contains(t, toolDef, "function")

			function := toolDef["function"].(map[string]any)
			assert.Contains(t, function, "name")
			assert.Contains(t, function, "description")
			assert.Contains(t, function, "parameters")
		}
	})

	t.Run("createToolContext", func(t *testing.T) {
		context := toolCaller.createToolContext()
		assert.NotEmpty(t, context)
		assert.Contains(t, context, "You have access to the following tools")
		assert.Contains(t, context, "tool_calls")
		assert.Contains(t, context, "JSON")
	})

	t.Run("extractToolCalls", func(t *testing.T) {
		// Test JSON format
		jsonContent := `{"tool_calls":[{"name":"execute_bash","arguments":{"command":"ls -la"}}]}`
		toolCalls := toolCaller.extractToolCalls(jsonContent)

		assert.Len(t, toolCalls, 1)
		assert.Equal(t, "execute_bash", toolCalls[0].Name)
		assert.Equal(t, "ls -la", toolCalls[0].Arguments["command"])

		// Test with thinking blocks
		thinkingContent := `<think>I need to list files</think>{"tool_calls":[{"name":"execute_bash","arguments":{"command":"ls"}}]}`
		toolCalls = toolCaller.extractToolCalls(thinkingContent)

		assert.Len(t, toolCalls, 1)
		assert.Equal(t, "execute_bash", toolCalls[0].Name)

		// Test no tool calls
		noToolContent := "This is just regular text"
		toolCalls = toolCaller.extractToolCalls(noToolContent)
		assert.Empty(t, toolCalls)
	})
}

func TestOllamaFunctionsAgent(t *testing.T) {
	// Create a mock LLM
	mockLLM := &MockLLM{}

	// Create a tool registry
	registry := tools.NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)

	// Create the agent
	agent := CreateOllamaFunctionsAgent(mockLLM, registry, nil)
	assert.NotNil(t, agent)

	t.Run("Call", func(t *testing.T) {
		// Set up mock to return a simple response
		mockLLM.response = "Hello, this is a test response"

		inputs := map[string]any{
			"input": "Test message",
		}

		result, err := agent.Call(context.Background(), inputs, nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello, this is a test response", result)
	})

	t.Run("CallMissingInput", func(t *testing.T) {
		inputs := map[string]any{}

		_, err := agent.Call(context.Background(), inputs, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'input'")
	})
}

// MockLLM is a simple mock implementation for testing
type MockLLM struct {
	response string
	err      error
}

func (m *MockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: m.response,
			},
		},
	}, nil
}

func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}
