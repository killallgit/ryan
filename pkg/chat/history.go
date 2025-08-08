package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// History manages chat message history
type History struct {
	Messages []*Message `json:"messages"`
	mu       sync.RWMutex
	filePath string
}

// NewHistory creates a new chat history manager
func NewHistory(filePath string) (*History, error) {
	h := &History{
		Messages: make([]*Message, 0),
		filePath: filePath,
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create history directory: %w", err)
	}

	// Load existing history if file exists
	if _, err := os.Stat(filePath); err == nil {
		if err := h.Load(); err != nil {
			return nil, fmt.Errorf("failed to load history: %w", err)
		}
	}

	return h, nil
}

// Add adds a message to the history
func (h *History) Add(msg *Message) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Messages = append(h.Messages, msg)
	return h.Save()
}

// GetMessages returns all messages in the history
func (h *History) GetMessages() []*Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	msgs := make([]*Message, len(h.Messages))
	copy(msgs, h.Messages)
	return msgs
}

// Clear clears the history
func (h *History) Clear() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Messages = make([]*Message, 0)
	return h.Save()
}

// Save saves the history to disk
func (h *History) Save() error {
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(h.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// Load loads the history from disk
func (h *History) Load() error {
	data, err := os.ReadFile(h.filePath)
	if err != nil {
		return fmt.Errorf("failed to read history file: %w", err)
	}

	if err := json.Unmarshal(data, h); err != nil {
		return fmt.Errorf("failed to unmarshal history: %w", err)
	}

	return nil
}

// GetLastN returns the last N messages from history
func (h *History) GetLastN(n int) []*Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if n <= 0 || len(h.Messages) == 0 {
		return []*Message{}
	}

	if n > len(h.Messages) {
		n = len(h.Messages)
	}

	result := make([]*Message, n)
	copy(result, h.Messages[len(h.Messages)-n:])
	return result
}
