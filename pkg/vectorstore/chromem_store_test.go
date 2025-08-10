package vectorstore

import (
	"context"
	"testing"

	"github.com/killallgit/ryan/pkg/embeddings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChromemStore(t *testing.T) {
	// Create mock embedder
	mockEmbedder := embeddings.NewMockEmbedder(384)

	// Create chromem store
	config := ChromemConfig{
		CollectionName: "test",
		Embedder:       mockEmbedder,
	}

	store, err := NewChromemStore(config)
	require.NoError(t, err)
	require.NotNil(t, store)
	defer store.Close()

	ctx := context.Background()

	t.Run("AddAndRetrieveDocuments", func(t *testing.T) {
		// Create test documents
		docs := []Document{
			{
				ID:      "doc1",
				Content: "The quick brown fox jumps over the lazy dog",
				Metadata: map[string]interface{}{
					"source": "test1",
				},
			},
			{
				ID:      "doc2",
				Content: "Machine learning is a subset of artificial intelligence",
				Metadata: map[string]interface{}{
					"source": "test2",
				},
			},
			{
				ID:      "doc3",
				Content: "Go is a statically typed programming language",
				Metadata: map[string]interface{}{
					"source": "test3",
				},
			},
		}

		// Add documents
		err := store.AddDocuments(ctx, docs)
		assert.NoError(t, err)

		// Count documents
		count, err := store.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 3, count)

		// Search for similar documents
		results, err := store.SimilaritySearch(ctx, "artificial intelligence and ML", 2)
		assert.NoError(t, err)
		assert.Len(t, results, 2)

		// Check that we got 2 results (with mock embedder, order is not deterministic)
		resultIDs := make(map[string]bool)
		for _, r := range results {
			resultIDs[r.Document.ID] = true
		}
		// Should have 2 unique document IDs
		assert.Len(t, resultIDs, 2)
	})

	t.Run("SimilaritySearchWithScore", func(t *testing.T) {
		results, err := store.SimilaritySearchWithScore(ctx, "programming", 3, 0.0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)

		// Check that scores are populated
		for _, result := range results {
			assert.GreaterOrEqual(t, result.Score, float32(0.0))
			assert.LessOrEqual(t, result.Score, float32(1.0))
		}
	})

	t.Run("DeleteDocuments", func(t *testing.T) {
		// Delete a document
		err := store.DeleteDocuments(ctx, []string{"doc1"})
		assert.NoError(t, err)

		// Verify count decreased
		count, err := store.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("Clear", func(t *testing.T) {
		// Clear all documents
		err := store.Clear(ctx)
		assert.NoError(t, err)

		// Verify store is empty
		count, err := store.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestVectorStoreRetriever(t *testing.T) {
	// Create mock embedder and store
	mockEmbedder := embeddings.NewMockEmbedder(384)
	config := ChromemConfig{
		CollectionName: "test_retriever",
		Embedder:       mockEmbedder,
	}

	store, err := NewChromemStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add test documents
	docs := []Document{
		{
			ID:      "doc1",
			Content: "Python is a popular programming language for data science",
		},
		{
			ID:      "doc2",
			Content: "JavaScript is widely used for web development",
		},
		{
			ID:      "doc3",
			Content: "Go is known for its concurrency features",
		},
	}

	err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Create retriever
	retriever := NewVectorStoreRetriever(store, RetrieverConfig{
		K:              2,
		ScoreThreshold: 0.0,
	})

	t.Run("GetRelevantDocuments", func(t *testing.T) {
		results, err := retriever.GetRelevantDocuments(ctx, "web programming")
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("GetRelevantDocumentsWithScore", func(t *testing.T) {
		results, err := retriever.GetRelevantDocumentsWithScore(ctx, "concurrent programming")
		assert.NoError(t, err)
		assert.Len(t, results, 2)

		// Check scores are present
		for _, result := range results {
			assert.NotEmpty(t, result.Document.Content)
			assert.GreaterOrEqual(t, result.Score, float32(0.0))
		}
	})
}
