package vectorstore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChromemStore_ListCollections(t *testing.T) {
	mockEmbedder := NewMockEmbedder(384)
	store, err := NewChromemStore(mockEmbedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	// Initially should be empty
	collections, err := store.ListCollections()
	require.NoError(t, err)
	assert.Empty(t, collections)

	// Create a collection
	_, err = store.CreateCollection("test-collection", nil)
	require.NoError(t, err)

	// Should now have one collection
	collections, err = store.ListCollections()
	require.NoError(t, err)
	assert.Len(t, collections, 1)
	assert.Contains(t, collections, "test-collection")
}

func TestChromemStore_DeleteCollection(t *testing.T) {
	mockEmbedder := NewMockEmbedder(384)
	store, err := NewChromemStore(mockEmbedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	// Create a collection
	_, err = store.CreateCollection("test-collection", nil)
	require.NoError(t, err)

	// Verify it exists
	collections, err := store.ListCollections()
	require.NoError(t, err)
	assert.Len(t, collections, 1)

	// Delete the collection
	err = store.DeleteCollection("test-collection")
	require.NoError(t, err)

	// Verify it's gone
	collections, err = store.ListCollections()
	require.NoError(t, err)
	assert.Empty(t, collections)

	// Try to delete non-existent collection (may or may not error depending on implementation)
	err = store.DeleteCollection("non-existent")
	if err != nil {
		// If error is returned, check it's wrapped in our Error type
		var vectorErr *Error
		assert.ErrorAs(t, err, &vectorErr)
		assert.Equal(t, ErrCodeCollectionNotFound, vectorErr.Code)
	}
}

func TestChromemCollection_Name(t *testing.T) {
	mockEmbedder := NewMockEmbedder(384)
	store, err := NewChromemStore(mockEmbedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	collectionName := "test-collection"
	collection, err := store.CreateCollection(collectionName, nil)
	require.NoError(t, err)

	assert.Equal(t, collectionName, collection.Name())
}

func TestChromemCollection_QueryWithEmbedding(t *testing.T) {
	mockEmbedder := NewMockEmbedder(384)
	store, err := NewChromemStore(mockEmbedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	collection, err := store.CreateCollection("test-collection", nil)
	require.NoError(t, err)

	// Add some test documents first
	docs := []Document{
		{ID: "doc1", Content: "This is a test document", Metadata: map[string]interface{}{"type": "test"}},
		{ID: "doc2", Content: "Another test document", Metadata: map[string]interface{}{"type": "test"}},
	}
	err = collection.AddDocuments(context.Background(), docs)
	require.NoError(t, err)

	// Create a query embedding (mock embedder creates predictable embeddings)
	// Create a query embedding with the correct dimensions (384)
	queryEmbedding := make([]float32, 384)
	for i := range queryEmbedding {
		queryEmbedding[i] = 0.1 // Simple test embedding with correct dimensions
	}

	// Query with embedding
	results, err := collection.QueryWithEmbedding(context.Background(), queryEmbedding, 5)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// Results should have the expected structure
	for _, result := range results {
		assert.NotEmpty(t, result.Document.ID)
		assert.NotEmpty(t, result.Document.Content)
		assert.GreaterOrEqual(t, result.Score, float32(0.0))
		assert.LessOrEqual(t, result.Score, float32(1.0))
	}
}

func TestChromemCollection_Delete(t *testing.T) {
	mockEmbedder := NewMockEmbedder(384)
	store, err := NewChromemStore(mockEmbedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	collection, err := store.CreateCollection("test-collection", nil)
	require.NoError(t, err)

	// Add a test document
	docs := []Document{
		{ID: "doc1", Content: "This is a test document", Metadata: map[string]interface{}{"type": "test"}},
	}
	err = collection.AddDocuments(context.Background(), docs)
	require.NoError(t, err)

	// Verify document exists by counting
	count, err := collection.Count()
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Delete the document
	err = collection.Delete(context.Background(), []string{"doc1"})
	require.NoError(t, err)

	// Verify document is gone
	count, err = collection.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestChromemCollection_Count(t *testing.T) {
	mockEmbedder := NewMockEmbedder(384)
	store, err := NewChromemStore(mockEmbedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	collection, err := store.CreateCollection("test-collection", nil)
	require.NoError(t, err)

	// Initially should be zero
	count, err := collection.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Add some documents
	docs := []Document{
		{ID: "doc1", Content: "Document 1", Metadata: map[string]interface{}{"type": "test"}},
		{ID: "doc2", Content: "Document 2", Metadata: map[string]interface{}{"type": "test"}},
		{ID: "doc3", Content: "Document 3", Metadata: map[string]interface{}{"type": "test"}},
	}
	err = collection.AddDocuments(context.Background(), docs)
	require.NoError(t, err)

	// Should now be 3
	count, err = collection.Count()
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestChromemCollection_Clear(t *testing.T) {
	mockEmbedder := NewMockEmbedder(384)
	store, err := NewChromemStore(mockEmbedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	collection, err := store.CreateCollection("test-collection", nil)
	require.NoError(t, err)

	// Add some documents
	docs := []Document{
		{ID: "doc1", Content: "Document 1", Metadata: map[string]interface{}{"type": "test"}},
		{ID: "doc2", Content: "Document 2", Metadata: map[string]interface{}{"type": "test"}},
	}
	err = collection.AddDocuments(context.Background(), docs)
	require.NoError(t, err)

	// Verify documents exist
	count, err := collection.Count()
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Clear the collection
	err = collection.Clear(context.Background())
	require.NoError(t, err)

	// Verify collection is empty
	count, err = collection.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestBuildChromemWhere(t *testing.T) {
	tests := []struct {
		name     string
		filter   map[string]interface{}
		expected map[string]string
	}{
		{
			name:     "empty filter",
			filter:   map[string]interface{}{},
			expected: nil,
		},
		{
			name:     "nil filter",
			filter:   nil,
			expected: nil,
		},
		{
			name: "string values",
			filter: map[string]interface{}{
				"type":   "document",
				"status": "active",
			},
			expected: map[string]string{
				"type":   "document",
				"status": "active",
			},
		},
		{
			name: "mixed types",
			filter: map[string]interface{}{
				"type":   "document",
				"count":  42,
				"active": true,
			},
			expected: map[string]string{
				"type":   "document",
				"count":  "42",
				"active": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to create a ChromemCollection to call the method
			mockEmbedder := NewMockEmbedder(384)
			store, err := NewChromemStore(mockEmbedder, "", false)
			require.NoError(t, err)
			defer store.Close()

			// Test buildChromemWhere as a standalone function
			result := buildChromemWhere(&queryOptions{filter: tt.filter})

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
