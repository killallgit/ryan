package vectorstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// Manager handles vector store lifecycle and operations
type Manager struct {
	store     VectorStore
	embedder  Embedder
	processor *DocumentProcessor
	config    Config
	log       *logger.Logger
	mu        sync.RWMutex
}

// NewManager creates a new vector store manager
func NewManager(config Config) (*Manager, error) {
	log := logger.WithComponent("vectorstore")

	// Create embedder
	embedder, err := CreateEmbedder(config.EmbedderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Create persistence directory if needed
	if config.EnablePersistence && config.PersistenceDir != "" {
		if err := os.MkdirAll(config.PersistenceDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create persistence directory: %w", err)
		}
	}

	// Create vector store
	var store VectorStore
	switch config.Provider {
	case "chromem", "": // Default to chromem
		store, err = NewChromemStore(embedder, config.PersistenceDir, config.EnablePersistence)
		if err != nil {
			return nil, fmt.Errorf("failed to create chromem store: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported vector store provider: %s", config.Provider)
	}

	// Create document processor
	processor := NewDocumentProcessor(embedder, config.ChunkSize, config.ChunkOverlap)

	manager := &Manager{
		store:     store,
		embedder:  embedder,
		processor: processor,
		config:    config,
		log:       log,
	}

	// Initialize default collections
	if err := manager.initializeCollections(); err != nil {
		return nil, fmt.Errorf("failed to initialize collections: %w", err)
	}

	log.Info("Vector store manager initialized",
		"provider", config.Provider,
		"persistence", config.EnablePersistence,
		"embedder", config.EmbedderConfig.Provider)

	return manager, nil
}

// initializeCollections creates default collections if they don't exist
func (m *Manager) initializeCollections() error {
	for _, colConfig := range m.config.Collections {
		_, err := GetOrCreateCollection(m.store, colConfig.Name, colConfig.Metadata)
		if err != nil {
			return fmt.Errorf("failed to initialize collection %s: %w", colConfig.Name, err)
		}
		m.log.Debug("Initialized collection", "name", colConfig.Name)
	}
	return nil
}

// GetStore returns the underlying vector store
func (m *Manager) GetStore() VectorStore {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store
}

// GetEmbedder returns the embedder
func (m *Manager) GetEmbedder() Embedder {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.embedder
}

// GetConfig returns the configuration
func (m *Manager) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetCollection gets or creates a collection
func (m *Manager) GetCollection(name string) (Collection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return GetOrCreateCollection(m.store, name, nil)
}

// IndexDocument indexes a single document
func (m *Manager) IndexDocument(ctx context.Context, collectionName string, doc Document) error {
	// Validate collection name
	if err := validateCollectionName(collectionName); err != nil {
		return err
	}

	// Process document (validates and generates embedding if needed)
	if err := m.processor.ProcessDocument(ctx, &doc); err != nil {
		return err
	}

	collection, err := m.GetCollection(collectionName)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	if err := collection.AddDocuments(ctx, []Document{doc}); err != nil {
		return fmt.Errorf("failed to add document: %w", err)
	}

	m.log.Debug("Indexed document", "collection", collectionName, "id", doc.ID)
	return nil
}

// IndexDocuments indexes multiple documents
func (m *Manager) IndexDocuments(ctx context.Context, collectionName string, docs []Document) error {
	// Validate collection name
	if err := validateCollectionName(collectionName); err != nil {
		return err
	}

	if len(docs) == 0 {
		return errors.New("no documents to index")
	}

	// Process documents (validates and generates embeddings)
	if err := m.processor.ProcessDocuments(ctx, docs); err != nil {
		return err
	}

	collection, err := m.GetCollection(collectionName)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	if err := collection.AddDocuments(ctx, docs); err != nil {
		return fmt.Errorf("failed to add documents: %w", err)
	}

	m.log.Debug("Indexed documents", "collection", collectionName, "count", len(docs))
	return nil
}

// Search performs a semantic search across a collection
func (m *Manager) Search(ctx context.Context, collectionName string, query string, k int, opts ...QueryOption) ([]Result, error) {
	collection, err := m.GetCollection(collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	results, err := collection.Query(ctx, query, k, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	m.log.Debug("Search completed", "collection", collectionName, "query", query, "results", len(results))
	return results, nil
}

// ClearCollection removes all documents from a collection
func (m *Manager) ClearCollection(ctx context.Context, collectionName string) error {
	collection, err := m.GetCollection(collectionName)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	if err := collection.Clear(ctx); err != nil {
		return fmt.Errorf("failed to clear collection: %w", err)
	}

	m.log.Info("Cleared collection", "name", collectionName)
	return nil
}

// DeleteCollection removes a collection
func (m *Manager) DeleteCollection(collectionName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.store.DeleteCollection(collectionName); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	m.log.Info("Deleted collection", "name", collectionName)
	return nil
}

// ListCollections returns all collection names
func (m *Manager) ListCollections() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.store.ListCollections()
}

// GetCollectionInfo returns information about a collection
func (m *Manager) GetCollectionInfo(collectionName string) (*CollectionMetadata, error) {
	collection, err := m.GetCollection(collectionName)
	if err != nil {
		return nil, err
	}

	count, err := collection.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to get document count: %w", err)
	}

	return &CollectionMetadata{
		Name:          collectionName,
		DocumentCount: count,
	}, nil
}

// ChunkAndIndexDocument chunks a document and indexes all chunks
func (m *Manager) ChunkAndIndexDocument(ctx context.Context, collectionName string, doc Document, metadata map[string]any) error {
	// Validate collection name
	if err := validateCollectionName(collectionName); err != nil {
		return err
	}

	// Chunk the document
	chunks, err := m.processor.ChunkDocument(doc, metadata)
	if err != nil {
		return fmt.Errorf("failed to chunk document: %w", err)
	}

	// Index all chunks
	return m.IndexDocuments(ctx, collectionName, chunks)
}

// GetDocumentProcessor returns the document processor for custom operations
func (m *Manager) GetDocumentProcessor() *DocumentProcessor {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.processor
}

// Close closes the vector store manager
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.store.Close(); err != nil {
		return fmt.Errorf("failed to close vector store: %w", err)
	}

	m.log.Info("Vector store manager closed")
	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	defaultPersistenceDir := filepath.Join(homeDir, ".ryan", "vectorstore")

	return Config{
		Provider:          "chromem",
		PersistenceDir:    defaultPersistenceDir,
		EnablePersistence: true,
		ChunkSize:         1000,
		ChunkOverlap:      200,
		Collections: []CollectionConfig{
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
		},
		EmbedderConfig: EmbedderConfig{
			Provider:     "ollama",
			Model:        "nomic-embed-text",
			BaseURL:      "http://localhost:11434",
			HTTPTimeout:  30 * time.Second,
			MaxRetries:   3,
			RetryBackoff: 100 * time.Millisecond,
		},
	}
}

// ConfigFromViper creates a Config from viper settings
func ConfigFromViper(persistenceDir string) Config {
	config := DefaultConfig()

	// Override with custom persistence directory if provided
	if persistenceDir != "" {
		config.PersistenceDir = persistenceDir
	}

	// In the future, we can read more settings from viper here
	// For now, we'll use defaults

	return config
}

// indexDocumentsInBatches processes large document sets in batches
func (m *Manager) indexDocumentsInBatches(ctx context.Context, collectionName string, docs []Document) error {
	collection, err := m.GetCollection(collectionName)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	// Process in batches
	for i := 0; i < len(docs); i += MaxBatchSize {
		end := i + MaxBatchSize
		if end > len(docs) {
			end = len(docs)
		}

		batch := docs[i:end]

		// Process batch (validates and generates embeddings)
		if err := m.processor.ProcessDocuments(ctx, batch); err != nil {
			return fmt.Errorf("failed to process batch starting at index %d: %w", i, err)
		}

		// Index batch
		if err := collection.AddDocuments(ctx, batch); err != nil {
			return fmt.Errorf("failed to add batch starting at index %d: %w", i, err)
		}

		m.log.Debug("Indexed batch", "collection", collectionName, "start", i, "size", len(batch))
	}

	return nil
}
