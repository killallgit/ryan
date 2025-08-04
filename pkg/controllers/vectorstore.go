package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/vectorstore"
)

type CollectionInfo struct {
	Name          string
	DocumentCount int
	EmbedderModel string
	LastUpdated   time.Time
	Metadata      map[string]interface{}
}

type VectorStoreStats struct {
	TotalCollections int
	TotalDocuments   int
	Provider         string
	PersistenceDir   string
	IsEnabled        bool
}

type EmbedderInfo struct {
	Provider   string
	Model      string
	Dimensions int
	BaseURL    string
}

type VectorStoreController struct {
	log *logger.Logger
}

func NewVectorStoreController() *VectorStoreController {
	return &VectorStoreController{
		log: logger.WithComponent("vectorstore_controller"),
	}
}

func (c *VectorStoreController) GetCollections() ([]CollectionInfo, error) {
	manager, err := vectorstore.GetGlobalManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get vector store manager: %w", err)
	}
	if manager == nil {
		return nil, fmt.Errorf("vector store is not enabled")
	}

	store := manager.GetStore()
	collectionNames, err := store.ListCollections()
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	collections := make([]CollectionInfo, 0, len(collectionNames))
	embedderInfo := c.getEmbedderInfo(manager)

	for _, name := range collectionNames {
		collection, err := store.GetCollection(name)
		if err != nil {
			c.log.Warn("Failed to get collection", "name", name, "error", err)
			continue
		}

		count, err := collection.Count()
		if err != nil {
			c.log.Warn("Failed to get collection count", "name", name, "error", err)
			count = -1
		}

		info := CollectionInfo{
			Name:          name,
			DocumentCount: count,
			EmbedderModel: embedderInfo.Model,
			LastUpdated:   time.Now(), // TODO: Get actual last updated time from metadata
			Metadata:      make(map[string]interface{}),
		}

		collections = append(collections, info)
	}

	return collections, nil
}

func (c *VectorStoreController) GetCollectionStats(name string) (*CollectionInfo, error) {
	manager, err := vectorstore.GetGlobalManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get vector store manager: %w", err)
	}
	if manager == nil {
		return nil, fmt.Errorf("vector store is not enabled")
	}

	store := manager.GetStore()
	collection, err := store.GetCollection(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	count, err := collection.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to get collection count: %w", err)
	}

	embedderInfo := c.getEmbedderInfo(manager)

	return &CollectionInfo{
		Name:          name,
		DocumentCount: count,
		EmbedderModel: embedderInfo.Model,
		LastUpdated:   time.Now(),
		Metadata:      make(map[string]interface{}),
	}, nil
}

func (c *VectorStoreController) GetStoreMetadata() (*VectorStoreStats, error) {
	// Check if vector store is configured to be enabled
	cfg := config.Get()
	configEnabled := cfg != nil && cfg.VectorStore.Enabled

	manager, err := vectorstore.GetGlobalManager()
	if err != nil {
		c.log.Error("Failed to get vector store manager", "error", err)
		return &VectorStoreStats{
			IsEnabled:        false,
			Provider:         "error",
			TotalCollections: 0,
			TotalDocuments:   0,
		}, nil
	}
	if manager == nil {
		// Vector store is configured as disabled
		return &VectorStoreStats{
			IsEnabled:        configEnabled, // Show true if config says enabled but manager is nil
			Provider:         "disabled",
			TotalCollections: 0,
			TotalDocuments:   0,
		}, nil
	}

	store := manager.GetStore()
	collectionNames, err := store.ListCollections()
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	totalDocs := 0
	for _, name := range collectionNames {
		collection, err := store.GetCollection(name)
		if err != nil {
			continue
		}
		count, err := collection.Count()
		if err != nil {
			continue
		}
		totalDocs += count
	}

	config := manager.GetConfig()

	return &VectorStoreStats{
		TotalCollections: len(collectionNames),
		TotalDocuments:   totalDocs,
		Provider:         config.Provider,
		PersistenceDir:   config.PersistenceDir,
		IsEnabled:        true,
	}, nil
}

func (c *VectorStoreController) GetEmbedderInfo() (*EmbedderInfo, error) {
	manager, err := vectorstore.GetGlobalManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get vector store manager: %w", err)
	}
	if manager == nil {
		return nil, fmt.Errorf("vector store is not enabled")
	}

	return c.getEmbedderInfo(manager), nil
}

func (c *VectorStoreController) getEmbedderInfo(manager *vectorstore.Manager) *EmbedderInfo {
	embedder := manager.GetEmbedder()
	config := manager.GetConfig()

	return &EmbedderInfo{
		Provider:   config.EmbedderConfig.Provider,
		Model:      config.EmbedderConfig.Model,
		Dimensions: embedder.Dimensions(),
		BaseURL:    config.EmbedderConfig.BaseURL,
	}
}

func (c *VectorStoreController) SearchInCollection(ctx context.Context, collectionName string, query string, k int) ([]vectorstore.Result, error) {
	manager, err := vectorstore.GetGlobalManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get vector store manager: %w", err)
	}
	if manager == nil {
		return nil, fmt.Errorf("vector store is not enabled")
	}

	return manager.Search(ctx, collectionName, query, k)
}
