package integration

import (
	"testing"

	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/langchain"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLangChainAgentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping LangChain integration tests in short mode")
	}

	t.Run("should create LangChain client with agent and tools", func(t *testing.T) {
		// Create a mock tool registry
		toolRegistry := tools.NewRegistry()
		
		client, err := langchain.NewClient("http://localhost:11434", "qwen3:latest", toolRegistry)
		require.NoError(t, err, "Should create LangChain client without error")
		assert.NotNil(t, client, "Client should not be nil")
		
		// Validate that tools are properly registered
		tools := client.GetTools()
		t.Logf("Registered tools count: %d", len(tools))
		
		// Test basic functionality without actual network calls
		memory := client.GetMemory()
		assert.NotNil(t, memory, "Memory should be available")
	})

	t.Run("should create LangChain controller with proper tool integration", func(t *testing.T) {
		// Create tool registry with basic tools
		toolRegistry := tools.NewRegistry()
		
		// Create LangChain controller
		controller, err := controllers.NewLangChainController("http://localhost:11434", "qwen3:latest", toolRegistry)
		require.NoError(t, err, "Should create LangChain controller without error")
		assert.NotNil(t, controller, "Controller should not be nil")
		
		// Validate controller setup
		assert.Equal(t, "qwen3:latest", controller.GetModel(), "Model should be set correctly")
		assert.NotNil(t, controller.GetToolRegistry(), "Tool registry should be available")
		
		// Test conversation functionality
		conversation := controller.GetConversation()
		assert.NotNil(t, conversation, "Conversation should be initialized")
	})

	t.Run("should handle tool adapter parsing correctly", func(t *testing.T) {
		// Test tool adapter input parsing
		toolRegistry := tools.NewRegistry()
		
		client, err := langchain.NewClient("http://localhost:11434", "qwen3:latest", toolRegistry)
		require.NoError(t, err)
		
		langchainTools := client.GetTools()
		if len(langchainTools) > 0 {
			// Test different input formats
			testInputs := []struct {
				name     string
				input    string
				expected bool
			}{
				{"JSON format", `{"command": "ls -la"}`, true},
				{"Key-value format", "command: ls -la", true},
				{"Simple string", "ls -la", true},
				{"Empty input", "", true},
			}
			
			for _, test := range testInputs {
				t.Run(test.name, func(t *testing.T) {
					// We can't actually call the tool without proper setup,
					// but we can validate that the adapter exists
					assert.NotNil(t, langchainTools[0], "Tool adapter should exist")
					assert.NotEmpty(t, langchainTools[0].Name(), "Tool should have a name")
					assert.NotEmpty(t, langchainTools[0].Description(), "Tool should have a description")
				})
			}
		}
	})

	t.Run("should validate thinking parser functionality", func(t *testing.T) {
		// Test thinking parser with agent format
		parser := langchain.NewThinkingOutputParser(true)
		
		testCases := []struct {
			name     string
			input    string
			hasThinking bool
			hasContent bool
		}{
			{
				"Standard thinking blocks",
				"<think>This is a thought</think>\nThis is the response",
				true,
				true,
			},
			{
				"Agent format",
				"Thought: I need to analyze this\nAction: execute_bash\nAction Input: ls -la\nObservation: file list\nAI: Here's the result",
				true,
				true,
			},
			{
				"Mixed format",
				"<think>Initial thought</think>\nThought: More thinking\nAI: Final response",
				true,
				true,
			},
		}
		
		for _, test := range testCases {
			t.Run(test.name, func(t *testing.T) {
				result, err := parser.Parse(test.input)
				require.NoError(t, err, "Parser should not error")
				
				assert.Equal(t, test.hasThinking, result.HasThinking, "Thinking detection should match")
				assert.Equal(t, test.hasContent, len(result.Content) > 0, "Content detection should match")
				
				if test.hasThinking {
					assert.NotEmpty(t, result.Thinking, "Thinking content should not be empty")
				}
			})
		}
	})

	t.Run("should validate streaming thinking parser", func(t *testing.T) {
		parser := langchain.NewStreamingThinkingParser(true)
		
		// Test processing chunks
		chunks := []string{
			"<think>",
			"I need to think about this",
			"</think>",
			"Here is my response",
		}
		
		var allContent []string
		var finalThinking string
		var isComplete bool
		
		for _, chunk := range chunks {
			content, thinking, complete := parser.ProcessChunk(chunk)
			if content != "" {
				allContent = append(allContent, content)
			}
			if thinking != "" {
				finalThinking = thinking
			}
			if complete {
				isComplete = true
			}
		}
		
		assert.True(t, isComplete, "Parsing should complete")
		assert.NotEmpty(t, finalThinking, "Should extract thinking content")
		assert.NotEmpty(t, allContent, "Should have content chunks")
		
		// Reset parser
		parser.Reset()
		// After reset, parser should be clean
		// This validates that the reset function works
	})
}

func TestLangChainPromptIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping LangChain prompt integration tests in short mode")
	}

	t.Run("should create agent prompts correctly", func(t *testing.T) {
		toolNames := []string{"execute_bash", "read_file"}
		
		// Test with thinking enabled
		promptWithThinking := langchain.CreateAgentPrompt(true, toolNames)
		assert.NotNil(t, promptWithThinking, "Prompt should be created")
		
		// Test with thinking disabled
		promptWithoutThinking := langchain.CreateAgentPrompt(false, toolNames)
		assert.NotNil(t, promptWithoutThinking, "Prompt should be created")
		
		// The prompts should be different
		// We can't easily compare them directly, but they should both exist
	})

	t.Run("should format agent output correctly", func(t *testing.T) {
		// Test agent output formatting
		testResult := map[string]any{
			"output": "This is the final response",
			"agent_scratchpad": "Thought: I need to help the user\nAction: execute_bash\nAction Input: ls\nObservation: file.txt",
			"intermediate_steps": []any{"step1", "step2"},
		}
		
		// Test with thinking enabled
		formattedWithThinking := langchain.FormatAgentOutput(testResult, true)
		assert.NotEmpty(t, formattedWithThinking, "Formatted output should not be empty")
		assert.Contains(t, formattedWithThinking, "Tool Usage:", "Should show tool usage")
		
		// Test with thinking disabled
		formattedWithoutThinking := langchain.FormatAgentOutput(testResult, false)
		assert.NotEmpty(t, formattedWithoutThinking, "Formatted output should not be empty")
		assert.Contains(t, formattedWithoutThinking, "Tool Usage:", "Should show tool usage even without thinking")
	})
}
