package vectorstore

import (
	"fmt"
	"os"

	"github.com/killallgit/ryan/pkg/embeddings"
	"github.com/spf13/viper"
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

// LoadConfig loads vector store configuration from Viper
func LoadConfig() Config {
	return Config{
		Enabled:        viper.GetBool("vectorstore.enabled"),
		Provider:       viper.GetString("vectorstore.provider"),
		CollectionName: viper.GetString("vectorstore.collection.name"),
		Persistence: PersistenceConfig{
			Enabled: viper.GetBool("vectorstore.persistence.enabled"),
			Path:    viper.GetString("vectorstore.persistence.path"),
		},
		Embedding: embeddings.Config{
			Provider: viper.GetString("vectorstore.embedding.provider"),
			Model:    viper.GetString("vectorstore.embedding.model"),
			Endpoint: viper.GetString("vectorstore.embedding.endpoint"),
			APIKey:   viper.GetString("vectorstore.embedding.api_key"),
		},
		Retrieval: RetrieverConfig{
			K:              viper.GetInt("vectorstore.retrieval.k"),
			ScoreThreshold: float32(viper.GetFloat64("vectorstore.retrieval.score_threshold")),
		},
	}
}

// SetDefaults sets default configuration values
func SetDefaults() {
	viper.SetDefault("vectorstore.enabled", false)
	viper.SetDefault("vectorstore.provider", "chromem")
	viper.SetDefault("vectorstore.collection.name", "default")
	viper.SetDefault("vectorstore.persistence.enabled", false)
	viper.SetDefault("vectorstore.persistence.path", "./data/vectors")
	viper.SetDefault("vectorstore.embedding.provider", "ollama")
	viper.SetDefault("vectorstore.embedding.model", "nomic-embed-text")

	// Use OLLAMA_HOST environment variable if set
	if ollamaHost := os.Getenv("OLLAMA_HOST"); ollamaHost != "" {
		viper.SetDefault("vectorstore.embedding.endpoint", ollamaHost)
	}

	viper.SetDefault("vectorstore.retrieval.k", 4)
	viper.SetDefault("vectorstore.retrieval.score_threshold", 0.0)
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
