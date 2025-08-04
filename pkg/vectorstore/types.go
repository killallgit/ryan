package vectorstore

import (
	"context"
	"time"
)

// VectorStore represents the main interface for vector storage operations
type VectorStore interface {
	// CreateCollection creates a new collection with the given name
	CreateCollection(name string, metadata map[string]interface{}) (Collection, error)

	// GetCollection retrieves an existing collection by name
	GetCollection(name string) (Collection, error)

	// ListCollections returns all collection names
	ListCollections() ([]string, error)

	// DeleteCollection removes a collection and all its documents
	DeleteCollection(name string) error

	// Close closes the vector store and releases resources
	Close() error
}

// Collection represents a collection of documents with vector embeddings
type Collection interface {
	// Name returns the collection name
	Name() string

	// AddDocuments adds documents to the collection
	AddDocuments(ctx context.Context, docs []Document) error

	// Query performs a similarity search
	Query(ctx context.Context, query string, k int, opts ...QueryOption) ([]Result, error)

	// QueryWithEmbedding performs a similarity search using a pre-computed embedding
	QueryWithEmbedding(ctx context.Context, embedding []float32, k int, opts ...QueryOption) ([]Result, error)

	// Delete removes documents by ID
	Delete(ctx context.Context, ids []string) error

	// Count returns the number of documents in the collection
	Count() (int, error)

	// Clear removes all documents from the collection
	Clear(ctx context.Context) error
}

// Document represents a document to be stored in the vector store
type Document struct {
	// ID is a unique identifier for the document
	ID string

	// Content is the text content of the document
	Content string

	// Metadata contains additional information about the document
	Metadata map[string]interface{}

	// Embedding is the vector representation (optional - will be generated if not provided)
	Embedding []float32
}

// Result represents a search result
type Result struct {
	// Document is the matched document
	Document Document

	// Score is the similarity score (higher is better)
	Score float32

	// Distance is the distance metric (lower is better)
	Distance float32
}

// QueryOption represents options for querying
type QueryOption interface {
	apply(*queryOptions)
}

// queryOptions holds query configuration
type queryOptions struct {
	// Filter for metadata-based filtering
	filter map[string]interface{}

	// MinScore sets a minimum similarity score threshold
	minScore float32

	// IncludeEmbeddings includes embeddings in results
	includeEmbeddings bool
}

// WithFilter adds metadata filtering to queries
func WithFilter(filter map[string]interface{}) QueryOption {
	return filterOption{filter: filter}
}

type filterOption struct {
	filter map[string]interface{}
}

func (f filterOption) apply(opts *queryOptions) {
	opts.filter = f.filter
}

// WithMinScore sets a minimum similarity score threshold
func WithMinScore(score float32) QueryOption {
	return minScoreOption{score: score}
}

type minScoreOption struct {
	score float32
}

func (m minScoreOption) apply(opts *queryOptions) {
	opts.minScore = m.score
}

// WithEmbeddings includes embeddings in query results
func WithEmbeddings() QueryOption {
	return embeddingsOption{}
}

type embeddingsOption struct{}

func (e embeddingsOption) apply(opts *queryOptions) {
	opts.includeEmbeddings = true
}

// Embedder generates embeddings from text
type Embedder interface {
	// EmbedText generates an embedding for a single text
	EmbedText(ctx context.Context, text string) ([]float32, error)

	// EmbedTexts generates embeddings for multiple texts
	EmbedTexts(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the embedding dimensions
	Dimensions() int
}

// Config represents vector store configuration
type Config struct {
	// Provider specifies the vector store implementation
	Provider string

	// PersistenceDir is the directory for persistent storage
	PersistenceDir string

	// EnablePersistence controls whether to persist data
	EnablePersistence bool

	// Collections to create on initialization
	Collections []CollectionConfig

	// Embedder configuration
	EmbedderConfig EmbedderConfig
}

// CollectionConfig represents configuration for a collection
type CollectionConfig struct {
	// Name of the collection
	Name string

	// Metadata for the collection
	Metadata map[string]interface{}
}

// EmbedderConfig represents embedder configuration
type EmbedderConfig struct {
	// Provider (e.g., "ollama", "openai", "local")
	Provider string

	// Model name
	Model string

	// API key (if required)
	APIKey string

	// Base URL (for custom endpoints)
	BaseURL string

	// HTTP timeout for embedding requests
	HTTPTimeout time.Duration

	// Maximum retries for failed requests
	MaxRetries int

	// Base backoff duration for retries
	RetryBackoff time.Duration
}

// StoreMetadata represents metadata about the vector store
type StoreMetadata struct {
	// Provider name
	Provider string

	// Version of the provider
	Version string

	// Collections in the store
	Collections []CollectionMetadata

	// CreatedAt timestamp
	CreatedAt time.Time

	// UpdatedAt timestamp
	UpdatedAt time.Time
}

// CollectionMetadata represents metadata about a collection
type CollectionMetadata struct {
	// Name of the collection
	Name string

	// DocumentCount is the number of documents
	DocumentCount int

	// Metadata associated with the collection
	Metadata map[string]interface{}

	// CreatedAt timestamp
	CreatedAt time.Time

	// UpdatedAt timestamp
	UpdatedAt time.Time
}

// Error types for vector store operations
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e Error) Unwrap() error {
	return e.Cause
}

// ErrorCode represents vector store error codes
type ErrorCode int

const (
	ErrCodeUnknown ErrorCode = iota
	ErrCodeCollectionNotFound
	ErrCodeCollectionExists
	ErrCodeDocumentNotFound
	ErrCodeInvalidEmbedding
	ErrCodeEmbeddingGeneration
	ErrCodePersistence
	ErrCodeNotImplemented
)
