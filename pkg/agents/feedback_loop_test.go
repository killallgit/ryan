package agents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeedbackLoop_ProcessFeedback(t *testing.T) {
	fl := NewFeedbackLoop()
	require.NotNil(t, fl)

	// Test with validation error type which has simpler handling
	feedback := &FeedbackRequest{
		ID:         "feedback1",
		SourceTask: "task1",
		TargetTask: "task2",
		Type:       FeedbackTypeValidationError,
		Content:    "Validation failed: missing required field",
		Context: &ExecutionContext{
			SessionID:  "test-session",
			RequestID:  "test-request",
			SharedData: make(map[string]interface{}),
		},
	}

	// Process feedback - validation errors are logged but don't fail
	err := fl.ProcessFeedback(context.Background(), feedback)
	assert.NoError(t, err)
}

func TestFeedbackLoop_SetOrchestrator(t *testing.T) {
	fl := NewFeedbackLoop()
	o := NewOrchestrator()

	// Set orchestrator
	fl.SetOrchestrator(o)

	// Verify it doesn't panic
	assert.NotNil(t, fl)
}
