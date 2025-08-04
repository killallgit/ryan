package vectorstore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DocumentIndexer handles indexing of files and documents into vector stores
type DocumentIndexer struct {
	manager        *Manager
	collectionName string
	processor      *DocumentProcessor
}

// IndexerConfig configures the document indexer
type IndexerConfig struct {
	CollectionName string
	ChunkSize      int // Size of text chunks in characters
	ChunkOverlap   int // Overlap between chunks
}

// DefaultIndexerConfig returns default indexer configuration
func DefaultIndexerConfig() IndexerConfig {
	return IndexerConfig{
		CollectionName: "documents",
		ChunkSize:      1000,
		ChunkOverlap:   200,
	}
}

// NewDocumentIndexer creates a new document indexer using a Manager
func NewDocumentIndexer(manager *Manager, config IndexerConfig) (*DocumentIndexer, error) {
	// Ensure collection exists
	_, err := manager.GetCollection(config.CollectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create collection: %w", err)
	}

	return &DocumentIndexer{
		manager:        manager,
		collectionName: config.CollectionName,
		processor:      manager.GetDocumentProcessor(),
	}, nil
}

// IndexFile indexes a single file
func (di *DocumentIndexer) IndexFile(ctx context.Context, filePath string) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	// Create metadata
	metadata := map[string]interface{}{
		"source":     filePath,
		"filename":   filepath.Base(filePath),
		"extension":  filepath.Ext(filePath),
		"size":       fileInfo.Size(),
		"modified":   fileInfo.ModTime().Unix(),
		"indexed_at": time.Now().Unix(),
	}

	// Index based on file type
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".txt", ".md", ".log":
		return di.indexTextFile(ctx, filePath, string(content), metadata)
	case ".go", ".py", ".js", ".java", ".cpp", ".c", ".rs":
		return di.indexCodeFile(ctx, filePath, string(content), metadata)
	case ".json", ".yaml", ".yml", ".toml":
		return di.indexStructuredFile(ctx, filePath, string(content), metadata)
	default:
		return di.indexTextFile(ctx, filePath, string(content), metadata)
	}
}

// indexTextFile indexes a text file
func (di *DocumentIndexer) indexTextFile(ctx context.Context, filePath string, content string, metadata map[string]interface{}) error {
	metadata["type"] = "text"

	// Create base document
	doc := Document{
		ID:       filePath,
		Content:  content,
		Metadata: metadata,
	}

	// Use manager to chunk and index
	return di.manager.ChunkAndIndexDocument(ctx, di.collectionName, doc, metadata)
}

// indexCodeFile indexes a code file with language-aware chunking
func (di *DocumentIndexer) indexCodeFile(ctx context.Context, filePath string, content string, metadata map[string]interface{}) error {
	metadata["type"] = "code"
	metadata["language"] = getLanguageFromExt(filepath.Ext(filePath))

	// For code files, use code chunking
	chunks := di.processor.ChunkCode(content)
	docs := make([]Document, 0, len(chunks))


	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}

		// Create document ID
		docID := fmt.Sprintf("%s_chunk_%d", filePath, i)

		// Copy metadata and add chunk info
		chunkMeta := make(map[string]interface{})
		for k, v := range metadata {
			chunkMeta[k] = v
		}
		chunkMeta["chunk_index"] = i
		chunkMeta["chunk_total"] = len(chunks)

		docs = append(docs, Document{
			ID:       docID,
			Content:  chunk,
			Metadata: chunkMeta,
		})
	}

	return di.manager.IndexDocuments(ctx, di.collectionName, docs)
}

// indexStructuredFile indexes structured data files
func (di *DocumentIndexer) indexStructuredFile(ctx context.Context, filePath string, content string, metadata map[string]interface{}) error {
	metadata["type"] = "structured"

	// For structured files, treat the whole content as one chunk
	doc := Document{
		ID:       filePath,
		Content:  content,
		Metadata: metadata,
	}

	return di.manager.IndexDocument(ctx, di.collectionName, doc)
}

// IndexDirectory recursively indexes all files in a directory
func (di *DocumentIndexer) IndexDirectory(ctx context.Context, dirPath string, patterns []string) error {
	var indexErrors []error

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file matches patterns
		if len(patterns) > 0 {
			matched := false
			for _, pattern := range patterns {
				if match, _ := filepath.Match(pattern, filepath.Base(path)); match {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}

		// Index the file
		if err := di.IndexFile(ctx, path); err != nil {
			indexErrors = append(indexErrors, fmt.Errorf("failed to index %s: %w", path, err))
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(indexErrors) > 0 {
		return fmt.Errorf("encountered %d errors during indexing: %v", len(indexErrors), indexErrors)
	}

	return nil
}

// IndexReader indexes content from an io.Reader
func (di *DocumentIndexer) IndexReader(ctx context.Context, reader io.Reader, source string, metadata map[string]interface{}) error {
	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["source"] = source
	metadata["indexed_at"] = time.Now().Unix()

	// Create document
	doc := Document{
		ID:       source,
		Content:  string(content),
		Metadata: metadata,
	}

	// Use manager to chunk and index
	return di.manager.ChunkAndIndexDocument(ctx, di.collectionName, doc, metadata)
}

// SearchDocuments searches for documents matching the query
func (di *DocumentIndexer) SearchDocuments(ctx context.Context, query string, k int, opts ...QueryOption) ([]Document, error) {
	results, err := di.manager.Search(ctx, di.collectionName, query, k, opts...)
	if err != nil {
		return nil, err
	}

	docs := make([]Document, len(results))
	for i, result := range results {
		docs[i] = result.Document
	}

	return docs, nil
}

// GetCollectionName returns the collection name used by this indexer
func (di *DocumentIndexer) GetCollectionName() string {
	return di.collectionName
}

// getLanguageFromExt returns the programming language based on file extension
func getLanguageFromExt(ext string) string {
	langMap := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".java":  "java",
		".cpp":   "cpp",
		".c":     "c",
		".rs":    "rust",
		".rb":    "ruby",
		".php":   "php",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".r":     "r",
	}

	if lang, ok := langMap[strings.ToLower(ext)]; ok {
		return lang
	}
	return "unknown"
}

