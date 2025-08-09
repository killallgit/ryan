package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockMemory(t *testing.T) {
	t.Run("mock basic operations", func(t *testing.T) {
		mock := NewMockMemory()

		assert.True(t, mock.IsEnabled())

		err := mock.AddUserMessage("Test")
		require.NoError(t, err)

		err = mock.AddAssistantMessage("Response")
		require.NoError(t, err)

		messages, err := mock.GetMessages()
		require.NoError(t, err)
		assert.Len(t, messages, 2)

		llmMessages, err := mock.ConvertToLLMMessages()
		require.NoError(t, err)
		assert.Len(t, llmMessages, 2)

		err = mock.Clear()
		require.NoError(t, err)

		messages, err = mock.GetMessages()
		require.NoError(t, err)
		assert.Len(t, messages, 0)

		err = mock.Close()
		require.NoError(t, err)
		assert.True(t, mock.Closed)
	})

	t.Run("mock error injection", func(t *testing.T) {
		mock := NewMockMemory()

		// Test error injection
		mock.AddUserError = assert.AnError
		err := mock.AddUserMessage("Test")
		assert.Error(t, err)

		mock.AddAssistantError = assert.AnError
		err = mock.AddAssistantMessage("Test")
		assert.Error(t, err)

		mock.GetMessagesError = assert.AnError
		_, err = mock.GetMessages()
		assert.Error(t, err)

		mock.ClearError = assert.AnError
		err = mock.Clear()
		assert.Error(t, err)
	})

	t.Run("disabled mock returns empty", func(t *testing.T) {
		mock := NewMockMemory()

		assert.True(t, mock.IsEnabled())

		// Add messages
		err := mock.AddUserMessage("Test")
		require.NoError(t, err)

		// Should have the message
		messages, err := mock.GetMessages()
		require.NoError(t, err)
		assert.Len(t, messages, 1)

		llmMessages, err := mock.ConvertToLLMMessages()
		require.NoError(t, err)
		assert.Len(t, llmMessages, 1)
	})
}
