package chat

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/vectorstore"
)

// HybridMemoryConfig configures the hybrid memory system
type HybridMemoryConfig struct {
	// Working memory (recent messages)
	WorkingMemorySize int // Number of recent messages to keep in working memory

	// Vector memory (semantic retrieval)
	VectorConfig VectorMemoryConfig

	// Context assembly
	MaxContextTokens    int     // Maximum tokens for assembled context
	SemanticWeight      float32 // Weight for semantic relevance (0.0-1.0)
	RecencyWeight       float32 // Weight for recency (0.0-1.0)
	DeduplicationWindow int     // Number of recent messages to deduplicate from vector results

	// Tool indexing
	EnableToolIndexing bool   // Whether to index tool outputs separately
	ToolsCollection    string // Collection name for tool outputs
}

// DefaultHybridMemoryConfig returns optimal hybrid memory configuration
func DefaultHybridMemoryConfig() HybridMemoryConfig {
	return HybridMemoryConfig{
		WorkingMemorySize:   10, // Keep last 10 messages in working memory
		VectorConfig:        DefaultVectorMemoryConfig(),
		MaxContextTokens:    4000,    // Conservative token limit
		SemanticWeight:      0.7,     // Favor semantic relevance
		RecencyWeight:       0.3,     // But still consider recency
		DeduplicationWindow: 5,       // Avoid duplicating last 5 messages
		EnableToolIndexing:  true,    // Enable tool output indexing
		ToolsCollection:     "tools", // Store tool outputs in tools collection
	}
}

// HybridMemory combines working memory with semantic vector retrieval
type HybridMemory struct {
	workingMemory   *LangChainMemory       // Recent messages buffer
	vectorMemory    *LangChainVectorMemory // Semantic vector store
	documentIndexer *DocumentIndexer       // Document and file indexer
	config          HybridMemoryConfig
}

// NewHybridMemory creates a new hybrid memory system
func NewHybridMemory(manager *vectorstore.Manager, config HybridMemoryConfig) (*HybridMemory, error) {
	// Create working memory (regular buffer)
	workingMemory := NewLangChainMemory()

	// Create vector memory for semantic retrieval
	vectorMemory, err := NewLangChainVectorMemory(manager, config.VectorConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector memory: %w", err)
	}

	// Create document indexer
	docIndexerConfig := DefaultDocumentIndexerConfig()
	documentIndexer := NewDocumentIndexer(manager, docIndexerConfig)

	return &HybridMemory{
		workingMemory:   workingMemory,
		vectorMemory:    vectorMemory,
		documentIndexer: documentIndexer,
		config:          config,
	}, nil
}

// NewHybridMemoryWithConversation creates hybrid memory with existing conversation
func NewHybridMemoryWithConversation(manager *vectorstore.Manager, config HybridMemoryConfig, conv Conversation) (*HybridMemory, error) {
	// Create working memory with conversation
	workingMemory, err := NewLangChainMemoryWithConversation(conv)
	if err != nil {
		return nil, fmt.Errorf("failed to create working memory: %w", err)
	}

	// Create vector memory with conversation
	vectorMemory, err := NewLangChainVectorMemoryWithConversation(manager, config.VectorConfig, conv)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector memory: %w", err)
	}

	// Create document indexer
	docIndexerConfig := DefaultDocumentIndexerConfig()
	documentIndexer := NewDocumentIndexer(manager, docIndexerConfig)

	hybrid := &HybridMemory{
		workingMemory:   workingMemory,
		vectorMemory:    vectorMemory,
		documentIndexer: documentIndexer,
		config:          config,
	}

	// Maintain working memory size limit
	hybrid.maintainWorkingMemorySize()

	return hybrid, nil
}

