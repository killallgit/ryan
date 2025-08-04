package vectorstore

import (
	"context"
	"fmt"
	"sync"
)

var (
	globalManager     *Manager
	globalManagerOnce sync.Once
	globalManagerErr  error
)

// GetGlobalManager returns the global vector store manager instance
// It initializes it on first call using the global configuration
func GetGlobalManager() (*Manager, error) {
	globalManagerOnce.Do(func() {
		globalManager, globalManagerErr = InitializeVectorStore()
	})
	return globalManager, globalManagerErr
}

// ResetGlobalManager resets the global manager instance
// This is mainly useful for testing
func ResetGlobalManager() {
	if globalManager != nil {
		globalManager.Close()
		globalManager = nil
	}
	globalManagerOnce = sync.Once{}
	globalManagerErr = nil
}

// GetOrCreateCollectionGlobal gets or creates a collection using the global manager
func GetOrCreateCollectionGlobal(name string, metadata map[string]any) (Collection, error) {
	manager, err := GetGlobalManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get global manager: %w", err)
	}
	if manager == nil {
		return nil, fmt.Errorf("vector store is not enabled")
	}
	return GetOrCreateCollection(manager.GetStore(), name, metadata)
}

// IndexDocumentsGlobal indexes documents using the global manager
func IndexDocumentsGlobal(collectionName string, docs []Document) error {
	manager, err := GetGlobalManager()
	if err != nil {
		return fmt.Errorf("failed to get global manager: %w", err)
	}
	if manager == nil {
		return fmt.Errorf("vector store is not enabled")
	}
	return manager.IndexDocuments(context.Background(), collectionName, docs)
}

// SearchGlobal searches documents using the global manager
func SearchGlobal(collectionName string, query string, k int, opts ...QueryOption) ([]Result, error) {
	manager, err := GetGlobalManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get global manager: %w", err)
	}
	if manager == nil {
		return nil, fmt.Errorf("vector store is not enabled")
	}
	return manager.Search(context.Background(), collectionName, query, k, opts...)
}
