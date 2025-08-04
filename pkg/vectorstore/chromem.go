package vectorstore

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/philippgille/chromem-go"
)

// ChromemStore implements VectorStore using chromem-go
type ChromemStore struct {
	db             *chromem.DB
	embedder       Embedder
	persistenceDir string
	mu             sync.RWMutex
}

// NewChromemStore creates a new chromem-based vector store
func NewChromemStore(embedder Embedder, persistenceDir string, enablePersistence bool) (*ChromemStore, error) {
	var db *chromem.DB

	if enablePersistence && persistenceDir != "" {
		// Create persistent DB
		var err error
		db, err = chromem.NewPersistentDB(persistenceDir, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create persistent chromem DB: %w", err)
		}
	} else {
		// Create in-memory DB
		db = chromem.NewDB()
	}

	return &ChromemStore{
		db:             db,
		embedder:       embedder,
		persistenceDir: persistenceDir,
	}, nil
}

// CreateCollection creates a new collection
func (cs *ChromemStore) CreateCollection(name string, metadata map[string]any) (Collection, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Check if collection already exists
	if existingCol := cs.db.GetCollection(name, nil); existingCol != nil {
		return nil, &Error{
			Code:    ErrCodeCollectionExists,
			Message: fmt.Sprintf("collection %s already exists", name),
		}
	}

	// Create embedding function that uses our embedder
	embedFunc := func(ctx context.Context, text string) ([]float32, error) {
		return cs.embedder.EmbedText(ctx, text)
	}

	// Convert metadata to map[string]string for chromem
	stringMetadata := make(map[string]string)
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			stringMetadata[k] = str
		} else {
			stringMetadata[k] = fmt.Sprintf("%v", v)
		}
	}

	// Create collection in chromem
	col, err := cs.db.CreateCollection(name, stringMetadata, embedFunc)
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeCollectionExists,
			Message: fmt.Sprintf("failed to create collection %s", name),
			Cause:   err,
		}
	}

	return &ChromemCollection{
		collection: col,
		embedder:   cs.embedder,
		name:       name,
	}, nil
}

// GetCollection retrieves an existing collection
func (cs *ChromemStore) GetCollection(name string) (Collection, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Create embedding function for the collection
	embedFunc := func(ctx context.Context, text string) ([]float32, error) {
		return cs.embedder.EmbedText(ctx, text)
	}

	col := cs.db.GetCollection(name, embedFunc)
	if col == nil {
		return nil, &Error{
			Code:    ErrCodeCollectionNotFound,
			Message: fmt.Sprintf("collection %s not found", name),
		}
	}

	return &ChromemCollection{
		collection: col,
		embedder:   cs.embedder,
		name:       name,
	}, nil
}

// ListCollections returns all collection names
func (cs *ChromemStore) ListCollections() ([]string, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	collections := cs.db.ListCollections()

	names := make([]string, 0, len(collections))
	for name := range collections {
		names = append(names, name)
	}

	return names, nil
}

// DeleteCollection removes a collection and all its documents
func (cs *ChromemStore) DeleteCollection(name string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if err := cs.db.DeleteCollection(name); err != nil {
		return &Error{
			Code:    ErrCodeCollectionNotFound,
			Message: fmt.Sprintf("failed to delete collection %s", name),
			Cause:   err,
		}
	}

	return nil
}

// Close closes the vector store
func (cs *ChromemStore) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Chromem doesn't have an explicit close method
	// If we add persistence flushing, it would go here
	return nil
}

// ChromemCollection implements Collection using chromem
type ChromemCollection struct {
	collection *chromem.Collection
	embedder   Embedder
	name       string
	mu         sync.RWMutex
}

// Name returns the collection name
func (cc *ChromemCollection) Name() string {
	return cc.name
}

// AddDocuments adds documents to the collection
func (cc *ChromemCollection) AddDocuments(ctx context.Context, docs []Document) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// Generate embeddings for documents that don't have them
	for i := range docs {
		if len(docs[i].Embedding) == 0 && docs[i].Content != "" {
			embedding, err := cc.embedder.EmbedText(ctx, docs[i].Content)
			if err != nil {
				return fmt.Errorf("failed to generate embedding for document %s: %w", docs[i].ID, err)
			}
			docs[i].Embedding = embedding
		}
	}

	// Convert to chromem documents
	chromemDocs := make([]chromem.Document, len(docs))
	for i, doc := range docs {
		// Convert metadata to map[string]string
		stringMetadata := make(map[string]string)
		for k, v := range doc.Metadata {
			if str, ok := v.(string); ok {
				stringMetadata[k] = str
			} else {
				stringMetadata[k] = fmt.Sprintf("%v", v)
			}
		}

		chromemDocs[i] = chromem.Document{
			ID:        doc.ID,
			Content:   doc.Content,
			Metadata:  stringMetadata,
			Embedding: doc.Embedding,
		}
	}

	// Add documents to collection with concurrent workers
	if err := cc.collection.AddDocuments(ctx, chromemDocs, runtime.NumCPU()); err != nil {
		return fmt.Errorf("failed to add documents: %w", err)
	}

	return nil
}

