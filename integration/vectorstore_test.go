package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVectorStoreIntegration(t *testing.T) {
	// Skip if not in integration mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}

	// Create temporary directory for persistence
	tempDir := t.TempDir()

	// Test with mock embedder for consistent results
	embedder := vectorstore.NewMockEmbedder(384)

	t.Run("InMemoryStore", func(t *testing.T) {
		store, err := vectorstore.NewChromemStore(embedder, "", false)
		require.NoError(t, err)
		defer store.Close()

		runVectorStoreTests(t, store)
	})

	t.Run("PersistentStore", func(t *testing.T) {
		persistDir := filepath.Join(tempDir, "persist")
		store, err := vectorstore.NewChromemStore(embedder, persistDir, true)
		require.NoError(t, err)
		defer store.Close()

		runVectorStoreTests(t, store)

		// Test persistence by creating a new store with same directory
		t.Run("PersistenceAcrossRestarts", func(t *testing.T) {
			// Add some data
			col, err := store.CreateCollection("persistence-test", nil)
			require.NoError(t, err)

			docs := []vectorstore.Document{
				{ID: "1", Content: "First document"},
				{ID: "2", Content: "Second document"},
			}
			err = col.AddDocuments(context.Background(), docs)
			require.NoError(t, err)

			// Close the store
			store.Close()

			// Create new store with same directory
			store2, err := vectorstore.NewChromemStore(embedder, persistDir, true)
			require.NoError(t, err)
			defer store2.Close()

			// Check if collection exists
			col2, err := store2.GetCollection("persistence-test")
			require.NoError(t, err)

			// Verify documents are still there
			count, err := col2.Count()
			require.NoError(t, err)
			assert.Equal(t, 2, count)

			// Query to verify content
			results, err := col2.Query(context.Background(), "First", 1)
			require.NoError(t, err)
			assert.Len(t, results, 1)
			assert.Equal(t, "1", results[0].Document.ID)
		})
	})
}

