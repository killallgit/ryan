package vectorstore

import (
	"fmt"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/spf13/viper"
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

	// Handle API key from environment for security (via Viper)
	if embedderConfig.Provider == "openai" && embedderConfig.APIKey == "" {
		embedderConfig.APIKey = viper.GetString("OPENAI_API_KEY")
	}

	// Check if embedder can be created - NewManager will handle fallback
	_, err := CreateEmbedder(embedderConfig)
	if err != nil {
		// Log the error and use mock embedder for development/debugging
		fmt.Printf("Warning: Failed to create %s embedder (%v), falling back to mock embedder\n", embedderConfig.Provider, err)
		embedderConfig = EmbedderConfig{
			Provider: "mock",
			Model:    "mock",
		}
	}

	// Build full config with collections
	managerConfig := Config{
		Provider:          cfg.VectorStore.Provider,
		PersistenceDir:    cfg.VectorStore.PersistenceDir,
		EnablePersistence: cfg.VectorStore.EnablePersistence,
		ChunkSize:         1000, // Default chunk size
		ChunkOverlap:      200,  // Default overlap
		EmbedderConfig:    embedderConfig,
		Collections:       make([]CollectionConfig, 0),
	}

	// Add configured collections
	for _, col := range cfg.VectorStore.Collections {
		managerConfig.Collections = append(managerConfig.Collections, CollectionConfig{
			Name:     col.Name,
			Metadata: col.Metadata,
		})
	}

	// Add default collections if not specified
	if len(managerConfig.Collections) == 0 {
		managerConfig.Collections = []CollectionConfig{
			{
				Name: "conversations",
				Metadata: map[string]any{
					"description": "Chat conversation history",
					"type":        "conversation",
				},
			},
			{
				Name: "documents",
				Metadata: map[string]any{
					"description": "Indexed documents and files",
					"type":        "document",
				},
			},
			{
				Name: "tools",
				Metadata: map[string]any{
					"description": "Tool execution results and outputs",
					"type":        "tool_output",
				},
			},
		}
	}

	// Use NewManager to create properly initialized manager
	manager, err := NewManager(managerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
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
