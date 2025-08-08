package tokens

import (
	"testing"
)

func TestNewTokenCounter(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		wantErr   bool
	}{
		{
			name:      "GPT-4 model",
			modelName: "gpt-4",
			wantErr:   false,
		},
		{
			name:      "GPT-3.5 model",
			modelName: "gpt-3.5-turbo",
			wantErr:   false,
		},
		{
			name:      "Ollama model (fallback)",
			modelName: "qwen3:latest",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter, err := NewTokenCounter(tt.modelName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTokenCounter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && counter == nil {
				t.Error("NewTokenCounter() returned nil counter without error")
			}
		})
	}
}

func TestCountTokens(t *testing.T) {
	counter, err := NewTokenCounter("gpt-4")
	if err != nil {
		t.Fatalf("Failed to create token counter: %v", err)
	}

	tests := []struct {
		name     string
		text     string
		minCount int // Minimum expected tokens
		maxCount int // Maximum expected tokens
	}{
		{
			name:     "Simple text",
			text:     "Hello, world!",
			minCount: 2,
			maxCount: 5,
		},
		{
			name:     "Longer text",
			text:     "The quick brown fox jumps over the lazy dog.",
			minCount: 8,
			maxCount: 12,
		},
		{
			name:     "Empty text",
			text:     "",
			minCount: 0,
			maxCount: 0,
		},
		{
			name:     "Single word",
			text:     "Test",
			minCount: 1,
			maxCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := counter.CountTokens(tt.text)
			if count < tt.minCount || count > tt.maxCount {
				t.Errorf("CountTokens() = %v, want between %v and %v", count, tt.minCount, tt.maxCount)
			}
		})
	}
}

func TestCountMessages(t *testing.T) {
	counter, err := NewTokenCounter("gpt-4")
	if err != nil {
		t.Fatalf("Failed to create token counter: %v", err)
	}

	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello!"},
		{Role: "assistant", Content: "Hi there! How can I help you today?"},
	}

	count := counter.CountMessages(messages)

	// We expect at least the sum of individual token counts plus overhead
	minExpected := 10 // Very conservative minimum
	maxExpected := 50 // Liberal maximum to account for variations

	if count < minExpected || count > maxExpected {
		t.Errorf("CountMessages() = %v, want between %v and %v", count, minExpected, maxExpected)
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		minCount int
		maxCount int
	}{
		{
			name:     "Short text",
			text:     "Hello world",
			minCount: 2,
			maxCount: 3,
		},
		{
			name:     "Medium text",
			text:     "The quick brown fox jumps over the lazy dog",
			minCount: 9,
			maxCount: 12,
		},
		{
			name:     "Long word",
			text:     "Supercalifragilisticexpialidocious",
			minCount: 3,
			maxCount: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := estimateTokens(tt.text)
			if count < tt.minCount || count > tt.maxCount {
				t.Errorf("estimateTokens() = %v, want between %v and %v", count, tt.minCount, tt.maxCount)
			}
		})
	}
}

func TestGetEncodingForModel(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{"GPT-4", "gpt-4", "cl100k_base"},
		{"GPT-3.5", "gpt-3.5-turbo", "cl100k_base"},
		{"Davinci", "text-davinci-003", "p50k_base"},
		{"Code model", "code-davinci-002", "p50k_base"},
		{"Unknown model", "unknown-model", "cl100k_base"},
		{"Ollama model", "qwen3:latest", "cl100k_base"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoding := getEncodingForModel(tt.model)
			if encoding != tt.expected {
				t.Errorf("getEncodingForModel(%s) = %s, want %s", tt.model, encoding, tt.expected)
			}
		})
	}
}
