package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateString(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected string
	}{
		{"idle state", StateIdle, ""},
		{"sending state", StateSending, "sending"},
		{"receiving state", StateReceiving, "receiving"},
		{"thinking state", StateThinking, "thinking"},
		{"tool use state", StateToolUse, "tool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestStateGetIcon(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected string
	}{
		{"idle icon", StateIdle, ""},
		{"sending icon", StateSending, "↑"},
		{"receiving icon", StateReceiving, "↓"},
		{"thinking icon", StateThinking, "🤔"},
		{"tool use icon", StateToolUse, "🔨"},
		{"unknown state", State("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.GetIcon())
		})
	}
}

func TestStateGetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected string
	}{
		{"idle display", StateIdle, "Idle"},
		{"sending display", StateSending, "Sending"},
		{"receiving display", StateReceiving, "Receiving"},
		{"thinking display", StateThinking, "Thinking"},
		{"tool use display", StateToolUse, "Using tools"},
		{"unknown state", State("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.GetDisplayName())
		})
	}
}
