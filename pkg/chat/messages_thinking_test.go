package chat

import (
	"testing"
)

func TestRemoveThinkingBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple think block",
			input:    "<think>This is my internal thought</think>This is my response",
			expected: "This is my response",
		},
		{
			name:     "thinking block variant",
			input:    "<thinking>Processing the request...</thinking>Here's the answer",
			expected: "Here's the answer",
		},
		{
			name:     "case insensitive",
			input:    "<THINK>Uppercase thinking</THINK>Response text",
			expected: "Response text",
		},
		{
			name:     "multiline thinking",
			input:    "<think>\nLine 1\nLine 2\n</think>\nThe actual response",
			expected: "The actual response",
		},
		{
			name:     "multiple thinking blocks",
			input:    "<think>First thought</think>Part 1 <thinking>Second thought</thinking>Part 2",
			expected: "Part 1 Part 2",
		},
		{
			name:     "no thinking blocks",
			input:    "Just a regular message without any thinking",
			expected: "Just a regular message without any thinking",
		},
		{
			name:     "empty thinking block",
			input:    "<think></think>Response",
			expected: "Response",
		},
		{
			name:     "thinking block with extra whitespace",
			input:    "<think>  \n  Thought with spaces  \n  </think>  Response  ",
			expected: "Response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveThinkingBlocks(tt.input)
			if result != tt.expected {
				t.Errorf("RemoveThinkingBlocks(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
