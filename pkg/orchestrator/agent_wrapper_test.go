package orchestrator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentWrapper(t *testing.T) {
	t.Run("creation and initialization", func(t *testing.T) {
		mockLLM := NewSimpleMockLLM()
		orch, err := New(mockLLM)
		require.NoError(t, err)

		wrapper, err := NewAgentWrapper(orch)
		assert.NoError(t, err)
		assert.NotNil(t, wrapper)
		assert.NotNil(t, wrapper.orchestrator)

		// Cleanup
		err = wrapper.Close()
		assert.NoError(t, err)
	})

	t.Run("execute blocking", func(t *testing.T) {
		// Setup orchestrator with mock
		mockLLM := NewSimpleMockLLM()
		mockLLM.WithIntentResponse(&TaskIntent{
			Type:       "reasoning",
			Confidence: 0.9,
		})

		orch, err := New(mockLLM)
		require.NoError(t, err)

		// Register mock agent
		mockAgent := NewSimpleMockAgent(AgentReasoner)
		err = orch.RegisterAgent(AgentReasoner, mockAgent)
		require.NoError(t, err)

		// Create wrapper
		wrapper, err := NewAgentWrapper(orch)
		require.NoError(t, err)
		defer wrapper.Close()

		// Execute
		ctx := context.Background()
		response, err := wrapper.Execute(ctx, "Test query")

		assert.NoError(t, err)
		assert.NotEmpty(t, response)
		assert.Contains(t, response, "Orchestrator Routing Decision")
		assert.Contains(t, response, "Agent Flow")
		assert.Contains(t, response, "Result")

		// Check token stats updated
		sent, received := wrapper.GetTokenStats()
		assert.Greater(t, sent, 0)
		assert.Greater(t, received, 0)
	})

	t.Run("execute streaming", func(t *testing.T) {
		mockLLM := NewSimpleMockLLM()
		mockLLM.WithIntentResponse(&TaskIntent{
			Type:       "tool_use",
			Confidence: 0.85,
		})

		orch, err := New(mockLLM)
		require.NoError(t, err)

		mockAgent := NewSimpleMockAgent(AgentToolCaller)
		err = orch.RegisterAgent(AgentToolCaller, mockAgent)
		require.NoError(t, err)

		wrapper, err := NewAgentWrapper(orch)
		require.NoError(t, err)
		defer wrapper.Close()

		// Use mock stream handler
		mockHandler := NewMockStreamHandler()

		ctx := context.Background()
		err = wrapper.ExecuteStream(ctx, "Stream test", mockHandler)

		assert.NoError(t, err)

		// Verify streaming occurred
		assert.NotEmpty(t, mockHandler.chunks)
		fullContent := mockHandler.GetFullContent()
		// We removed the verbose messages, check for agent marker
		assert.Contains(t, fullContent, "[tool_caller]")
	})

	t.Run("format response", func(t *testing.T) {
		mockLLM := NewSimpleMockLLM()
		orch, err := New(mockLLM)
		require.NoError(t, err)

		wrapper, err := NewAgentWrapper(orch)
		require.NoError(t, err)
		defer wrapper.Close()

		// Create a result with history
		result := &TaskResult{
			Query:  "Test",
			Result: "Final result text",
			History: []AgentResponse{
				{
					AgentType: AgentToolCaller,
					Status:    "success",
					Response:  "Tool response",
					ToolsCalled: []ToolCall{
						{Name: "bash", Arguments: map[string]interface{}{"command": "ls"}},
					},
					Timestamp: time.Now(),
				},
				{
					AgentType: AgentReasoner,
					Status:    "failed",
					Response:  "Reasoning failed",
					Error:     stringPtr("Test error"),
					Timestamp: time.Now(),
				},
			},
		}

		formatted := wrapper.formatResponse(result)

		// Check formatting
		assert.Contains(t, formatted, "Orchestrator Routing Decision")
		assert.Contains(t, formatted, "Agent Flow")
		assert.Contains(t, formatted, "✅") // Success emoji
		assert.Contains(t, formatted, "❌") // Failed emoji
		assert.Contains(t, formatted, "tool_caller")
		assert.Contains(t, formatted, "reasoner")
		assert.Contains(t, formatted, "Tools: bash")
		assert.Contains(t, formatted, "Error: Test error")
		assert.Contains(t, formatted, "Final result text")
	})

	t.Run("clear memory", func(t *testing.T) {
		mockLLM := NewSimpleMockLLM()
		orch, err := New(mockLLM)
		require.NoError(t, err)

		wrapper, err := NewAgentWrapper(orch)
		require.NoError(t, err)
		defer wrapper.Close()

		// Clear memory should work even without history manager
		err = wrapper.ClearMemory()
		assert.NoError(t, err)
	})

	t.Run("token tracking", func(t *testing.T) {
		mockLLM := NewSimpleMockLLM()
		orch, err := New(mockLLM)
		require.NoError(t, err)

		wrapper, err := NewAgentWrapper(orch)
		require.NoError(t, err)
		defer wrapper.Close()

		// Initially zero
		sent, received := wrapper.GetTokenStats()
		assert.Equal(t, 0, sent)
		assert.Equal(t, 0, received)

		// Manually update for testing
		wrapper.tokensSent = 100
		wrapper.tokensReceived = 200

		sent, received = wrapper.GetTokenStats()
		assert.Equal(t, 100, sent)
		assert.Equal(t, 200, received)
	})

	t.Run("handle orchestrator failure", func(t *testing.T) {
		mockLLM := NewSimpleMockLLM()
		// Don't set intent response, causing JSON parse error
		mockLLM.responses = []string{"invalid json"}

		orch, err := New(mockLLM)
		require.NoError(t, err)

		wrapper, err := NewAgentWrapper(orch)
		require.NoError(t, err)
		defer wrapper.Close()

		ctx := context.Background()
		_, err = wrapper.Execute(ctx, "Test failure")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "orchestrator execution failed")
	})
}

