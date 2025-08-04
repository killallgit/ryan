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

// DocumentIndexer handles indexing of documents into vector stores
type DocumentIndexer struct {
	store      VectorStore
	collection Collection
	chunkSize  int
	chunkOverlap int
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

// NewDocumentIndexer creates a new document indexer
func NewDocumentIndexer(store VectorStore, config IndexerConfig) (*DocumentIndexer, error) {
	collection, err := GetOrCreateCollection(store, config.CollectionName, map[string]interface{}{
		"type": "document_index",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get or create collection: %w", err)
	}

	return &DocumentIndexer{
		store:        store,
		collection:   collection,
		chunkSize:    config.ChunkSize,
		chunkOverlap: config.ChunkOverlap,
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
		"source":      filePath,
		"filename":    filepath.Base(filePath),
		"extension":   filepath.Ext(filePath),
		"size":        fileInfo.Size(),
		"modified":    fileInfo.ModTime().Unix(),
		"indexed_at":  time.Now().Unix(),
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
	chunks := di.chunkText(content)
	return di.indexChunks(ctx, filePath, chunks, metadata)
}

// indexCodeFile indexes a code file with language-aware chunking
func (di *DocumentIndexer) indexCodeFile(ctx context.Context, filePath string, content string, metadata map[string]interface{}) error {
	// For code files, try to chunk by functions/methods
	chunks := di.chunkCode(content, filepath.Ext(filePath))
	metadata["type"] = "code"
	metadata["language"] = getLanguageFromExt(filepath.Ext(filePath))
	return di.indexChunks(ctx, filePath, chunks, metadata)
}

// indexStructuredFile indexes structured data files
func (di *DocumentIndexer) indexStructuredFile(ctx context.Context, filePath string, content string, metadata map[string]interface{}) error {
	// For structured files, treat the whole content as one chunk
	// In the future, we could parse and index individual fields
	metadata["type"] = "structured"
	chunks := []string{content}
	return di.indexChunks(ctx, filePath, chunks, metadata)
}

// indexChunks indexes text chunks into the vector store
func (di *DocumentIndexer) indexChunks(ctx context.Context, source string, chunks []string, baseMetadata map[string]interface{}) error {
	docs := make([]Document, 0, len(chunks))
	
	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}

		// Create document ID
		docID := fmt.Sprintf("%s_chunk_%d", source, i)

		// Copy metadata and add chunk-specific info
		metadata := make(map[string]interface{})
		for k, v := range baseMetadata {
			metadata[k] = v
		}
		metadata["chunk_index"] = i
		metadata["chunk_total"] = len(chunks)

		docs = append(docs, Document{
			ID:       docID,
			Content:  chunk,
			Metadata: metadata,
		})
	}

	if len(docs) > 0 {
		return di.collection.AddDocuments(ctx, docs)
	}

	return nil
}

// chunkText splits text into overlapping chunks
func (di *DocumentIndexer) chunkText(text string) []string {
	if len(text) <= di.chunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + di.chunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at a sentence or paragraph boundary
		if end < len(text) {
			// Look for sentence endings
			sentenceEnds := []string{". ", ".\n", "! ", "!\n", "? ", "?\n"}
			bestEnd := end
			
			for _, sep := range sentenceEnds {
				if idx := strings.LastIndex(text[start:end], sep); idx > 0 {
					bestEnd = start + idx + len(sep)
					break
				}
			}
			
			// If no sentence boundary, try paragraph
			if bestEnd == end {
				if idx := strings.LastIndex(text[start:end], "\n\n"); idx > 0 {
					bestEnd = start + idx + 2
				}
			}
			
			end = bestEnd
		}

		chunks = append(chunks, text[start:end])

		// Move start position with overlap
		if end < len(text) {
			start = end - di.chunkOverlap
			if start < 0 {
				start = 0
			}
		} else {
			break
		}
	}

	return chunks
}

// chunkCode splits code into logical chunks (functions, classes, etc.)
func (di *DocumentIndexer) chunkCode(code string, ext string) []string {
	// Simple line-based chunking for now
	// TODO: Implement language-aware chunking
	lines := strings.Split(code, "\n")
	
	var chunks []string
	var currentChunk []string
	currentSize := 0

	for _, line := range lines {
		lineSize := len(line) + 1 // +1 for newline
		
		// Check if adding this line would exceed chunk size
		if currentSize+lineSize > di.chunkSize && len(currentChunk) > 0 {
			chunks = append(chunks, strings.Join(currentChunk, "\n"))
			
			// Start new chunk with overlap
			overlapStart := len(currentChunk) - (di.chunkOverlap / 50) // Rough line count for overlap
			if overlapStart < 0 {
				overlapStart = 0
			}
			currentChunk = currentChunk[overlapStart:]
			currentSize = 0
			for _, l := range currentChunk {
				currentSize += len(l) + 1
			}
		}

		currentChunk = append(currentChunk, line)
		currentSize += lineSize
	}

	// Add the last chunk
	if len(currentChunk) > 0 {
		chunks = append(chunks, strings.Join(currentChunk, "\n"))
	}

	return chunks
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

	chunks := di.chunkText(string(content))
	return di.indexChunks(ctx, source, chunks, metadata)
}

// SearchDocuments searches for documents matching the query
func (di *DocumentIndexer) SearchDocuments(ctx context.Context, query string, k int, opts ...QueryOption) ([]Document, error) {
	results, err := di.collection.Query(ctx, query, k, opts...)
	if err != nil {
		return nil, err
	}

	docs := make([]Document, len(results))
	for i, result := range results {
		docs[i] = result.Document
	}

	return docs, nil
}

// GetCollection returns the underlying collection
func (di *DocumentIndexer) GetCollection() Collection {
	return di.collection
}

// getLanguageFromExt returns the programming language based on file extension
func getLanguageFromExt(ext string) string {
	langMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".java": "java",
		".cpp":  "cpp",
		".c":    "c",
		".rs":   "rust",
		".rb":   "ruby",
		".php":  "php",
		".swift": "swift",
		".kt":   "kotlin",
		".scala": "scala",
		".r":    "r",
	}

	if lang, ok := langMap[strings.ToLower(ext)]; ok {
		return lang
	}
	return "unknown"
}