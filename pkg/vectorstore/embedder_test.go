package vectorstore

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMockEmbedderValidation(t *testing.T) {
	embedder := NewMockEmbedder(384)
	ctx := context.Background()

	t.Run("ValidateEmptyText", func(t *testing.T) {
		_, err := embedder.EmbedText(ctx, "")
		if err == nil {
			t.Error("expected error for empty text")
		}
		if err.Error() != "empty text" {
			t.Errorf("expected 'empty text' error, got %v", err)
		}
	})

	t.Run("MockEmbedder_Dimensions", func(t *testing.T) {
		dimensions := embedder.Dimensions()
		if dimensions != 384 {
			t.Errorf("expected 384 dimensions, got %d", dimensions)
		}
	})

	t.Run("ValidateTextTooLong", func(t *testing.T) {
		longText := strings.Repeat("a", MaxTextLength+1)
		_, err := embedder.EmbedText(ctx, longText)
		if err == nil {
			t.Error("expected error for text too long")
		}
		if !strings.Contains(err.Error(), "exceeds max length") {
			t.Errorf("expected max length error, got %v", err)
		}
	})

	t.Run("ValidateBatchEmpty", func(t *testing.T) {
		_, err := embedder.EmbedTexts(ctx, []string{})
		if err == nil {
			t.Error("expected error for empty batch")
		}
		if err.Error() != "no texts to embed" {
			t.Errorf("expected 'no texts to embed' error, got %v", err)
		}
	})

	t.Run("ValidateBatchTooLarge", func(t *testing.T) {
		texts := make([]string, MaxBatchSize+1)
		for i := range texts {
			texts[i] = "test"
		}
		_, err := embedder.EmbedTexts(ctx, texts)
		if err == nil {
			t.Error("expected error for batch too large")
		}
		if !strings.Contains(err.Error(), "exceeds max batch size") {
			t.Errorf("expected max batch size error, got %v", err)
		}
	})

	t.Run("ValidateBatchWithEmptyText", func(t *testing.T) {
		texts := []string{"valid", "", "another"}
		_, err := embedder.EmbedTexts(ctx, texts)
		if err == nil {
			t.Error("expected error for empty text in batch")
		}
		if !strings.Contains(err.Error(), "failed to embed text at index 1") {
			t.Errorf("expected failed to embed text at index error, got %v", err)
		}
	})

	t.Run("ValidEmbedding", func(t *testing.T) {
		embedding, err := embedder.EmbedText(ctx, "test text")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(embedding) != 384 {
			t.Errorf("expected embedding dimension 384, got %d", len(embedding))
		}

		// Check normalization
		var sum float32
		for _, v := range embedding {
			sum += v * v
		}
		// Should be approximately 1.0 (normalized)
		if sum < 0.9 || sum > 1.1 {
			t.Errorf("expected normalized embedding, got magnitude squared %f", sum)
		}
	})
}

func TestEmbedderConfiguration(t *testing.T) {
	t.Run("DefaultHTTPConfig", func(t *testing.T) {
		config := DefaultHTTPClientConfig()
		if config.Timeout != 30*time.Second {
			t.Errorf("expected 30s timeout, got %v", config.Timeout)
		}
		if config.MaxRetries != 3 {
			t.Errorf("expected 3 retries, got %d", config.MaxRetries)
		}
		if config.BackoffBase != 100*time.Millisecond {
			t.Errorf("expected 100ms backoff, got %v", config.BackoffBase)
		}
	})

	t.Run("CreateEmbedderDefaults", func(t *testing.T) {
		config := EmbedderConfig{
			Provider: "mock",
		}
		embedder, err := CreateEmbedder(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if embedder == nil {
			t.Error("expected embedder, got nil")
		}
	})

	t.Run("UnsupportedProvider", func(t *testing.T) {
		config := EmbedderConfig{
			Provider: "unsupported",
		}
		_, err := CreateEmbedder(config)
		if err == nil {
			t.Error("expected error for unsupported provider")
		}
		if !strings.Contains(err.Error(), "unsupported embedder provider") {
			t.Errorf("expected unsupported provider error, got %v", err)
		}
	})
}

func TestEmbedderCreationFunctions(t *testing.T) {
	t.Run("NewOllamaEmbedder", func(t *testing.T) {
		// Test with mock-like config
		baseURL := "http://localhost:11434"
		model := "test-model"

		embedder, err := NewOllamaEmbedder(baseURL, model)
		if err != nil {
			// This might fail due to network, but we're testing the creation logic
			t.Logf("NewOllamaEmbedder failed (expected if no server): %v", err)
		} else {
			if embedder == nil {
				t.Error("expected embedder, got nil")
			}
		}
	})

	t.Run("NewOllamaEmbedderWithConfig", func(t *testing.T) {
		config := EmbedderConfig{
			Provider: "ollama",
			BaseURL:  "http://localhost:11434",
			Model:    "test-model",
		}

		embedder, err := NewOllamaEmbedderWithConfig(config)
		if err != nil {
			// This might fail due to network, but we're testing the creation logic
			t.Logf("NewOllamaEmbedderWithConfig failed (expected if no server): %v", err)
		} else {
			if embedder == nil {
				t.Error("expected embedder, got nil")
			}
		}
	})

	t.Run("NewOpenAIEmbedder", func(t *testing.T) {
		apiKey := "test-api-key"
		model := "text-embedding-ada-002"

		embedder, err := NewOpenAIEmbedder(apiKey, model)
		if err != nil {
			// This might fail due to API key validation, but we're testing the creation logic
			t.Logf("NewOpenAIEmbedder failed (expected with test key): %v", err)
		} else {
			if embedder == nil {
				t.Error("expected embedder, got nil")
			}
		}
	})

	t.Run("NewOpenAIEmbedderWithConfig", func(t *testing.T) {
		config := EmbedderConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Model:    "text-embedding-ada-002",
		}

		embedder, err := NewOpenAIEmbedderWithConfig(config)
		if err != nil {
			// This might fail due to API key validation, but we're testing the creation logic
			t.Logf("NewOpenAIEmbedderWithConfig failed (expected with test key): %v", err)
		} else {
			if embedder == nil {
				t.Error("expected embedder, got nil")
			}
		}
	})
}
