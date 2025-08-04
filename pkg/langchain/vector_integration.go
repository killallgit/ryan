package langchain

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/chroma"
	"github.com/tmc/langchaingo/vectorstores/pgvector"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// VectorStoreManager manages vector store operations for file context
type VectorStoreManager struct {
	vectorStore     vectorstores.VectorStore
	embedder        embeddings.Embedder
	fileContextMem  *FileContextMemory
	conversationMem *BranchingConversationMemory
	collections     map[string]string // collection names for different types
	log             *logger.Logger
}

// BranchingConversationMemory extends memory for conversation branching
type BranchingConversationMemory struct {
	branchingConv *chat.BranchingConversation
	baseMemory    schema.Memory
	log           *logger.Logger
}

// NewVectorStoreManager creates a new vector store manager
func NewVectorStoreManager(llm llms.Model, cfg *config.Config) (*VectorStoreManager, error) {
	// Create Ollama LLM client for embeddings using configured URL
	ollamaURL := "http://localhost:11434"
	if cfg != nil && cfg.Ollama.URL != "" {
		ollamaURL = cfg.Ollama.URL
	}
	
	ollamaClient, err := ollama.New(
		ollama.WithModel("nomic-embed-text"),
		ollama.WithServerURL(ollamaURL),
	)
	if err != nil {
		return nil, fmt.Errorf("creating Ollama client for embeddings: %w", err)
	}
	
	// Create embedder using the Ollama client
	embedder, err := embeddings.NewEmbedder(ollamaClient)
	if err != nil {
		return nil, fmt.Errorf("creating embedder: %w", err)
	}

	// Create vector store based on configuration
	vectorStore, err := createVectorStore(cfg, embedder)
	if err != nil {
		return nil, fmt.Errorf("creating vector store: %w", err)
	}

	return &VectorStoreManager{
		vectorStore: vectorStore,
		embedder:    embedder,
		collections: map[string]string{
			"files":         "ryan_files",
			"conversations": "ryan_conversations",
			"contexts":      "ryan_contexts",
		},
		log: logger.WithComponent("vector_store_manager"),
	}, nil
}

// createVectorStore creates a vector store based on configuration
func createVectorStore(cfg *config.Config, embedder embeddings.Embedder) (vectorstores.VectorStore, error) {
	ctx := context.Background()

	// Default to in-memory store if no configuration
	if cfg == nil || cfg.VectorStore == nil {
		return chroma.New(
			chroma.WithChromaURL("http://localhost:8000"),
			chroma.WithEmbedder(embedder),
		)
	}

	switch cfg.VectorStore.Type {
	case "chroma":
		return chroma.New(
			chroma.WithChromaURL(cfg.VectorStore.URL),
			chroma.WithEmbedder(embedder),
		)

	case "qdrant":
		qdrantURL, err := url.Parse(cfg.VectorStore.URL)
		if err != nil {
			return nil, fmt.Errorf("parsing Qdrant URL: %w", err)
		}
		return qdrant.New(
			qdrant.WithURL(*qdrantURL),
			qdrant.WithCollectionName(cfg.VectorStore.Collection),
			qdrant.WithEmbedder(embedder),
		)

	case "pgvector":
		return pgvector.New(
			ctx,
			pgvector.WithConnectionURL(cfg.VectorStore.URL),
			pgvector.WithEmbedder(embedder),
		)

	default:
		return chroma.New(
			chroma.WithEmbedder(embedder),
		)
	}
}

// AddFileToVectorStore adds a file and its chunks to the vector store
func (vsm *VectorStoreManager) AddFileToVectorStore(ctx context.Context, fc *chat.FileContext) error {
	vsm.log.Debug("Adding file to vector store", "path", fc.Path)

	// Create documents for the file chunks
	docs := make([]schema.Document, 0, len(fc.ChunkRefs))

	for i, chunk := range fc.ChunkRefs {
		// Extract content for this chunk (simplified - in practice you'd store chunk content)
		content := vsm.extractChunkContent(fc.Content, chunk)
		
		doc := schema.Document{
			PageContent: content,
			Metadata: map[string]any{
				"file_path":    fc.Path,
				"chunk_id":     chunk.ChunkID,
				"start_line":   chunk.StartLine,
				"end_line":     chunk.EndLine,
				"chunk_index":  i,
				"content_hash": fc.ContentHash,
				"last_edit":    fc.LastEdit.Unix(),
				"collection":   vsm.collections["files"],
			},
		}

		docs = append(docs, doc)
	}

	// Add documents to vector store
	_, err := vsm.vectorStore.AddDocuments(ctx, docs)
	if err != nil {
		return fmt.Errorf("adding documents to vector store: %w", err)
	}

	vsm.log.Debug("Added file chunks to vector store", "path", fc.Path, "chunks", len(docs))
	return nil
}

