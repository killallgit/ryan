package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OllamaEmbedder implements Embedder using Ollama's embedding API
type OllamaEmbedder struct {
	endpoint   string
	model      string
	dimensions int
	client     *http.Client
}

// OllamaConfig contains configuration for OllamaEmbedder
type OllamaConfig struct {
	// Endpoint is the Ollama API endpoint
	Endpoint string

	// Model is the embedding model to use (e.g., "nomic-embed-text")
	Model string

	// Timeout for API requests
	Timeout time.Duration
}

// NewOllamaEmbedder creates a new Ollama embedder
func NewOllamaEmbedder(config OllamaConfig) (*OllamaEmbedder, error) {
	if config.Endpoint == "" {
		// Check OLLAMA_HOST environment variable first
		if ollamaHost := os.Getenv("OLLAMA_HOST"); ollamaHost != "" {
			config.Endpoint = ollamaHost
		} else {
			config.Endpoint = "http://localhost:11434"
		}
	}

	if config.Model == "" {
		// Check OLLAMA_DEFAULT_MODEL for embedding model override
		if ollamaModel := os.Getenv("OLLAMA_EMBED_MODEL"); ollamaModel != "" {
			config.Model = ollamaModel
		} else {
			config.Model = "nomic-embed-text"
		}
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	embedder := &OllamaEmbedder{
		endpoint: config.Endpoint,
		model:    config.Model,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}

	// Get model dimensions by creating a test embedding
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testEmbed, err := embedder.EmbedText(ctx, "test")
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding dimensions: %w", err)
	}
	embedder.dimensions = len(testEmbed)

	return embedder, nil
}

// EmbedText creates an embedding for a single text
func (e *OllamaEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.EmbedTexts(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}

// EmbedTexts creates embeddings for multiple texts
func (e *OllamaEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))

	// Ollama API typically handles one text at a time for embeddings
	for i, text := range texts {
		embedding, err := e.embedSingle(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		results[i] = embedding
	}

	return results, nil
}

// embedSingle creates an embedding for a single text using Ollama API
func (e *OllamaEmbedder) embedSingle(ctx context.Context, text string) ([]float32, error) {
	// Prepare request
	reqBody := map[string]interface{}{
		"model":  e.model,
		"prompt": text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", e.endpoint+"/api/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert float64 to float32
	embedding := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// GetDimensions returns the dimensionality of the embeddings
func (e *OllamaEmbedder) GetDimensions() int {
	return e.dimensions
}

// Close releases any resources
func (e *OllamaEmbedder) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}

// NewOllamaEmbedderForTesting creates a new Ollama embedder for integration tests.
// This version PANICS if OLLAMA_HOST is not set and no endpoint is provided.
// This ensures integration tests don't accidentally fall back to localhost.
//
// Use this function in integration tests where OLLAMA_HOST must be set.
// Use NewOllamaEmbedder for production code where fallback to localhost is acceptable.
func NewOllamaEmbedderForTesting(config OllamaConfig) (*OllamaEmbedder, error) {
	if config.Endpoint == "" {
		// In testing, we REQUIRE OLLAMA_HOST to be set - no fallback to localhost
		ollamaHost := os.Getenv("OLLAMA_HOST")
		if ollamaHost == "" {
			panic("OLLAMA_HOST environment variable MUST be set for integration tests - no fallback allowed")
		}
		config.Endpoint = ollamaHost
	}

	// Now create the embedder directly without going through NewOllamaEmbedder
	// to avoid any fallback logic
	if config.Model == "" {
		// Check OLLAMA_DEFAULT_MODEL for embedding model override
		if ollamaModel := os.Getenv("OLLAMA_EMBED_MODEL"); ollamaModel != "" {
			config.Model = ollamaModel
		} else {
			config.Model = "nomic-embed-text"
		}
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	embedder := &OllamaEmbedder{
		endpoint: config.Endpoint,
		model:    config.Model,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}

	// Get model dimensions by creating a test embedding
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testEmbed, err := embedder.EmbedText(ctx, "test")
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding dimensions from %s: %w", config.Endpoint, err)
	}
	embedder.dimensions = len(testEmbed)

	return embedder, nil
}