func runVectorStoreTests(t *testing.T, store vectorstore.VectorStore) {
	ctx := context.Background()

	t.Run("CollectionOperations", func(t *testing.T) {
		// Create collection
		col, err := store.CreateCollection("test-collection", map[string]interface{}{
			"description": "Test collection",
			"created_at":  time.Now().Format(time.RFC3339),
		})
		require.NoError(t, err)
		assert.NotNil(t, col)
		assert.Equal(t, "test-collection", col.Name())

		// Try to create duplicate collection (should fail)
		_, err = store.CreateCollection("test-collection", nil)
		assert.Error(t, err)

		// Get collection
		col2, err := store.GetCollection("test-collection")
		require.NoError(t, err)
		assert.Equal(t, col.Name(), col2.Name())

		// List collections
		names, err := store.ListCollections()
		require.NoError(t, err)
		assert.Contains(t, names, "test-collection")

		// Delete collection
		err = store.DeleteCollection("test-collection")
		require.NoError(t, err)

		// Verify deletion
		_, err = store.GetCollection("test-collection")
		assert.Error(t, err)
	})

	t.Run("DocumentOperations", func(t *testing.T) {
		// Create collection
		col, err := store.CreateCollection("doc-test", nil)
		require.NoError(t, err)

		// Add documents
		docs := []vectorstore.Document{
			{
				ID:      "doc1",
				Content: "The quick brown fox jumps over the lazy dog",
				Metadata: map[string]interface{}{
					"type":   "sentence",
					"source": "test",
				},
			},
			{
				ID:      "doc2",
				Content: "Machine learning is a subset of artificial intelligence",
				Metadata: map[string]interface{}{
					"type":   "definition",
					"source": "test",
				},
			},
			{
				ID:      "doc3",
				Content: "Vector databases enable semantic search capabilities",
				Metadata: map[string]interface{}{
					"type":   "definition",
					"source": "test",
				},
			},
		}

		err = col.AddDocuments(ctx, docs)
		require.NoError(t, err)

		// Count documents
		count, err := col.Count()
		require.NoError(t, err)
		assert.Equal(t, 3, count)

		// Query documents
		results, err := col.Query(ctx, "artificial intelligence machine learning", 2)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		// First result should be the ML document
		assert.Equal(t, "doc2", results[0].Document.ID)

		// Query with filter
		results, err = col.Query(ctx, "search", 10, vectorstore.WithFilter(map[string]interface{}{
			"type": "definition",
		}))
		require.NoError(t, err)
		// Should only return definition documents
		for _, r := range results {
			assert.Equal(t, "definition", r.Document.Metadata["type"])
		}

		// Delete document
		err = col.Delete(ctx, []string{"doc1"})
		require.NoError(t, err)

		// Verify deletion
		count, err = col.Count()
		require.NoError(t, err)
		assert.Equal(t, 2, count)

		// Clear collection
		err = col.Clear(ctx)
		require.NoError(t, err)

		// Verify clear
		count, err = col.Count()
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		// Cleanup
		err = store.DeleteCollection("doc-test")
		require.NoError(t, err)
	})

	t.Run("SimilaritySearch", func(t *testing.T) {
		col, err := store.CreateCollection("similarity-test", nil)
		require.NoError(t, err)

		// Add documents with varying similarity
		docs := []vectorstore.Document{
			{ID: "1", Content: "cats and dogs are pets"},
			{ID: "2", Content: "dogs are loyal animals"},
			{ID: "3", Content: "cats are independent creatures"},
			{ID: "4", Content: "programming in Go is efficient"},
			{ID: "5", Content: "vector databases store embeddings"},
		}

		err = col.AddDocuments(ctx, docs)
		require.NoError(t, err)

		// Search for pet-related content
		results, err := col.Query(ctx, "pets and animals", 3)
		require.NoError(t, err)
		assert.Len(t, results, 3)

		// At least 2 of top 3 should be pet-related (allowing for some variance in mock embedder)
		petRelated := 0
		for _, r := range results {
			id := r.Document.ID
			if id == "1" || id == "2" || id == "3" {
				petRelated++
			}
		}
		assert.GreaterOrEqual(t, petRelated, 2, "At least 2 of top 3 results should be pet-related")

		// Test with minimum score threshold
		results, err = col.Query(ctx, "pets", 10, vectorstore.WithMinScore(0.5))
		require.NoError(t, err)
		// Should filter out low-scoring results
		for _, r := range results {
			assert.GreaterOrEqual(t, r.Score, float32(0.5))
		}

		// Cleanup
		err = store.DeleteCollection("similarity-test")
		require.NoError(t, err)
	})

	t.Run("EmbeddingQuery", func(t *testing.T) {
		col, err := store.CreateCollection("embedding-test", nil)
		require.NoError(t, err)

		// Add document
		doc := vectorstore.Document{
			ID:      "test-doc",
			Content: "This is a test document for embedding queries",
		}
		err = col.AddDocuments(ctx, []vectorstore.Document{doc})
		require.NoError(t, err)

		// Get embedding for query
		embedder := vectorstore.NewMockEmbedder(384)
		queryEmbedding, err := embedder.EmbedText(ctx, "test document queries")
		require.NoError(t, err)

		// Query with embedding
		results, err := col.QueryWithEmbedding(ctx, queryEmbedding, 1)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "test-doc", results[0].Document.ID)

		// Cleanup
		err = store.DeleteCollection("embedding-test")
		require.NoError(t, err)
	})
}

