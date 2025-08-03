package chat

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLangChainMemory(t *testing.T) {
	t.Run("should create new memory", func(t *testing.T) {
		memory := NewLangChainMemory()
		assert.NotNil(t, memory)
		assert.NotNil(t, memory.GetBuffer())

		conv := memory.GetConversation()
		assert.Empty(t, conv.Messages)
	})

	t.Run("should add messages to memory", func(t *testing.T) {
		memory := NewLangChainMemory()
		ctx := context.Background()

		// Add user message
		userMsg := NewUserMessage("Hello")
		err := memory.AddMessage(ctx, userMsg)
		require.NoError(t, err)

		// Add assistant message
		assistantMsg := NewAssistantMessage("Hi there!")
		err = memory.AddMessage(ctx, assistantMsg)
		require.NoError(t, err)

		// Check conversation
		conv := memory.GetConversation()
		assert.Len(t, conv.Messages, 2)
		assert.Equal(t, "Hello", conv.Messages[0].Content)
		assert.Equal(t, "Hi there!", conv.Messages[1].Content)
	})

	t.Run("should create memory from existing conversation", func(t *testing.T) {
		// Create conversation with messages
		conv := NewConversation("test-model")
		conv = AddMessage(conv, NewSystemMessage("You are helpful"))
		conv = AddMessage(conv, NewUserMessage("Hello"))
		conv = AddMessage(conv, NewAssistantMessage("Hi!"))

		// Create memory from conversation
		memory, err := NewLangChainMemoryWithConversation(conv)
		require.NoError(t, err)

		// Check that all messages were transferred
		memoryConv := memory.GetConversation()
		assert.Len(t, memoryConv.Messages, 3)
		assert.Equal(t, "You are helpful", memoryConv.Messages[0].Content)
		assert.Equal(t, "Hello", memoryConv.Messages[1].Content)
		assert.Equal(t, "Hi!", memoryConv.Messages[2].Content)
	})

	t.Run("should clear memory", func(t *testing.T) {
		memory := NewLangChainMemory()
		ctx := context.Background()

		// Add some messages
		memory.AddMessage(ctx, NewUserMessage("Hello"))
		memory.AddMessage(ctx, NewAssistantMessage("Hi!"))

		// Check messages exist
		conv := memory.GetConversation()
		assert.Len(t, conv.Messages, 2)

		// Clear memory
		err := memory.Clear(ctx)
		require.NoError(t, err)

		// Check messages are gone
		conv = memory.GetConversation()
		assert.Empty(t, conv.Messages)
	})

	t.Run("should handle different message types", func(t *testing.T) {
		memory := NewLangChainMemory()
		ctx := context.Background()

		// Add various message types
		messages := []Message{
			NewSystemMessage("System prompt"),
			NewUserMessage("User message"),
			NewAssistantMessage("Assistant response"),
			NewToolResultMessage("test_tool", "Tool result"),
			NewErrorMessage("Error occurred"),
		}

		for _, msg := range messages {
			err := memory.AddMessage(ctx, msg)
			assert.NoError(t, err)
		}

		conv := memory.GetConversation()
		assert.Len(t, conv.Messages, 5)
	})
}

func TestConvertToLangChainMessages(t *testing.T) {
	messages := []Message{
		NewSystemMessage("System"),
		NewUserMessage("User"),
		NewAssistantMessage("Assistant"),
		NewToolResultMessage("tool", "Tool result"),
		NewErrorMessage("Error"),
	}

	langchainMessages := ConvertToLangChainMessages(messages)
	assert.Len(t, langchainMessages, 5)

	// Convert back
	convertedMessages := ConvertFromLangChainMessages(langchainMessages)
	assert.Len(t, convertedMessages, 5)

	// Check first few conversions (system, user, assistant)
	assert.Equal(t, RoleSystem, convertedMessages[0].Role)
	assert.Equal(t, RoleUser, convertedMessages[1].Role)
	assert.Equal(t, RoleAssistant, convertedMessages[2].Role)
}

func TestMemoryAdapter(t *testing.T) {
	memory := NewLangChainMemory()
	adapter := &MemoryAdapter{LangChainMemory: memory}

	ctx := context.Background()

	t.Run("should implement schema.Memory interface", func(t *testing.T) {
		assert.Equal(t, "history", adapter.GetMemoryKey(ctx))

		variables := adapter.MemoryVariables(ctx)
		assert.Contains(t, variables, "history")
	})

	t.Run("should load and save memory variables", func(t *testing.T) {
		// Add a message
		memory.AddMessage(ctx, NewUserMessage("Test"))

		// Load memory variables
		vars, err := adapter.LoadMemoryVariables(ctx, map[string]any{})
		assert.NoError(t, err)
		assert.NotNil(t, vars)

		// Save context
		inputs := map[string]any{"input": "test"}
		outputs := map[string]any{"output": "response"}
		err = adapter.SaveContext(ctx, inputs, outputs)
		assert.NoError(t, err)
	})
}
