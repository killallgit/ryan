package stream

import "context"

// Source defines providers that can generate streaming responses
type Source interface {
	// Stream initiates a streaming response
	Stream(ctx context.Context, prompt string, handler Handler) error

	// StreamWithHistory streams with conversation history
	StreamWithHistory(ctx context.Context, messages []Message, handler Handler) error
}

// Message represents a conversation message
type Message struct {
	Role    string
	Content string
}

// ConvertMessage creates a Message from role and content
func ConvertMessage(role, content string) Message {
	return Message{
		Role:    role,
		Content: content,
	}
}