// Query performs a similarity search
func (cc *ChromemCollection) Query(ctx context.Context, query string, k int, opts ...QueryOption) ([]Result, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	// Apply options
	options := &queryOptions{}
	for _, opt := range opts {
		opt.apply(options)
	}

	// Convert where clause
	where := buildChromemWhere(options)

	// Ensure k doesn't exceed document count
	docCount := cc.collection.Count()
	if k > docCount {
		k = docCount
	}

	// Handle empty collection
	if docCount == 0 {
		return []Result{}, nil
	}

	// Perform query
	chromemResults, err := cc.collection.Query(ctx, query, k, where, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	// Filter by min score if needed
	filteredResults := chromemResults
	if options.minScore > 0 {
		filtered := make([]chromem.Result, 0, len(chromemResults))
		for _, r := range chromemResults {
			if r.Similarity >= options.minScore {
				filtered = append(filtered, r)
			}
		}
		filteredResults = filtered
	}

	// Convert results
	results := make([]Result, len(filteredResults))
	for i, cr := range filteredResults {
		// Convert metadata back to map[string]any
		metadata := make(map[string]any)
		for k, v := range cr.Metadata {
			metadata[k] = v
		}

		results[i] = Result{
			Document: Document{
				ID:        cr.ID,
				Content:   cr.Content,
				Metadata:  metadata,
				Embedding: cr.Embedding,
			},
			Score:    cr.Similarity,
			Distance: 1 - cr.Similarity, // Convert similarity to distance
		}
	}

	return results, nil
}

// QueryWithEmbedding performs a similarity search using a pre-computed embedding
func (cc *ChromemCollection) QueryWithEmbedding(ctx context.Context, embedding []float32, k int, opts ...QueryOption) ([]Result, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	// Apply options
	options := &queryOptions{}
	for _, opt := range opts {
		opt.apply(options)
	}

	// Convert where clause
	where := buildChromemWhere(options)

	// Ensure k doesn't exceed document count
	docCount := cc.collection.Count()
	if k > docCount {
		k = docCount
	}

	// Handle empty collection
	if docCount == 0 {
		return []Result{}, nil
	}

	// Perform query with embedding
	chromemResults, err := cc.collection.QueryEmbedding(ctx, embedding, k, where, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection with embedding: %w", err)
	}

	// Filter by min score if needed
	filteredResults := chromemResults
	if options.minScore > 0 {
		filtered := make([]chromem.Result, 0, len(chromemResults))
		for _, r := range chromemResults {
			if r.Similarity >= options.minScore {
				filtered = append(filtered, r)
			}
		}
		filteredResults = filtered
	}

	// Convert results
	results := make([]Result, len(filteredResults))
	for i, cr := range filteredResults {
		// Convert metadata back to map[string]any
		metadata := make(map[string]any)
		for k, v := range cr.Metadata {
			metadata[k] = v
		}

		results[i] = Result{
			Document: Document{
				ID:        cr.ID,
				Content:   cr.Content,
				Metadata:  metadata,
				Embedding: cr.Embedding,
			},
			Score:    cr.Similarity,
			Distance: 1 - cr.Similarity, // Convert similarity to distance
		}
	}

	return results, nil
}

// Delete removes documents by ID
func (cc *ChromemCollection) Delete(ctx context.Context, ids []string) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// Delete documents by IDs
	if err := cc.collection.Delete(ctx, nil, nil, ids...); err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	return nil
}

// Count returns the number of documents in the collection
func (cc *ChromemCollection) Count() (int, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return cc.collection.Count(), nil
}

// Clear removes all documents from the collection
func (cc *ChromemCollection) Clear(ctx context.Context) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// Get document count
	count := cc.collection.Count()
	if count == 0 {
		return nil // Already empty
	}

	// Get all documents by querying with a dummy query
	// We use a single space as query text since chromem requires non-empty query
	allResults, err := cc.collection.Query(ctx, " ", count, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to query all documents: %w", err)
	}

	if len(allResults) == 0 {
		return nil // No results, already empty
	}

	// Extract all IDs
	ids := make([]string, len(allResults))
	for i, r := range allResults {
		ids[i] = r.ID
	}

	// Delete all documents
	if err := cc.collection.Delete(ctx, nil, nil, ids...); err != nil {
		return fmt.Errorf("failed to clear collection: %w", err)
	}

	return nil
}

// Helper functions

func buildChromemWhere(opts *queryOptions) map[string]string {
	if opts.filter != nil && len(opts.filter) > 0 {
		// Convert map[string]any to map[string]string
		where := make(map[string]string)
		for k, v := range opts.filter {
			if str, ok := v.(string); ok {
				where[k] = str
			} else {
				where[k] = fmt.Sprintf("%v", v)
			}
		}
		return where
	}
	return nil
}

// GetOrCreateCollection is a helper that gets a collection or creates it if it doesn't exist
func GetOrCreateCollection(store VectorStore, name string, metadata map[string]any) (Collection, error) {
	// Try to get existing collection
	col, err := store.GetCollection(name)
	if err == nil {
		return col, nil
	}

	// If not found, create it
	if e, ok := err.(*Error); ok && e.Code == ErrCodeCollectionNotFound {
		return store.CreateCollection(name, metadata)
	}

	return nil, err
}
