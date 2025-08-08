package chat

import "time"

// Stream messages with source identification
type StreamStartMsg struct {
	StreamID   string
	SourceType string // "user", "assistant", "system", "agent", "tool"
	Metadata   map[string]interface{}
}

type StreamChunkMsg struct {
	StreamID   string
	Content    string
	SourceType string
}

type StreamEndMsg struct {
	StreamID     string
	Error        error
	FinalContent string
}

// Node creation message
type CreateNodeMsg struct {
	NodeType  string // "user", "assistant", "system", etc.
	Content   string
	StreamID  string
	Timestamp time.Time
}

// MessageNode represents a display element
type MessageNode struct {
	ID          string
	Type        string // "user", "assistant", "system", "tool", "agent"
	Content     string
	Timestamp   time.Time
	StreamID    string // Link to stream if applicable
	IsStreaming bool
}
