package vectorstore

import (
	"fmt"
	"os"

	"github.com/killallgit/ryan/pkg/config"
)

// InitializeVectorStore initializes the vector store from global configuration
func InitializeVectorStore() (*Manager, error) {
	cfg := config.Get()
	if cfg == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}

	if !cfg.VectorStore.Enabled {
		return nil, nil // Vector store is disabled
	}

	// Create embedder config
	embedderConfig := EmbedderConfig{
		Provider: cfg.VectorStore.Embedder.Provider,
		Model:    cfg.VectorStore.Embedder.Model,
		BaseURL:  cfg.VectorStore.Embedder.BaseURL,
		APIKey:   cfg.VectorStore.Embedder.APIKey,
	}

	// Handle API key from environment for security
	if embedderConfig.Provider == "openai" && embedderConfig.APIKey == "" {
		embedderConfig.APIKey = os.Getenv("OPENAI_API_KEY")
	}

	// Create embedder with fallback to mock
	embedder, err := CreateEmbedder(embedderConfig)
	if err != nil {
		// Log the error and fall back to mock embedder for development/debugging
		fmt.Printf("Warning: Failed to create %s embedder (%v), falling back to mock embedder\n", embedderConfig.Provider, err)
		mockConfig := EmbedderConfig{
			Provider: "mock",
			Model:    "mock",
		}
		embedder, err = CreateEmbedder(mockConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create fallback mock embedder: %w", err)
		}
	}

	// Create store
	var store VectorStore
	switch cfg.VectorStore.Provider {
	case "chromem":
		store, err = NewChromemStore(embedder, cfg.VectorStore.PersistenceDir, cfg.VectorStore.EnablePersistence)
		if err != nil {
			return nil, fmt.Errorf("failed to create chromem store: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported vector store provider: %s", cfg.VectorStore.Provider)
	}

	// Create manager with the store and embedder
	manager := &Manager{
		store:    store,
		embedder: embedder,
		config: Config{
			Provider:          cfg.VectorStore.Provider,
			PersistenceDir:    cfg.VectorStore.PersistenceDir,
			EnablePersistence: cfg.VectorStore.EnablePersistence,
			EmbedderConfig:    embedderConfig,
		},
	}

	// Create pre-configured collections
	for _, col := range cfg.VectorStore.Collections {
		if _, err := GetOrCreateCollection(store, col.Name, col.Metadata); err != nil {
			return nil, fmt.Errorf("failed to create collection %s: %w", col.Name, err)
		}
	}

	return manager, nil
}

// NewIndexerFromGlobalConfig creates a document indexer using global configuration
func NewIndexerFromGlobalConfig(collectionName string) (*DocumentIndexer, error) {
	// Get global manager
	manager, err := GetGlobalManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get global manager: %w", err)
	}
	if manager == nil {
		return nil, fmt.Errorf("vector store is not enabled")
	}

	cfg := config.Get()
	indexerConfig := IndexerConfig{
		CollectionName: collectionName,
		ChunkSize:      cfg.VectorStore.Indexer.ChunkSize,
		ChunkOverlap:   cfg.VectorStore.Indexer.ChunkOverlap,
	}

	// Use defaults if not set
	if indexerConfig.ChunkSize <= 0 {
		indexerConfig.ChunkSize = 1000
	}
	if indexerConfig.ChunkOverlap < 0 {
		indexerConfig.ChunkOverlap = 200
	}

	return NewDocumentIndexer(manager, indexerConfig)
}
