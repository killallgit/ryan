package chat

import "time"

type Conversation struct {
	Messages []Message
	Model    string
}

func NewConversation(model string) Conversation {
	return Conversation{
		Messages: make([]Message, 0),
		Model:    model,
	}
}

func NewConversationWithSystem(model, systemPrompt string) Conversation {
	conv := NewConversation(model)
	if systemPrompt != "" {
		conv = AddMessage(conv, NewSystemMessage(systemPrompt))
	}
	return conv
}

func AddMessage(conv Conversation, msg Message) Conversation {
	messages := make([]Message, len(conv.Messages)+1)
	copy(messages, conv.Messages)
	messages[len(conv.Messages)] = msg

	return Conversation{
		Messages: messages,
		Model:    conv.Model,
	}
}

func GetMessages(conv Conversation) []Message {
	result := make([]Message, len(conv.Messages))
	copy(result, conv.Messages)
	return result
}

func GetMessageCount(conv Conversation) int {
	return len(conv.Messages)
}

func GetLastMessage(conv Conversation) (Message, bool) {
	if len(conv.Messages) == 0 {
		return Message{}, false
	}
	return conv.Messages[len(conv.Messages)-1], true
}

func GetLastAssistantMessage(conv Conversation) (Message, bool) {
	for i := len(conv.Messages) - 1; i >= 0; i-- {
		msg := conv.Messages[i]
		if msg.IsAssistant() {
			return msg, true
		}
	}
	return Message{}, false
}

func GetLastUserMessage(conv Conversation) (Message, bool) {
	for i := len(conv.Messages) - 1; i >= 0; i-- {
		msg := conv.Messages[i]
		if msg.IsUser() {
			return msg, true
		}
	}
	return Message{}, false
}

func GetMessagesByRole(conv Conversation, role string) []Message {
	var result []Message
	for _, msg := range conv.Messages {
		if msg.Role == role {
			result = append(result, msg)
		}
	}
	return result
}

func GetMessagesAfter(conv Conversation, timestamp time.Time) []Message {
	var result []Message
	for _, msg := range conv.Messages {
		if msg.Timestamp.After(timestamp) {
			result = append(result, msg)
		}
	}
	return result
}

func GetMessagesBefore(conv Conversation, timestamp time.Time) []Message {
	var result []Message
	for _, msg := range conv.Messages {
		if msg.Timestamp.Before(timestamp) {
			result = append(result, msg)
		}
	}
	return result
}

func IsEmpty(conv Conversation) bool {
	return len(conv.Messages) == 0
}

func HasSystemMessage(conv Conversation) bool {
	for _, msg := range conv.Messages {
		if msg.IsSystem() {
			return true
		}
	}
	return false
}

func WithModel(conv Conversation, model string) Conversation {
	return Conversation{
		Messages: conv.Messages,
		Model:    model,
	}
}