func TestVectorStoreManager(t *testing.T) {
	// Skip if not in integration mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}

	tempDir := t.TempDir()

	config := vectorstore.Config{
		Provider:          "chromem",
		PersistenceDir:    tempDir,
		EnablePersistence: true,
		Collections: []vectorstore.CollectionConfig{
			{Name: "test-conversations", Metadata: map[string]interface{}{"type": "conversation"}},
			{Name: "test-documents", Metadata: map[string]interface{}{"type": "document"}},
		},
		EmbedderConfig: vectorstore.EmbedderConfig{
			Provider: "mock",
		},
	}

	manager, err := vectorstore.NewManager(config)
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	t.Run("DefaultCollections", func(t *testing.T) {
		// Check that default collections were created
		names, err := manager.ListCollections()
		require.NoError(t, err)
		assert.Contains(t, names, "test-conversations")
		assert.Contains(t, names, "test-documents")
	})

	t.Run("IndexAndSearch", func(t *testing.T) {
		// Index documents
		docs := []vectorstore.Document{
			{ID: "1", Content: "How to implement vector search in Go"},
			{ID: "2", Content: "Building semantic search with embeddings"},
			{ID: "3", Content: "LangChain integration patterns"},
		}

		err := manager.IndexDocuments(ctx, "test-documents", docs)
		require.NoError(t, err)

		// Search
		results, err := manager.Search(ctx, "test-documents", "vector embeddings Go", 2)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("CollectionInfo", func(t *testing.T) {
		info, err := manager.GetCollectionInfo("test-documents")
		require.NoError(t, err)
		assert.Equal(t, "test-documents", info.Name)
		assert.Equal(t, 3, info.DocumentCount) // From previous test
	})

	t.Run("ClearCollection", func(t *testing.T) {
		err := manager.ClearCollection(ctx, "test-documents")
		require.NoError(t, err)

		info, err := manager.GetCollectionInfo("test-documents")
		require.NoError(t, err)
		assert.Equal(t, 0, info.DocumentCount)
	})
}

// TestRealEmbeddings tests with actual embedding providers (requires services to be running)
func TestRealEmbeddings(t *testing.T) {
	// Skip unless explicitly enabled
	if os.Getenv("TEST_REAL_EMBEDDINGS") != "true" {
		t.Skip("Skipping real embeddings test. Set TEST_REAL_EMBEDDINGS=true to run.")
	}

	ctx := context.Background()
	tempDir := t.TempDir()

	testCases := []struct {
		name     string
		provider string
		model    string
		baseURL  string
		apiKey   string
		skip     bool
	}{
		{
			name:     "Ollama",
			provider: "ollama",
			model:    "nomic-embed-text",
			baseURL:  "http://localhost:11434",
			skip:     os.Getenv("OLLAMA_BASE_URL") == "",
		},
		{
			name:     "OpenAI",
			provider: "openai",
			model:    "text-embedding-3-small",
			apiKey:   os.Getenv("OPENAI_API_KEY"),
			skip:     os.Getenv("OPENAI_API_KEY") == "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skipf("Skipping %s test. Configure environment to run.", tc.name)
			}

			config := vectorstore.Config{
				Provider:          "chromem",
				PersistenceDir:    filepath.Join(tempDir, tc.name),
				EnablePersistence: false,
				EmbedderConfig: vectorstore.EmbedderConfig{
					Provider: tc.provider,
					Model:    tc.model,
					BaseURL:  tc.baseURL,
					APIKey:   tc.apiKey,
				},
			}

			embedder, err := vectorstore.CreateEmbedder(config.EmbedderConfig)
			require.NoError(t, err)

			// Test embedding generation
			embedding, err := embedder.EmbedText(ctx, "Test embedding generation")
			require.NoError(t, err)
			assert.NotEmpty(t, embedding)
			assert.Greater(t, len(embedding), 0)

			// Test dimensions
			dims := embedder.Dimensions()
			assert.Equal(t, len(embedding), dims)

			// Create store and test search
			store, err := vectorstore.NewChromemStore(embedder, "", false)
			require.NoError(t, err)
			defer store.Close()

			col, err := store.CreateCollection("embedding-test", nil)
			require.NoError(t, err)

			// Add test documents
			docs := []vectorstore.Document{
				{ID: "1", Content: "Machine learning and artificial intelligence"},
				{ID: "2", Content: "Natural language processing with transformers"},
				{ID: "3", Content: "Computer vision and image recognition"},
			}

			err = col.AddDocuments(ctx, docs)
			require.NoError(t, err)

			// Search
			results, err := col.Query(ctx, "NLP and language models", 2)
			require.NoError(t, err)
			assert.Len(t, results, 2)

			// The NLP document should rank high
			found := false
			for _, r := range results {
				if r.Document.ID == "2" {
					found = true
					break
				}
			}
			assert.True(t, found, "NLP document should be in top results")
		})
	}
}
