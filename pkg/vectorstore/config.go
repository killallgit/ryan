package vectorstore

import (
	"fmt"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/embeddings"
)

// Config contains configuration for vector stores
type Config struct {
	// Enabled determines if vector store is enabled
	Enabled bool

	// Provider is the vector store provider (e.g., "chromem")
	Provider string

	// CollectionName is the name of the collection
	CollectionName string

	// Persistence configuration
	Persistence PersistenceConfig

	// Embedding configuration
	Embedding embeddings.Config

	// Retrieval configuration
	Retrieval RetrieverConfig
}

// PersistenceConfig contains persistence settings
type PersistenceConfig struct {
	// Enabled determines if persistence is enabled
	Enabled bool

	// Path is the directory path for persistence
	Path string
}

// LoadConfig loads vector store configuration from global config
func LoadConfig() Config {
	settings := config.Get()
	return Config{
		Enabled:        settings.VectorStore.Enabled,
		Provider:       settings.VectorStore.Provider,
		CollectionName: settings.VectorStore.Collection.Name,
		Persistence: PersistenceConfig{
			Enabled: settings.VectorStore.Persistence.Enabled,
			Path:    settings.VectorStore.Persistence.Path,
		},
		Embedding: embeddings.Config{
			Provider: settings.VectorStore.Embedding.Provider,
			Model:    settings.VectorStore.Embedding.Model,
			Endpoint: settings.VectorStore.Embedding.Endpoint,
			APIKey:   settings.VectorStore.Embedding.APIKey,
		},
		Retrieval: RetrieverConfig{
			K:              settings.VectorStore.Retrieval.K,
			ScoreThreshold: settings.VectorStore.Retrieval.ScoreThreshold,
		},
	}
}

// NewVectorStore creates a new vector store based on configuration
func NewVectorStore(config Config, embedder embeddings.Embedder) (VectorStore, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("vector store is not enabled")
	}

	switch config.Provider {
	case "chromem":
		chromemConfig := ChromemConfig{
			CollectionName:   config.CollectionName,
			PersistDirectory: "",
			Embedder:         embedder,
		}

		if config.Persistence.Enabled {
			chromemConfig.PersistDirectory = config.Persistence.Path
		}

		return NewChromemStore(chromemConfig)
	default:
		return nil, fmt.Errorf("unsupported vector store provider: %s", config.Provider)
	}
}
