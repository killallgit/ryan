package vectorstore

import (
	"context"
)

// Document represents a document to be stored in the vector store
type Document struct {
	ID       string                 // Unique identifier
	Content  string                 // Document content
	Metadata map[string]interface{} // Additional metadata
	Vector   []float32              // Optional pre-computed embedding vector
}

// SearchResult represents a search result from the vector store
type SearchResult struct {
	Document Document
	Score    float32 // Similarity score (0-1, higher is better)
	Distance float32 // Distance metric (lower is better)
}

// VectorStore defines the interface for vector storage and retrieval
type VectorStore interface {
	// AddDocuments adds documents to the vector store
	AddDocuments(ctx context.Context, documents []Document) error

	// DeleteDocuments removes documents by their IDs
	DeleteDocuments(ctx context.Context, ids []string) error

	// SimilaritySearch performs a similarity search
	SimilaritySearch(ctx context.Context, query string, k int) ([]SearchResult, error)

	// SimilaritySearchWithScore performs a similarity search with scores
	SimilaritySearchWithScore(ctx context.Context, query string, k int, scoreThreshold float32) ([]SearchResult, error)

	// Clear removes all documents from the store
	Clear(ctx context.Context) error

	// Count returns the number of documents in the store
	Count(ctx context.Context) (int, error)

	// Close closes the vector store and releases resources
	Close() error
}

// Retriever defines the interface for document retrieval
type Retriever interface {
	// GetRelevantDocuments retrieves relevant documents for a query
	GetRelevantDocuments(ctx context.Context, query string) ([]Document, error)

	// GetRelevantDocumentsWithScore retrieves documents with similarity scores
	GetRelevantDocumentsWithScore(ctx context.Context, query string) ([]SearchResult, error)
}

// RetrieverConfig contains configuration for a retriever
type RetrieverConfig struct {
	// Number of documents to retrieve
	K int

	// Minimum similarity score threshold (0-1)
	ScoreThreshold float32

	// Maximum context length for augmentation
	MaxContextLength int

	// Metadata filters
	Filters map[string]interface{}
}

// VectorStoreRetriever implements Retriever using a VectorStore
type VectorStoreRetriever struct {
	store  VectorStore
	config RetrieverConfig
}

// NewVectorStoreRetriever creates a new retriever from a vector store
func NewVectorStoreRetriever(store VectorStore, config RetrieverConfig) *VectorStoreRetriever {
	if config.K == 0 {
		config.K = 4 // Default to 4 documents
	}
	return &VectorStoreRetriever{
		store:  store,
		config: config,
	}
}

// GetRelevantDocuments retrieves relevant documents for a query
func (r *VectorStoreRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]Document, error) {
	results, err := r.store.SimilaritySearchWithScore(ctx, query, r.config.K, r.config.ScoreThreshold)
	if err != nil {
		return nil, err
	}

	documents := make([]Document, len(results))
	for i, result := range results {
		documents[i] = result.Document
	}
	return documents, nil
}

// GetRelevantDocumentsWithScore retrieves documents with similarity scores
func (r *VectorStoreRetriever) GetRelevantDocumentsWithScore(ctx context.Context, query string) ([]SearchResult, error) {
	return r.store.SimilaritySearchWithScore(ctx, query, r.config.K, r.config.ScoreThreshold)
}
