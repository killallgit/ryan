package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/vectorstore"
	"github.com/tmc/langchaingo/schema"
)

// LangChainVectorMemory extends LangChainMemory with vector store capabilities for semantic retrieval
type LangChainVectorMemory struct {
	*LangChainMemory
	store          vectorstore.VectorStore
	collection     vectorstore.Collection
	collectionName string
	maxRetrieved   int
	scoreThreshold float32
}

// VectorMemoryConfig configures the vector memory
type VectorMemoryConfig struct {
	CollectionName string
	MaxRetrieved   int     // Maximum number of messages to retrieve
	ScoreThreshold float32 // Minimum similarity score
}

// DefaultVectorMemoryConfig returns default configuration
func DefaultVectorMemoryConfig() VectorMemoryConfig {
	return VectorMemoryConfig{
		CollectionName: "conversations",
		MaxRetrieved:   10,
		ScoreThreshold: 0.7,
	}
}

// NewLangChainVectorMemory creates a new vector-backed memory
func NewLangChainVectorMemory(store vectorstore.VectorStore, config VectorMemoryConfig) (*LangChainVectorMemory, error) {
	// Get or create collection
	collection, err := vectorstore.GetOrCreateCollection(store, config.CollectionName, map[string]interface{}{
		"type": "conversation_memory",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get or create collection: %w", err)
	}

	return &LangChainVectorMemory{
		LangChainMemory: NewLangChainMemory(),
		store:           store,
		collection:      collection,
		collectionName:  config.CollectionName,
		maxRetrieved:    config.MaxRetrieved,
		scoreThreshold:  config.ScoreThreshold,
	}, nil
}

// NewLangChainVectorMemoryWithConversation creates vector memory with existing conversation
func NewLangChainVectorMemoryWithConversation(store vectorstore.VectorStore, config VectorMemoryConfig, conv Conversation) (*LangChainVectorMemory, error) {
	lm, err := NewLangChainMemoryWithConversation(conv)
	if err != nil {
		return nil, err
	}

	collection, err := vectorstore.GetOrCreateCollection(store, config.CollectionName, map[string]interface{}{
		"type": "conversation_memory",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get or create collection: %w", err)
	}

	vm := &LangChainVectorMemory{
		LangChainMemory: lm,
		store:           store,
		collection:      collection,
		collectionName:  config.CollectionName,
		maxRetrieved:    config.MaxRetrieved,
		scoreThreshold:  config.ScoreThreshold,
	}

	// Index existing conversation
	ctx := context.Background()
	if err := vm.indexConversation(ctx); err != nil {
		return nil, fmt.Errorf("failed to index existing conversation: %w", err)
	}

	return vm, nil
}

// AddMessage adds a message to memory and indexes it in the vector store
func (vm *LangChainVectorMemory) AddMessage(ctx context.Context, msg Message) error {
	// Add to base memory
	if err := vm.LangChainMemory.AddMessage(ctx, msg); err != nil {
		return err
	}

	// Index the message
	return vm.indexMessage(ctx, msg, len(vm.conversation.Messages)-1)
}

// indexMessage indexes a single message in the vector store
func (vm *LangChainVectorMemory) indexMessage(ctx context.Context, msg Message, position int) error {
	// Create document ID based on timestamp and position
	docID := fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), position)

	// Prepare content for indexing
	content := vm.formatMessageForIndexing(msg)

	// Create metadata
	metadata := map[string]interface{}{
		"role":      string(msg.Role),
		"position":  position,
		"timestamp": time.Now().Unix(),
	}

	if msg.ToolName != "" {
		metadata["tool_name"] = msg.ToolName
	}

	// Create and add document
	doc := vectorstore.Document{
		ID:       docID,
		Content:  content,
		Metadata: metadata,
	}

	return vm.collection.AddDocuments(ctx, []vectorstore.Document{doc})
}

// formatMessageForIndexing formats a message for vector indexing
func (vm *LangChainVectorMemory) formatMessageForIndexing(msg Message) string {
	switch msg.Role {
	case RoleUser:
		return fmt.Sprintf("User: %s", msg.Content)
	case RoleAssistant:
		return fmt.Sprintf("Assistant: %s", msg.Content)
	case RoleSystem:
		return fmt.Sprintf("System: %s", msg.Content)
	case RoleTool:
		return fmt.Sprintf("Tool (%s): %s", msg.ToolName, msg.Content)
	case RoleError:
		return fmt.Sprintf("Error: %s", msg.Content)
	default:
		return msg.Content
	}
}

// indexConversation indexes the entire conversation
func (vm *LangChainVectorMemory) indexConversation(ctx context.Context) error {
	docs := make([]vectorstore.Document, 0, len(vm.conversation.Messages))

	for i, msg := range vm.conversation.Messages {
		docID := fmt.Sprintf("msg_init_%d", i)
		content := vm.formatMessageForIndexing(msg)

		metadata := map[string]interface{}{
			"role":      string(msg.Role),
			"position":  i,
			"timestamp": time.Now().Unix(),
		}

		if msg.ToolName != "" {
			metadata["tool_name"] = msg.ToolName
		}

		docs = append(docs, vectorstore.Document{
			ID:       docID,
			Content:  content,
			Metadata: metadata,
		})
	}

	if len(docs) > 0 {
		return vm.collection.AddDocuments(ctx, docs)
	}

	return nil
}

