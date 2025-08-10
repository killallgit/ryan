package embeddings

import (
	"context"
)

// Embedder defines the interface for creating embeddings
type Embedder interface {
	// EmbedText creates an embedding for a single text
	EmbedText(ctx context.Context, text string) ([]float32, error)

	// EmbedTexts creates embeddings for multiple texts
	EmbedTexts(ctx context.Context, texts []string) ([][]float32, error)

	// GetDimensions returns the dimensionality of the embeddings
	GetDimensions() int

	// Close releases any resources
	Close() error
}

// Config contains configuration for embedders
type Config struct {
	// Provider name (e.g., "ollama", "openai")
	Provider string

	// Model name for embeddings
	Model string

	// API endpoint (if applicable)
	Endpoint string

	// API key (if applicable)
	APIKey string

	// Additional provider-specific options
	Options map[string]interface{}
}
