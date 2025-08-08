package chat

import "time"

// MessageNode represents a display element
type MessageNode struct {
	ID          string
	Type        string // "user", "assistant", "system", "tool", "agent"
	Content     string
	Timestamp   time.Time
	StreamID    string // Link to stream if applicable
	IsStreaming bool
}
