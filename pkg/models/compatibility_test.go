package models

import (
	"context"
	"testing"
)

func TestGetModelInfo(t *testing.T) {
	tests := []struct {
		modelName           string
		expectedSupport     bool
		expectedRecommended bool
		minCompatibility    ToolCompatibility
	}{
		// Known excellent models based on inference
		{"llama3.1:8b", true, true, ToolCompatibilityExcellent},
		{"llama3.2:3b", true, true, ToolCompatibilityExcellent},
		{"qwen2.5:7b", true, true, ToolCompatibilityExcellent},
		{"qwen2.5-coder:1.5b", true, true, ToolCompatibilityExcellent},
		{"qwen3:latest", true, true, ToolCompatibilityExcellent},

		// Known good models
		{"mistral:7b", true, true, ToolCompatibilityGood},
		{"deepseek-coder:6.7b", true, true, ToolCompatibilityGood},
		{"deepseek-r1:7b", true, true, ToolCompatibilityGood},

		// Basic support
		{"llama2:7b", true, false, ToolCompatibilityBasic},
		{"qwen:7b", true, false, ToolCompatibilityBasic},

		// Models with no tool support
		{"gemma:2b", false, false, ToolCompatibilityNone},
		{"gemma:7b", false, false, ToolCompatibilityNone},
		{"phi:3b", false, false, ToolCompatibilityNone},
		{"completely-unknown-model", false, false, ToolCompatibilityUnknown},

		// Models with version variations
		{"llama3.1:8b-base", true, true, ToolCompatibilityExcellent},
		{"llama3.1:8b-instruct", true, true, ToolCompatibilityExcellent},
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

			// Check minimum compatibility level
			if info.ToolCompatibility < tt.minCompatibility {
				t.Errorf("GetModelInfo(%s).ToolCompatibility = %v, want at least %v",
					tt.modelName, info.ToolCompatibility, tt.minCompatibility)
			}

			// Basic validation that we got some info back
			if info.Name == "" {
				t.Errorf("GetModelInfo(%s) returned empty name", tt.modelName)
			}
		})
	}
}

func TestInferModelFamily(t *testing.T) {
	tests := []struct {
		modelName string
		expected  string
	}{
		{"llama3.1:8b", "llama"},
		{"LLaMA3.2:3b", "llama"},
		{"qwen2.5:7b", "qwen"},
		{"Qwen2.5-coder:1.5b", "qwen"},
		{"mistral:7b", "mistral"},
		{"Mistral-7B-Instruct", "mistral"},
		{"deepseek-r1:latest", "deepseek"},
		{"gemma:2b", "gemma"},
		{"phi:3b", "phi"},
		{"codellama:7b", "codellama"},
		{"starcoder:15b", "starcoder"},
		{"wizardcoder:13b", "wizardcoder"},
		{"unknown-model:test", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			result := InferModelFamily(tt.modelName)
			if result != tt.expected {
				t.Errorf("InferModelFamily(%s) = %s, want %s", tt.modelName, result, tt.expected)
			}
		})
	}
}

func TestInferToolCompatibility(t *testing.T) {
	tests := []struct {
		modelName string
		expected  ToolCompatibility
	}{
		// Llama 3.1+ should be excellent
		{"llama3.1:8b", ToolCompatibilityExcellent},
		{"llama3.2:3b", ToolCompatibilityExcellent},
		{"llama3.3:70b", ToolCompatibilityExcellent},

		// Llama 3 should be good
		{"llama3:8b", ToolCompatibilityGood},

		// Older Llama should be basic
		{"llama2:7b", ToolCompatibilityBasic},

		// Qwen2.5 and Qwen3 should be excellent
		{"qwen2.5:7b", ToolCompatibilityExcellent},
		{"qwen3:latest", ToolCompatibilityExcellent},
		{"qwen2.5-coder:1.5b", ToolCompatibilityExcellent},

		// Qwen2 should be good
		{"qwen2:7b", ToolCompatibilityGood},

		// Older Qwen should be basic
		{"qwen:7b", ToolCompatibilityBasic},

		// Mistral should be good
		{"mistral:7b", ToolCompatibilityGood},
		{"mixtral:8x7b", ToolCompatibilityExcellent},

		// DeepSeek should be good
		{"deepseek-r1:latest", ToolCompatibilityGood},
		{"deepseek-coder:6.7b", ToolCompatibilityGood},

		// Code models should be good
		{"codellama:7b", ToolCompatibilityGood},
		{"starcoder:15b", ToolCompatibilityGood},

		// Models with no tool support
		{"gemma:2b", ToolCompatibilityNone},
		{"phi:3b", ToolCompatibilityNone},
		{"unknown-model", ToolCompatibilityUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			result := InferToolCompatibility(tt.modelName)
			if result != tt.expected {
				t.Errorf("InferToolCompatibility(%s) = %v, want %v", tt.modelName, result, tt.expected)
			}
		})
	}
}

func TestNormalizeModelName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"llama3.1:8b", "llama3.1:8b"},
		{"llama3.1:latest", "llama3.1"},
		{"QWEN2.5:7B", "qwen2.5:7b"},
		{"mistral:latest", "mistral"},
		{"  mistral:7b  ", "mistral:7b"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeModelName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeModelName(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCompareModelNames(t *testing.T) {
	tests := []struct {
		name1    string
		name2    string
		expected bool
	}{
		{"llama3.1:8b", "llama3.1:8b", true},
		{"llama3.1:8b", "LLAMA3.1:8B", true},
		{"llama3.1:latest", "llama3.1", true},
		{"llama3.1", "llama3.1:latest", true},
		{"qwen2.5:7b", "qwen2.5:7b", true},
		{"llama3.1:8b", "llama3.2:8b", false},
		{"mistral", "mixtral", false},
	}

	for _, tt := range tests {
		t.Run(tt.name1+"_vs_"+tt.name2, func(t *testing.T) {
			result := CompareModelNames(tt.name1, tt.name2)
			if result != tt.expected {
				t.Errorf("CompareModelNames(%s, %s) = %v, want %v", tt.name1, tt.name2, result, tt.expected)
			}
		})
	}
}

func TestGetRecommendedModels(t *testing.T) {
	// Without a provider set, should return empty list
	ctx := context.Background()
	recommended, err := GetRecommendedModels(ctx)
	if err != nil {
		t.Errorf("GetRecommendedModels() returned error: %v", err)
	}

	// Should return empty list when no provider is set
	if len(recommended) != 0 {
		t.Errorf("GetRecommendedModels() without provider should return empty list, got %d models", len(recommended))
	}
}

func TestGetToolCompatibleModels(t *testing.T) {
	// Without a provider set, should return empty list
	ctx := context.Background()
	compatible, err := GetToolCompatibleModels(ctx)
	if err != nil {
		t.Errorf("GetToolCompatibleModels() returned error: %v", err)
	}

	// Should return empty list when no provider is set
	if len(compatible) != 0 {
		t.Errorf("GetToolCompatibleModels() without provider should return empty list, got %d models", len(compatible))
	}
}

func TestRefreshModelCache(t *testing.T) {
	// Without a provider set, should not error
	ctx := context.Background()
	err := RefreshModelCache(ctx)
	if err != nil {
		t.Errorf("RefreshModelCache() returned error: %v", err)
	}
}
