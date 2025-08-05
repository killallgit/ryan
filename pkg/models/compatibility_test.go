package models

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetModelInfo(t *testing.T) {
	tests := []struct {
		modelName           string
		expectedSupport     bool
		expectedRecommended bool
	}{
		// Known excellent models
		{"llama3.1:8b", true, true},
		{"qwen2.5:7b", true, true},
		{"qwen2.5-coder:1.5b", true, true},

		// Known good models
		{"llama3.2:3b", true, true},
		{"mistral:7b", true, true},

		// Known unsupported models
		{"gemma:2b", false, false},
		{"gemma:7b", false, false},

		// Models with version variations
		{"llama3.1:8b-base", true, true},
		{"llama3.1:8b-instruct", true, true},

		// Unknown models that should be inferred
		{"llama3.4:unknown", true, true},           // Should infer from llama3 family
		{"qwen4:test", true, true},                 // Should infer from qwen family
		{"phi:3b", false, false},                   // Should infer no support
		{"completely-unknown-model", false, false}, // Should be unknown
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			info := GetModelInfo(tt.modelName)

			isSupported := IsToolCompatible(tt.modelName)
			if isSupported != tt.expectedSupport {
				t.Errorf("IsToolCompatible(%s) = %v, want %v", tt.modelName, isSupported, tt.expectedSupport)
			}

			isRecommended := IsRecommendedForTools(tt.modelName)
			if isRecommended != tt.expectedRecommended {
				t.Errorf("IsRecommendedForTools(%s) = %v, want %v", tt.modelName, isRecommended, tt.expectedRecommended)
			}

			// Basic validation that we got some info back
			if info.Name == "" {
				t.Errorf("GetModelInfo(%s) returned empty name", tt.modelName)
			}
		})
	}
}

func TestGetRecommendedModels(t *testing.T) {
	recommended := GetRecommendedModels()

	if len(recommended) == 0 {
		t.Error("GetRecommendedModels() returned empty list")
	}

	// Check that some known good models are in the list
	expectedModels := []string{"llama3.1:8b", "qwen2.5:7b", "qwen2.5-coder:1.5b"}
	for _, expected := range expectedModels {
		found := false
		for _, actual := range recommended {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected model %s not found in recommended models", expected)
		}
	}
}

func TestGetModelsByCompatibility(t *testing.T) {
	byCompatibility := GetModelsByCompatibility()

	// Should have models in different compatibility categories
	if len(byCompatibility[ToolCompatibilityExcellent]) == 0 {
		t.Error("No models marked as excellent compatibility")
	}

	if len(byCompatibility[ToolCompatibilityNone]) == 0 {
		t.Error("No models marked as no compatibility (should include gemma models)")
	}

	// Check that gemma models are in the none category
	noneModels := byCompatibility[ToolCompatibilityNone]
	foundGemma := false
	for _, model := range noneModels {
		if model == "gemma:2b" || model == "gemma:7b" {
			foundGemma = true
			break
		}
	}
	if !foundGemma {
		t.Error("Gemma models should be in ToolCompatibilityNone category")
	}
}

func TestNormalizeModelName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"llama3.1:8b", "llama3.1:8b"},
		{"llama3.1:8b-base", "llama3.1:8b"},
		{"llama3.1:8b-instruct", "llama3.1:8b"},
		{"QWEN2.5:7B", "qwen2.5:7b"},
		{"  mistral:7b  ", "mistral:7b"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeModelName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeModelName(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractBaseModelName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"llama3.1:8b", "llama3.1"},
		{"qwen2.5-coder:1.5b", "qwen2.5-coder"},
		{"mistral", "mistral"},
		{"model:version:tag", "model"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractBaseModelName(tt.input)
			if result != tt.expected {
				t.Errorf("extractBaseModelName(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToolCompatibility_String(t *testing.T) {
	tests := []struct {
		tc       ToolCompatibility
		expected string
	}{
		{ToolCompatibilityUnknown, "Unknown"},
		{ToolCompatibilityNone, "None"},
		{ToolCompatibilityBasic, "Basic"},
		{ToolCompatibilityGood, "Good"},
		{ToolCompatibilityExcellent, "Excellent"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.tc.String()
			if result != tt.expected {
				t.Errorf("ToolCompatibility(%d).String() = %s, want %s", tt.tc, result, tt.expected)
			}
		})
	}
}

func TestVersionSupportsTools(t *testing.T) {
	tests := []struct {
		version  string
		expected bool
	}{
		// Supported versions
		{"1.0.0", true},
		{"1.5.2", true},
		{"2.0.0", true},
		{"0.4.0", true},
		{"0.4.5", true},
		{"0.10.0", true},

		// Unsupported versions
		{"0.3.9", false},
		{"0.2.0", false},
		{"0.1.0", false},

		// Invalid version strings
		{"invalid", false},
		{"1.x.y", false},
		{"v1.0.0", false}, // No 'v' prefix support
		{"", false},
		{"1", false}, // Missing minor version
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := VersionSupportsTools(tt.version)
			if result != tt.expected {
				t.Errorf("VersionSupportsTools(%s) = %v, want %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestCheckOllamaVersion(t *testing.T) {
	t.Run("successful version check", func(t *testing.T) {
		// Create a test server that returns a version response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/version" {
				t.Errorf("Expected path /api/version, got %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version": "1.0.0"}`))
		}))
		defer server.Close()

		version, supported, err := CheckOllamaVersion(server.URL)
		if err != nil {
			t.Errorf("CheckOllamaVersion() returned error: %v", err)
		}
		if version != "1.0.0" {
			t.Errorf("CheckOllamaVersion() version = %s, want 1.0.0", version)
		}
		if !supported {
			t.Errorf("CheckOllamaVersion() supported = %v, want true", supported)
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version": "0.3.0"}`))
		}))
		defer server.Close()

		version, supported, err := CheckOllamaVersion(server.URL)
		if err != nil {
			t.Errorf("CheckOllamaVersion() returned error: %v", err)
		}
		if version != "0.3.0" {
			t.Errorf("CheckOllamaVersion() version = %s, want 0.3.0", version)
		}
		if supported {
			t.Errorf("CheckOllamaVersion() supported = %v, want false", supported)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		_, _, err := CheckOllamaVersion(server.URL)
		if err == nil {
			t.Error("CheckOllamaVersion() should return error for server error")
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{invalid json`))
		}))
		defer server.Close()

		_, _, err := CheckOllamaVersion(server.URL)
		if err == nil {
			t.Error("CheckOllamaVersion() should return error for invalid JSON")
		}
	})

	t.Run("network error", func(t *testing.T) {
		// Use an invalid URL to trigger network error
		_, _, err := CheckOllamaVersion("http://invalid-host:99999")
		if err == nil {
			t.Error("CheckOllamaVersion() should return error for network error")
		}
	})
}
