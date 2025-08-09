package memory

import (
	"github.com/killallgit/ryan/pkg/llm"
	"github.com/tmc/langchaingo/llms"
)

// MockMemory is a mock implementation of MemoryStore for testing
type MockMemory struct {
	messages []llms.ChatMessage
	Closed   bool

	// Error injection for testing
	AddUserError      error
	AddAssistantError error
	GetMessagesError  error
	ClearError        error
}

// NewMockMemory creates a new mock memory store
func NewMockMemory() *MockMemory {
	return &MockMemory{
		messages: []llms.ChatMessage{},
	}
}

func (m *MockMemory) IsEnabled() bool {
	return true
}

func (m *MockMemory) AddUserMessage(content string) error {
	if m.AddUserError != nil {
		return m.AddUserError
	}
	m.messages = append(m.messages, llms.HumanChatMessage{Content: content})
	return nil
}

func (m *MockMemory) AddAssistantMessage(content string) error {
	if m.AddAssistantError != nil {
		return m.AddAssistantError
	}
	m.messages = append(m.messages, llms.AIChatMessage{Content: content})
	return nil
}

func (m *MockMemory) GetMessages() ([]llms.ChatMessage, error) {
	if m.GetMessagesError != nil {
		return nil, m.GetMessagesError
	}
	return m.messages, nil
}

func (m *MockMemory) ConvertToLLMMessages() ([]llm.Message, error) {

	messages, err := m.GetMessages()
	if err != nil {
		return nil, err
	}

	var result []llm.Message
	for _, msg := range messages {
		switch msg.GetType() {
		case llms.ChatMessageTypeHuman:
			result = append(result, llm.Message{
				Role:    "user",
				Content: msg.GetContent(),
			})
		case llms.ChatMessageTypeAI:
			result = append(result, llm.Message{
				Role:    "assistant",
				Content: msg.GetContent(),
			})
		case llms.ChatMessageTypeSystem:
			result = append(result, llm.Message{
				Role:    "system",
				Content: msg.GetContent(),
			})
		}
	}
	return result, nil
}

func (m *MockMemory) Clear() error {
	if m.ClearError != nil {
		return m.ClearError
	}
	m.messages = []llms.ChatMessage{}
	return nil
}

func (m *MockMemory) Close() error {
	m.Closed = true
	return nil
}
