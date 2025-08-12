package integration

import (
	"context"
	"testing"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenTracking(t *testing.T) {
	if !isOllamaAvailable() {
		t.Skip("Skipping test: Ollama is not available")
	}

	// Setup viper configuration
	setupViperForTest(t)

	// Create Ollama client
	ollamaClient := ollama.NewClient()

	// Create executor agent
	executorAgent, err := agent.NewExecutorAgent(ollamaClient.LLM)
	require.NoError(t, err)
	defer executorAgent.Close()

	t.Run("Tracks tokens for simple prompts", func(t *testing.T) {
		// Initial state should have no tokens
		sentBefore, recvBefore := executorAgent.GetTokenStats()
		assert.Equal(t, 0, sentBefore)
		assert.Equal(t, 0, recvBefore)

		// Execute a simple prompt
		ctx := context.Background()
		response, err := executorAgent.Execute(ctx, "What is 2+2?")
		require.NoError(t, err)
		assert.NotEmpty(t, response)

		// Check that tokens were tracked
		sentAfter, recvAfter := executorAgent.GetTokenStats()
		assert.Greater(t, sentAfter, 0, "Should have tracked sent tokens")
		assert.Greater(t, recvAfter, 0, "Should have tracked received tokens")
	})

	t.Run("Accumulates tokens across multiple prompts", func(t *testing.T) {
		// Clear memory to reset tokens
		err := executorAgent.ClearMemory()
		require.NoError(t, err)

		// Verify tokens were reset
		sent1, recv1 := executorAgent.GetTokenStats()
		assert.Equal(t, 0, sent1)
		assert.Equal(t, 0, recv1)

		// First prompt
		ctx := context.Background()
		_, err = executorAgent.Execute(ctx, "Hello")
		require.NoError(t, err)

		sent2, recv2 := executorAgent.GetTokenStats()
		assert.Greater(t, sent2, 0)
		assert.Greater(t, recv2, 0)

		// Second prompt
		_, err = executorAgent.Execute(ctx, "How are you?")
		require.NoError(t, err)

		sent3, recv3 := executorAgent.GetTokenStats()
		assert.Greater(t, sent3, sent2, "Tokens should accumulate")
		assert.Greater(t, recv3, recv2, "Tokens should accumulate")
	})

	t.Run("Tracks tokens during streaming", func(t *testing.T) {
		// Clear memory to reset tokens
		err := executorAgent.ClearMemory()
		require.NoError(t, err)

		// Create a simple stream handler
		handler := &testStreamHandler{}

		// Execute streaming
		ctx := context.Background()
		err = executorAgent.ExecuteStream(ctx, "Count to 3", handler)
		require.NoError(t, err)

		// Check that tokens were tracked
		sent, recv := executorAgent.GetTokenStats()
		assert.Greater(t, sent, 0, "Should have tracked sent tokens during streaming")
		assert.Greater(t, recv, 0, "Should have tracked received tokens during streaming")
	})
}

type testStreamHandler struct {
	chunks []string
	final  string
}

func (h *testStreamHandler) OnChunk(chunk []byte) error {
	h.chunks = append(h.chunks, string(chunk))
	return nil
}

func (h *testStreamHandler) OnComplete(finalContent string) error {
	h.final = finalContent
	return nil
}

func (h *testStreamHandler) OnError(err error) {
	// No-op for test
}
