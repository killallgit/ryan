package chat

import (
	"strings"
	"time"
)

type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Timestamp time.Time  `json:"timestamp"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	ToolName  string     `json:"tool_name,omitempty"`
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

func NewUserMessage(content string) Message {
	return Message{
		Role:      RoleUser,
		Content:   strings.TrimSpace(content),
		Timestamp: time.Now(),
	}
}

func NewAssistantMessage(content string) Message {
	return Message{
		Role:      RoleAssistant,
		Content:   content,
		Timestamp: time.Now(),
	}
}

func NewSystemMessage(content string) Message {
	return Message{
		Role:      RoleSystem,
		Content:   content,
		Timestamp: time.Now(),
	}
}

func NewErrorMessage(content string) Message {
	return Message{
		Role:      RoleError,
		Content:   content,
		Timestamp: time.Now(),
	}
}

func NewAssistantMessageWithToolCalls(toolCalls []ToolCall) Message {
	return Message{
		Role:      RoleAssistant,
		Content:   "",
		ToolCalls: toolCalls,
		Timestamp: time.Now(),
	}
}

func NewToolResultMessage(toolName, content string) Message {
	return Message{
		Role:      RoleTool,
		Content:   content,
		ToolName:  toolName,
		Timestamp: time.Now(),
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
	}
}
