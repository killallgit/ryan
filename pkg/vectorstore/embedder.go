package vectorstore

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// LangChainEmbedder wraps a LangChain embedder to implement our Embedder interface
type LangChainEmbedder struct {
	embedder   embeddings.Embedder
	dimensions int
}

// NewOllamaEmbedder creates an embedder using Ollama
func NewOllamaEmbedder(model string, baseURL string) (*LangChainEmbedder, error) {
	opts := []ollama.Option{
		ollama.WithModel(model),
	}

	if baseURL != "" {
		opts = append(opts, ollama.WithServerURL(baseURL))
	}

	llm, err := ollama.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama LLM: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Get dimensions by generating a test embedding
	dims, err := getEmbeddingDimensions(context.Background(), embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to determine embedding dimensions: %w", err)
	}

	return &LangChainEmbedder{
		embedder:   embedder,
		dimensions: dims,
	}, nil
}

// NewOpenAIEmbedder creates an embedder using OpenAI
func NewOpenAIEmbedder(apiKey string, model string) (*LangChainEmbedder, error) {
	opts := []openai.Option{}

	if apiKey != "" {
		opts = append(opts, openai.WithToken(apiKey))
	}

	if model != "" {
		opts = append(opts, openai.WithEmbeddingModel(model))
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI LLM: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Get dimensions by generating a test embedding
	dims, err := getEmbeddingDimensions(context.Background(), embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to determine embedding dimensions: %w", err)
	}

	return &LangChainEmbedder{
		embedder:   embedder,
		dimensions: dims,
	}, nil
}

// EmbedText generates an embedding for a single text
func (le *LangChainEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := le.embedder.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("failed to embed text: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embeddings[0], nil
}

// EmbedTexts generates embeddings for multiple texts
func (le *LangChainEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings, err := le.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to embed texts: %w", err)
	}

	return embeddings, nil
}

// Dimensions returns the embedding dimensions
func (le *LangChainEmbedder) Dimensions() int {
	return le.dimensions
}

// getEmbeddingDimensions determines the embedding dimensions by generating a test embedding
func getEmbeddingDimensions(ctx context.Context, embedder embeddings.Embedder) (int, error) {
	testEmbedding, err := embedder.EmbedDocuments(ctx, []string{"test"})
	if err != nil {
		return 0, err
	}

	if len(testEmbedding) == 0 || len(testEmbedding[0]) == 0 {
		return 0, fmt.Errorf("empty test embedding")
	}

	return len(testEmbedding[0]), nil
}

// MockEmbedder is a simple embedder for testing that returns fixed-size random embeddings
type MockEmbedder struct {
	dims int
}

// NewMockEmbedder creates a mock embedder for testing
func NewMockEmbedder(dimensions int) *MockEmbedder {
	return &MockEmbedder{dims: dimensions}
}

// EmbedText generates a mock embedding for a single text
func (me *MockEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	// Generate a deterministic embedding based on text content
	embedding := make([]float32, me.dims)

	// Simple semantic features for testing
	words := strings.Fields(strings.ToLower(text))
	wordSet := make(map[string]bool)
	for _, w := range words {
		wordSet[w] = true
	}

	// Create feature vectors based on common words
	features := map[string]int{
		"machine":      0,
		"learning":     1,
		"artificial":   2,
		"intelligence": 3,
		"vector":       4,
		"database":     5,
		"search":       6,
		"cat":          7,
		"cats":         7,
		"dog":          8,
		"dogs":         8,
		"pet":          9,
		"pets":         9,
		"animal":       10,
		"animals":      10,
		"programming":  11,
		"go":           12,
		"efficient":    13,
		"embeddings":   14,
		"store":        15,
	}

	// Initialize with small random-like values
	for i := range embedding {
		embedding[i] = float32(i%7) * 0.01
	}

	// Set features based on words present
	for word := range wordSet {
		if idx, exists := features[word]; exists && idx < me.dims {
			embedding[idx] = 0.9
			// Add some spread to nearby dimensions
			if idx > 0 {
				embedding[idx-1] = 0.6
			}
			if idx < me.dims-1 {
				embedding[idx+1] = 0.6
			}
		}
	}

	// Add text length as a feature
	if me.dims > 20 {
		embedding[20] = float32(len(text)) / 100.0
	}

	// Normalize the embedding
	var sum float32
	for _, v := range embedding {
		sum += v * v
	}
	if sum > 0 {
		norm := float32(math.Sqrt(float64(sum)))
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding, nil
}

// EmbedTexts generates mock embeddings for multiple texts
func (me *MockEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		emb, err := me.EmbedText(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings[i] = emb
	}

	return embeddings, nil
}

// Dimensions returns the embedding dimensions
func (me *MockEmbedder) Dimensions() int {
	return me.dims
}

// CreateEmbedder creates an embedder based on configuration
func CreateEmbedder(config EmbedderConfig) (Embedder, error) {
	switch config.Provider {
	case "ollama":
		model := config.Model
		if model == "" {
			model = "nomic-embed-text" // Default embedding model
		}
		return NewOllamaEmbedder(model, config.BaseURL)

	case "openai":
		model := config.Model
		if model == "" {
			model = "text-embedding-3-small" // Default OpenAI embedding model
		}
		return NewOpenAIEmbedder(config.APIKey, model)

	case "mock":
		return NewMockEmbedder(384), nil // Common embedding size

	default:
		return nil, fmt.Errorf("unsupported embedder provider: %s", config.Provider)
	}
}
