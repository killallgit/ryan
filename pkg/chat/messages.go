package chat

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID        string           `json:"id"`         // UUID for unique identification
	ParentID  *string          `json:"parent_id"`  // Reference to parent message
	ContextID string           `json:"context_id"` // Which conversation branch this belongs to
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	Timestamp time.Time        `json:"timestamp"`
	ToolCalls []ToolCall       `json:"tool_calls,omitempty"`
	ToolName  string           `json:"tool_name,omitempty"`
	Metadata  *MessageMetadata `json:"metadata,omitempty"`
}

// MessageMetadata contains message lifecycle and processing information
type MessageMetadata struct {
	StreamID    string `json:"stream_id,omitempty"`    // ID of the streaming session
	ChunkIndex  int    `json:"chunk_index,omitempty"`  // Order in streaming chunks
	IsStreaming bool   `json:"is_streaming,omitempty"` // Whether this is a partial message
	Source      string `json:"source,omitempty"`       // "optimistic", "streaming", "final"

	// Context-aware fields for conversation branching
	BranchPoint bool    `json:"branch_point,omitempty"` // True if this message has children
	ChildCount  int     `json:"child_count,omitempty"`  // Number of response branches
	Depth       int     `json:"depth,omitempty"`        // Distance from conversation root
	ThreadTitle *string `json:"thread_title,omitempty"` // User-defined branch name
}

type ToolCall struct {
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

const (
	RoleUser         = "user"
	RoleAssistant    = "assistant"
	RoleSystem       = "system"
	RoleError        = "error"
	RoleTool         = "tool"
	RoleToolProgress = "tool_progress"
)

// Message source constants for lifecycle tracking
const (
	MessageSourceOptimistic = "optimistic" // From UI optimistic updates
	MessageSourceStreaming  = "streaming"  // From streaming chunks
	MessageSourceFinal      = "final"      // Completed message
)

// generateMessageID creates a new unique message ID
func generateMessageID() string {
	return uuid.New().String()
}

// generateContextID creates a new unique context ID
func generateContextID() string {
	return uuid.New().String()
}

func NewUserMessage(content string) Message {
	return Message{
		ID:        generateMessageID(),
		ParentID:  nil,
		ContextID: "", // Will be set by context manager
		Role:      RoleUser,
		Content:   strings.TrimSpace(content),
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
			Depth:  0,
		},
	}
}

func NewAssistantMessage(content string) Message {
	return Message{
		ID:        generateMessageID(),
		ParentID:  nil,
		ContextID: "", // Will be set by context manager
		Role:      RoleAssistant,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
			Depth:  0,
		},
	}
}

func NewSystemMessage(content string) Message {
	return Message{
		ID:        generateMessageID(),
		ParentID:  nil,
		ContextID: "", // Will be set by context manager
		Role:      RoleSystem,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
			Depth:  0,
		},
	}
}

func NewErrorMessage(content string) Message {
	return Message{
		ID:        generateMessageID(),
		ParentID:  nil,
		ContextID: "", // Will be set by context manager
		Role:      RoleError,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
			Depth:  0,
		},
	}
}

func NewAssistantMessageWithToolCalls(toolCalls []ToolCall) Message {
	return Message{
		ID:        generateMessageID(),
		ParentID:  nil,
		ContextID: "", // Will be set by context manager
		Role:      RoleAssistant,
		Content:   "",
		ToolCalls: toolCalls,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
			Depth:  0,
		},
	}
}

func NewToolResultMessage(toolName, content string) Message {
	return Message{
		ID:        generateMessageID(),
		ParentID:  nil,
		ContextID: "", // Will be set by context manager
		Role:      RoleTool,
		Content:   content,
		ToolName:  toolName,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
			Depth:  0,
		},
	}
}

// NewToolProgressMessage creates a message showing tool execution progress
func NewToolProgressMessage(toolName, command string) Message {
	// Truncate command if too long (like Claude Code does)
	truncatedCommand := command
	if len(command) > 50 {
		truncatedCommand = command[:47] + "..."
	}

	content := fmt.Sprintf("%s(%s)", toolName, truncatedCommand)

	return Message{
		ID:        generateMessageID(),
		ParentID:  nil,
		ContextID: "", // Will be set by context manager
		Role:      RoleToolProgress,
		Content:   content,
		ToolName:  toolName,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceOptimistic,
			Depth:  0,
		},
	}
}

// Enhanced constructors with metadata support

// NewUserMessageWithSource creates a user message with source metadata
func NewUserMessageWithSource(content, source string) Message {
	return Message{
		ID:        generateMessageID(),
		ParentID:  nil,
		ContextID: "", // Will be set by context manager
		Role:      RoleUser,
		Content:   strings.TrimSpace(content),
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: source,
			Depth:  0,
		},
	}
}

// NewOptimisticUserMessage creates a user message marked as optimistic
func NewOptimisticUserMessage(content string) Message {
	return NewUserMessageWithSource(content, MessageSourceOptimistic)
}

// NewStreamingMessage creates a message marked as streaming
func NewStreamingMessage(role, content, streamID string, chunkIndex int) Message {
	return Message{
		ID:        generateMessageID(),
		ParentID:  nil,
		ContextID: "", // Will be set by context manager
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			StreamID:    streamID,
			ChunkIndex:  chunkIndex,
			IsStreaming: true,
			Source:      MessageSourceStreaming,
			Depth:       0,
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
		ID:        m.ID,
		ParentID:  m.ParentID,
		ContextID: m.ContextID,
		Role:      m.Role,
		Content:   m.Content,
		Timestamp: t,
		ToolCalls: m.ToolCalls,
		ToolName:  m.ToolName,
		Metadata:  m.Metadata,
	}
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

// WithMetadata adds or updates metadata to a message
func (m Message) WithMetadata(metadata MessageMetadata) Message {
	return Message{
		ID:        m.ID,
		ParentID:  m.ParentID,
		ContextID: m.ContextID,
		Role:      m.Role,
		Content:   m.Content,
		Timestamp: m.Timestamp,
		ToolCalls: m.ToolCalls,
		ToolName:  m.ToolName,
		Metadata:  &metadata,
	}
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

// RemoveThinkingBlocks removes <think> or <thinking> blocks from content
func RemoveThinkingBlocks(content string) string {
	// Remove <think>...</think> blocks (case insensitive)
	thinkRegex := regexp.MustCompile(`(?is)<think(?:ing)?>\s*.*?\s*</think(?:ing)?>`)
	cleanContent := thinkRegex.ReplaceAllString(content, "")

	// Trim any extra whitespace that might be left
	cleanContent = strings.TrimSpace(cleanContent)

	// Clean up multiple consecutive newlines that might result from removal
	multiNewlineRegex := regexp.MustCompile(`\n{3,}`)
	cleanContent = multiNewlineRegex.ReplaceAllString(cleanContent, "\n\n")

	return cleanContent
}
