package vectorstore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentIndexer_IndexFile(t *testing.T) {
	// Create manager with mock embedder
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	config := Config{
		Provider:          "chromem",
		EnablePersistence: false,
		ChunkSize:         1000,
		ChunkOverlap:      200,
		EmbedderConfig: EmbedderConfig{
			Provider: "mock",
		},
	}

	processor := NewDocumentProcessor(embedder, config.ChunkSize, config.ChunkOverlap)
	log := logger.WithComponent("test")
	manager := &Manager{
		store:     store,
		embedder:  embedder,
		processor: processor,
		config:    config,
		log:       log,
	}
	// Initialize collections for testing
	_, err = GetOrCreateCollection(store, "documents", nil)
	require.NoError(t, err)

	// Create indexer
	indexerConfig := DefaultIndexerConfig()
	indexer, err := NewDocumentIndexer(manager, indexerConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "This is a test document. It contains some text that should be indexed. " +
		"The indexer should be able to find this content when we search for it."

	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Index the file
	err = indexer.IndexFile(ctx, testFile)
	require.NoError(t, err)

	// Search for content
	docs, err := indexer.SearchDocuments(ctx, "test document indexed", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, docs)

	// Verify metadata
	firstDoc := docs[0]
	assert.Equal(t, testFile, firstDoc.Metadata["source"])
	assert.Equal(t, "test.txt", firstDoc.Metadata["filename"])
	assert.Equal(t, ".txt", firstDoc.Metadata["extension"])
	assert.Equal(t, "text", firstDoc.Metadata["type"])
}

func TestDocumentIndexer_ChunkText(t *testing.T) {
	// Create manager
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	config := Config{
		ChunkSize:    100,
		ChunkOverlap: 20,
	}

	processor := NewDocumentProcessor(embedder, config.ChunkSize, config.ChunkOverlap)
	log := logger.WithComponent("test")
	manager := &Manager{
		store:     store,
		embedder:  embedder,
		processor: processor,
		config:    config,
		log:       log,
	}
	// Initialize collections for testing
	_, err = GetOrCreateCollection(store, "documents", nil)
	require.NoError(t, err)

	// Test text chunking
	longText := strings.Repeat("This is a test sentence. ", 10)
	chunks := manager.GetDocumentProcessor().ChunkText(longText)

	assert.Greater(t, len(chunks), 1, "Long text should be split into multiple chunks")

	// Test overlap
	if len(chunks) > 1 {
		assert.True(t, len(chunks[0]) > config.ChunkOverlap)
		assert.True(t, len(chunks[1]) > config.ChunkOverlap)
	}
}

func TestDocumentIndexer_IndexDirectory(t *testing.T) {
	// Create manager
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	config := Config{
		Provider:          "chromem",
		EnablePersistence: false,
		ChunkSize:         1000,
		ChunkOverlap:      200,
		EmbedderConfig: EmbedderConfig{
			Provider: "mock",
		},
	}

	processor := NewDocumentProcessor(embedder, config.ChunkSize, config.ChunkOverlap)
	log := logger.WithComponent("test")
	manager := &Manager{
		store:     store,
		embedder:  embedder,
		processor: processor,
		config:    config,
		log:       log,
	}
	// Initialize collections for testing
	_, err = GetOrCreateCollection(store, "documents", nil)
	require.NoError(t, err)

	// Create indexer
	indexerConfig := DefaultIndexerConfig()
	indexer, err := NewDocumentIndexer(manager, indexerConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// Create temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"file1.txt":  "This is the first test file.",
		"file2.md":   "# This is a markdown file\n\nWith some content.",
		"file3.go":   "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}",
		"ignore.pdf": "This should be ignored",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		err = os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Index directory with pattern filter
	patterns := []string{"*.txt", "*.md", "*.go"}
	err = indexer.IndexDirectory(ctx, tmpDir, patterns)
	require.NoError(t, err)

	// Search for content
	docs, err := indexer.SearchDocuments(ctx, "test file", 10)
	require.NoError(t, err)

	// Should find txt and md files, not PDF
	foundFiles := make(map[string]bool)
	for _, doc := range docs {
		if source, ok := doc.Metadata["source"].(string); ok {
			foundFiles[filepath.Base(source)] = true
		}
	}

	assert.True(t, foundFiles["file1.txt"] || foundFiles["file2.md"], "Should find at least one text file")
	assert.False(t, foundFiles["ignore.pdf"], "Should not index PDF file")
}

func TestDocumentIndexer_IndexReader(t *testing.T) {
	// Create manager
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	config := Config{
		Provider:          "chromem",
		EnablePersistence: false,
		ChunkSize:         1000,
		ChunkOverlap:      200,
		EmbedderConfig: EmbedderConfig{
			Provider: "mock",
		},
	}

	processor := NewDocumentProcessor(embedder, config.ChunkSize, config.ChunkOverlap)
	log := logger.WithComponent("test")
	manager := &Manager{
		store:     store,
		embedder:  embedder,
		processor: processor,
		config:    config,
		log:       log,
	}
	// Initialize collections for testing
	_, err = GetOrCreateCollection(store, "documents", nil)
	require.NoError(t, err)

	// Create indexer
	indexerConfig := DefaultIndexerConfig()
	indexer, err := NewDocumentIndexer(manager, indexerConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// Test content
	content := "This is content from a reader. It should be indexed properly."
	reader := strings.NewReader(content)

	// Index from reader
	metadata := map[string]interface{}{
		"type": "reader_test",
	}
	err = indexer.IndexReader(ctx, reader, "test_source", metadata)
	require.NoError(t, err)

	// Search for content
	docs, err := indexer.SearchDocuments(ctx, "reader indexed", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, docs)

	// Verify metadata
	firstDoc := docs[0]
	assert.Equal(t, "test_source", firstDoc.Metadata["source"])
	assert.Equal(t, "reader_test", firstDoc.Metadata["type"])
}

func TestDocumentIndexer_CodeFileIndexing(t *testing.T) {
	// Create manager
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	config := Config{
		Provider:          "chromem",
		EnablePersistence: false,
		ChunkSize:         1000,
		ChunkOverlap:      200,
		EmbedderConfig: EmbedderConfig{
			Provider: "mock",
		},
	}

	processor := NewDocumentProcessor(embedder, config.ChunkSize, config.ChunkOverlap)
	log := logger.WithComponent("test")
	manager := &Manager{
		store:     store,
		embedder:  embedder,
		processor: processor,
		config:    config,
		log:       log,
	}
	// Initialize collections for testing
	_, err = GetOrCreateCollection(store, "documents", nil)
	require.NoError(t, err)

	// Create indexer
	indexerConfig := DefaultIndexerConfig()
	indexer, err := NewDocumentIndexer(manager, indexerConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a temporary Go file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

import "fmt"

// TestFunction demonstrates a simple function
func TestFunction() {
	fmt.Println("Hello, World!")
}

func main() {
	TestFunction()
}`

	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Index the file
	err = indexer.IndexFile(ctx, testFile)
	require.NoError(t, err)

	// Search for content
	docs, err := indexer.SearchDocuments(ctx, "TestFunction", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, docs)

	// Verify it's marked as code
	found := false
	for _, doc := range docs {
		if doc.Metadata["type"] == "code" && doc.Metadata["language"] == "go" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find code document with correct type and language")
}

func TestDocumentIndexer_GetCollectionName(t *testing.T) {
	// Create manager
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	config := Config{
		Provider:          "chromem",
		EnablePersistence: false,
		ChunkSize:         1000,
		ChunkOverlap:      200,
		EmbedderConfig: EmbedderConfig{
			Provider: "mock",
		},
	}

	processor := NewDocumentProcessor(embedder, config.ChunkSize, config.ChunkOverlap)
	log := logger.WithComponent("test")
	manager := &Manager{
		store:     store,
		embedder:  embedder,
		processor: processor,
		config:    config,
		log:       log,
	}
	// Initialize collections for testing
	_, err = GetOrCreateCollection(store, "documents", nil)
	require.NoError(t, err)

	// Create indexer with custom collection name
	indexerConfig := IndexerConfig{
		CollectionName: "test_collection",
		ChunkSize:      500,
		ChunkOverlap:   100,
	}
	indexer, err := NewDocumentIndexer(manager, indexerConfig)
	require.NoError(t, err)

	assert.Equal(t, "test_collection", indexer.GetCollectionName())
}