// AddMessage adds a message to both working and vector memory
func (hm *HybridMemory) AddMessage(ctx context.Context, msg Message) error {
	// Add to working memory
	if err := hm.workingMemory.AddMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to add to working memory: %w", err)
	}

	// Add to vector memory for semantic indexing
	if err := hm.vectorMemory.AddMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to add to vector memory: %w", err)
	}

	// Special handling for tool results - index them in tools collection
	if msg.Role == RoleTool {
		if err := hm.indexToolOutput(ctx, msg); err != nil {
			// Log but don't fail - tool indexing is supplementary
			// TODO: Add proper logging
		}
	}

	// Maintain working memory size limit
	hm.maintainWorkingMemorySize()

	return nil
}

// maintainWorkingMemorySize keeps working memory within configured size
func (hm *HybridMemory) maintainWorkingMemorySize() {
	conv := hm.workingMemory.GetConversation()
	messages := GetMessages(conv)

	if len(messages) > hm.config.WorkingMemorySize {
		// Keep system messages and recent messages
		var systemMessages []Message
		var otherMessages []Message

		for _, msg := range messages {
			if msg.Role == RoleSystem {
				systemMessages = append(systemMessages, msg)
			} else {
				otherMessages = append(otherMessages, msg)
			}
		}

		// Keep only the most recent non-system messages
		recentStart := len(otherMessages) - (hm.config.WorkingMemorySize - len(systemMessages))
		if recentStart < 0 {
			recentStart = 0
		}

		// Rebuild conversation with system messages + recent messages
		newMessages := systemMessages
		newMessages = append(newMessages, otherMessages[recentStart:]...)

		newConv := NewConversation(conv.Model)
		for _, msg := range newMessages {
			newConv = AddMessage(newConv, msg)
		}

		// Update working memory conversation
		hm.workingMemory.conversation = &newConv
	}
}

// GetHybridContext assembles optimized context from working + semantic memory
func (hm *HybridMemory) GetHybridContext(ctx context.Context, query string) ([]Message, error) {
	// Get working memory (recent messages)
	workingMessages := GetMessages(hm.workingMemory.GetConversation())

	// Get semantically relevant messages
	relevantMessages, err := hm.vectorMemory.GetRelevantMessages(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get relevant messages: %w", err)
	}

	// Deduplicate: remove messages that are already in recent working memory
	deduplicatedRelevant := hm.deduplicateMessages(workingMessages, relevantMessages)

	// Score and combine messages
	scoredMessages := hm.scoreMessages(workingMessages, deduplicatedRelevant, query)

	// Assemble final context within token limits
	finalContext := hm.assembleContext(scoredMessages)

	return finalContext, nil
}

// deduplicateMessages removes vector results that overlap with working memory
func (hm *HybridMemory) deduplicateMessages(workingMessages, vectorMessages []Message) []Message {
	if len(workingMessages) == 0 {
		return vectorMessages
	}

	// Get recent messages for deduplication (last N messages)
	recentCount := hm.config.DeduplicationWindow
	if recentCount > len(workingMessages) {
		recentCount = len(workingMessages)
	}

	recentMessages := workingMessages[len(workingMessages)-recentCount:]

	// Create set of recent message contents for fast lookup
	recentContents := make(map[string]bool)
	for _, msg := range recentMessages {
		recentContents[msg.Content] = true
	}

	// Filter out duplicates
	var deduplicated []Message
	for _, msg := range vectorMessages {
		if !recentContents[msg.Content] {
			deduplicated = append(deduplicated, msg)
		}
	}

	return deduplicated
}

// ScoredMessage holds a message with relevance score
type ScoredMessage struct {
	Message        Message
	RelevanceScore float32
	IsRecent       bool
}

