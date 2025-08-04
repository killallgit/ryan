package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeStreamingChatClient(t *testing.T) {
	ctx := context.Background()

	t.Run("should stream response in chunks", func(t *testing.T) {
		client := NewFakeStreamingChatClient("test-model", "Hello world!")
		client.SetChunkSize(3)
		client.SetChunkDelay(5 * time.Millisecond)

		req := chat.ChatRequest{
			Model: "test-model",
			Messages: []chat.Message{
				chat.NewUserMessage("Hi"),
			},
		}

		chunkChan, err := client.StreamMessage(ctx, req)
		require.NoError(t, err)

		var chunks []chat.MessageChunk
		for chunk := range chunkChan {
			chunks = append(chunks, chunk)
		}

		// Should have 4 chunks: "Hel", "lo ", "wor", "ld!"
		assert.Len(t, chunks, 4)
		assert.Equal(t, "Hel", chunks[0].Content)
		assert.Equal(t, "lo ", chunks[1].Content)
		assert.Equal(t, "wor", chunks[2].Content)
		assert.Equal(t, "ld!", chunks[3].Content)

		// Only last chunk should be marked as done
		assert.False(t, chunks[0].Done)
		assert.False(t, chunks[1].Done)
		assert.False(t, chunks[2].Done)
		assert.True(t, chunks[3].Done)

		// Check accumulated content
		assert.Equal(t, "Hello world!", chunks[3].Message.Content)
	})

	t.Run("should support cancellation", func(t *testing.T) {
		client := NewFakeStreamingChatClient("test-model", "This is a long message that will be cancelled")
		client.SetChunkSize(2)
		client.SetChunkDelay(50 * time.Millisecond)

		req := chat.ChatRequest{
			Model: "test-model",
			Messages: []chat.Message{
				chat.NewUserMessage("Test"),
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // Ensure cancel is called on all paths

		chunkChan, err := client.StreamMessage(ctx, req)
		require.NoError(t, err)

		// Cancel after receiving first chunk
		var chunks []chat.MessageChunk
		for chunk := range chunkChan {
			chunks = append(chunks, chunk)
			if len(chunks) == 1 {
				cancel()
			}
		}

		// Should have received only 1 or 2 chunks before cancellation
		assert.LessOrEqual(t, len(chunks), 2)
	})

	t.Run("should simulate streaming errors", func(t *testing.T) {
		client := NewFakeStreamingChatClient("test-model", "This will fail after some chunks")
		client.SetChunkSize(4)
		client.SetFailAfter(2, "network error")

		req := chat.ChatRequest{
			Model: "test-model",
			Messages: []chat.Message{
				chat.NewUserMessage("Test"),
			},
		}

		chunkChan, err := client.StreamMessage(ctx, req)
		require.NoError(t, err)

		var chunks []chat.MessageChunk
		var errorChunk *chat.MessageChunk
		for chunk := range chunkChan {
			if chunk.Error != nil {
				errorChunk = &chunk
			} else {
				chunks = append(chunks, chunk)
			}
		}

		// Should have received 2 chunks before error
		assert.Len(t, chunks, 2)
		assert.NotNil(t, errorChunk)
		assert.Equal(t, "network error", errorChunk.Error.Error())
	})

	t.Run("should stream tool calls", func(t *testing.T) {
		toolResponse := `{"tool_calls": [{"name": "calculator", "arguments": {"a": 1, "b": 2}}]}`
		client := NewFakeStreamingChatClient("test-model", toolResponse)

		req := chat.ChatRequest{
			Model: "test-model",
			Messages: []chat.Message{
				chat.NewUserMessage("What is 1 + 2?"),
			},
			Tools: []map[string]any{
				{"name": "calculator"},
			},
		}

		chunkChan, err := client.StreamMessage(ctx, req)
		require.NoError(t, err)

		var chunks []chat.MessageChunk
		for chunk := range chunkChan {
			chunks = append(chunks, chunk)
		}

		// Tool calls should come as a single chunk
		assert.Len(t, chunks, 1)
		assert.True(t, chunks[0].Done)
		assert.Len(t, chunks[0].Message.ToolCalls, 1)
		assert.Equal(t, "calculator", chunks[0].Message.ToolCalls[0].Function.Name)
	})

	t.Run("should support streaming capability check", func(t *testing.T) {
		client := NewFakeStreamingChatClient("test-model")
		assert.True(t, client.SupportsStreaming())
	})

	t.Run("should handle multiple responses cycling", func(t *testing.T) {
		client := NewFakeStreamingChatClient("test-model", "First", "Second", "Third")
		client.SetChunkSize(10) // Larger than any response

		req := chat.ChatRequest{
			Model: "test-model",
			Messages: []chat.Message{
				chat.NewUserMessage("Test"),
			},
		}

		// First call
		chunkChan1, err := client.StreamMessage(ctx, req)
		require.NoError(t, err)
		var content1 string
		for chunk := range chunkChan1 {
			content1 = chunk.Message.Content
		}
		assert.Equal(t, "First", content1)

		// Second call
		chunkChan2, err := client.StreamMessage(ctx, req)
		require.NoError(t, err)
		var content2 string
		for chunk := range chunkChan2 {
			content2 = chunk.Message.Content
		}
		assert.Equal(t, "Second", content2)
	})
}
