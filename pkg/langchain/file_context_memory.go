package langchain

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

// FileContextMemory extends LangChain memory to include file context tracking
type FileContextMemory struct {
	// Embed the base memory for conversation history
	baseMemory schema.Memory
	
	// File context management
	fileContexts     map[string]*chat.FileContext
	contextMutex     sync.RWMutex
	branchingConv    *chat.BranchingConversation
	
	// Configuration
	maxContextSize   int
	includeThinking  bool
	embedFileContent bool
	
	log *logger.Logger
}

// NewFileContextMemory creates a new file-aware memory instance
func NewFileContextMemory(baseMemory schema.Memory, branchingConv *chat.BranchingConversation) *FileContextMemory {
	return &FileContextMemory{
		baseMemory:       baseMemory,
		fileContexts:     make(map[string]*chat.FileContext),
		branchingConv:    branchingConv,
		maxContextSize:   4000, // Token limit for context
		includeThinking:  false,
		embedFileContent: true,
		log:             logger.WithComponent("file_context_memory"),
	}
}

// MemoryVariables returns the memory variables this memory class returns
func (m *FileContextMemory) MemoryVariables(ctx context.Context) []string {
	baseVars := m.baseMemory.MemoryVariables(ctx)
	// Add our custom variables
	return append(baseVars, "file_context", "current_files", "branch_info")
}

// GetMemoryKey returns the memory key for this memory implementation
func (m *FileContextMemory) GetMemoryKey(ctx context.Context) string {
	return "file_context_memory"
}

// LoadMemoryVariables returns the memory variables
func (m *FileContextMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	// Get base memory variables
	vars, err := m.baseMemory.LoadMemoryVariables(ctx, inputs)
	if err != nil {
		return nil, err
	}

	// Add file context information
	m.contextMutex.RLock()
	defer m.contextMutex.RUnlock()

	// Build file context summary
	fileContextStr := m.buildFileContextSummary()
	vars["file_context"] = fileContextStr

	// Add current files list
	currentFiles := make([]string, 0, len(m.fileContexts))
	for path := range m.fileContexts {
		currentFiles = append(currentFiles, path)
	}
	vars["current_files"] = strings.Join(currentFiles, ", ")

	// Add branch information
	if m.branchingConv != nil {
		vars["branch_info"] = fmt.Sprintf("Branch: %s", m.branchingConv.CurrentBranch)
	}

	return vars, nil
}

// SaveContext saves the context to memory
func (m *FileContextMemory) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
	// Save to base memory
	if err := m.baseMemory.SaveContext(ctx, inputs, outputs); err != nil {
		return err
	}

	// Extract file contexts if present
	if fileContexts, ok := inputs["file_contexts"].([]chat.FileContext); ok {
		m.updateFileContexts(fileContexts)
	}

	// Handle branching conversation updates
	if m.branchingConv != nil && inputs["message"] != nil {
		if msg, ok := inputs["message"].(chat.Message); ok {
			if contexts, ok := inputs["file_contexts"].([]chat.FileContext); ok {
				_, err := m.branchingConv.AddMessageWithContext(msg, contexts)
				if err != nil {
					m.log.Error("Failed to add message with context", "error", err)
				}
			}
		}
	}

	return nil
}

// Clear clears the memory
func (m *FileContextMemory) Clear(ctx context.Context) error {
	m.contextMutex.Lock()
	defer m.contextMutex.Unlock()

	// Clear file contexts
	m.fileContexts = make(map[string]*chat.FileContext)

	// Clear base memory
	return m.baseMemory.Clear(ctx)
}


// Additional methods for file context management

// AddFileContext adds or updates a file context
func (m *FileContextMemory) AddFileContext(fc chat.FileContext) {
	m.contextMutex.Lock()
	defer m.contextMutex.Unlock()
	
	m.fileContexts[fc.Path] = &fc
	m.log.Debug("Added file context", "path", fc.Path, "content_length", len(fc.Content))
}

// GetFileContext retrieves a file context
func (m *FileContextMemory) GetFileContext(path string) (*chat.FileContext, bool) {
	m.contextMutex.RLock()
	defer m.contextMutex.RUnlock()
	
	fc, exists := m.fileContexts[path]
	return fc, exists
}

