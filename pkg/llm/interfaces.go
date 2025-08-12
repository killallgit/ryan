package llm

import (
	"context"

	"github.com/killallgit/ryan/pkg/stream"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Generate generates a response for the given prompt
	Generate(ctx context.Context, prompt string) (string, error)

	// GenerateStream generates a streaming response for the given prompt
	GenerateStream(ctx context.Context, prompt string, handler stream.Handler) error

	// GetName returns the provider name
	GetName() string

	// GetModel returns the current model name
	GetModel() string
}

// StreamHandler defines the interface for handling streaming responses
type StreamHandler = stream.Handler

// TokenCounter provides token counting capabilities
type TokenCounter interface {
	// CountTokens counts the tokens in the given text
	CountTokens(text string) (int, error)
}

// Message represents a message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ConversationalProvider extends Provider with conversation support
type ConversationalProvider interface {
	Provider

	// GenerateWithHistory generates a response considering conversation history
	GenerateWithHistory(ctx context.Context, messages []Message) (string, error)

	// GenerateStreamWithHistory generates a streaming response with history
	GenerateStreamWithHistory(ctx context.Context, messages []Message, handler stream.Handler) error
}

// ProviderConfig contains configuration for an LLM provider
type ProviderConfig struct {
	Name     string                 `json:"name"`
	Model    string                 `json:"model"`
	Endpoint string                 `json:"endpoint,omitempty"`
	APIKey   string                 `json:"api_key,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// Registry manages LLM providers
type Registry interface {
	// Register registers a new provider
	Register(name string, provider Provider) error

	// Get retrieves a provider by name
	Get(name string) (Provider, error)

	// List returns all registered provider names
	List() []string

	// SetDefault sets the default provider
	SetDefault(name string) error

	// GetDefault returns the default provider
	GetDefault() (Provider, error)
}
