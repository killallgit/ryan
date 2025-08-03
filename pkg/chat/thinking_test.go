package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMessageThinking(t *testing.T) {
	t.Run("should parse content with thinking block", func(t *testing.T) {
		content := "<think>Complex reasoning here</think>Final response to user"
		parsed := ParseMessageThinking(content)

		assert.True(t, parsed.HasThinking)
		assert.Equal(t, "Complex reasoning here", parsed.ThinkingContent)
		assert.Equal(t, "Final response to user", parsed.ResponseContent)
	})

	t.Run("should handle content without thinking block", func(t *testing.T) {
		content := "Just a regular response"
		parsed := ParseMessageThinking(content)

		assert.False(t, parsed.HasThinking)
		assert.Equal(t, "", parsed.ThinkingContent)
		assert.Equal(t, "Just a regular response", parsed.ResponseContent)
	})

	t.Run("should handle multiple thinking blocks", func(t *testing.T) {
		content := "<think>First thought</think>Some text<think>Second thought</think>Final response"
		parsed := ParseMessageThinking(content)

		assert.True(t, parsed.HasThinking)
		assert.Equal(t, "First thought\n\nSecond thought", parsed.ThinkingContent)
		assert.Equal(t, "Some textFinal response", parsed.ResponseContent)
	})

	t.Run("should handle thinking tags with variation", func(t *testing.T) {
		content := "<thinking>Detailed analysis</thinking>My answer is correct"
		parsed := ParseMessageThinking(content)

		assert.True(t, parsed.HasThinking)
		assert.Equal(t, "Detailed analysis", parsed.ThinkingContent)
		assert.Equal(t, "My answer is correct", parsed.ResponseContent)
	})

	t.Run("should handle empty thinking block", func(t *testing.T) {
		content := "<think></think>Response only"
		parsed := ParseMessageThinking(content)

		assert.False(t, parsed.HasThinking) // Empty thinking should be treated as no thinking
		assert.Equal(t, "", parsed.ThinkingContent)
		assert.Equal(t, "Response only", parsed.ResponseContent)
	})

	t.Run("should trim whitespace properly", func(t *testing.T) {
		content := "  <think>  Thinking with spaces  </think>  Response with spaces  "
		parsed := ParseMessageThinking(content)

		assert.True(t, parsed.HasThinking)
		assert.Equal(t, "Thinking with spaces", parsed.ThinkingContent)
		assert.Equal(t, "Response with spaces", parsed.ResponseContent)
	})
}

func TestExtractResponseContent(t *testing.T) {
	t.Run("should extract only response content", func(t *testing.T) {
		content := "<think>Internal reasoning</think>This is the user response"
		response := ExtractResponseContent(content)

		assert.Equal(t, "This is the user response", response)
	})

	t.Run("should return full content when no thinking blocks", func(t *testing.T) {
		content := "Regular message with no thinking"
		response := ExtractResponseContent(content)

		assert.Equal(t, "Regular message with no thinking", response)
	})

	t.Run("should handle complex mixed content", func(t *testing.T) {
		content := "Prefix <think>reasoning step 1</think> middle <think>reasoning step 2</think> suffix"
		response := ExtractResponseContent(content)

		assert.Equal(t, "Prefix  middle  suffix", response)
	})
}