// SearchFileContent searches for content across all files
func (vsm *VectorStoreManager) SearchFileContent(ctx context.Context, query string, numResults int) ([]schema.Document, error) {
	// Search with file collection filter
	return vsm.vectorStore.SimilaritySearch(ctx, query, numResults,
		vectorstores.WithFilters(map[string]any{
			"collection": vsm.collections["files"],
		}))
}

// SearchInFile searches for content within a specific file
func (vsm *VectorStoreManager) SearchInFile(ctx context.Context, filePath, query string, numResults int) ([]schema.Document, error) {
	return vsm.vectorStore.SimilaritySearch(ctx, query, numResults,
		vectorstores.WithFilters(map[string]any{
			"collection": vsm.collections["files"],
			"file_path":  filePath,
		}))
}

// AddConversationToVectorStore adds conversation messages to vector store
func (vsm *VectorStoreManager) AddConversationToVectorStore(ctx context.Context, conv *chat.BranchingConversation) error {
	docs := make([]schema.Document, 0)

	for i, msg := range conv.Messages {
		// Skip system messages and empty messages
		if msg.IsSystem() || msg.IsEmpty() {
			continue
		}

		content := msg.Content
		if msg.HasThinking() {
			content = fmt.Sprintf("Thinking: %s\n\nResponse: %s", msg.Thinking.Content, msg.Content)
		}

		doc := schema.Document{
			PageContent: content,
			Metadata: map[string]any{
				"role":         msg.Role,
				"timestamp":    msg.Timestamp.Unix(),
				"message_id":   msg.Metadata.MessageID,
				"branch_id":    conv.CurrentBranch,
				"message_index": i,
				"has_thinking": msg.HasThinking(),
				"collection":   vsm.collections["conversations"],
			},
		}

		docs = append(docs, doc)
	}

	if len(docs) > 0 {
		_, err := vsm.vectorStore.AddDocuments(ctx, docs)
		if err != nil {
			return fmt.Errorf("adding conversation to vector store: %w", err)
		}
	}

	return nil
}

// SearchConversationHistory searches for relevant messages in conversation history
func (vsm *VectorStoreManager) SearchConversationHistory(ctx context.Context, query string, numResults int) ([]schema.Document, error) {
	return vsm.vectorStore.SimilaritySearch(ctx, query, numResults,
		vectorstores.WithFilters(map[string]any{
			"collection": vsm.collections["conversations"],
		}))
}

// CreateContextualSearch combines file and conversation search
func (vsm *VectorStoreManager) CreateContextualSearch(ctx context.Context, query string, maxResults int) (*ContextualSearchResult, error) {
	// Search in files
	fileResults, err := vsm.SearchFileContent(ctx, query, maxResults/2)
	if err != nil {
		return nil, fmt.Errorf("searching files: %w", err)
	}

	// Search in conversations
	convResults, err := vsm.SearchConversationHistory(ctx, query, maxResults/2)
	if err != nil {
		return nil, fmt.Errorf("searching conversations: %w", err)
	}

	return &ContextualSearchResult{
		Query:               query,
		FileResults:         fileResults,
		ConversationResults: convResults,
		TotalResults:        len(fileResults) + len(convResults),
	}, nil
}

// ContextualSearchResult contains results from both files and conversations
type ContextualSearchResult struct {
	Query               string
	FileResults         []schema.Document
	ConversationResults []schema.Document
	TotalResults        int
}