// scoreMessages assigns relevance scores to messages
func (hm *HybridMemory) scoreMessages(workingMessages, relevantMessages []Message, query string) []ScoredMessage {
	var scored []ScoredMessage

	// Score working memory messages (high recency, variable semantic relevance)
	for _, msg := range workingMessages {
		score := hm.config.RecencyWeight * 1.0 // Full recency weight for working memory

		// Add basic semantic relevance for working memory
		if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(query)) {
			score += hm.config.SemanticWeight * 0.8 // Boost if query terms present
		} else {
			score += hm.config.SemanticWeight * 0.3 // Base semantic score
		}

		scored = append(scored, ScoredMessage{
			Message:        msg,
			RelevanceScore: score,
			IsRecent:       true,
		})
	}

	// Score vector memory messages (high semantic relevance, lower recency)
	for _, msg := range relevantMessages {
		score := hm.config.SemanticWeight*0.9 + hm.config.RecencyWeight*0.2

		scored = append(scored, ScoredMessage{
			Message:        msg,
			RelevanceScore: score,
			IsRecent:       false,
		})
	}

	// Sort by relevance score (descending)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].RelevanceScore > scored[j].RelevanceScore
	})

	return scored
}

// assembleContext creates final context within token limits
func (hm *HybridMemory) assembleContext(scoredMessages []ScoredMessage) []Message {
	var context []Message
	var systemMessages []Message
	estimatedTokens := 0

	// Always include system messages first
	for _, sm := range scoredMessages {
		if sm.Message.Role == RoleSystem {
			systemMessages = append(systemMessages, sm.Message)
			estimatedTokens += hm.estimateTokens(sm.Message.Content)
		}
	}

	// Add system messages to context
	context = append(context, systemMessages...)

	// Add other messages in order of relevance, respecting token limits
	for _, sm := range scoredMessages {
		if sm.Message.Role == RoleSystem {
			continue // Already added
		}

		messageTokens := hm.estimateTokens(sm.Message.Content)
		if estimatedTokens+messageTokens <= hm.config.MaxContextTokens {
			context = append(context, sm.Message)
			estimatedTokens += messageTokens
		} else {
			break // Stop when token limit would be exceeded
		}
	}

	// Sort final context chronologically while preserving system messages at the start
	hm.sortContextChronologically(context)

	return context
}

// estimateTokens provides rough token estimation (4 chars ≈ 1 token)
func (hm *HybridMemory) estimateTokens(content string) int {
	return len(content) / 4
}

// sortContextChronologically sorts messages by timestamp while keeping system messages first
func (hm *HybridMemory) sortContextChronologically(messages []Message) {
	var systemMsgs []Message
	var otherMsgs []Message

	for _, msg := range messages {
		if msg.Role == RoleSystem {
			systemMsgs = append(systemMsgs, msg)
		} else {
			otherMsgs = append(otherMsgs, msg)
		}
	}

	// System messages stay at the beginning
	// Other messages are already in chronological order from the conversation
	copy(messages, systemMsgs)
	copy(messages[len(systemMsgs):], otherMsgs)
}

// GetConversation returns the current working conversation
func (hm *HybridMemory) GetConversation() Conversation {
	return hm.workingMemory.GetConversation()
}

// GetWorkingMemory returns the working memory component
func (hm *HybridMemory) GetWorkingMemory() *LangChainMemory {
	return hm.workingMemory
}

// GetVectorMemory returns the vector memory component
func (hm *HybridMemory) GetVectorMemory() *LangChainVectorMemory {
	return hm.vectorMemory
}

// GetDocumentIndexer returns the document indexer component
func (hm *HybridMemory) GetDocumentIndexer() *DocumentIndexer {
	return hm.documentIndexer
}

// Clear clears both working and vector memory
func (hm *HybridMemory) Clear(ctx context.Context) error {
	if err := hm.workingMemory.Clear(ctx); err != nil {
		return fmt.Errorf("failed to clear working memory: %w", err)
	}

	if err := hm.vectorMemory.Clear(ctx); err != nil {
		return fmt.Errorf("failed to clear vector memory: %w", err)
	}

	return nil
}

