package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/killallgit/ryan/pkg/llm"
	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory/sqlite3"
)

type Memory struct {
	store     *sqlite3.SqliteChatMessageHistory
	dbPath    string
	sessionID string
}

func New(sessionID string) (*Memory, error) {

	configRoot := filepath.Dir(viper.ConfigFileUsed())
	if configRoot == "" || configRoot == "." {
		configRoot = ".ryan"
	}
	// Create context directory for memory database
	contextDir := filepath.Join(configRoot, "context")
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create context directory: %w", err)
	}
	dbPath := filepath.Join(contextDir, "memory.db")

	connectionString := fmt.Sprintf("file:%s?cache=shared&mode=rwc", dbPath)
	chatHistory := sqlite3.NewSqliteChatMessageHistory(
		sqlite3.WithDBAddress(connectionString),
		sqlite3.WithSession(sessionID),
	)

	return &Memory{
		store:     chatHistory,
		dbPath:    dbPath,
		sessionID: sessionID,
	}, nil
}

func (m *Memory) IsEnabled() bool {
	return true
}

func (m *Memory) AddUserMessage(content string) error {
	return m.store.AddUserMessage(context.Background(), content)
}

func (m *Memory) AddAssistantMessage(content string) error {
	return m.store.AddAIMessage(context.Background(), content)
}

func (m *Memory) GetMessages() ([]llms.ChatMessage, error) {
	return m.store.Messages(context.Background())
}

func (m *Memory) ConvertToLLMMessages() ([]llm.Message, error) {

	chatMessages, err := m.GetMessages()
	if err != nil {
		return nil, err
	}

	var messages []llm.Message
	for _, msg := range chatMessages {
		switch msg.GetType() {
		case llms.ChatMessageTypeHuman:
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: msg.GetContent(),
			})
		case llms.ChatMessageTypeAI:
			messages = append(messages, llm.Message{
				Role:    "assistant",
				Content: msg.GetContent(),
			})
		case llms.ChatMessageTypeSystem:
			messages = append(messages, llm.Message{
				Role:    "system",
				Content: msg.GetContent(),
			})
		case llms.ChatMessageTypeTool:
			// Skip tool messages for now
			continue
		case llms.ChatMessageTypeGeneric, llms.ChatMessageTypeFunction:
			// Skip these message types
			continue
		}
	}

	windowSize := viper.GetInt("langchain.memory_window_size")
	if windowSize > 0 && len(messages) > windowSize {
		messages = messages[len(messages)-windowSize:]
	}

	return messages, nil
}

func (m *Memory) Clear() error {
	return m.store.Clear(context.Background())
}

func (m *Memory) Close() error {
	return nil
}

// ChatMessageHistory returns the underlying chat message history for use with agents
func (m *Memory) ChatMessageHistory() *sqlite3.SqliteChatMessageHistory {
	return m.store
}
