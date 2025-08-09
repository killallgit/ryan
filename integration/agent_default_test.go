package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAgentResponses(t *testing.T) {
	if !isOllamaAvailable() {
		t.Skip("Skipping test: Ollama is not available")
	}

	t.Run("It responds to a basic prompt", func(t *testing.T) {
		// Setup viper configuration
		setupViperForTest(t)

		// Create executorAgent
		executorAgent, err := agent.NewExecutorAgent()
		require.NoError(t, err)
		defer executorAgent.Close()

		// Execute prompt
		ctx := context.Background()
		response, err := executorAgent.Execute(ctx, "Say hello and nothing else")
		require.NoError(t, err, "Should execute successfully")

		t.Logf("Agent response: %s", response)

		// Should have some response
		assert.NotEmpty(t, response, "Agent should produce output")

		// Should contain something related to hello
		responseLower := strings.ToLower(response)
		assert.True(t,
			strings.Contains(responseLower, "hello") ||
			strings.Contains(responseLower, "hi") ||
			strings.Contains(responseLower, "greet"),
			"Response should be related to the prompt")
	})

	t.Run("It outputs response for math questions", func(t *testing.T) {
		setupViperForTest(t)
		executorAgent, err := agent.NewExecutorAgent()
		require.NoError(t, err)
		defer executorAgent.Close()

		ctx := context.Background()
		response, err := executorAgent.Execute(ctx, "What is 2+2? Answer with just the number.")
		require.NoError(t, err)

		t.Logf("Math response: %s", response)

		// Should contain 4 somewhere in the response
		assert.Contains(t, response, "4", "Response should contain the answer")
	})

	t.Run("It handles multi-line prompts", func(t *testing.T) {
		setupViperForTest(t)
		executorAgent, err := agent.NewExecutorAgent()
		require.NoError(t, err)
		defer executorAgent.Close()

		// Multi-line prompt
		prompt := `List three colors:
1. Red
2. Blue
3. ?
Complete the list with one more color`

		ctx := context.Background()
		response, err := executorAgent.Execute(ctx, prompt)
		require.NoError(t, err)

		responseLower := strings.ToLower(response)
		t.Logf("Response to multi-line prompt: %s", response)

		// Should mention a color
		hasColor := strings.Contains(responseLower, "green") ||
			strings.Contains(responseLower, "yellow") ||
			strings.Contains(responseLower, "purple") ||
			strings.Contains(responseLower, "orange") ||
			strings.Contains(responseLower, "black") ||
			strings.Contains(responseLower, "white") ||
			strings.Contains(responseLower, "pink")

		assert.True(t, hasColor, "Response should include a color")
	})

	t.Run("It preserves conversation context", func(t *testing.T) {
		t.Skip("Memory persistence with LangChain agents needs further investigation")

		setupViperForTest(t)
		executorAgent, err := agent.NewExecutorAgent()
		require.NoError(t, err)
		defer executorAgent.Close()

		ctx := context.Background()

		// First conversation: establish context
		response1, err := executorAgent.Execute(ctx, "My favorite number is 42. Remember this.")
		require.NoError(t, err)
		t.Logf("First response: %s", response1)

		// Second conversation: test context retention
		response2, err := executorAgent.Execute(ctx, "What was my favorite number?")
		require.NoError(t, err)
		t.Logf("Second response: %s", response2)

		// Should remember the number 42
		assert.Contains(t, response2, "42", "Agent should remember the favorite number")
	})
}
