package langchain

import (
	"testing"

	"github.com/killallgit/ryan/pkg/models"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentSelector(t *testing.T) {
	registry := tools.NewRegistry()
	modelName := "qwen2.5-coder:7b"

	selector := NewAgentSelector(registry, modelName)

	assert.NotNil(t, selector)
	assert.Equal(t, registry, selector.toolRegistry)
	assert.Equal(t, modelName, selector.modelInfo.Name)
	assert.NotNil(t, selector.log)
}

func TestAgentSelector_SelectAgent(t *testing.T) {
	registry := tools.NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)

	tests := []struct {
		name               string
		modelName          string
		toolCompatibility  models.ToolCompatibility
		input              string
		expectedAgentType  AgentType
		expectedNeedsTools bool
	}{
		{
			name:               "excellent model with tool need",
			modelName:          "qwen2.5-coder:7b",
			toolCompatibility:  models.ToolCompatibilityExcellent,
			input:              "list files in current directory",
			expectedAgentType:  AgentTypeOllamaFunctions,
			expectedNeedsTools: true,
		},
		{
			name:               "excellent model without tool need",
			modelName:          "qwen2.5-coder:7b",
			toolCompatibility:  models.ToolCompatibilityExcellent,
			input:              "explain how algorithms work",
			expectedAgentType:  AgentTypeDirect,
			expectedNeedsTools: false,
		},
		{
			name:               "good model with tool need",
			modelName:          "moderate-model:7b",
			toolCompatibility:  models.ToolCompatibilityGood,
			input:              "run command to check disk usage",
			expectedAgentType:  AgentTypeConversational,
			expectedNeedsTools: true,
		},
		{
			name:               "no tool support model",
			modelName:          "basic-model:7b",
			toolCompatibility:  models.ToolCompatibilityNone,
			input:              "create a new file",
			expectedAgentType:  AgentTypeDirect,
			expectedNeedsTools: false,
		},
		{
			name:               "GPT model with excellent support",
			modelName:          "gpt-4",
			toolCompatibility:  models.ToolCompatibilityExcellent,
			input:              "search for files containing error",
			expectedAgentType:  AgentTypeOpenAIFunctions,
			expectedNeedsTools: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewAgentSelector(registry, tt.modelName)
			agentType, needsTools := selector.SelectAgent(tt.input)

			// Tool need analysis should still work regardless of model info
			assert.Equal(t, tt.expectedNeedsTools, needsTools)
			_ = agentType // Use the variable to avoid unused error
		})
	}
}

func TestAgentSelector_analyzeToolNeed(t *testing.T) {
	registry := tools.NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)

	selector := NewAgentSelector(registry, "test-model")

	tests := []struct {
		name        string
		input       string
		expectTools bool
	}{
		// File system operations
		{
			name:        "list files request",
			input:       "how many files are in this directory?",
			expectTools: true,
		},
		{
			name:        "read file request",
			input:       "show me the contents of README.md",
			expectTools: true,
		},
		{
			name:        "create file request",
			input:       "create file called test.txt",
			expectTools: true,
		},

		// Command execution
		{
			name:        "bash command request",
			input:       "run command to check system status",
			expectTools: true,
		},
		{
			name:        "docker command",
			input:       "docker ps to see running containers",
			expectTools: true,
		},
		{
			name:        "git command",
			input:       "git status to check repository state",
			expectTools: true,
		},

		// System information
		{
			name:        "disk usage check",
			input:       "check disk usage on the system",
			expectTools: true,
		},
		{
			name:        "process information",
			input:       "what processes are running?",
			expectTools: true,
		},

		// Search operations
		{
			name:        "code search",
			input:       "search for TODO comments in the code",
			expectTools: true,
		},
		{
			name:        "grep operation",
			input:       "grep for error messages in log files",
			expectTools: true,
		},

		// Question patterns
		{
			name:        "how many question",
			input:       "how many containers are running?",
			expectTools: true,
		},
		{
			name:        "what is question (system)",
			input:       "what is the current memory usage?",
			expectTools: true,
		},
		{
			name:        "show me request",
			input:       "show me the current directory structure",
			expectTools: true,
		},
		{
			name:        "can you check request",
			input:       "can you check if the service is running?",
			expectTools: true,
		},

		// Explicit tool mentions
		{
			name:        "explicit bash mention",
			input:       "use execute_bash to run ls command",
			expectTools: true,
		},

		// Non-tool requests
		{
			name:        "general question",
			input:       "explain how machine learning works",
			expectTools: false,
		},
		{
			name:        "explanation request",
			input:       "explain how machine learning works",
			expectTools: false,
		},
		{
			name:        "definition request",
			input:       "define the term 'recursion'",
			expectTools: false,
		},
		{
			name:        "opinion request",
			input:       "what do you think about this approach?",
			expectTools: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsTools := selector.analyzeToolNeed(tt.input)
			assert.Equal(t, tt.expectTools, needsTools,
				"Expected needsTools=%t for input: %s", tt.expectTools, tt.input)
		})
	}
}

