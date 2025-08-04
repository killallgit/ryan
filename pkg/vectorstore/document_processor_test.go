package vectorstore

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentProcessor_ProcessDocument(t *testing.T) {
	embedder := NewMockEmbedder(384)
	processor := NewDocumentProcessor(embedder, 1000, 200)
	ctx := context.Background()

	// Test document without embedding
	doc := Document{
		ID:      "test-doc",
		Content: "This is a test document",
	}

	err := processor.ProcessDocument(ctx, &doc)
	require.NoError(t, err)

	// Should have generated embedding
	assert.NotEmpty(t, doc.Embedding)
	assert.Equal(t, 384, len(doc.Embedding))

	// Test document with existing embedding
	doc2 := Document{
		ID:        "test-doc-2",
		Content:   "Another test document",
		Embedding: []float32{0.1, 0.2, 0.3},
	}

	originalEmbedding := doc2.Embedding
	err = processor.ProcessDocument(ctx, &doc2)
	require.NoError(t, err)

	// Should keep existing embedding
	assert.Equal(t, originalEmbedding, doc2.Embedding)
}

func TestDocumentProcessor_ProcessDocuments(t *testing.T) {
	embedder := NewMockEmbedder(384)
	processor := NewDocumentProcessor(embedder, 1000, 200)
	ctx := context.Background()

	docs := []Document{
		{ID: "doc1", Content: "First document"},
		{ID: "doc2", Content: "Second document"},
		{ID: "doc3", Content: "Third document", Embedding: []float32{0.1, 0.2}},
	}

	err := processor.ProcessDocuments(ctx, docs)
	require.NoError(t, err)

	// First two should have generated embeddings
	assert.NotEmpty(t, docs[0].Embedding)
	assert.NotEmpty(t, docs[1].Embedding)

	// Third should keep existing embedding
	assert.Equal(t, []float32{0.1, 0.2}, docs[2].Embedding)
}

func TestDocumentProcessor_ChunkDocument(t *testing.T) {
	embedder := NewMockEmbedder(384)
	processor := NewDocumentProcessor(embedder, 100, 20)

	doc := Document{
		ID:      "test-doc",
		Content: strings.Repeat("This is a test sentence. ", 10),
		Metadata: map[string]interface{}{
			"original": "metadata",
		},
	}

	baseMetadata := map[string]interface{}{
		"source": "test_source",
	}

	chunks, err := processor.ChunkDocument(doc, baseMetadata)
	require.NoError(t, err)

	// Should create multiple chunks
	assert.Greater(t, len(chunks), 1)

	// Check first chunk
	firstChunk := chunks[0]
	assert.Equal(t, "test-doc_chunk_0", firstChunk.ID)
	assert.NotEmpty(t, firstChunk.Content)

	// Check metadata inheritance
	assert.Equal(t, "test_source", firstChunk.Metadata["source"])
	assert.Equal(t, "metadata", firstChunk.Metadata["original"])
	assert.Equal(t, 0, firstChunk.Metadata["chunk_index"])
	assert.Equal(t, len(chunks), firstChunk.Metadata["chunk_total"])
	assert.Equal(t, "test-doc", firstChunk.Metadata["parent_doc_id"])
}

func TestDocumentProcessor_ChunkText(t *testing.T) {
	embedder := NewMockEmbedder(384)
	processor := NewDocumentProcessor(embedder, 50, 10)

	// Test short text (no chunking needed)
	shortText := "Short text"
	chunks := processor.ChunkText(shortText)
	assert.Equal(t, 1, len(chunks))
	assert.Equal(t, shortText, chunks[0])

	// Test long text (requires chunking)
	longText := strings.Repeat("This is a sentence. ", 10)
	chunks = processor.ChunkText(longText)
	assert.Greater(t, len(chunks), 1)

	// Test overlap
	if len(chunks) > 1 {
		// Should have some content overlap between chunks
		firstChunk := chunks[0]
		secondChunk := chunks[1]
		assert.True(t, len(firstChunk) > 10) // Should be longer than overlap
		assert.True(t, len(secondChunk) > 10)
	}
}

func TestDocumentProcessor_ChunkCode(t *testing.T) {
	embedder := NewMockEmbedder(384)
	processor := NewDocumentProcessor(embedder, 100, 20)

	code := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	fmt.Println("This is a test")
	fmt.Println("Another line")
}

func helper() {
	fmt.Println("Helper function")
}`

	chunks := processor.ChunkCode(code)
	assert.Greater(t, len(chunks), 0)

	// Each chunk should contain valid code snippets
	for _, chunk := range chunks {
		assert.NotEmpty(t, strings.TrimSpace(chunk))
	}
}

func TestDocumentProcessor_Validation(t *testing.T) {
	embedder := NewMockEmbedder(384)
	processor := NewDocumentProcessor(embedder, 1000, 200)
	ctx := context.Background()

	// Test invalid document ID
	doc := Document{
		ID:      "", // Invalid empty ID
		Content: "Test content",
	}

	err := processor.ProcessDocument(ctx, &doc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document ID cannot be empty")
}

func TestDocumentProcessor_LargeBatch(t *testing.T) {
	embedder := NewMockEmbedder(384)
	processor := NewDocumentProcessor(embedder, 1000, 200)
	ctx := context.Background()

	// Create more documents than MaxBatchSize
	docs := make([]Document, MaxBatchSize+10)
	for i := range docs {
		docs[i] = Document{
			ID:      fmt.Sprintf("doc-%d", i),
			Content: fmt.Sprintf("Document %d content", i),
		}
	}

	err := processor.ProcessDocuments(ctx, docs)
	require.NoError(t, err)

	// All documents should have embeddings
	for _, doc := range docs {
		assert.NotEmpty(t, doc.Embedding)
	}
}
