package vectorstore

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/killallgit/ryan/pkg/embeddings"
	chromem "github.com/philippgille/chromem-go"
)

// ChromemStore implements VectorStore using chromem-go
type ChromemStore struct {
	db         *chromem.DB
	collection *chromem.Collection
	embedder   embeddings.Embedder
	docIDs     map[string]bool // Track document IDs for clearing
	mu         sync.RWMutex
}

// ChromemConfig contains configuration for ChromemStore
type ChromemConfig struct {
	// CollectionName is the name of the collection to use
	CollectionName string

	// PersistDirectory is the directory for persistence (empty for in-memory only)
	PersistDirectory string

	// Embedder to use for creating embeddings
	Embedder embeddings.Embedder

	// Metadata for the collection
	Metadata map[string]string
}

// NewChromemStore creates a new ChromemStore
func NewChromemStore(config ChromemConfig) (*ChromemStore, error) {
	if config.Embedder == nil {
		return nil, fmt.Errorf("embedder is required")
	}

	if config.CollectionName == "" {
		config.CollectionName = "default"
	}

	// Create the database
	var db *chromem.DB
	var err error
	if config.PersistDirectory != "" {
		db, err = chromem.NewPersistentDB(config.PersistDirectory, false)
	} else {
		db = chromem.NewDB()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create chromem database: %w", err)
	}

	// Create embedding function adapter
	embeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		return config.Embedder.EmbedText(ctx, text)
	}

	// Create or get collection
	collection, err := db.GetOrCreateCollection(
		config.CollectionName,
		config.Metadata,
		embeddingFunc,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	return &ChromemStore{
		db:         db,
		collection: collection,
		embedder:   config.Embedder,
		docIDs:     make(map[string]bool),
	}, nil
}

// AddDocuments adds documents to the vector store
func (s *ChromemStore) AddDocuments(ctx context.Context, documents []Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Convert to chromem documents
	chromemDocs := make([]chromem.Document, len(documents))
	for i, doc := range documents {
		// Convert metadata to string map
		metadata := make(map[string]string)
		for k, v := range doc.Metadata {
			if str, ok := v.(string); ok {
				metadata[k] = str
			} else {
				metadata[k] = fmt.Sprintf("%v", v)
			}
		}

		chromemDocs[i] = chromem.Document{
			ID:        doc.ID,
			Content:   doc.Content,
			Metadata:  metadata,
			Embedding: doc.Vector,
		}
	}

	// Add to collection
	if err := s.collection.AddDocuments(ctx, chromemDocs, runtime.NumCPU()); err != nil {
		return fmt.Errorf("failed to add documents: %w", err)
	}

	// Track document IDs
	for _, doc := range documents {
		s.docIDs[doc.ID] = true
	}

	return nil
}

// DeleteDocuments removes documents by their IDs
func (s *ChromemStore) DeleteDocuments(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete documents by IDs
	err := s.collection.Delete(ctx, nil, nil, ids...)
	if err == nil {
		// Remove from tracked IDs
		for _, id := range ids {
			delete(s.docIDs, id)
		}
	}
	return err
}

// SimilaritySearch performs a similarity search
func (s *ChromemStore) SimilaritySearch(ctx context.Context, query string, k int) ([]SearchResult, error) {
	return s.SimilaritySearchWithScore(ctx, query, k, 0)
}

// SimilaritySearchWithScore performs a similarity search with scores
func (s *ChromemStore) SimilaritySearchWithScore(ctx context.Context, query string, k int, scoreThreshold float32) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Perform query
	chromemResults, err := s.collection.Query(ctx, query, k, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Convert results
	results := make([]SearchResult, 0, len(chromemResults))
	for _, cr := range chromemResults {
		// Use similarity score directly (chromem returns cosine similarity)
		score := cr.Similarity

		// Apply threshold
		if scoreThreshold > 0 && score < scoreThreshold {
			continue
		}

		// Convert metadata back to interface{} map
		metadata := make(map[string]interface{})
		for k, v := range cr.Metadata {
			metadata[k] = v
		}

		results = append(results, SearchResult{
			Document: Document{
				ID:       cr.ID,
				Content:  cr.Content,
				Metadata: metadata,
				Vector:   cr.Embedding,
			},
			Score:    score,
			Distance: 1 - score, // Convert similarity to distance
		})
	}

	return results, nil
}

// Clear removes all documents from the store
func (s *ChromemStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use tracked IDs to delete all documents
	if len(s.docIDs) == 0 {
		return nil // Already empty
	}

	// Extract IDs
	ids := make([]string, 0, len(s.docIDs))
	for id := range s.docIDs {
		ids = append(ids, id)
	}

	// Delete all documents by their IDs
	err := s.collection.Delete(ctx, nil, nil, ids...)
	if err == nil {
		// Clear tracked IDs
		s.docIDs = make(map[string]bool)
	}
	return err
}

// Count returns the number of documents in the store
func (s *ChromemStore) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.collection.Count(), nil
}

// Close closes the vector store and releases resources
func (s *ChromemStore) Close() error {
	// Chromem doesn't require explicit closing for in-memory mode
	// For persistent mode, it auto-saves on each operation
	return nil
}
