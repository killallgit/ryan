package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockChatClient implements the ChatClient interface for testing
type MockChatClient struct {
	mock.Mock
}

func (m *MockChatClient) SendMessage(req chat.ChatRequest) (chat.Message, error) {
	args := m.Called(req)
	return args.Get(0).(chat.Message), args.Error(1)
}

func (m *MockChatClient) SendMessageWithResponse(req chat.ChatRequest) (chat.ChatResponse, error) {
	args := m.Called(req)
	return args.Get(0).(chat.ChatResponse), args.Error(1)
}

// MockStreamingChatClient implements the StreamingChatClient interface for testing
type MockStreamingChatClient struct {
	MockChatClient // Embed MockChatClient
}

func (m *MockStreamingChatClient) StreamMessage(ctx context.Context, req chat.ChatRequest) (<-chan chat.MessageChunk, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(<-chan chat.MessageChunk), args.Error(1)
}

func TestChatControllerStreaming(t *testing.T) {
	t.Run("should start streaming successfully", func(t *testing.T) {
		mockClient := &MockStreamingChatClient{}
		controller := NewChatController(mockClient, "test-model", nil)

		// Create a mock channel with test chunks
		chunksChan := make(chan chat.MessageChunk, 3)
		chunksChan <- chat.MessageChunk{
			StreamID:  "test-stream",
			Content:   "Hello",
			Done:      false,
			Timestamp: time.Now(),
			Message:   chat.Message{Role: chat.RoleAssistant},
		}
		chunksChan <- chat.MessageChunk{
			StreamID:  "test-stream",
			Content:   " world!",
			Done:      true,
			Timestamp: time.Now(),
			Message:   chat.Message{Role: chat.RoleAssistant},
		}
		close(chunksChan)

		mockClient.On("StreamMessage", mock.Anything, mock.Anything).Return((<-chan chat.MessageChunk)(chunksChan), nil)

		ctx := context.Background()
		updates, err := controller.StartStreaming(ctx, "test message")

		require.NoError(t, err)
		require.NotNil(t, updates)

		// Collect all updates
		var receivedUpdates []StreamingUpdate
		for update := range updates {
			receivedUpdates = append(receivedUpdates, update)
		}

		// Should receive: StreamStarted, ChunkReceived, ChunkReceived, MessageComplete
		assert.GreaterOrEqual(t, len(receivedUpdates), 3)

		// Check first update is stream started
		assert.Equal(t, StreamStarted, receivedUpdates[0].Type)

		// Check we got chunk updates
		chunkUpdates := 0
		for _, update := range receivedUpdates {
			if update.Type == ChunkReceived {
				chunkUpdates++
			}
		}
		assert.GreaterOrEqual(t, chunkUpdates, 1)

		mockClient.AssertExpectations(t)
	})

	t.Run("should fallback to non-streaming when client doesn't support it", func(t *testing.T) {
		// Create a proper mock for regular ChatClient
		mockClient := &MockChatClient{}

		// Mock the SendMessageWithResponse for fallback
		mockResponse := chat.ChatResponse{
			Message: chat.Message{
				Role:    chat.RoleAssistant,
				Content: "Fallback response",
			},
		}
		mockClient.On("SendMessageWithResponse", mock.Anything).Return(mockResponse, nil)

		controller := NewChatController(mockClient, "test-model", nil)

		ctx := context.Background()
		updates, err := controller.StartStreaming(ctx, "test message")

		require.NoError(t, err)
		require.NotNil(t, updates)

		// Should get MessageComplete from fallback
		var receivedUpdate StreamingUpdate
		select {
		case receivedUpdate = <-updates:
			assert.Equal(t, MessageComplete, receivedUpdate.Type)
			assert.Equal(t, "Fallback response", receivedUpdate.Message.Content)
		case <-time.After(1 * time.Second):
			t.Fatal("Expected to receive update from fallback")
		}

		mockClient.AssertExpectations(t)
	})

	t.Run("should reject empty messages", func(t *testing.T) {
		mockClient := &MockStreamingChatClient{}
		controller := NewChatController(mockClient, "test-model", nil)

		ctx := context.Background()
		updates, err := controller.StartStreaming(ctx, "")

		assert.Error(t, err)
		assert.Nil(t, updates)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("should handle streaming errors", func(t *testing.T) {
		mockClient := &MockStreamingChatClient{}
		controller := NewChatController(mockClient, "test-model", nil)

		// Create error channel
		errorChan := make(chan chat.MessageChunk, 1)
		errorChan <- chat.MessageChunk{
			StreamID: "error-stream",
			Error:    assert.AnError,
		}
		close(errorChan)

		mockClient.On("StreamMessage", mock.Anything, mock.Anything).Return((<-chan chat.MessageChunk)(errorChan), nil)

		ctx := context.Background()
		updates, err := controller.StartStreaming(ctx, "test message")

		require.NoError(t, err)
		require.NotNil(t, updates)

		// Should get error update
		var receivedUpdates []StreamingUpdate
		for update := range updates {
			receivedUpdates = append(receivedUpdates, update)
			if update.Type == StreamError {
				break
			}
		}

		errorFound := false
		for _, update := range receivedUpdates {
			if update.Type == StreamError {
				errorFound = true
				break
			}
		}
		assert.True(t, errorFound, "Should receive StreamError update")

		mockClient.AssertExpectations(t)
	})
}

func TestStreamingUpdateTypes(t *testing.T) {
	t.Run("should have correct streaming update constants", func(t *testing.T) {
		assert.Equal(t, StreamingUpdateType(0), StreamStarted)
		assert.Equal(t, StreamingUpdateType(1), ChunkReceived)
		assert.Equal(t, StreamingUpdateType(2), MessageComplete)
		assert.Equal(t, StreamingUpdateType(3), StreamError)
		assert.Equal(t, StreamingUpdateType(4), ToolExecutionStarted)
		assert.Equal(t, StreamingUpdateType(5), ToolExecutionComplete)
	})
}

func TestStreamingMetadata(t *testing.T) {
	t.Run("should create streaming metadata correctly", func(t *testing.T) {
		metadata := StreamingMetadata{
			ChunkCount:    5,
			ContentLength: 100,
			Duration:      2 * time.Second,
			Model:         "test-model",
		}

		assert.Equal(t, 5, metadata.ChunkCount)
		assert.Equal(t, 100, metadata.ContentLength)
		assert.Equal(t, 2*time.Second, metadata.Duration)
		assert.Equal(t, "test-model", metadata.Model)
	})
}

// Test that streaming preserves conversation state
func TestStreamingConversationState(t *testing.T) {
	t.Run("should preserve conversation state during streaming", func(t *testing.T) {
		mockClient := &MockStreamingChatClient{}
		controller := NewChatController(mockClient, "test-model", nil)

		// Add initial message
		controller.AddUserMessage("Initial message")
		initialCount := controller.GetMessageCount()

		// Mock streaming response
		chunksChan := make(chan chat.MessageChunk, 2)
		chunksChan <- chat.MessageChunk{
			StreamID:  "conversation-test",
			Content:   "Response content",
			Done:      true,
			Timestamp: time.Now(),
			Message:   chat.Message{Role: chat.RoleAssistant, Content: "Response content"},
		}
		close(chunksChan)

		mockClient.On("StreamMessage", mock.Anything, mock.Anything).Return((<-chan chat.MessageChunk)(chunksChan), nil)

		ctx := context.Background()
		updates, err := controller.StartStreaming(ctx, "New message")

		require.NoError(t, err)

		// Process all updates
		for range updates {
			// Just consume the updates
		}

		// Conversation should have grown
		finalCount := controller.GetMessageCount()
		assert.Greater(t, finalCount, initialCount)

		// Should have both user message and assistant response
		assert.GreaterOrEqual(t, finalCount, initialCount+2)

		mockClient.AssertExpectations(t)
	})
}