// RemoveFileContext removes a file from context
func (m *FileContextMemory) RemoveFileContext(path string) {
	m.contextMutex.Lock()
	defer m.contextMutex.Unlock()
	
	delete(m.fileContexts, path)
}

// GetRelevantFileContexts returns file contexts relevant to the current query
func (m *FileContextMemory) GetRelevantFileContexts(query string) []chat.FileContext {
	m.contextMutex.RLock()
	defer m.contextMutex.RUnlock()

	relevant := make([]chat.FileContext, 0)
	
	// Simple relevance: return all files mentioned in query
	// In production, this would use embeddings and similarity search
	for path, fc := range m.fileContexts {
		if strings.Contains(query, path) || strings.Contains(query, getFileName(path)) {
			relevant = append(relevant, *fc)
		}
	}

	// If no specific files mentioned, return most recently edited
	if len(relevant) == 0 && len(m.fileContexts) > 0 {
		var mostRecent *chat.FileContext
		for _, fc := range m.fileContexts {
			if mostRecent == nil || fc.LastEdit.After(mostRecent.LastEdit) {
				mostRecent = fc
			}
		}
		if mostRecent != nil {
			relevant = append(relevant, *mostRecent)
		}
	}

	return relevant
}

// buildFileContextSummary creates a summary of current file contexts
func (m *FileContextMemory) buildFileContextSummary() string {
	if len(m.fileContexts) == 0 {
		return "No files in context"
	}

	var parts []string
	for path, fc := range m.fileContexts {
		summary := fmt.Sprintf("File: %s (edited %s, %d edits)", 
			path, 
			fc.LastEdit.Format("15:04:05"),
			len(fc.EditHistory))
		
		// Add recent edits summary
		if len(fc.EditHistory) > 0 {
			lastEdit := fc.EditHistory[len(fc.EditHistory)-1]
			summary += fmt.Sprintf(", last: %s", lastEdit.EditType)
		}
		
		parts = append(parts, summary)
	}

	return strings.Join(parts, "\n")
}

// updateFileContexts updates multiple file contexts
func (m *FileContextMemory) updateFileContexts(contexts []chat.FileContext) {
	m.contextMutex.Lock()
	defer m.contextMutex.Unlock()

	for _, fc := range contexts {
		m.fileContexts[fc.Path] = &fc
	}
}

// CreateLangChainMessages converts our enhanced messages to LangChain format
func (m *FileContextMemory) CreateLangChainMessages(messages []chat.Message) []llms.MessageContent {
	var langchainMessages []llms.MessageContent

	for _, msg := range messages {
		// Skip optimistic messages
		if msg.IsOptimistic() {
			continue
		}

		// Build content with file context if relevant
		content := msg.Content
		
		// Add thinking if visible
		if msg.HasThinking() && msg.IsThinkingVisible() && m.includeThinking {
			content = fmt.Sprintf("<think>\n%s\n</think>\n\n%s", msg.Thinking.Content, content)
		}

		// Convert based on role
		switch msg.Role {
		case chat.RoleUser:
			langchainMessages = append(langchainMessages, llms.TextParts(llms.ChatMessageTypeHuman, content))
		case chat.RoleAssistant:
			langchainMessages = append(langchainMessages, llms.TextParts(llms.ChatMessageTypeAI, content))
		case chat.RoleSystem:
			langchainMessages = append(langchainMessages, llms.TextParts(llms.ChatMessageTypeSystem, content))
		case chat.RoleTool:
			langchainMessages = append(langchainMessages, llms.TextParts(llms.ChatMessageTypeTool, content))
		}
	}

	return langchainMessages
}

// Helper functions

func getFileName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

// WithMaxContextSize sets the maximum context size
func (m *FileContextMemory) WithMaxContextSize(size int) *FileContextMemory {
	m.maxContextSize = size
	return m
}

// WithThinking enables/disables thinking blocks in context
func (m *FileContextMemory) WithThinking(include bool) *FileContextMemory {
	m.includeThinking = include
	return m
}

// WithEmbedding enables/disables file content embedding
func (m *FileContextMemory) WithEmbedding(embed bool) *FileContextMemory {
	m.embedFileContent = embed
	return m
}