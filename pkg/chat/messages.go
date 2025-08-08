package chat

import (
	"time"
)

// MessageRole represents the role of a message sender
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

// Message represents a chat message independent of any UI framework
type Message struct {
	ID        string      `json:"id"`
	Role      MessageRole `json:"role"`
	Content   string      `json:"content"`
	Timestamp time.Time   `json:"timestamp"`
	Metadata  Metadata    `json:"metadata,omitempty"`
}

// Metadata contains optional message metadata
type Metadata struct {
	Model       string `json:"model,omitempty"`
	TokensUsed  int    `json:"tokens_used,omitempty"`
	StreamID    string `json:"stream_id,omitempty"`
	IsStreaming bool   `json:"is_streaming,omitempty"`
	Error       string `json:"error,omitempty"`
}

// NewMessage creates a new message with the given role and content
func NewMessage(role MessageRole, content string) *Message {
	return &Message{
		ID:        generateMessageID(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// generateMessageID creates a unique message ID
func generateMessageID() string {
	return time.Now().Format("20060102-150405") + "-" + generateRandomString(8)
}

// generateRandomString generates a random string of the specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
