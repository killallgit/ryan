package langchain

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores"
)

// FileProcessingChain handles loading, chunking, and embedding files
type FileProcessingChain struct {
	llm         llms.Model
	embedder    embeddings.Embedder
	splitter    textsplitter.TextSplitter
	vectorStore vectorstores.VectorStore
	memory      *FileContextMemory
	log         *logger.Logger
}

// NewFileProcessingChain creates a new file processing chain
func NewFileProcessingChain(llm llms.Model, memory *FileContextMemory, cfg *config.Config) (*FileProcessingChain, error) {
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

	// Create a code-aware splitter
	splitter := textsplitter.NewRecursiveCharacter()
	splitter.ChunkSize = 1000        // Smaller chunks for code
	splitter.ChunkOverlap = 200      // Good overlap for context
	splitter.Separators = []string{
		"\n\n",    // Paragraphs
		"\n",      // Lines
		". ",      // Sentences
		", ",      // Phrases
		" ",       // Words
		"",        // Characters
	}

	return &FileProcessingChain{
		llm:      llm,
		embedder: embedder,
		splitter: splitter,
		memory:   memory,
		log:      logger.WithComponent("file_processing_chain"),
	}, nil
}

// Call implements the Chain interface
func (c *FileProcessingChain) Call(ctx context.Context, inputs map[string]any, options ...chains.ChainCallOption) (map[string]any, error) {
	filePath, ok := inputs["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path input required")
	}

	// Process the file
	fileContext, err := c.ProcessFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Add to memory
	c.memory.AddFileContext(*fileContext)

	return map[string]any{
		"file_context": fileContext,
		"chunks_count": len(fileContext.ChunkRefs),
		"content_hash": fileContext.ContentHash,
	}, nil
}

// ProcessFile loads and processes a file
func (c *FileProcessingChain) ProcessFile(ctx context.Context, filePath string) (*chat.FileContext, error) {
	c.log.Debug("Processing file", "path", filePath)

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Create file context
	fc := &chat.FileContext{
		Path:        filePath,
		Content:     string(content),
		ContentHash: c.hashContent(content),
		LastEdit:    time.Now(),
		EditHistory: []chat.FileEdit{
			{
				Timestamp:  time.Now(),
				NewContent: string(content),
				EditType:   "create",
			},
		},
	}

	// Load and split document
	docs, err := c.loadAndSplitFile(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("loading and splitting file: %w", err)
	}

	// Create embeddings and chunk references
	chunks, err := c.createChunks(ctx, docs, filePath)
	if err != nil {
		return nil, fmt.Errorf("creating chunks: %w", err)
	}

	fc.ChunkRefs = chunks

	// If vector store is available, add documents
	if c.vectorStore != nil {
		_, err := c.vectorStore.AddDocuments(ctx, docs)
		if err != nil {
			c.log.Error("Failed to add documents to vector store", "error", err)
		} else {
			fc.VectorStoreID = c.generateVectorStoreID(filePath)
		}
	}

	return fc, nil
}

// loadAndSplitFile loads a file and splits it into documents
func (c *FileProcessingChain) loadAndSplitFile(ctx context.Context, filePath string) ([]schema.Document, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Choose loader based on file type
	var loader documentloaders.Loader
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".go", ".py", ".js", ".ts", ".java", ".cpp", ".c", ".rs":
		// Use text loader for code files
		loader = documentloaders.NewText(file)
	case ".md", ".txt", ".log":
		// Use text loader for text files
		loader = documentloaders.NewText(file)
	case ".json", ".yaml", ".yml", ".toml":
		// Use text loader for config files
		loader = documentloaders.NewText(file)
	default:
		// Default to text loader
		loader = documentloaders.NewText(file)
	}

	// Load and split
	return loader.LoadAndSplit(ctx, c.splitter)
}

