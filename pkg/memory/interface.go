package memory

import (
	"github.com/killallgit/ryan/pkg/llm"
	"github.com/tmc/langchaingo/llms"
)

// MemoryStore defines the interface for conversation memory storage
type MemoryStore interface {
	// IsEnabled returns whether memory is enabled
	IsEnabled() bool

	// AddUserMessage adds a user message to memory
	AddUserMessage(content string) error

	// AddAssistantMessage adds an assistant message to memory
	AddAssistantMessage(content string) error

	// GetMessages retrieves all messages from memory
	GetMessages() ([]llms.ChatMessage, error)

	// ConvertToLLMMessages converts memory messages to LLM format
	ConvertToLLMMessages() ([]llm.Message, error)

	// Clear clears all messages from memory
	Clear() error

	// Close closes the memory store
	Close() error
}
