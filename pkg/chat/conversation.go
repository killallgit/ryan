package chat

import (
	"strings"
	"time"
)

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

// Enhanced conversation management functions

// AddMessageWithDeduplication adds a message while preventing duplicates based on source and content
func AddMessageWithDeduplication(conv Conversation, msg Message) Conversation {
	// If this is a final message, remove any optimistic messages with similar content
	if msg.GetSource() == MessageSourceFinal && msg.IsUser() {
		conv = RemoveOptimisticMessages(conv, msg.Content)
	}

	// Add the new message
	return AddMessage(conv, msg)
}

// RemoveOptimisticMessages removes optimistic messages that match the given content
func RemoveOptimisticMessages(conv Conversation, content string) Conversation {
	var filteredMessages []Message

	for _, existingMsg := range conv.Messages {
		// Keep message if it's not optimistic or doesn't match content
		if !existingMsg.IsOptimistic() ||
			strings.TrimSpace(existingMsg.Content) != strings.TrimSpace(content) {
			filteredMessages = append(filteredMessages, existingMsg)
		}
	}

	return Conversation{
		Messages: filteredMessages,
		Model:    conv.Model,
	}
}

// ReplaceOptimisticMessage replaces an optimistic message with a final one
func ReplaceOptimisticMessage(conv Conversation, optimisticContent string, finalMsg Message) Conversation {
	for i, msg := range conv.Messages {
		if msg.IsOptimistic() && strings.TrimSpace(msg.Content) == strings.TrimSpace(optimisticContent) {
			// Replace the optimistic message with the final one
			messages := make([]Message, len(conv.Messages))
			copy(messages, conv.Messages)
			messages[i] = finalMsg

			return Conversation{
				Messages: messages,
				Model:    conv.Model,
			}
		}
	}

	// If no optimistic message found, just add the final message
	return AddMessage(conv, finalMsg)
}

// GetOptimisticMessages returns all optimistic messages
func GetOptimisticMessages(conv Conversation) []Message {
	var optimistic []Message
	for _, msg := range conv.Messages {
		if msg.IsOptimistic() {
			optimistic = append(optimistic, msg)
		}
	}
	return optimistic
}

// GetStreamingMessages returns all streaming messages
func GetStreamingMessages(conv Conversation) []Message {
	var streaming []Message
	for _, msg := range conv.Messages {
		if msg.IsStreaming() {
			streaming = append(streaming, msg)
		}
	}
	return streaming
}

// RemoveStreamingMessages removes all streaming messages (useful when streaming completes)
func RemoveStreamingMessages(conv Conversation, streamID string) Conversation {
	var filteredMessages []Message

	for _, msg := range conv.Messages {
		// Keep message if it's not streaming or has different stream ID
		if !msg.IsStreaming() || msg.GetStreamID() != streamID {
			filteredMessages = append(filteredMessages, msg)
		}
	}

	return Conversation{
		Messages: filteredMessages,
		Model:    conv.Model,
	}
}
