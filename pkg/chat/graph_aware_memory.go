package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

// GraphAwareLangChainMemory extends LangChain memory with context tree awareness
type GraphAwareLangChainMemory struct {
	*LangChainVectorMemory
	contextTree    *ContextTree
	vectorManager  *VectorContextManager
	currentContext string
	maxRetrieved   int
	includeContext bool // Whether to include full conversation path
}

// GraphAwareMemoryConfig configures the graph-aware memory
type GraphAwareMemoryConfig struct {
	VectorConfig   VectorMemoryConfig
	MaxRetrieved   int
	IncludeContext bool // Include full conversation path from root
	ContextDepth   int  // How many levels of context to include
}

// DefaultGraphAwareMemoryConfig returns default configuration
func DefaultGraphAwareMemoryConfig() GraphAwareMemoryConfig {
	return GraphAwareMemoryConfig{
		VectorConfig:   DefaultVectorMemoryConfig(),
		MaxRetrieved:   20,
		IncludeContext: true,
		ContextDepth:   3,
	}
}

// NewGraphAwareLangChainMemory creates a new graph-aware memory
func NewGraphAwareLangChainMemory(
	contextTree *ContextTree,
	vectorManager *VectorContextManager,
	config GraphAwareMemoryConfig,
) (*GraphAwareLangChainMemory, error) {
	// Create base vector memory (we'll override its methods)
	baseMemory := NewLangChainMemory()
	vectorMemory := &LangChainVectorMemory{
		LangChainMemory: baseMemory,
		manager:         vectorManager.manager,
		collectionName:  config.VectorConfig.CollectionName,
		maxRetrieved:    config.VectorConfig.MaxRetrieved,
		scoreThreshold:  config.VectorConfig.ScoreThreshold,
	}

	return &GraphAwareLangChainMemory{
		LangChainVectorMemory: vectorMemory,
		contextTree:           contextTree,
		vectorManager:         vectorManager,
		currentContext:        contextTree.ActiveContext,
		maxRetrieved:          config.MaxRetrieved,
		includeContext:        config.IncludeContext,
	}, nil
}

// AddMessage adds a message to the current active context
func (galm *GraphAwareLangChainMemory) AddMessage(ctx context.Context, msg Message) error {
	// Set message context to current active context
	msg.ContextID = galm.currentContext

	// Add to context tree
	if err := galm.contextTree.AddMessage(msg, galm.currentContext); err != nil {
		return fmt.Errorf("failed to add message to context tree: %w", err)
	}

	// Index in vector store
	if err := galm.vectorManager.IndexMessage(ctx, &msg); err != nil {
		return fmt.Errorf("failed to index message in vector store: %w", err)
	}

	// Add to base LangChain memory
	return galm.addMessageToBuffer(ctx, msg)
}

// SwitchContext changes the active conversation context
func (galm *GraphAwareLangChainMemory) SwitchContext(contextID string) error {
	if err := galm.contextTree.SwitchContext(contextID); err != nil {
		return err
	}

	galm.currentContext = contextID
	return nil
}

// BranchFromMessage creates a new conversation branch from any message
func (galm *GraphAwareLangChainMemory) BranchFromMessage(messageID, title string) (*Context, error) {
	return galm.contextTree.BranchFromMessage(messageID, title)
}

// GetRelevantMessages retrieves messages semantically similar to the query with context awareness
func (galm *GraphAwareLangChainMemory) GetRelevantMessages(ctx context.Context, query string) ([]Message, error) {
	// Use hybrid search that combines context-specific and cross-context results
	relevantMessages, err := galm.vectorManager.HybridSearch(ctx, galm.currentContext, query)
	if err != nil {
		return nil, fmt.Errorf("failed to perform hybrid search: %w", err)
	}

	// If including context, add full conversation path
	if galm.includeContext {
		contextPath, err := galm.GetConversationPath()
		if err != nil {
			return relevantMessages, nil // Return relevant messages even if context path fails
		}

		// Merge context path with relevant messages, avoiding duplicates
		mergedMessages := galm.mergeAndDeduplicateMessages(contextPath, relevantMessages)

		// Limit total results
		if len(mergedMessages) > galm.maxRetrieved {
			mergedMessages = mergedMessages[:galm.maxRetrieved]
		}

		return mergedMessages, nil
	}

	return relevantMessages, nil
}

