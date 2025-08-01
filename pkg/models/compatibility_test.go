package models

import (
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