// createChunks creates chunk references with line numbers
func (c *FileProcessingChain) createChunks(ctx context.Context, docs []schema.Document, filePath string) ([]chat.ChunkRef, error) {
	chunks := make([]chat.ChunkRef, 0, len(docs))

	for i, doc := range docs {
		// Extract line numbers from metadata if available
		startLine := 1
		endLine := 1

		if lineStart, ok := doc.Metadata["start_line"].(int); ok {
			startLine = lineStart
		}
		if lineEnd, ok := doc.Metadata["end_line"].(int); ok {
			endLine = lineEnd
		}

		// Create embedding for chunk
		var embedding []float32
		if c.embedder != nil {
			embResult, err := c.embedder.EmbedQuery(ctx, doc.PageContent)
			if err != nil {
				c.log.Warn("Failed to create embedding", "error", err)
			} else {
				embedding = embResult
			}
		}

		chunk := chat.ChunkRef{
			ChunkID:   fmt.Sprintf("%s_chunk_%d", c.hashContent([]byte(filePath)), i),
			StartLine: startLine,
			EndLine:   endLine,
			Embedding: embedding,
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// GetMemory returns the chain's memory
func (c *FileProcessingChain) GetMemory() schema.Memory {
	return c.memory
}

// GetInputKeys returns expected input keys
func (c *FileProcessingChain) GetInputKeys() []string {
	return []string{"file_path"}
}

// GetOutputKeys returns output keys
func (c *FileProcessingChain) GetOutputKeys() []string {
	return []string{"file_context", "chunks_count", "content_hash"}
}

// Helper methods

func (c *FileProcessingChain) hashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash[:8])
}

func (c *FileProcessingChain) generateVectorStoreID(filePath string) string {
	return fmt.Sprintf("vs_%s_%d", c.hashContent([]byte(filePath)), time.Now().Unix())
}

// WithVectorStore sets the vector store for the chain
func (c *FileProcessingChain) WithVectorStore(vs vectorstores.VectorStore) *FileProcessingChain {
	c.vectorStore = vs
	return c
}

// FileSearchChain searches for content across files using embeddings
type FileSearchChain struct {
	embedder    embeddings.Embedder
	vectorStore vectorstores.VectorStore
	memory      *FileContextMemory
	log         *logger.Logger
}

// NewFileSearchChain creates a new file search chain
func NewFileSearchChain(embedder embeddings.Embedder, memory *FileContextMemory) *FileSearchChain {
	return &FileSearchChain{
		embedder: embedder,
		memory:   memory,
		log:      logger.WithComponent("file_search_chain"),
	}
}

// Call implements the Chain interface
func (c *FileSearchChain) Call(ctx context.Context, inputs map[string]any, options ...chains.ChainCallOption) (map[string]any, error) {
	query, ok := inputs["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query input required")
	}

	numResults := 5
	if n, ok := inputs["num_results"].(int); ok {
		numResults = n
	}

	// Search using vector store if available
	if c.vectorStore != nil {
		docs, err := c.vectorStore.SimilaritySearch(ctx, query, numResults)
		if err != nil {
			return nil, fmt.Errorf("similarity search failed: %w", err)
		}

		return map[string]any{
			"results":    docs,
			"num_found":  len(docs),
			"search_type": "vector",
		}, nil
	}

	// Fallback to memory-based search
	relevantContexts := c.memory.GetRelevantFileContexts(query)
	
	return map[string]any{
		"results":     relevantContexts,
		"num_found":   len(relevantContexts),
		"search_type": "memory",
	}, nil
}

// GetMemory returns the chain's memory
func (c *FileSearchChain) GetMemory() schema.Memory {
	return c.memory
}

// GetInputKeys returns expected input keys
func (c *FileSearchChain) GetInputKeys() []string {
	return []string{"query"}
}

// GetOutputKeys returns output keys
func (c *FileSearchChain) GetOutputKeys() []string {
	return []string{"results", "num_found", "search_type"}
}

// WithVectorStore sets the vector store
func (c *FileSearchChain) WithVectorStore(vs vectorstores.VectorStore) *FileSearchChain {
	c.vectorStore = vs
	return c
}

// FileLoaderChain loads files with appropriate loaders
type FileLoaderChain struct {
	memory *FileContextMemory
	log    *logger.Logger
}

// NewFileLoaderChain creates a new file loader chain
func NewFileLoaderChain(memory *FileContextMemory) *FileLoaderChain {
	return &FileLoaderChain{
		memory: memory,
		log:    logger.WithComponent("file_loader_chain"),
	}
}

// Call implements the Chain interface
func (c *FileLoaderChain) Call(ctx context.Context, inputs map[string]any, options ...chains.ChainCallOption) (map[string]any, error) {
	filePath, ok := inputs["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path input required")
	}

	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Create basic file context
	hash := sha256.Sum256(content)
	fc := chat.FileContext{
		Path:        filePath,
		Content:     string(content),
		ContentHash: fmt.Sprintf("%x", hash[:8]),
		LastEdit:    info.ModTime(),
	}

	// Add to memory
	c.memory.AddFileContext(fc)

	return map[string]any{
		"content":      string(content),
		"size":         info.Size(),
		"modified":     info.ModTime(),
		"file_context": fc,
	}, nil
}

// GetMemory returns the chain's memory
func (c *FileLoaderChain) GetMemory() schema.Memory {
	return c.memory
}

// GetInputKeys returns expected input keys
func (c *FileLoaderChain) GetInputKeys() []string {
	return []string{"file_path"}
}

// GetOutputKeys returns output keys
func (c *FileLoaderChain) GetOutputKeys() []string {
	return []string{"content", "size", "modified", "file_context"}
}