// BuildContextFromSearch creates a context string from search results
func (vsm *VectorStoreManager) BuildContextFromSearch(result *ContextualSearchResult) string {
	var context strings.Builder

	context.WriteString(fmt.Sprintf("# Contextual Search Results for: %s\n\n", result.Query))

	if len(result.FileResults) > 0 {
		context.WriteString("## Relevant File Content:\n\n")
		for i, doc := range result.FileResults {
			filePath, _ := doc.Metadata["file_path"].(string)
			startLine, _ := doc.Metadata["start_line"].(int)
			endLine, _ := doc.Metadata["end_line"].(int)

			context.WriteString(fmt.Sprintf("### %d. %s (lines %d-%d)\n", i+1, filePath, startLine, endLine))
			context.WriteString("```\n")
			context.WriteString(doc.PageContent)
			context.WriteString("\n```\n\n")
		}
	}

	if len(result.ConversationResults) > 0 {
		context.WriteString("## Relevant Conversation History:\n\n")
		for i, doc := range result.ConversationResults {
			role, _ := doc.Metadata["role"].(string)
			timestamp, _ := doc.Metadata["timestamp"].(int64)

			context.WriteString(fmt.Sprintf("### %d. %s message\n", i+1, role))
			context.WriteString(fmt.Sprintf("*Timestamp: %d*\n\n", timestamp))
			context.WriteString(doc.PageContent)
			context.WriteString("\n\n")
		}
	}

	return context.String()
}

// Helper methods

func (vsm *VectorStoreManager) extractChunkContent(fullContent string, chunk chat.ChunkRef) string {
	// Simple line-based extraction
	lines := strings.Split(fullContent, "\n")
	
	startIdx := chunk.StartLine - 1
	endIdx := chunk.EndLine - 1
	
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= len(lines) {
		endIdx = len(lines) - 1
	}
	if startIdx > endIdx {
		return ""
	}

	return strings.Join(lines[startIdx:endIdx+1], "\n")
}

// WithFileContextMemory sets the file context memory
func (vsm *VectorStoreManager) WithFileContextMemory(mem *FileContextMemory) *VectorStoreManager {
	vsm.fileContextMem = mem
	return vsm
}

// WithConversationMemory sets the conversation memory
func (vsm *VectorStoreManager) WithConversationMemory(mem *BranchingConversationMemory) *VectorStoreManager {
	vsm.conversationMem = mem
	return vsm
}

// BranchingConversationMemory methods

// NewBranchingConversationMemory creates memory for branching conversations
func NewBranchingConversationMemory(branchingConv *chat.BranchingConversation, baseMemory schema.Memory) *BranchingConversationMemory {
	return &BranchingConversationMemory{
		branchingConv: branchingConv,
		baseMemory:    baseMemory,
		log:           logger.WithComponent("branching_conversation_memory"),
	}
}

// GetCurrentBranchMessages returns messages for the current branch
func (bcm *BranchingConversationMemory) GetCurrentBranchMessages() []chat.Message {
	return bcm.branchingConv.Messages
}

// GetBranchContext returns context for a specific branch
func (bcm *BranchingConversationMemory) GetBranchContext(branchID string) ([]chat.Message, error) {
	// Switch to branch temporarily to get context
	originalBranch := bcm.branchingConv.CurrentBranch
	
	switched, err := bcm.branchingConv.SwitchBranch(branchID)
	if err != nil {
		return nil, err
	}
	
	messages := switched.Messages
	
	// Switch back
	_, err = bcm.branchingConv.SwitchBranch(originalBranch)
	if err != nil {
		bcm.log.Error("Failed to switch back to original branch", "error", err)
	}

	return messages, nil
}

// AddMessageToBranch adds a message to a specific branch
func (bcm *BranchingConversationMemory) AddMessageToBranch(branchID string, msg chat.Message, fileContexts []chat.FileContext) error {
	// Switch to target branch
	originalBranch := bcm.branchingConv.CurrentBranch
	
	switched, err := bcm.branchingConv.SwitchBranch(branchID)
	if err != nil {
		return err
	}
	
	// Add message with context
	updated, err := switched.AddMessageWithContext(msg, fileContexts)
	if err != nil {
		return err
	}
	
	*bcm.branchingConv = *updated
	
	// Switch back if needed
	if originalBranch != branchID {
		_, err = bcm.branchingConv.SwitchBranch(originalBranch)
		if err != nil {
			bcm.log.Error("Failed to switch back to original branch", "error", err)
		}
	}

	return nil
}