// GetConversationPath returns the full conversation path from root to current context
func (galm *GraphAwareLangChainMemory) GetConversationPath() ([]Message, error) {
	return galm.contextTree.GetConversationPath(galm.currentContext)
}

// GetCurrentContextMessages returns all messages in the current context
func (galm *GraphAwareLangChainMemory) GetCurrentContextMessages() []Message {
	return galm.contextTree.GetContextMessages(galm.currentContext)
}

// GetContextBranches returns all child contexts (branches) of the current context
func (galm *GraphAwareLangChainMemory) GetContextBranches() []*Context {
	return galm.contextTree.GetContextBranches(galm.currentContext)
}

// GetMessageBranches returns all child messages of a specific message
func (galm *GraphAwareLangChainMemory) GetMessageBranches(messageID string) []*Message {
	return galm.contextTree.GetMessageBranches(messageID)
}

// mergeAndDeduplicateMessages combines context path with relevant messages, removing duplicates
func (galm *GraphAwareLangChainMemory) mergeAndDeduplicateMessages(contextPath, relevantMessages []Message) []Message {
	seen := make(map[string]bool)
	var merged []Message

	// Add context path first (higher priority)
	for _, msg := range contextPath {
		if !seen[msg.ID] {
			seen[msg.ID] = true
			merged = append(merged, msg)
		}
	}

	// Add relevant messages that aren't already in context path
	for _, msg := range relevantMessages {
		if !seen[msg.ID] {
			seen[msg.ID] = true
			merged = append(merged, msg)
		}
	}

	return merged
}

// GetMemoryVariables returns memory variables with context-aware relevant context
func (galm *GraphAwareLangChainMemory) GetMemoryVariables(ctx context.Context) (map[string]any, error) {
	// Get base memory variables from LangChain
	vars := make(map[string]any)

	// Get current conversation path
	conversationPath, err := galm.GetConversationPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation path: %w", err)
	}

	// Convert to LangChain format and add to history
	if len(conversationPath) > 0 {
		langchainMessages := ConvertToLangChainMessages(conversationPath)
		vars["history"] = galm.formatMessagesForPrompt(langchainMessages)
	}

	// Add context information
	currentContext := galm.contextTree.GetActiveContext()
	if currentContext != nil {
		vars["context_info"] = map[string]any{
			"context_id":    currentContext.ID,
			"context_title": currentContext.Title,
			"branch_point":  currentContext.BranchPoint,
			"created":       currentContext.Created,
			"message_count": len(currentContext.MessageIDs),
		}

		// Add branch information if there are branches
		branches := galm.GetContextBranches()
		if len(branches) > 0 {
			var branchInfo []map[string]any
			for _, branch := range branches {
				branchInfo = append(branchInfo, map[string]any{
					"id":      branch.ID,
					"title":   branch.Title,
					"created": branch.Created,
				})
			}
			vars["available_branches"] = branchInfo
		}
	}

	return vars, nil
}

// formatMessagesForPrompt formats messages for use in prompts
func (galm *GraphAwareLangChainMemory) formatMessagesForPrompt(messages []llms.ChatMessage) string {
	var formatted []string

	for _, msg := range messages {
		switch m := msg.(type) {
		case llms.HumanChatMessage:
			formatted = append(formatted, fmt.Sprintf("Human: %s", m.Content))
		case llms.AIChatMessage:
			formatted = append(formatted, fmt.Sprintf("AI: %s", m.Content))
		case llms.SystemChatMessage:
			formatted = append(formatted, fmt.Sprintf("System: %s", m.Content))
		case llms.GenericChatMessage:
			formatted = append(formatted, fmt.Sprintf("%s: %s", strings.Title(m.Role), m.Content))
		}
	}

	return strings.Join(formatted, "\n")
}

// SaveContext saves the context with graph awareness
func (galm *GraphAwareLangChainMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	// Extract and add input message
	if input, ok := inputs["input"].(string); ok && input != "" {
		userMsg := NewUserMessage(input)
		if err := galm.AddMessage(ctx, userMsg); err != nil {
			return fmt.Errorf("failed to save user message: %w", err)
		}
	}

	// Extract and add output message
	if output, ok := outputs["output"].(string); ok && output != "" {
		assistantMsg := NewAssistantMessage(output)
		if err := galm.AddMessage(ctx, assistantMsg); err != nil {
			return fmt.Errorf("failed to save assistant message: %w", err)
		}
	}

	return nil
}