// GetMemoryVariables returns memory variables with hybrid context
func (hm *HybridMemory) GetMemoryVariables(ctx context.Context) (map[string]any, error) {
	// Get base variables from working memory
	vars, err := hm.workingMemory.GetMemoryVariables(ctx)
	if err != nil {
		return nil, err
	}

	// Add hybrid context if we have a recent user message
	conv := hm.workingMemory.GetConversation()
	messages := GetMessages(conv)
	if len(messages) > 0 {
		lastMessage := messages[len(messages)-1]
		if lastMessage.Role == RoleUser {
			// Get hybrid context for the last user query
			hybridContext, err := hm.GetHybridContext(ctx, lastMessage.Content)
			if err == nil && len(hybridContext) > 0 {
				// Format hybrid context
				contextParts := make([]string, 0, len(hybridContext))
				for _, msg := range hybridContext {
					switch msg.Role {
					case RoleUser:
						contextParts = append(contextParts, fmt.Sprintf("User: %s", msg.Content))
					case RoleAssistant:
						contextParts = append(contextParts, fmt.Sprintf("Assistant: %s", msg.Content))
					case RoleSystem:
						contextParts = append(contextParts, fmt.Sprintf("System: %s", msg.Content))
					case RoleTool:
						contextParts = append(contextParts, fmt.Sprintf("Tool (%s): %s", msg.ToolName, msg.Content))
					}
				}

				vars["hybrid_context"] = strings.Join(contextParts, "\n")
			}
		}
	}

	return vars, nil
}

// SaveContext saves context to both memory systems
func (hm *HybridMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	if err := hm.workingMemory.SaveContext(ctx, inputs, outputs); err != nil {
		return fmt.Errorf("failed to save to working memory: %w", err)
	}

	if err := hm.vectorMemory.SaveContext(ctx, inputs, outputs); err != nil {
		return fmt.Errorf("failed to save to vector memory: %w", err)
	}

	// Maintain working memory size after saving
	hm.maintainWorkingMemorySize()

	return nil
}

// indexToolOutput indexes tool outputs in a separate tools collection for enhanced retrieval
func (hm *HybridMemory) indexToolOutput(ctx context.Context, toolMsg Message) error {
	if !hm.config.EnableToolIndexing || toolMsg.Role != RoleTool {
		return nil
	}

	// Get the vector store manager from the vector memory
	manager := hm.vectorMemory.manager
	if manager == nil {
		return fmt.Errorf("vector store manager not available")
	}

	// Create a detailed document for the tool output
	docID := fmt.Sprintf("tool_%s_%d", toolMsg.ToolName, time.Now().UnixNano())

	// Enhanced content format for tool outputs
	content := fmt.Sprintf("Tool: %s\nOutput: %s", toolMsg.ToolName, toolMsg.Content)

	// Rich metadata for tool outputs
	metadata := map[string]any{
		"type":         "tool_output",
		"tool_name":    toolMsg.ToolName,
		"content_type": "tool_result",
		"timestamp":    time.Now().Unix(),
		"indexed_at":   time.Now().Format(time.RFC3339),
	}

	// Analyze content type for better categorization
	contentLower := strings.ToLower(toolMsg.Content)
	if strings.Contains(contentLower, "error") || strings.Contains(contentLower, "failed") {
		metadata["result_type"] = "error"
	} else if strings.Contains(contentLower, "success") || strings.Contains(contentLower, "completed") {
		metadata["result_type"] = "success"
	} else {
		metadata["result_type"] = "info"
	}

	// Create document
	doc := vectorstore.Document{
		ID:       docID,
		Content:  content,
		Metadata: metadata,
	}

	// Index in tools collection
	return manager.IndexDocument(ctx, hm.config.ToolsCollection, doc)
}

// GetRelevantToolOutputs retrieves semantically relevant tool outputs
func (hm *HybridMemory) GetRelevantToolOutputs(ctx context.Context, query string, maxResults int) ([]vectorstore.Result, error) {
	if !hm.config.EnableToolIndexing {
		return nil, nil
	}

	manager := hm.vectorMemory.manager
	if manager == nil {
		return nil, fmt.Errorf("vector store manager not available")
	}

	// Search tool outputs with lower threshold for broader results
	return manager.Search(ctx, hm.config.ToolsCollection, query, maxResults)
}
