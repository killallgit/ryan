package chat

import (
	"context"
	"fmt"

	"github.com/killallgit/ryan/pkg/vectorstore"
)

// VectorContextManager manages vector store collections for each conversation context
type VectorContextManager struct {
	manager            *vectorstore.Manager
	tree               *ContextTree
	globalCollection   string
	contextCollections map[string]string // contextID -> collectionName
	config             VectorContextConfig
}

// VectorContextConfig configures the vector context manager
type VectorContextConfig struct {
	GlobalCollection    string
	ContextPrefix       string // e.g., "ctx_"
	EnableCrossSearch   bool
	MaxContextsPerQuery int
	ScoreThreshold      float32
	MaxRetrieved        int
}

// DefaultVectorContextConfig returns default configuration
func DefaultVectorContextConfig() VectorContextConfig {
	return VectorContextConfig{
		GlobalCollection:    "global_conversations",
		ContextPrefix:       "ctx_",
		EnableCrossSearch:   true,
		MaxContextsPerQuery: 5,
		ScoreThreshold:      0.5,
		MaxRetrieved:        20,
	}
}

// NewVectorContextManager creates a new vector context manager
func NewVectorContextManager(manager *vectorstore.Manager, tree *ContextTree, config VectorContextConfig) (*VectorContextManager, error) {
	// Ensure global collection exists
	_, err := manager.GetCollection(config.GlobalCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create global collection: %w", err)
	}

	return &VectorContextManager{
		manager:            manager,
		tree:               tree,
		globalCollection:   config.GlobalCollection,
		contextCollections: make(map[string]string),
		config:             config,
	}, nil
}

// getCollectionName returns the collection name for a context
func (vcm *VectorContextManager) getCollectionName(contextID string) string {
	if contextID == "" {
		return vcm.config.GlobalCollection
	}
	return vcm.config.ContextPrefix + contextID
}

// ensureContextCollection ensures a collection exists for the given context
func (vcm *VectorContextManager) ensureContextCollection(ctx context.Context, contextID string) error {
	collectionName := vcm.getCollectionName(contextID)

	// Check if we already have this collection cached
	if _, exists := vcm.contextCollections[contextID]; exists {
		return nil
	}

	// Try to get or create the collection
	_, err := vcm.manager.GetCollection(collectionName)
	if err != nil {
		return fmt.Errorf("failed to get or create collection %s: %w", collectionName, err)
	}

	// Cache the collection name
	vcm.contextCollections[contextID] = collectionName
	return nil
}

// IndexMessage indexes a message in both global and context-specific collections
func (vcm *VectorContextManager) IndexMessage(ctx context.Context, msg *Message) error {
	// Ensure context collection exists
	if err := vcm.ensureContextCollection(ctx, msg.ContextID); err != nil {
		return err
	}

	// Index in global collection for cross-context search
	if err := vcm.indexInCollection(ctx, vcm.config.GlobalCollection, msg); err != nil {
		return fmt.Errorf("failed to index in global collection: %w", err)
	}

	// Index in context-specific collection
	contextCollection := vcm.getCollectionName(msg.ContextID)
	if err := vcm.indexInCollection(ctx, contextCollection, msg); err != nil {
		return fmt.Errorf("failed to index in context collection: %w", err)
	}

	return nil
}

// indexInCollection indexes a message in a specific collection
func (vcm *VectorContextManager) indexInCollection(ctx context.Context, collectionName string, msg *Message) error {
	// Create document ID
	docID := fmt.Sprintf("msg_%s", msg.ID)

	// Prepare content for indexing
	content := vcm.formatMessageForIndexing(*msg)

	// Create metadata
	metadata := map[string]interface{}{
		"message_id": msg.ID,
		"context_id": msg.ContextID,
		"role":       msg.Role,
		"timestamp":  msg.Timestamp.Unix(),
	}

	if msg.ParentID != nil {
		metadata["parent_id"] = *msg.ParentID
	}

	if msg.ToolName != "" {
		metadata["tool_name"] = msg.ToolName
	}

	if msg.Metadata != nil {
		metadata["depth"] = msg.Metadata.Depth
		metadata["branch_point"] = msg.Metadata.BranchPoint
		metadata["child_count"] = msg.Metadata.ChildCount

		if msg.Metadata.ThreadTitle != nil {
			metadata["thread_title"] = *msg.Metadata.ThreadTitle
		}
	}

	// Create and add document
	doc := vectorstore.Document{
		ID:       docID,
		Content:  content,
		Metadata: metadata,
	}

	return vcm.manager.IndexDocument(ctx, collectionName, doc)
}

