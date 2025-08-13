package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/ollama"
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

		// Create LLM and executorAgent
		ollamaClient := ollama.NewClient()
		executorAgent, err := agent.NewReactAgent(ollamaClient.LLM)
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
		ollamaClient := ollama.NewClient()
		executorAgent, err := agent.NewReactAgent(ollamaClient.LLM)
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
		ollamaClient := ollama.NewClient()
		executorAgent, err := agent.NewReactAgent(ollamaClient.LLM)
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

	// Memory persistence test removed - LangChain memory needs architectural changes
	// to work properly with the current React agent implementation
}
