package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/ollama"
)

func TestOllamaModelProvider_ListModels(t *testing.T) {
	// Create a test server that mimics Ollama's /api/tags endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("Expected /api/tags, got %s", r.URL.Path)
		}

		response := map[string]interface{}{
			"models": []map[string]interface{}{
				{
					"name":   "llama3.1:8b",
					"size":   4661224448,
					"digest": "abc123",
					"details": map[string]interface{}{
						"parameter_size":     "8B",
						"quantization_level": "Q4_0",
						"format":             "gguf",
					},
				},
				{
					"name":   "qwen2.5:7b",
					"size":   4000000000,
					"digest": "def456",
					"details": map[string]interface{}{
						"parameter_size":     "7B",
						"quantization_level": "Q4_K_M",
						"format":             "gguf",
					},
				},
				{
					"name":   "gemma:2b",
					"size":   1500000000,
					"digest": "ghi789",
					"details": map[string]interface{}{
						"parameter_size":     "2B",
						"quantization_level": "Q8_0",
						"format":             "gguf",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create Ollama client and provider
	client := ollama.NewClient(server.URL)
	provider := NewOllamaModelProvider(client)

	// Test ListModels
	ctx := context.Background()
	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	// Check we got the expected models
	if len(models) != 3 {
		t.Errorf("Expected 3 models, got %d", len(models))
	}

	// Check model details
	expectedModels := map[string]struct {
		compat      ToolCompatibility
		recommended bool
	}{
		"llama3.1:8b": {ToolCompatibilityExcellent, true},
		"qwen2.5:7b":  {ToolCompatibilityExcellent, true},
		"gemma:2b":    {ToolCompatibilityNone, false},
	}

	for _, model := range models {
		expected, ok := expectedModels[model.Name]
		if !ok {
			t.Errorf("Unexpected model: %s", model.Name)
			continue
		}

		if model.ToolCompatibility != expected.compat {
			t.Errorf("Model %s: expected compatibility %v, got %v",
				model.Name, expected.compat, model.ToolCompatibility)
		}

		if model.RecommendedForTools != expected.recommended {
			t.Errorf("Model %s: expected recommended=%v, got %v",
				model.Name, expected.recommended, model.RecommendedForTools)
		}
	}
}

func TestOllamaModelProvider_GetModelInfo(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"models": []map[string]interface{}{
				{
					"name": "llama3.1:8b",
					"size": 4661224448,
					"details": map[string]interface{}{
						"parameter_size": "8B",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	provider := NewOllamaModelProvider(client)

	ctx := context.Background()

	// Test getting info for a model that exists
	info, err := provider.GetModelInfo(ctx, "llama3.1:8b")
	if err != nil {
		t.Fatalf("GetModelInfo failed: %v", err)
	}

	if info.Name != "llama3.1:8b" {
		t.Errorf("Expected name llama3.1:8b, got %s", info.Name)
	}

	if info.ToolCompatibility != ToolCompatibilityExcellent {
		t.Errorf("Expected excellent compatibility, got %v", info.ToolCompatibility)
	}

	// Test getting info for a model that doesn't exist (should infer)
	info2, err := provider.GetModelInfo(ctx, "unknown-model:test")
	if err != nil {
		t.Fatalf("GetModelInfo for unknown model failed: %v", err)
	}

	if info2.ToolCompatibility != ToolCompatibilityUnknown {
		t.Errorf("Expected unknown compatibility for unknown model, got %v", info2.ToolCompatibility)
	}
}

func TestOllamaModelProvider_Cache(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		response := map[string]interface{}{
			"models": []map[string]interface{}{
				{"name": "test-model", "size": 1000000},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	provider := NewOllamaModelProvider(client)
	provider.SetCacheTTL(100 * time.Millisecond)

	ctx := context.Background()

	// First call should hit the server
	_, err := provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("First ListModels failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 server call, got %d", callCount)
	}

	// Second call should use cache
	_, err = provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("Second ListModels failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected still 1 server call (cached), got %d", callCount)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call should hit server again
	_, err = provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("Third ListModels failed: %v", err)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 server calls (cache expired), got %d", callCount)
	}
}

func TestOllamaModelProvider_IsModelAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"models": []map[string]interface{}{
				{"name": "available-model", "size": 1000000},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	provider := NewOllamaModelProvider(client)

	ctx := context.Background()

	// Test available model
	available, err := provider.IsModelAvailable(ctx, "available-model")
	if err != nil {
		t.Fatalf("IsModelAvailable failed: %v", err)
	}
	if !available {
		t.Error("Expected available-model to be available")
	}

	// Test unavailable model
	available, err = provider.IsModelAvailable(ctx, "unavailable-model")
	if err != nil {
		t.Fatalf("IsModelAvailable failed: %v", err)
	}
	if available {
		t.Error("Expected unavailable-model to not be available")
	}
}
