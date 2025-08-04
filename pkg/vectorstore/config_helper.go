package vectorstore

import (
	"fmt"
	"os"

	"github.com/killallgit/ryan/pkg/config"
)

// NewFromConfig creates a vector store manager from configuration
func NewFromConfig(cfg *config.VectorStoreConfig) (*Manager, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, fmt.Errorf("vector store is not enabled in configuration")
	}

	// Create embedder config
	embedderConfig := EmbedderConfig{
		Provider: cfg.Embedder.Provider,
		Model:    cfg.Embedder.Model,
		BaseURL:  cfg.Embedder.BaseURL,
		APIKey:   cfg.Embedder.APIKey,
	}

	// Handle API key from environment for security
	if embedderConfig.Provider == "openai" && embedderConfig.APIKey == "" {
		embedderConfig.APIKey = os.Getenv("OPENAI_API_KEY")
	}

	// Create manager config
	managerConfig := Config{
		Provider:          cfg.Provider,
		PersistenceDir:    cfg.PersistenceDir,
		EnablePersistence: cfg.EnablePersistence,
		EmbedderConfig:    embedderConfig,
	}

	// Convert collections config
	for _, col := range cfg.Collections {
		managerConfig.Collections = append(managerConfig.Collections, CollectionConfig{
			Name:     col.Name,
			Metadata: col.Metadata,
		})
	}

	// Create manager
	manager, err := NewManager(managerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store manager: %w", err)
	}

	// Create collections if specified
	if len(managerConfig.Collections) > 0 {
		for _, col := range managerConfig.Collections {
			if _, err := GetOrCreateCollection(manager.GetStore(), col.Name, col.Metadata); err != nil {
				return nil, fmt.Errorf("failed to create collection %s: %w", col.Name, err)
			}
		}
	}

	return manager, nil
}

// NewIndexerFromConfig creates a document indexer from configuration
func NewIndexerFromConfig(store VectorStore, cfg *config.VectorStoreIndexerConfig, collectionName string) (*DocumentIndexer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("indexer configuration is nil")
	}

	indexerConfig := IndexerConfig{
		CollectionName: collectionName,
		ChunkSize:      cfg.ChunkSize,
		ChunkOverlap:   cfg.ChunkOverlap,
	}

	// Validate chunk configuration
	if indexerConfig.ChunkSize <= 0 {
		indexerConfig.ChunkSize = 1000 // Default
	}
	if indexerConfig.ChunkOverlap < 0 {
		indexerConfig.ChunkOverlap = 200 // Default
	}
	if indexerConfig.ChunkOverlap >= indexerConfig.ChunkSize {
		return nil, fmt.Errorf("chunk overlap must be less than chunk size")
	}

	return NewDocumentIndexer(store, indexerConfig)
}

// InitializeVectorStore is a convenience function to initialize vector store from app config
func InitializeVectorStore() (*Manager, error) {
	cfg := config.Get()
	if cfg == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}

	if !cfg.VectorStore.Enabled {
		return nil, nil // Vector store is disabled
	}

	return NewFromConfig(&cfg.VectorStore)
}