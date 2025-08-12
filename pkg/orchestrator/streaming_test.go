package orchestrator

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStreamHandler for testing streaming
type MockStreamHandler struct {
	chunks       []string
	finalContent string
	errors       []error
}

func NewMockStreamHandler() *MockStreamHandler {
	return &MockStreamHandler{
		chunks: []string{},
		errors: []error{},
	}
}

func (h *MockStreamHandler) OnChunk(chunk []byte) error {
	h.chunks = append(h.chunks, string(chunk))
	return nil
}

func (h *MockStreamHandler) OnComplete(finalContent string) error {
	h.finalContent = finalContent
	return nil
}

func (h *MockStreamHandler) OnError(err error) {
	h.errors = append(h.errors, err)
}

func (h *MockStreamHandler) GetFullContent() string {
	return strings.Join(h.chunks, "")
}

func TestStreamingHandler(t *testing.T) {
	t.Run("basic streaming", func(t *testing.T) {
		mockHandler := NewMockStreamHandler()
		streamHandler := NewStreamingHandler(mockHandler, true)

		// Test OnChunk
		err := streamHandler.OnChunk([]byte("Hello "))
		assert.NoError(t, err)
		err = streamHandler.OnChunk([]byte("World"))
		assert.NoError(t, err)

		assert.Equal(t, 2, len(mockHandler.chunks))
		assert.Equal(t, "Hello World", mockHandler.GetFullContent())

		// Test OnComplete
		err = streamHandler.OnComplete("Final content")
		assert.NoError(t, err)
		assert.Equal(t, "Final content", mockHandler.finalContent)

		// Test OnError
		streamHandler.OnError(assert.AnError)
		assert.Equal(t, 1, len(mockHandler.errors))
	})

	t.Run("routing decision display", func(t *testing.T) {
		mockHandler := NewMockStreamHandler()
		streamHandler := NewStreamingHandler(mockHandler, true)

		decision := &RouteDecision{
			TargetAgent: AgentReasoner,
			Instruction: "Test task",
		}

		err := streamHandler.OnRoutingDecision(decision)
		assert.NoError(t, err)
		assert.Contains(t, mockHandler.GetFullContent(), "Routing to reasoner agent")
		assert.Contains(t, mockHandler.GetFullContent(), "Test task")
	})

	t.Run("routing info disabled", func(t *testing.T) {
		mockHandler := NewMockStreamHandler()
		streamHandler := NewStreamingHandler(mockHandler, false) // Routing disabled

		decision := &RouteDecision{
			TargetAgent: AgentReasoner,
		}

		err := streamHandler.OnRoutingDecision(decision)
		assert.NoError(t, err)
		assert.Empty(t, mockHandler.chunks) // No output when disabled
	})

	t.Run("agent status notifications", func(t *testing.T) {
		mockHandler := NewMockStreamHandler()
		streamHandler := NewStreamingHandler(mockHandler, true)

		// Test agent start
		err := streamHandler.OnAgentStart(AgentToolCaller)
		assert.NoError(t, err)
		assert.Contains(t, mockHandler.GetFullContent(), "tool_caller agent processing")

		// Test agent complete - success
		err = streamHandler.OnAgentComplete(AgentToolCaller, "success")
		assert.NoError(t, err)
		assert.Contains(t, mockHandler.GetFullContent(), "✅")
		assert.Contains(t, mockHandler.GetFullContent(), "tool_caller agent success")

		// Test agent complete - failure
		mockHandler.chunks = []string{} // Reset
		err = streamHandler.OnAgentComplete(AgentReasoner, "failed")
		assert.NoError(t, err)
		assert.Contains(t, mockHandler.GetFullContent(), "❌")
		assert.Contains(t, mockHandler.GetFullContent(), "reasoner agent failed")
	})
}

