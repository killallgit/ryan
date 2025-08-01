package chat

import (
	"strings"
	"time"
)

type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
}

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
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

func (m Message) IsUser() bool {
	return m.Role == RoleUser
}

func (m Message) IsAssistant() bool {
	return m.Role == RoleAssistant
}

func (m Message) IsSystem() bool {
	return m.Role == RoleSystem
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