func TestAgentWrapperFormatResponse(t *testing.T) {
	tests := []struct {
		name     string
		result   *TaskResult
		expected []string
	}{
		{
			name: "empty history",
			result: &TaskResult{
				Result:  "Simple result",
				History: []AgentResponse{},
			},
			expected: []string{
				"Orchestrator Routing Decision",
				"Result:",
				"Simple result",
			},
		},
		{
			name: "with in_progress status",
			result: &TaskResult{
				Result: "In progress",
				History: []AgentResponse{
					{
						AgentType: AgentPlanner,
						Status:    "in_progress",
					},
				},
			},
			expected: []string{
				"⏳", // In progress emoji
				"planner",
			},
		},
		{
			name: "multiple tools called",
			result: &TaskResult{
				Result: "Done",
				History: []AgentResponse{
					{
						AgentType: AgentToolCaller,
						Status:    "success",
						ToolsCalled: []ToolCall{
							{Name: "bash"},
							{Name: "git"},
							{Name: "file_read"},
						},
					},
				},
			},
			expected: []string{
				"Tools: bash, git, file_read",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := NewSimpleMockLLM()
			orch, _ := New(mockLLM)
			wrapper, _ := NewAgentWrapper(orch)
			defer wrapper.Close()

			formatted := wrapper.formatResponse(tt.result)

			for _, expected := range tt.expected {
				assert.Contains(t, formatted, expected)
			}
		})
	}
}

func TestAgentWrapperIntegration(t *testing.T) {
	t.Run("full execution flow", func(t *testing.T) {
		// Create a more complex mock scenario
		mockLLM := NewSimpleMockLLM()
		mockLLM.WithIntentResponse(&TaskIntent{
			Type:                 "tool_use",
			Confidence:           0.95,
			RequiredCapabilities: []string{"bash_commands"},
		})

		orch, err := New(mockLLM, WithMaxIterations(3))
		require.NoError(t, err)

		// Register multiple agents
		toolAgent := NewSimpleMockAgent(AgentToolCaller)
		reasonerAgent := NewSimpleMockAgent(AgentReasoner)

		err = orch.RegisterAgent(AgentToolCaller, toolAgent)
		require.NoError(t, err)
		err = orch.RegisterAgent(AgentReasoner, reasonerAgent)
		require.NoError(t, err)

		wrapper, err := NewAgentWrapper(orch)
		require.NoError(t, err)
		defer wrapper.Close()

		// Execute and verify
		ctx := context.Background()
		response, err := wrapper.Execute(ctx, "Run ls command")

		assert.NoError(t, err)
		assert.NotEmpty(t, response)

		// Verify formatted output structure
		lines := strings.Split(response, "\n")
		assert.Greater(t, len(lines), 5) // Should have multiple lines of output

		// Check for key sections
		assert.Contains(t, response, "Orchestrator Routing Decision")
		assert.Contains(t, response, "Agent Flow")
		assert.Contains(t, response, "Result")
	})
}