func TestOrchestratorExecuteStream(t *testing.T) {
	t.Run("basic streaming execution", func(t *testing.T) {
		// Create mock LLM with intent response
		mockLLM := NewSimpleMockLLM()
		mockLLM.WithIntentResponse(&TaskIntent{
			Type:       "reasoning",
			Confidence: 0.9,
		})

		// Create orchestrator
		orch, err := New(mockLLM)
		require.NoError(t, err)

		// Register a mock agent
		mockAgent := NewSimpleMockAgent(AgentReasoner)
		err = orch.RegisterAgent(AgentReasoner, mockAgent)
		require.NoError(t, err)

		// Create mock handler
		mockHandler := NewMockStreamHandler()

		// Execute with streaming
		ctx := context.Background()
		result, err := orch.ExecuteStream(ctx, "Test query", mockHandler)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, StatusCompleted, result.Status)

		// Check that streaming output was generated
		fullContent := mockHandler.GetFullContent()
		assert.Contains(t, fullContent, "Orchestrator Processing")
		assert.Contains(t, fullContent, "Analyzing task intent")
		assert.Contains(t, fullContent, "Intent**: reasoning") // Format includes **
		assert.Contains(t, fullContent, "Execution Summary")
	})

	t.Run("streaming with agent failure", func(t *testing.T) {
		mockLLM := NewSimpleMockLLM()
		mockLLM.WithIntentResponse(&TaskIntent{
			Type:       "tool_use",
			Confidence: 0.8,
		})

		orch, err := New(mockLLM, WithMaxIterations(2))
		require.NoError(t, err)

		// Register a failing mock agent
		mockAgent := &SimpleMockAgent{
			agentType: AgentToolCaller,
			responses: []AgentResponse{
				{
					AgentType: AgentToolCaller,
					Status:    "failed",
					Error:     stringPtr("Mock failure"),
				},
			},
		}
		err = orch.RegisterAgent(AgentToolCaller, mockAgent)
		require.NoError(t, err)

		mockHandler := NewMockStreamHandler()
		ctx := context.Background()

		result, err := orch.ExecuteStream(ctx, "Test with failure", mockHandler)

		// Should still return result even with failure
		assert.NotNil(t, result)

		// Check streaming output shows failure
		fullContent := mockHandler.GetFullContent()
		assert.Contains(t, fullContent, "tool_caller")
	})
}

func TestStreamingFeedbackLoop(t *testing.T) {
	t.Run("successful completion", func(t *testing.T) {
		mockLLM := NewSimpleMockLLM()
		orch, err := New(mockLLM)
		require.NoError(t, err)

		// Register successful agent
		mockAgent := NewSimpleMockAgent(AgentReasoner)
		err = orch.RegisterAgent(AgentReasoner, mockAgent)
		require.NoError(t, err)

		mockHandler := NewMockStreamHandler()
		streamHandler := NewStreamingHandler(mockHandler, true)

		state := &TaskState{
			ID:     "test-123",
			Query:  "Test query",
			Intent: &TaskIntent{Type: "reasoning", Confidence: 0.9},
			Status: StatusInProgress,
		}

		ctx := context.Background()
		result, err := orch.streamingFeedbackLoop(ctx, state, streamHandler)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, StatusCompleted, state.Status)
		assert.Equal(t, "test-123", result.ID)
	})

	t.Run("context cancellation", func(t *testing.T) {
		mockLLM := NewSimpleMockLLM()
		orch, err := New(mockLLM)
		require.NoError(t, err)

		mockHandler := NewMockStreamHandler()
		streamHandler := NewStreamingHandler(mockHandler, true)

		state := &TaskState{
			ID:     "test-cancel",
			Query:  "Test query",
			Intent: &TaskIntent{Type: "reasoning"},
			Status: StatusInProgress,
		}

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		result, err := orch.streamingFeedbackLoop(ctx, state, streamHandler)

		assert.Error(t, err)
		assert.Equal(t, StatusCancelled, state.Status)
		assert.NotNil(t, result)
	})
}

func TestFormatStreamingSummary(t *testing.T) {
	orch := &Orchestrator{}

	result := &TaskResult{
		Duration: 5 * 1000000000, // 5 seconds
		Status:   StatusCompleted,
		History: []AgentResponse{
			{AgentType: AgentReasoner},
			{AgentType: AgentToolCaller},
		},
	}

	summary := orch.formatStreamingSummary(result)

	assert.Contains(t, summary, "Execution Summary")
	assert.Contains(t, summary, "Duration: 5s")
	assert.Contains(t, summary, "Agent interactions: 2")
	assert.Contains(t, summary, "Status: completed")
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}