// formatMessageForIndexing formats a message for vector indexing
func (vcm *VectorContextManager) formatMessageForIndexing(msg Message) string {
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

// SearchInContext searches for relevant messages within a specific context
func (vcm *VectorContextManager) SearchInContext(ctx context.Context, contextID, query string) ([]Message, error) {
	collectionName := vcm.getCollectionName(contextID)

	// Search in context-specific collection
	results, err := vcm.manager.Search(ctx, collectionName, query, vcm.config.MaxRetrieved)
	if err != nil {
		return nil, fmt.Errorf("failed to search in context %s: %w", contextID, err)
	}

	return vcm.resultsToMessages(results), nil
}

// SearchAcrossContexts searches for relevant messages across all contexts
func (vcm *VectorContextManager) SearchAcrossContexts(ctx context.Context, query string) ([]Message, error) {
	// Search in global collection
	results, err := vcm.manager.Search(ctx, vcm.config.GlobalCollection, query, vcm.config.MaxRetrieved)
	if err != nil {
		return nil, fmt.Errorf("failed to search across contexts: %w", err)
	}

	return vcm.resultsToMessages(results), nil
}

// SearchRelatedContexts searches for relevant messages in contexts related to the current one
func (vcm *VectorContextManager) SearchRelatedContexts(ctx context.Context, currentContextID, query string) ([]Message, error) {
	// Get related contexts (parent and sibling contexts)
	relatedContexts := vcm.getRelatedContexts(currentContextID)

	var allMessages []Message

	// Search in each related context
	for _, contextID := range relatedContexts {
		if contextID == currentContextID {
			continue // Skip current context
		}

		messages, err := vcm.SearchInContext(ctx, contextID, query)
		if err != nil {
			continue // Skip failed searches
		}

		allMessages = append(allMessages, messages...)
	}

	// Sort by relevance/score and limit results
	if len(allMessages) > vcm.config.MaxRetrieved/2 {
		allMessages = allMessages[:vcm.config.MaxRetrieved/2]
	}

	return allMessages, nil
}

// getRelatedContexts returns context IDs related to the given context
func (vcm *VectorContextManager) getRelatedContexts(contextID string) []string {
	var related []string

	currentContext := vcm.tree.GetContext(contextID)
	if currentContext == nil {
		return related
	}

	// Add parent context
	if currentContext.ParentID != nil {
		related = append(related, *currentContext.ParentID)
	}

	// Add sibling contexts (other branches from same parent)
	if currentContext.ParentID != nil {
		siblings := vcm.tree.GetContextBranches(*currentContext.ParentID)
		for _, sibling := range siblings {
			if sibling.ID != contextID {
				related = append(related, sibling.ID)
			}
		}
	}

	// Add child contexts
	children := vcm.tree.GetContextBranches(contextID)
	for _, child := range children {
		related = append(related, child.ID)
	}

	return related
}

// resultsToMessages converts search results to messages
func (vcm *VectorContextManager) resultsToMessages(results []vectorstore.Result) []Message {
	var messages []Message

	for _, result := range results {
		if result.Score < vcm.config.ScoreThreshold {
			continue
		}

		// Extract message ID from metadata
		messageID, ok := result.Document.Metadata["message_id"].(string)
		if !ok {
			continue
		}

		// Get message from tree
		if msg := vcm.tree.GetMessage(messageID); msg != nil {
			messages = append(messages, *msg)
		}
	}

	return messages
}

// GetBranchPointMessages returns messages that provide context for branching
func (vcm *VectorContextManager) GetBranchPointMessages(contextID string) []Message {
	context := vcm.tree.GetContext(contextID)
	if context == nil || context.BranchPoint == nil {
		return []Message{}
	}

	// Get the branch point message and its immediate context
	branchMsg := vcm.tree.GetMessage(*context.BranchPoint)
	if branchMsg == nil {
		return []Message{}
	}

	messages := []Message{*branchMsg}

	// Include parent message for context
	if branchMsg.ParentID != nil {
		if parentMsg := vcm.tree.GetMessage(*branchMsg.ParentID); parentMsg != nil {
			messages = append([]Message{*parentMsg}, messages...)
		}
	}

	return messages
}

// IndexExistingMessages indexes all existing messages in the tree
func (vcm *VectorContextManager) IndexExistingMessages(ctx context.Context) error {
	for _, msg := range vcm.tree.Messages {
		if err := vcm.IndexMessage(ctx, msg); err != nil {
			return fmt.Errorf("failed to index message %s: %w", msg.ID, err)
		}
	}
	return nil
}

// ClearContext clears all messages from a context's collection
func (vcm *VectorContextManager) ClearContext(ctx context.Context, contextID string) error {
	collectionName := vcm.getCollectionName(contextID)
	return vcm.manager.ClearCollection(ctx, collectionName)
}

// ClearAllContexts clears all collections (global and context-specific)
func (vcm *VectorContextManager) ClearAllContexts(ctx context.Context) error {
	// Clear global collection
	if err := vcm.manager.ClearCollection(ctx, vcm.config.GlobalCollection); err != nil {
		return fmt.Errorf("failed to clear global collection: %w", err)
	}

	// Clear all context collections
	for contextID, collectionName := range vcm.contextCollections {
		if err := vcm.manager.ClearCollection(ctx, collectionName); err != nil {
			// Log error but continue with other collections
			fmt.Printf("Warning: failed to clear context collection %s for context %s: %v\n", collectionName, contextID, err)
		}
	}

	return nil
}

// DeleteContextCollection removes the collection for a specific context
func (vcm *VectorContextManager) DeleteContextCollection(ctx context.Context, contextID string) error {
	collectionName := vcm.getCollectionName(contextID)

	// Clear the collection
	if err := vcm.manager.ClearCollection(ctx, collectionName); err != nil {
		return fmt.Errorf("failed to clear context collection: %w", err)
	}

	// Remove from cache
	delete(vcm.contextCollections, contextID)

	return nil
}

// HybridSearch performs a hybrid search combining context-specific and cross-context results
func (vcm *VectorContextManager) HybridSearch(ctx context.Context, contextID, query string) ([]Message, error) {
	var allMessages []Message

	// 1. Search in current context (higher priority)
	contextMessages, err := vcm.SearchInContext(ctx, contextID, query)
	if err == nil {
		allMessages = append(allMessages, contextMessages...)
	}

	// 2. If not enough results, search related contexts
	if len(allMessages) < vcm.config.MaxRetrieved/2 {
		relatedMessages, err := vcm.SearchRelatedContexts(ctx, contextID, query)
		if err == nil {
			allMessages = append(allMessages, relatedMessages...)
		}
	}

	// 3. Always include branch point messages for context continuity
	branchMessages := vcm.GetBranchPointMessages(contextID)
	allMessages = append(branchMessages, allMessages...)

	// Remove duplicates and limit results
	seen := make(map[string]bool)
	var uniqueMessages []Message

	for _, msg := range allMessages {
		if !seen[msg.ID] {
			seen[msg.ID] = true
			uniqueMessages = append(uniqueMessages, msg)
		}

		if len(uniqueMessages) >= vcm.config.MaxRetrieved {
			break
		}
	}

	return uniqueMessages, nil
}
