package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDispatcherAgent_CanHandle(t *testing.T) {
	orchestrator := NewOrchestrator()
	dispatcher := NewDispatcherAgent(orchestrator)

	tests := []string{
		"analyze this code",
		"create a new file",
		"what is the weather like",
		"read all files in directory",
		"complex multi-step task",
		"",
		"single word",
	}

	for _, request := range tests {
		t.Run(request, func(t *testing.T) {
			canHandle, confidence := dispatcher.CanHandle(request)
			
			// Dispatcher should handle any request
			assert.True(t, canHandle)
			assert.Equal(t, 1.0, confidence)
		})
	}
}

func TestDispatcherAgent_Basic(t *testing.T) {
	orchestrator := NewOrchestrator()
	dispatcher := NewDispatcherAgent(orchestrator)

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "dispatcher", dispatcher.Name())
	})

	t.Run("Description", func(t *testing.T) {
		desc := dispatcher.Description()
		assert.Contains(t, desc, "prompt")
		assert.Contains(t, desc, "execution")
		assert.Contains(t, desc, "plan")
	})
}