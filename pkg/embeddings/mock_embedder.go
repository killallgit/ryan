package embeddings

import (
	"context"
	"hash/fnv"
)

// MockEmbedder is a mock implementation of Embedder for testing
type MockEmbedder struct {
	dimensions int
	calls      []string // Track calls for testing
}

// NewMockEmbedder creates a new mock embedder
func NewMockEmbedder(dimensions int) *MockEmbedder {
	if dimensions == 0 {
		dimensions = 384 // Default dimensions
	}
	return &MockEmbedder{
		dimensions: dimensions,
		calls:      make([]string, 0),
	}
}

// EmbedText creates a deterministic mock embedding for a single text
func (m *MockEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	m.calls = append(m.calls, text)

	// Generate deterministic embedding based on text hash
	h := fnv.New32a()
	h.Write([]byte(text))
	seed := h.Sum32()

	embedding := make([]float32, m.dimensions)
	for i := range embedding {
		// Generate pseudo-random values based on seed
		seed = seed*1664525 + 1013904223                   // Linear congruential generator
		embedding[i] = (float32(seed%1000) / 1000.0) - 0.5 // Range [-0.5, 0.5]
	}

	return embedding, nil
}

// EmbedTexts creates mock embeddings for multiple texts
func (m *MockEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := m.EmbedText(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = embedding
	}
	return results, nil
}

// GetDimensions returns the dimensionality of the embeddings
func (m *MockEmbedder) GetDimensions() int {
	return m.dimensions
}

// Close releases any resources
func (m *MockEmbedder) Close() error {
	return nil
}

// GetCalls returns the list of texts that were embedded (for testing)
func (m *MockEmbedder) GetCalls() []string {
	return m.calls
}

// Reset clears the call history (for testing)
func (m *MockEmbedder) Reset() {
	m.calls = make([]string, 0)
}
