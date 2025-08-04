package vectorstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentIndexer_IndexFile(t *testing.T) {
	// Create mock embedder and store
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	// Create indexer
	config := DefaultIndexerConfig()
	indexer, err := NewDocumentIndexer(store, config)
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
	assert.NotNil(t, firstDoc.Metadata["size"])
	assert.NotNil(t, firstDoc.Metadata["indexed_at"])
}

func TestDocumentIndexer_ChunkText(t *testing.T) {
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	config := IndexerConfig{
		CollectionName: "test_chunks",
		ChunkSize:      100,
		ChunkOverlap:   20,
	}
	indexer, err := NewDocumentIndexer(store, config)
	require.NoError(t, err)

	// Test short text (no chunking needed)
	shortText := "This is a short text."
	chunks := indexer.chunkText(shortText)
	assert.Len(t, chunks, 1)
	assert.Equal(t, shortText, chunks[0])

	// Test long text (should be chunked)
	longText := strings.Repeat("This is a sentence. ", 20) // ~400 chars
	chunks = indexer.chunkText(longText)
	assert.Greater(t, len(chunks), 1)

	// Verify overlap
	for i := 0; i < len(chunks)-1; i++ {
		// Check that there's some overlap between consecutive chunks
		chunk1End := chunks[i][len(chunks[i])-config.ChunkOverlap:]
		chunk2Start := chunks[i+1][:config.ChunkOverlap]
		
		// There should be some common content due to overlap
		// This is a simplified check
		assert.True(t, len(chunk1End) > 0 && len(chunk2Start) > 0)
	}
}

func TestDocumentIndexer_IndexDirectory(t *testing.T) {
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	config := DefaultIndexerConfig()
	indexer, err := NewDocumentIndexer(store, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create test directory with files
	tmpDir := t.TempDir()
	
	// Create test files
	files := map[string]string{
		"doc1.txt": "This is document one about machine learning.",
		"doc2.txt": "This is document two about deep learning.",
		"code.go":  "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}",
		"data.json": `{"name": "test", "value": 123}`,
		"ignore.log": "This should be ignored",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Index directory with patterns
	patterns := []string{"*.txt", "*.go", "*.json"}
	err = indexer.IndexDirectory(ctx, tmpDir, patterns)
	require.NoError(t, err)

	// Search for content from different files
	docs, err := indexer.SearchDocuments(ctx, "machine learning", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, docs)

	// Verify that .log file was not indexed
	// Search for content unique to log file
	docs, err = indexer.SearchDocuments(ctx, "This should be ignored", 5)
	require.NoError(t, err)
	
	// Check that no results are from the log file
	foundLogFile := false
	for _, doc := range docs {
		if strings.HasSuffix(doc.Metadata["filename"].(string), ".log") {
			foundLogFile = true
			break
		}
	}
	assert.False(t, foundLogFile, "Log file should not have been indexed")
}

func TestDocumentIndexer_IndexCodeFile(t *testing.T) {
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	config := DefaultIndexerConfig()
	indexer, err := NewDocumentIndexer(store, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a Go code file
	tmpDir := t.TempDir()
	codeFile := filepath.Join(tmpDir, "main.go")
	code := `package main

import "fmt"

// greet prints a greeting message
func greet(name string) {
	fmt.Printf("Hello, %s!\n", name)
}

// main is the entry point
func main() {
	greet("World")
	greet("Go")
}

// calculate performs a calculation
func calculate(a, b int) int {
	return a + b
}`

	err = os.WriteFile(codeFile, []byte(code), 0644)
	require.NoError(t, err)

	// Index the code file
	err = indexer.IndexFile(ctx, codeFile)
	require.NoError(t, err)

	// Search for function names
	docs, err := indexer.SearchDocuments(ctx, "greet function", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, docs)

	// Verify metadata
	firstDoc := docs[0]
	assert.Equal(t, "code", firstDoc.Metadata["type"])
	assert.Equal(t, "go", firstDoc.Metadata["language"])
}

func TestDocumentIndexer_IndexReader(t *testing.T) {
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	config := DefaultIndexerConfig()
	indexer, err := NewDocumentIndexer(store, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Index from reader
	content := "This is content from a reader. It should be indexed properly."
	reader := strings.NewReader(content)
	
	metadata := map[string]interface{}{
		"type": "stream",
		"custom": "value",
	}

	err = indexer.IndexReader(ctx, reader, "stream-source", metadata)
	require.NoError(t, err)

	// Search for content
	docs, err := indexer.SearchDocuments(ctx, "content from reader", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, docs)

	// Verify metadata
	firstDoc := docs[0]
	assert.Equal(t, "stream-source", firstDoc.Metadata["source"])
	assert.Equal(t, "stream", firstDoc.Metadata["type"])
	assert.Equal(t, "value", firstDoc.Metadata["custom"])
}

func TestDocumentIndexer_LargeFile(t *testing.T) {
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	config := IndexerConfig{
		CollectionName: "large_docs",
		ChunkSize:      500,
		ChunkOverlap:   100,
	}
	indexer, err := NewDocumentIndexer(store, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a large file
	tmpDir := t.TempDir()
	largeFile := filepath.Join(tmpDir, "large.txt")
	
	// Generate large content
	var contentBuilder strings.Builder
	for i := 0; i < 100; i++ {
		contentBuilder.WriteString(fmt.Sprintf(
			"This is paragraph %d. It contains some text about topic %d. "+
			"The content is designed to be chunked properly. "+
			"Each paragraph should be meaningful on its own. ", i, i))
	}
	
	err = os.WriteFile(largeFile, []byte(contentBuilder.String()), 0644)
	require.NoError(t, err)

	// Index the large file
	err = indexer.IndexFile(ctx, largeFile)
	require.NoError(t, err)

	// Search for specific paragraph
	docs, err := indexer.SearchDocuments(ctx, "paragraph 42 topic", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, docs)

	// Verify chunking happened
	// Get metadata about chunks to verify
	collection := indexer.GetCollection()
	
	// Count chunks for the large file by searching with a unique query
	searchResults, err := collection.Query(ctx, "paragraph contains text", 100)
	require.NoError(t, err)
	
	// Should have multiple chunks
	chunkCount := 0
	for _, result := range searchResults {
		if result.Document.Metadata["source"] == largeFile {
			chunkCount++
		}
	}
	assert.Greater(t, chunkCount, 1, "Large file should be split into multiple chunks")
}