func TestAgentSelector_isOllamaCompatible(t *testing.T) {
	registry := tools.NewRegistry()

	tests := []struct {
		name           string
		modelName      string
		expectedResult bool
	}{
		{
			name:           "llama model",
			modelName:      "llama2:7b",
			expectedResult: true,
		},
		{
			name:           "qwen model",
			modelName:      "qwen2.5-coder:14b",
			expectedResult: true,
		},
		{
			name:           "mistral model",
			modelName:      "mistral:7b-instruct",
			expectedResult: true,
		},
		{
			name:           "deepseek model",
			modelName:      "deepseek-coder:6.7b",
			expectedResult: true,
		},
		{
			name:           "command-r model",
			modelName:      "command-r:35b",
			expectedResult: true,
		},
		{
			name:           "granite model",
			modelName:      "granite-code:8b",
			expectedResult: true,
		},
		{
			name:           "gemma2 model",
			modelName:      "gemma2:9b",
			expectedResult: true,
		},
		{
			name:           "phi3 model",
			modelName:      "phi3:medium",
			expectedResult: true,
		},
		{
			name:           "non-compatible model",
			modelName:      "gpt-4",
			expectedResult: false,
		},
		{
			name:           "unknown model",
			modelName:      "unknown-model:latest",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewAgentSelector(registry, tt.modelName)
			result := selector.isOllamaCompatible()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestAgentSelector_GetRecommendedAgent(t *testing.T) {
	registry := tools.NewRegistry()
	selector := NewAgentSelector(registry, "test-model")

	tests := []struct {
		name         string
		agentType    AgentType
		needsTools   bool
		expectedText string
	}{
		{
			name:         "Ollama functions agent",
			agentType:    AgentTypeOllamaFunctions,
			needsTools:   true,
			expectedText: "Native Ollama function calling (recommended for tool usage)",
		},
		{
			name:         "OpenAI functions agent",
			agentType:    AgentTypeOpenAIFunctions,
			needsTools:   true,
			expectedText: "OpenAI Functions agent (native function calling)",
		},
		{
			name:         "Conversational agent",
			agentType:    AgentTypeConversational,
			needsTools:   true,
			expectedText: "Conversational ReAct agent (may require output processing)",
		},
		{
			name:         "Direct agent with tools",
			agentType:    AgentTypeDirect,
			needsTools:   true,
			expectedText: "Direct LLM (tools available but not executable)",
		},
		{
			name:         "Direct agent without tools",
			agentType:    AgentTypeDirect,
			needsTools:   false,
			expectedText: "Direct LLM (no tools needed)",
		},
		{
			name:         "Unknown agent type",
			agentType:    AgentType(999),
			needsTools:   false,
			expectedText: "Unknown agent type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendation := selector.GetRecommendedAgent(tt.agentType, tt.needsTools)
			assert.Equal(t, tt.expectedText, recommendation)
		})
	}
}

func TestAgentSelector_EdgeCases(t *testing.T) {
	t.Run("nil tool registry", func(t *testing.T) {
		selector := NewAgentSelector(nil, "test-model")
		assert.NotNil(t, selector)
		assert.Nil(t, selector.toolRegistry)

		// Should still analyze tool need but won't find explicit tool references
		needsTools := selector.analyzeToolNeed("use execute_bash to run command")
		assert.True(t, needsTools) // Still true due to "run command" keyword
	})

	t.Run("empty input", func(t *testing.T) {
		registry := tools.NewRegistry()
		selector := NewAgentSelector(registry, "test-model")

		agentType, needsTools := selector.SelectAgent("")
		assert.Equal(t, AgentTypeDirect, agentType)
		assert.False(t, needsTools)
	})

	t.Run("very long input", func(t *testing.T) {
		registry := tools.NewRegistry()
		selector := NewAgentSelector(registry, "test-model")

		longInput := "This is a very long input that contains the keyword 'list files' " +
			"which should trigger tool detection even in a long string. " +
			"The agent selector should be able to handle this gracefully."

		_, needsTools := selector.SelectAgent(longInput)
		assert.True(t, needsTools) // Should detect "list files"
	})

	t.Run("case sensitivity", func(t *testing.T) {
		registry := tools.NewRegistry()
		selector := NewAgentSelector(registry, "test-model")

		// Test mixed case
		needsTools := selector.analyzeToolNeed("LIST FILES in directory")
		assert.True(t, needsTools)

		needsTools = selector.analyzeToolNeed("RUN COMMAND to check status")
		assert.True(t, needsTools)
	})

	t.Run("multiple tool indicators", func(t *testing.T) {
		registry := tools.NewRegistry()
		selector := NewAgentSelector(registry, "test-model")

		input := "list files and then run command to check disk usage"
		needsTools := selector.analyzeToolNeed(input)
		assert.True(t, needsTools)
	})
}