// GetRelevantMessages retrieves messages semantically similar to the query
func (vm *LangChainVectorMemory) GetRelevantMessages(ctx context.Context, query string) ([]Message, error) {
	// Search for relevant messages - don't filter by score for now to debug
	results, err := vm.collection.Query(ctx, query, vm.maxRetrieved)
	if err != nil {
		return nil, fmt.Errorf("failed to query vector store: %w", err)
	}

	// Extract positions and sort by position
	type posMsg struct {
		position int
		message  Message
		score    float32
	}

	posMessages := make([]posMsg, 0, len(results))
	for _, result := range results {
		// Apply score threshold manually
		if result.Score < vm.scoreThreshold {
			continue
		}

		// Try different type assertions for position
		var position int
		var found bool

		switch v := result.Document.Metadata["position"].(type) {
		case int:
			position = v
			found = true
		case float64:
			position = int(v)
			found = true
		case int64:
			position = int(v)
			found = true
		case string:
			// Try to parse string as int
			if _, err := fmt.Sscanf(v, "%d", &position); err == nil {
				found = true
			}
		}

		if !found {
			continue // Skip if position is not found or can't be parsed
		}

		if position < len(vm.conversation.Messages) {
			posMessages = append(posMessages, posMsg{
				position: position,
				message:  vm.conversation.Messages[position],
				score:    result.Score,
			})
		}
	}

	// Sort by position to maintain chronological order
	for i := 0; i < len(posMessages)-1; i++ {
		for j := i + 1; j < len(posMessages); j++ {
			if posMessages[i].position > posMessages[j].position {
				posMessages[i], posMessages[j] = posMessages[j], posMessages[i]
			}
		}
	}

	// Extract messages
	messages := make([]Message, len(posMessages))
	for i, pm := range posMessages {
		messages[i] = pm.message
	}

	return messages, nil
}

// GetMemoryVariables returns memory variables with relevant context
func (vm *LangChainVectorMemory) GetMemoryVariables(ctx context.Context) (map[string]any, error) {
	// Get base memory variables
	vars, err := vm.LangChainMemory.GetMemoryVariables(ctx)
	if err != nil {
		return nil, err
	}

	// Check if we should add relevant context
	// This is a simple heuristic - you might want to make this configurable
	if len(vm.conversation.Messages) > 0 {
		lastMessage := vm.conversation.Messages[len(vm.conversation.Messages)-1]
		if lastMessage.Role == RoleUser {
			// Get relevant messages for the last user input
			relevant, err := vm.GetRelevantMessages(ctx, lastMessage.Content)
			if err == nil && len(relevant) > 0 {
				// Format relevant messages as context
				contextParts := make([]string, 0, len(relevant))
				for _, msg := range relevant {
					contextParts = append(contextParts, vm.formatMessageForIndexing(msg))
				}

				vars["relevant_context"] = strings.Join(contextParts, "\n")
			}
		}
	}

	return vars, nil
}

// Clear clears both conversation and vector store collection
func (vm *LangChainVectorMemory) Clear(ctx context.Context) error {
	// Clear base memory
	if err := vm.LangChainMemory.Clear(ctx); err != nil {
		return err
	}

	// Clear vector store collection
	return vm.collection.Clear(ctx)
}

// SaveContext saves context and indexes new messages
func (vm *LangChainVectorMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	// Get current message count
	prevCount := len(vm.conversation.Messages)

	// Save to base memory
	if err := vm.LangChainMemory.SaveContext(ctx, inputs, outputs); err != nil {
		return err
	}

	// Index any new messages
	for i := prevCount; i < len(vm.conversation.Messages); i++ {
		if err := vm.indexMessage(ctx, vm.conversation.Messages[i], i); err != nil {
			return fmt.Errorf("failed to index message %d: %w", i, err)
		}
	}

	return nil
}

// VectorMemoryAdapter allows using LangChainVectorMemory as schema.Memory
type VectorMemoryAdapter struct {
	*LangChainVectorMemory
}

// Ensure VectorMemoryAdapter implements schema.Memory
var _ schema.Memory = (*VectorMemoryAdapter)(nil)

// GetMemoryKey returns the memory key
func (vma *VectorMemoryAdapter) GetMemoryKey(ctx context.Context) string {
	return "history"
}

// MemoryVariables returns the memory variables
func (vma *VectorMemoryAdapter) MemoryVariables(ctx context.Context) []string {
	return []string{"history", "relevant_context"}
}

// LoadMemoryVariables loads the memory variables with relevant context
func (vma *VectorMemoryAdapter) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	return vma.LangChainVectorMemory.GetMemoryVariables(ctx)
}

// SaveContext saves the context
func (vma *VectorMemoryAdapter) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	return vma.LangChainVectorMemory.SaveContext(ctx, inputs, outputs)
}

// Clear clears the memory
func (vma *VectorMemoryAdapter) Clear(ctx context.Context) error {
	return vma.LangChainVectorMemory.Clear(ctx)
}
