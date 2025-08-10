package chat

import (
	"context"
	"fmt"
	"sync"

	"github.com/killallgit/ryan/pkg/llm"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/memory"
)

// StreamCallback is called when new content is streamed
type StreamCallback func(content string) error

// Manager manages chat conversations
type Manager struct {
	history        *History
	memory         *memory.Memory
	currentStream  *StreamingMessage
	streamCallback StreamCallback
	mu             sync.RWMutex
}

// StreamingMessage represents a message being streamed
type StreamingMessage struct {
	Message       *Message
	ContentBuffer string
}

// NewManager creates a new chat manager
func NewManager(historyPath string) (*Manager, error) {
	logger.Debug("Creating chat manager with history path: %s", historyPath)

	history, err := NewHistory(historyPath)
	if err != nil {
		logger.Error("Failed to create history: %v", err)
		return nil, fmt.Errorf("failed to create history: %w", err)
	}

	// Create memory with a session ID based on the history path
	sessionID := fmt.Sprintf("session_%s", historyPath)
	logger.Debug("Creating memory with session ID: %s", sessionID)

	mem, err := memory.New(sessionID)
	if err != nil {
		logger.Error("Failed to create memory: %v", err)
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}

	logger.Info("Chat manager created successfully")

	return &Manager{
		history: history,
		memory:  mem,
	}, nil
}

// SetStreamCallback sets the callback for streaming content
func (m *Manager) SetStreamCallback(callback StreamCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamCallback = callback
}

// AddMessage adds a message to the chat
func (m *Manager) AddMessage(role MessageRole, content string) error {
	logger.Debug("Adding message - Role: %s, Content length: %d", role, len(content))

	msg := NewMessage(role, content)

	// Add to memory if enabled
	if m.memory != nil {
		switch role {
		case RoleUser:
			if err := m.memory.AddUserMessage(content); err != nil {
				logger.Error("Failed to add user message to memory: %v", err)
				return fmt.Errorf("failed to add user message to memory: %w", err)
			}
			logger.Debug("User message added to memory")
		case RoleAssistant:
			if err := m.memory.AddAssistantMessage(content); err != nil {
				logger.Error("Failed to add assistant message to memory: %v", err)
				return fmt.Errorf("failed to add assistant message to memory: %w", err)
			}
			logger.Debug("Assistant message added to memory")
		}
	}

	err := m.history.Add(msg)
	if err != nil {
		logger.Error("Failed to add message to history: %v", err)
		return err
	}

	logger.Debug("Message added to history successfully")
	return nil
}

// AddMessageWithMetadata adds a message with metadata to the chat
func (m *Manager) AddMessageWithMetadata(msg *Message) error {
	return m.history.Add(msg)
}

// StartStreaming starts streaming a new message
func (m *Manager) StartStreaming(role MessageRole) (*StreamingMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := NewMessage(role, "")
	msg.Metadata.IsStreaming = true

	m.currentStream = &StreamingMessage{
		Message:       msg,
		ContentBuffer: "",
	}

	return m.currentStream, nil
}

// AppendToStream appends content to the current streaming message
func (m *Manager) AppendToStream(content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentStream == nil {
		return fmt.Errorf("no active stream")
	}

	m.currentStream.ContentBuffer += content
	m.currentStream.Message.Content = m.currentStream.ContentBuffer

	// Call the stream callback if set
	if m.streamCallback != nil {
		if err := m.streamCallback(content); err != nil {
			return fmt.Errorf("stream callback error: %w", err)
		}
	}

	return nil
}

// EndStreaming ends the current streaming message
func (m *Manager) EndStreaming() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentStream == nil {
		return fmt.Errorf("no active stream")
	}

	// Add to memory if it's an assistant message
	if m.memory != nil && m.currentStream.Message.Role == RoleAssistant {
		if err := m.memory.AddAssistantMessage(m.currentStream.Message.Content); err != nil {
			return fmt.Errorf("failed to add assistant message to memory: %w", err)
		}
	}

	m.currentStream.Message.Metadata.IsStreaming = false
	if err := m.history.Add(m.currentStream.Message); err != nil {
		return fmt.Errorf("failed to add message to history: %w", err)
	}

	m.currentStream = nil
	return nil
}

// EndStreamingWithTokens ends the current streaming message with token count
func (m *Manager) EndStreamingWithTokens(tokens int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentStream == nil {
		return fmt.Errorf("no active stream")
	}

	m.currentStream.Message.Metadata.IsStreaming = false
	m.currentStream.Message.Metadata.TokensUsed = tokens
	if err := m.history.Add(m.currentStream.Message); err != nil {
		return fmt.Errorf("failed to add message to history: %w", err)
	}

	m.currentStream = nil
	return nil
}

// GetHistory returns the chat history
func (m *Manager) GetHistory() []*Message {
	return m.history.GetMessages()
}

// ClearHistory clears the chat history
func (m *Manager) ClearHistory() error {
	return m.history.Clear()
}

// GetMemoryMessages returns messages from memory for LLM context
func (m *Manager) GetMemoryMessages() ([]llm.Message, error) {
	if m.memory == nil {
		return []llm.Message{}, nil
	}
	return m.memory.ConvertToLLMMessages()
}

// GetMemory returns the memory instance
func (m *Manager) GetMemory() *memory.Memory {
	return m.memory
}

// ProcessWithContext processes a message with a given context
func (m *Manager) ProcessWithContext(ctx context.Context, content string, processor func(context.Context, string) (string, error)) error {
	// Add user message
	if err := m.AddMessage(RoleUser, content); err != nil {
		return fmt.Errorf("failed to add user message: %w", err)
	}

	// Process with the provided processor
	response, err := processor(ctx, content)
	if err != nil {
		// Add error message
		errMsg := NewMessage(RoleAssistant, fmt.Sprintf("Error: %v", err))
		errMsg.Metadata.Error = err.Error()
		m.history.Add(errMsg)
		return fmt.Errorf("processor error: %w", err)
	}

	// Add assistant response
	if err := m.AddMessage(RoleAssistant, response); err != nil {
		return fmt.Errorf("failed to add assistant message: %w", err)
	}

	return nil
}