// Clear clears the current context (not the entire tree)
func (galm *GraphAwareLangChainMemory) Clear(ctx context.Context) error {
	// Clear current context in vector store
	if err := galm.vectorManager.ClearContext(ctx, galm.currentContext); err != nil {
		return fmt.Errorf("failed to clear vector context: %w", err)
	}

	// Clear current context messages (but preserve the context structure)
	currentContext := galm.contextTree.GetActiveContext()
	if currentContext != nil {
		// Remove messages from tree
		for _, msgID := range currentContext.MessageIDs {
			delete(galm.contextTree.Messages, msgID)
		}
		currentContext.MessageIDs = []string{}
	}

	// Clear base LangChain memory
	return galm.LangChainMemory.Clear(ctx)
}

// ClearAll clears all contexts and the entire tree
func (galm *GraphAwareLangChainMemory) ClearAll(ctx context.Context) error {
	// Clear all vector store collections
	if err := galm.vectorManager.ClearAllContexts(ctx); err != nil {
		return fmt.Errorf("failed to clear all vector contexts: %w", err)
	}

	// Reset context tree to fresh state
	newTree := NewContextTree()
	galm.contextTree = newTree
	galm.currentContext = newTree.ActiveContext

	// Clear base LangChain memory
	return galm.LangChainMemory.Clear(ctx)
}

// GetCurrentContext returns the current active context
func (galm *GraphAwareLangChainMemory) GetCurrentContext() *Context {
	return galm.contextTree.GetActiveContext()
}

// GetContextTree returns the entire context tree (for advanced operations)
func (galm *GraphAwareLangChainMemory) GetContextTree() *ContextTree {
	return galm.contextTree
}

// GraphAwareMemoryAdapter allows using GraphAwareLangChainMemory as schema.Memory
type GraphAwareMemoryAdapter struct {
	*GraphAwareLangChainMemory
}

// Ensure GraphAwareMemoryAdapter implements schema.Memory
var _ schema.Memory = (*GraphAwareMemoryAdapter)(nil)

// GetMemoryKey returns the memory key
func (gama *GraphAwareMemoryAdapter) GetMemoryKey(ctx context.Context) string {
	return "history"
}

// MemoryVariables returns the memory variables
func (gama *GraphAwareMemoryAdapter) MemoryVariables(ctx context.Context) []string {
	return []string{"history", "context_info", "available_branches"}
}

// LoadMemoryVariables loads the memory variables with context awareness
func (gama *GraphAwareMemoryAdapter) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	return gama.GraphAwareLangChainMemory.GetMemoryVariables(ctx)
}

// SaveContext saves the context
func (gama *GraphAwareMemoryAdapter) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	return gama.GraphAwareLangChainMemory.SaveContext(ctx, inputs, outputs)
}

// Clear clears the memory
func (gama *GraphAwareMemoryAdapter) Clear(ctx context.Context) error {
	return gama.GraphAwareLangChainMemory.Clear(ctx)
}

// NewGraphAwareLangChainMemoryFromGlobalConfig creates graph-aware memory using global configuration
func NewGraphAwareLangChainMemoryFromGlobalConfig() (*GraphAwareLangChainMemory, error) {
	// This would integrate with your existing global vector store configuration
	// For now, return an error indicating it needs to be implemented
	return nil, fmt.Errorf("global config integration not yet implemented - use NewGraphAwareLangChainMemory directly")
}

// Message creation functions for context-aware messages

// NewUserMessageInContext creates a user message in a specific context
func NewUserMessageInContext(content, contextID string) Message {
	msg := NewUserMessage(content)
	msg.ContextID = contextID
	return msg
}

// NewAssistantMessageInContext creates an assistant message in a specific context
func NewAssistantMessageInContext(content, contextID string) Message {
	msg := NewAssistantMessage(content)
	msg.ContextID = contextID
	return msg
}

// NewBranchMessage creates a message that starts a new conversation branch
func NewBranchMessage(content, sourceMessageID, newContextID string, role string) Message {
	msg := Message{
		ID:        generateMessageID(),
		ParentID:  &sourceMessageID,
		ContextID: newContextID,
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
			Depth:  0, // Will be calculated when added to context
		},
	}
	return msg
}
