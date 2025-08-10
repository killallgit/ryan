package embeddings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOllamaEmbedderConfig(t *testing.T) {
	t.Run("requires endpoint in config", func(t *testing.T) {
		// Create embedder with empty config
		config := OllamaConfig{}
		embedder, err := NewOllamaEmbedder(config)

		// We expect an error because endpoint is required
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint not provided in config")
		assert.Nil(t, embedder)
	})

	t.Run("uses provided config endpoint", func(t *testing.T) {
		// Create embedder with explicit config
		configEndpoint := "http://config-server:9090"
		config := OllamaConfig{
			Endpoint: configEndpoint,
		}
		embedder, err := NewOllamaEmbedder(config)

		// We expect an error because config server doesn't exist
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get embedding dimensions")

		// Embedder will be nil, but the error shows it tried the config endpoint
		assert.Nil(t, embedder)
	})
}
