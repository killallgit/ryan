package langchain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/tmc/langchaingo/llms"
)

// ContextManager handles persistence of LangChain conversation context
type ContextManager struct {
	contextDir string
	sessionID  string
	log        *logger.Logger
}

// ContextData represents the serializable context state
type ContextData struct {
	SessionID string                 `json:"session_id"`
	Messages  []llms.ChatMessage     `json:"messages"`
	Variables map[string]interface{} `json:"variables"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// NewContextManager creates a new context manager
func NewContextManager(sessionID string) (*ContextManager, error) {
	cfg := config.Get()
	if cfg == nil {
		return nil, fmt.Errorf("config not initialized")
	}

	contextDir := cfg.Directories.Contexts
	if contextDir == "" {
		contextDir = "./.ryan/contexts"
	}

	// Ensure context directory exists
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create context directory: %w", err)
	}

	return &ContextManager{
		contextDir: contextDir,
		sessionID:  sessionID,
		log:        logger.WithComponent("context_manager"),
	}, nil
}

// SaveContext persists the current conversation context to disk
func (cm *ContextManager) SaveContext(ctx context.Context, messages []llms.ChatMessage) error {
	if messages == nil {
		return fmt.Errorf("messages is nil")
	}

	// Create basic variables map (can be enhanced later)
	variables := make(map[string]interface{})
	variables["total_messages"] = len(messages)

	// Count message types for metadata
	humanCount, aiCount, systemCount := 0, 0, 0
	for _, msg := range messages {
		switch msg.GetType() {
		case llms.ChatMessageTypeHuman:
			humanCount++
		case llms.ChatMessageTypeAI:
			aiCount++
		case llms.ChatMessageTypeSystem:
			systemCount++
		}
	}

	variables["human_messages"] = humanCount
	variables["ai_messages"] = aiCount
	variables["system_messages"] = systemCount

	// Create context data
	contextData := ContextData{
		SessionID: cm.sessionID,
		Messages:  messages,
		Variables: variables,
		Metadata: map[string]interface{}{
			"model":      "ollama", // Could be enhanced to get actual model
			"tool_count": 0,        // Could be enhanced to get actual tool count
		},
		UpdatedAt: time.Now(),
	}

	// Set created time if this is a new context
	contextFile := cm.getContextFilePath()
	if _, err := os.Stat(contextFile); os.IsNotExist(err) {
		contextData.CreatedAt = time.Now()
	} else {
		// Load existing context to preserve created time
		if existingData, loadErr := cm.loadContextData(); loadErr == nil {
			contextData.CreatedAt = existingData.CreatedAt
		} else {
			contextData.CreatedAt = time.Now()
		}
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(contextData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(contextFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}

	cm.log.Debug("Context saved successfully",
		"session_id", cm.sessionID,
		"message_count", len(messages),
		"file", contextFile)

	return nil
}

// LoadContext restores conversation context from disk
func (cm *ContextManager) LoadContext(ctx context.Context) (*ContextData, error) {
	contextData, err := cm.loadContextData()
	if err != nil {
		return nil, err
	}

	cm.log.Debug("Context loaded successfully",
		"session_id", contextData.SessionID,
		"message_count", len(contextData.Messages),
		"created_at", contextData.CreatedAt)

	return contextData, nil
}

// loadContextData loads raw context data from file
func (cm *ContextManager) loadContextData() (*ContextData, error) {
	contextFile := cm.getContextFilePath()

	// Check if context file exists
	if _, err := os.Stat(contextFile); os.IsNotExist(err) {
		cm.log.Debug("No existing context found", "file", contextFile)
		return nil, fmt.Errorf("no context file found for session %s", cm.sessionID)
	}

	// Read the context file
	data, err := os.ReadFile(contextFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read context file: %w", err)
	}

	// Parse JSON
	var contextData ContextData
	if err := json.Unmarshal(data, &contextData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context data: %w", err)
	}

	return &contextData, nil
}

// ClearContext removes the persisted context for this session
func (cm *ContextManager) ClearContext() error {
	contextFile := cm.getContextFilePath()

	if err := os.Remove(contextFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove context file: %w", err)
	}

	cm.log.Debug("Context cleared", "session_id", cm.sessionID, "file", contextFile)
	return nil
}

// ListContexts returns all available context sessions
func (cm *ContextManager) ListContexts() ([]string, error) {
	files, err := os.ReadDir(cm.contextDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read context directory: %w", err)
	}

	var sessions []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			// Extract session ID from filename (remove .json extension)
			sessionID := file.Name()[:len(file.Name())-5]
			sessions = append(sessions, sessionID)
		}
	}

	return sessions, nil
}

// getContextFilePath returns the full path to the context file for this session
func (cm *ContextManager) getContextFilePath() string {
	return filepath.Join(cm.contextDir, fmt.Sprintf("%s.json", cm.sessionID))
}

// GetContextStats returns statistics about the persisted context
func (cm *ContextManager) GetContextStats() (*ContextStats, error) {
	contextData, err := cm.loadContextData()
	if err != nil {
		return nil, err
	}

	stats := &ContextStats{
		SessionID:    contextData.SessionID,
		MessageCount: len(contextData.Messages),
		CreatedAt:    contextData.CreatedAt,
		UpdatedAt:    contextData.UpdatedAt,
		SizeBytes:    0, // Will be calculated
	}

	// Calculate file size
	contextFile := cm.getContextFilePath()
	if fileInfo, err := os.Stat(contextFile); err == nil {
		stats.SizeBytes = fileInfo.Size()
	}

	// Calculate message types
	for _, msg := range contextData.Messages {
		switch msg.GetType() {
		case llms.ChatMessageTypeHuman:
			stats.HumanMessages++
		case llms.ChatMessageTypeAI:
			stats.AIMessages++
		case llms.ChatMessageTypeSystem:
			stats.SystemMessages++
		case llms.ChatMessageTypeGeneric:
			stats.OtherMessages++
		}
	}

	return stats, nil
}

// ContextStats provides statistics about a context session
type ContextStats struct {
	SessionID      string    `json:"session_id"`
	MessageCount   int       `json:"message_count"`
	HumanMessages  int       `json:"human_messages"`
	AIMessages     int       `json:"ai_messages"`
	SystemMessages int       `json:"system_messages"`
	OtherMessages  int       `json:"other_messages"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	SizeBytes      int64     `json:"size_bytes"`
}
