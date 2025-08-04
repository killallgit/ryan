package chat

import (
	"strings"
	"time"
)

type Message struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	Timestamp time.Time        `json:"timestamp"`
	ToolCalls []ToolCall       `json:"tool_calls,omitempty"`
	ToolName  string           `json:"tool_name,omitempty"`
	Thinking  *ThinkingBlock   `json:"thinking,omitempty"`
	Metadata  *MessageMetadata `json:"metadata,omitempty"`
}

// ThinkingBlock represents separated thinking content
type ThinkingBlock struct {
	Content string `json:"content"`
	Visible bool   `json:"visible"`
}

// MessageMetadata contains message lifecycle and processing information
type MessageMetadata struct {
	StreamID    string `json:"stream_id,omitempty"`    // ID of the streaming session
	ChunkIndex  int    `json:"chunk_index,omitempty"`  // Order in streaming chunks
	IsStreaming bool   `json:"is_streaming,omitempty"` // Whether this is a partial message
	Source      string `json:"source,omitempty"`       // "optimistic", "streaming", "final"
}

type ToolCall struct {
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleError     = "error"
	RoleTool      = "tool"
)

// Message source constants for lifecycle tracking
const (
	MessageSourceOptimistic = "optimistic" // From UI optimistic updates
	MessageSourceStreaming  = "streaming"  // From streaming chunks
	MessageSourceFinal      = "final"      // Completed message
)

func NewUserMessage(content string) Message {
	return Message{
		Role:      RoleUser,
		Content:   strings.TrimSpace(content),
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
		},
	}
}

func NewAssistantMessage(content string) Message {
	return Message{
		Role:      RoleAssistant,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
		},
	}
}

func NewSystemMessage(content string) Message {
	return Message{
		Role:      RoleSystem,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
		},
	}
}

func NewErrorMessage(content string) Message {
	return Message{
		Role:      RoleError,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
		},
	}
}

func NewAssistantMessageWithToolCalls(toolCalls []ToolCall) Message {
	return Message{
		Role:      RoleAssistant,
		Content:   "",
		ToolCalls: toolCalls,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
		},
	}
}

func NewToolResultMessage(toolName, content string) Message {
	return Message{
		Role:      RoleTool,
		Content:   content,
		ToolName:  toolName,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
		},
	}
}

// Enhanced constructors with metadata support

// NewUserMessageWithSource creates a user message with source metadata
func NewUserMessageWithSource(content, source string) Message {
	return Message{
		Role:      RoleUser,
		Content:   strings.TrimSpace(content),
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: source,
		},
	}
}

// NewOptimisticUserMessage creates a user message marked as optimistic
func NewOptimisticUserMessage(content string) Message {
	return NewUserMessageWithSource(content, MessageSourceOptimistic)
}

// NewAssistantMessageWithThinking creates an assistant message with separated thinking
func NewAssistantMessageWithThinking(content, thinking string, showThinking bool) Message {
	msg := Message{
		Role:      RoleAssistant,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
		},
	}

	if thinking != "" {
		msg.Thinking = &ThinkingBlock{
			Content: thinking,
			Visible: showThinking,
		}
	}

	return msg
}

// NewStreamingMessage creates a message marked as streaming
func NewStreamingMessage(role, content, streamID string, chunkIndex int) Message {
	return Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			StreamID:    streamID,
			ChunkIndex:  chunkIndex,
			IsStreaming: true,
			Source:      MessageSourceStreaming,
		},
	}
}

func (m Message) IsUser() bool {
	return m.Role == RoleUser
}

func (m Message) IsAssistant() bool {
	return m.Role == RoleAssistant
}

func (m Message) IsSystem() bool {
	return m.Role == RoleSystem
}

func (m Message) IsError() bool {
	return m.Role == RoleError
}

func (m Message) IsTool() bool {
	return m.Role == RoleTool
}

func (m Message) HasToolCalls() bool {
	return len(m.ToolCalls) > 0
}

func (m Message) IsEmpty() bool {
	return strings.TrimSpace(m.Content) == ""
}

func (m Message) WithTimestamp(t time.Time) Message {
	return Message{
		Role:      m.Role,
		Content:   m.Content,
		Timestamp: t,
		ToolCalls: m.ToolCalls,
		ToolName:  m.ToolName,
		Thinking:  m.Thinking,
		Metadata:  m.Metadata,
	}
}

// Helper methods for thinking blocks
func (m Message) HasThinking() bool {
	return m.Thinking != nil && strings.TrimSpace(m.Thinking.Content) != ""
}

func (m Message) IsThinkingVisible() bool {
	return m.HasThinking() && m.Thinking.Visible
}

// Helper methods for metadata
func (m Message) HasMetadata() bool {
	return m.Metadata != nil
}

func (m Message) GetSource() string {
	if m.HasMetadata() {
		return m.Metadata.Source
	}
	return MessageSourceFinal // Default for existing messages
}

func (m Message) IsOptimistic() bool {
	return m.GetSource() == MessageSourceOptimistic
}

func (m Message) IsStreaming() bool {
	return m.HasMetadata() && m.Metadata.IsStreaming
}

func (m Message) GetStreamID() string {
	if m.HasMetadata() {
		return m.Metadata.StreamID
	}
	return ""
}

// WithThinking adds or updates thinking content to a message
func (m Message) WithThinking(content string, visible bool) Message {
	updated := m
	if content != "" {
		updated.Thinking = &ThinkingBlock{
			Content: content,
			Visible: visible,
		}
	}
	return updated
}

// WithMetadata adds or updates metadata to a message
func (m Message) WithMetadata(metadata MessageMetadata) Message {
	updated := m
	updated.Metadata = &metadata
	return updated
}

// WithSource sets the message source in metadata
func (m Message) WithSource(source string) Message {
	if m.Metadata == nil {
		return m.WithMetadata(MessageMetadata{Source: source})
	}

	metadata := *m.Metadata
	metadata.Source = source
	return m.WithMetadata(metadata)
}
