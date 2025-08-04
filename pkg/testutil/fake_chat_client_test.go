package testutil

import (
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeChatClient(t *testing.T) {
	t.Run("should implement ChatClient interface", func(t *testing.T) {
		client := NewFakeChatClient("test-model", "test response")

		// Ensure it implements the interface
		var _ chat.ChatClient = client
	})

	t.Run("should send message and return response", func(t *testing.T) {
		client := NewFakeChatClient("test-model", "Hello from fake LLM!")

		req := chat.ChatRequest{
			Model: "test-model",
			Messages: []chat.Message{
				{Role: "user", Content: "Hello"},
			},
		}

		msg, err := client.SendMessage(req)
		require.NoError(t, err)
		assert.Equal(t, "assistant", msg.Role)
		assert.Equal(t, "Hello from fake LLM!", msg.Content)
	})

	t.Run("should return full response with metadata", func(t *testing.T) {
		client := NewFakeChatClient("test-model", "Test response")
		client.SetResponseTime(50 * time.Millisecond)

		req := chat.ChatRequest{
			Model: "test-model",
			Messages: []chat.Message{
				{Role: "user", Content: "Test prompt"},
			},
		}

		start := time.Now()
		resp, err := client.SendMessageWithResponse(req)
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.Equal(t, "test-model", resp.Model)
		assert.True(t, resp.Done)
		assert.Equal(t, "complete", resp.DoneReason)
		assert.Greater(t, resp.PromptEvalCount, 0)
		assert.Greater(t, resp.EvalCount, 0)
		assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
	})

	t.Run("should handle tool calls", func(t *testing.T) {
		toolResponse := `{"tool_calls": [{"name": "calculator", "arguments": {"a": 1, "b": 2}}]}`
		client := NewFakeChatClient("test-model", toolResponse)

		req := chat.ChatRequest{
			Model: "test-model",
			Messages: []chat.Message{
				{Role: "user", Content: "What is 1 + 2?"},
			},
			Tools: []map[string]any{
				{"name": "calculator", "description": "Adds two numbers"},
			},
		}

		resp, err := client.SendMessageWithResponse(req)
		require.NoError(t, err)
		assert.Len(t, resp.Message.ToolCalls, 1)
		assert.Equal(t, "calculator", resp.Message.ToolCalls[0].Function.Name)
		assert.Equal(t, float64(1), resp.Message.ToolCalls[0].Function.Arguments["a"])
		assert.Equal(t, float64(2), resp.Message.ToolCalls[0].Function.Arguments["b"])
	})

	t.Run("should track calls through fake LLM", func(t *testing.T) {
		client := NewFakeChatClient("test-model", "response1", "response2")
		fakeLLM := client.GetFakeLLM()

		req := chat.ChatRequest{
			Model: "test-model",
			Messages: []chat.Message{
				{Role: "user", Content: "First message"},
			},
		}

		_, err := client.SendMessage(req)
		require.NoError(t, err)
		assert.Equal(t, 1, fakeLLM.GetCallCount())
		assert.Contains(t, fakeLLM.GetLastPrompt(), "First message")

		req.Messages = []chat.Message{
			{Role: "user", Content: "Second message"},
		}

		_, err = client.SendMessage(req)
		require.NoError(t, err)
		assert.Equal(t, 2, fakeLLM.GetCallCount())
		assert.Contains(t, fakeLLM.GetLastPrompt(), "Second message")
	})

	t.Run("should propagate errors from fake LLM", func(t *testing.T) {
		client := NewFakeChatClient("test-model", "response")
		client.GetFakeLLM().SetErrorOnCall(1, "simulated LLM error")

		req := chat.ChatRequest{
			Model: "test-model",
			Messages: []chat.Message{
				{Role: "user", Content: "Test"},
			},
		}

		_, err := client.SendMessage(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "simulated LLM error")
	})
}
