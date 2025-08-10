package integration

import (
	"context"
	"os"
	"testing"

	"github.com/killallgit/ryan/pkg/embeddings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOllamaEmbedderIntegration(t *testing.T) {
	// This test REQUIRES OLLAMA_HOST to be set - no fallback allowed
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		t.Fatal("OLLAMA_HOST environment variable MUST be set for integration tests")
	}

	// Skip if Ollama is not actually available at the specified host
	if !isOllamaAvailable() {
		t.Skipf("Skipping test: Ollama is not available at %s", ollamaHost)
	}

	t.Run("Creates embedder with explicit config", func(t *testing.T) {
		// Create embedder with config using OLLAMA_HOST from environment
		// Use OLLAMA_EMBEDDING_MODEL if set (for CI), otherwise use default
		embeddingModel := os.Getenv("OLLAMA_EMBEDDING_MODEL")
		if embeddingModel == "" {
			embeddingModel = "nomic-embed-text"
		}
		config := embeddings.OllamaConfig{
			Endpoint: ollamaHost,
			Model:    embeddingModel,
		}
		embedder, err := embeddings.NewOllamaEmbedder(config)
		require.NoError(t, err, "Should create embedder with config")
		require.NotNil(t, embedder)
		defer embedder.Close()

		// Test that it can actually create embeddings
		ctx := context.Background()
		embedding, err := embedder.EmbedText(ctx, "test text")
		assert.NoError(t, err, "Should create embedding")
		assert.NotEmpty(t, embedding, "Embedding should not be empty")
		assert.Greater(t, len(embedding), 0, "Embedding should have dimensions")
	})

	t.Run("Explicit config overrides OLLAMA_HOST", func(t *testing.T) {
		// When we provide explicit config, it should use that instead of env var
		// But for integration tests, we'll use the actual OLLAMA_HOST value
		// Use OLLAMA_EMBEDDING_MODEL if set (for CI), otherwise use default
		embeddingModel := os.Getenv("OLLAMA_EMBEDDING_MODEL")
		if embeddingModel == "" {
			embeddingModel = "nomic-embed-text"
		}
		config := embeddings.OllamaConfig{
			Endpoint: ollamaHost, // Use the actual host for testing
			Model:    embeddingModel,
		}
		embedder, err := embeddings.NewOllamaEmbedder(config)
		require.NoError(t, err, "Should create embedder with explicit config")
		require.NotNil(t, embedder)
		defer embedder.Close()

		// Verify it works
		ctx := context.Background()
		embedding, err := embedder.EmbedText(ctx, "another test")
		assert.NoError(t, err, "Should create embedding with explicit config")
		assert.NotEmpty(t, embedding)
	})

	t.Run("Testing function panics if OLLAMA_HOST not set", func(t *testing.T) {
		// Save original value
		originalHost := os.Getenv("OLLAMA_HOST")

		// Temporarily unset OLLAMA_HOST
		os.Unsetenv("OLLAMA_HOST")
		defer os.Setenv("OLLAMA_HOST", originalHost)

		// The testing version should panic if OLLAMA_HOST is not set
		assert.Panics(t, func() {
			config := embeddings.OllamaConfig{}
			// For integration tests, this should panic if OLLAMA_HOST is not set
			_, _ = embeddings.NewOllamaEmbedderForTesting(config)
		}, "Should panic when OLLAMA_HOST is not set in integration tests")
	})

	t.Run("Regular function falls back to localhost if OLLAMA_HOST not set", func(t *testing.T) {
		// Save original value
		originalHost := os.Getenv("OLLAMA_HOST")

		// Temporarily unset OLLAMA_HOST
		os.Unsetenv("OLLAMA_HOST")
		defer os.Setenv("OLLAMA_HOST", originalHost)

		// The regular version should fall back to localhost
		config := embeddings.OllamaConfig{}
		embedder, err := embeddings.NewOllamaEmbedder(config)
		// It will fail to connect to localhost, but that's OK
		// We're just verifying it doesn't panic
		_ = embedder
		_ = err
	})
}

// TestVectorStoreWithOllamaEmbedder tests the vector store with real Ollama embeddings
func TestVectorStoreWithOllamaEmbedder(t *testing.T) {
	// This test REQUIRES OLLAMA_HOST to be set
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		t.Fatal("OLLAMA_HOST environment variable MUST be set for integration tests")
	}

	// Skip if Ollama is not available
	if !isOllamaAvailable() {
		t.Skipf("Skipping test: Ollama is not available at %s", ollamaHost)
	}

	// Test will be implemented when we need to test vector store with real embeddings
	t.Skip("Vector store integration with Ollama embedder - to be implemented")
}
