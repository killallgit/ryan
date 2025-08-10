package vectorstore

import (
	"context"
	"fmt"
	"sync"

	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
)

// MockVectorStore implements a mock vectorstore for testing
type MockVectorStore struct {
	documents  []schema.Document
	embeddings map[string][]float32
	mu         sync.RWMutex
	idCounter  int
}

// NewMockVectorStore creates a new mock vectorstore
func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		documents:  []schema.Document{},
		embeddings: make(map[string][]float32),
		idCounter:  0,
	}
}

// AddDocuments adds documents to the mock store
func (m *MockVectorStore) AddDocuments(ctx context.Context, docs []schema.Document, options ...vectorstores.Option) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]string, len(docs))
	for i, doc := range docs {
		// Generate ID
		id := fmt.Sprintf("doc_%d", m.idCounter)
		m.idCounter++
		ids[i] = id

		// Store document
		m.documents = append(m.documents, doc)

		// Generate mock embedding (just a simple fake embedding)
		embedding := []float32{
			float32(m.idCounter) * 0.1,
			float32(m.idCounter) * 0.2,
			float32(m.idCounter) * 0.3,
		}
		m.embeddings[id] = embedding
	}

	return ids, nil
}

// SimilaritySearch performs a mock similarity search
func (m *MockVectorStore) SimilaritySearch(ctx context.Context, query string, numDocuments int, options ...vectorstores.Option) ([]schema.Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// For mock, just return the most recent documents
	if numDocuments > len(m.documents) {
		numDocuments = len(m.documents)
	}

	// Return documents in reverse order (most recent first)
	results := make([]schema.Document, 0, numDocuments)
	for i := len(m.documents) - 1; i >= 0 && len(results) < numDocuments; i-- {
		results = append(results, m.documents[i])
	}

	return results, nil
}

// Clear removes all documents from the store
func (m *MockVectorStore) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.documents = []schema.Document{}
	m.embeddings = make(map[string][]float32)
	m.idCounter = 0
	return nil
}

// DocumentCount returns the number of documents in the store
func (m *MockVectorStore) DocumentCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.documents)
}

// GetDocuments returns all documents (for testing purposes)
func (m *MockVectorStore) GetDocuments() []schema.Document {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid external modifications
	docs := make([]schema.Document, len(m.documents))
	copy(docs, m.documents)
	return docs
}

// AsRetriever returns a retriever for the mock store using langchain's ToRetriever
func (m *MockVectorStore) AsRetriever(k int) vectorstores.Retriever {
	if k <= 0 {
		k = 4 // Default to 4 documents
	}
	return vectorstores.ToRetriever(m, k)
}
