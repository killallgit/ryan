package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestFakeLLM(t *testing.T) {
	ctx := context.Background()

	t.Run("should cycle through responses", func(t *testing.T) {
		llm := NewFakeLLM("response1", "response2", "response3")

		resp1, err := llm.Call(ctx, "prompt1")
		require.NoError(t, err)
		assert.Equal(t, "response1", resp1)

		resp2, err := llm.Call(ctx, "prompt2")
		require.NoError(t, err)
		assert.Equal(t, "response2", resp2)

		resp3, err := llm.Call(ctx, "prompt3")
		require.NoError(t, err)
		assert.Equal(t, "response3", resp3)

		// Should cycle back
		resp4, err := llm.Call(ctx, "prompt4")
		require.NoError(t, err)
		assert.Equal(t, "response1", resp4)
	})

	t.Run("should track call count and prompts", func(t *testing.T) {
		llm := NewFakeLLM("test response")

		assert.Equal(t, 0, llm.GetCallCount())

		_, err := llm.Call(ctx, "test prompt")
		require.NoError(t, err)

		assert.Equal(t, 1, llm.GetCallCount())
		assert.Equal(t, "test prompt", llm.GetLastPrompt())
	})

	t.Run("should return error when configured", func(t *testing.T) {
		llm := NewFakeLLM("response1", "response2")
		llm.SetErrorOnCall(2, "simulated error")

		_, err := llm.Call(ctx, "prompt1")
		require.NoError(t, err)

		_, err = llm.Call(ctx, "prompt2")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "simulated error")
	})

	t.Run("should add responses dynamically", func(t *testing.T) {
		llm := NewFakeLLM("initial")

		resp1, err := llm.Call(ctx, "prompt1")
		require.NoError(t, err)
		assert.Equal(t, "initial", resp1)

		llm.AddResponse("dynamic")

		// Next call should cycle back to initial since we're at index 1
		resp2, err := llm.Call(ctx, "prompt2")
		require.NoError(t, err)
		assert.Equal(t, "initial", resp2)

		// Third call should now get the dynamic response
		resp3, err := llm.Call(ctx, "prompt3")
		require.NoError(t, err)
		assert.Equal(t, "dynamic", resp3)
	})

	t.Run("should reset state", func(t *testing.T) {
		llm := NewFakeLLM("response1", "response2")

		_, err := llm.Call(ctx, "prompt1")
		require.NoError(t, err)
		_, err = llm.Call(ctx, "prompt2")
		require.NoError(t, err)

		assert.Equal(t, 2, llm.GetCallCount())

		llm.Reset()

		assert.Equal(t, 0, llm.GetCallCount())
		assert.Equal(t, "", llm.GetLastPrompt())

		// Should start from first response again
		resp, err := llm.Call(ctx, "new prompt")
		require.NoError(t, err)
		assert.Equal(t, "response1", resp)
	})

	t.Run("should handle GenerateContent", func(t *testing.T) {
		llm := NewFakeLLM("generated content")

		messages := []llms.MessageContent{
			{
				Parts: []llms.ContentPart{
					llms.TextContent{Text: "Hello"},
				},
			},
			{
				Parts: []llms.ContentPart{
					llms.TextContent{Text: "World"},
				},
			},
		}

		resp, err := llm.GenerateContent(ctx, messages)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Choices, 1)
		assert.Equal(t, "generated content", resp.Choices[0].Content)
	})

	t.Run("should use predefined responses", func(t *testing.T) {
		llm := NewFakeLLM(PredefinedResponses.SimpleChat...)

		resp1, err := llm.Call(ctx, "Hi")
		require.NoError(t, err)
		assert.Equal(t, "Hello! How can I help you today?", resp1)

		resp2, err := llm.Call(ctx, "Tell me more")
		require.NoError(t, err)
		assert.Equal(t, "I understand your question. Here's my response.", resp2)
	})
}
