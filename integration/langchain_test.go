package integration

import (
	"testing"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLangChainIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping LangChain integration tests in short mode")
	}

	t.Run("should create LangChain client without errors", func(t *testing.T) {
		config := chat.LangChainConfig("http://localhost:11434", "llama2")
		client, err := chat.NewChatClient(config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("should create LangChain streaming client without errors", func(t *testing.T) {
		config := chat.LangChainStreamingConfig("http://localhost:11434", "llama2") 
		streamingClient, err := chat.NewStreamingChatClient(config)
		require.NoError(t, err)
		assert.NotNil(t, streamingClient)
	})

	t.Run("should handle invalid server URLs gracefully", func(t *testing.T) {
		config := chat.LangChainConfig("http://invalid-server:11434", "llama2")
		client, err := chat.NewChatClient(config)
		require.NoError(t, err)
		assert.NotNil(t, client)

		// Test sending a message (should fail gracefully)
		req := chat.ChatRequest{
			Model:    "llama2",
			Messages: []chat.Message{{Role: "user", Content: "test"}},
		}

		_, err = client.SendMessage(req)
		// We expect an error here since the server doesn't exist
		assert.Error(t, err)
	})
}