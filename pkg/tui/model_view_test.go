package tui

import (
	"testing"

	"github.com/killallgit/ryan/pkg/models"
)

func TestToolCompatibilityDisplay(t *testing.T) {
	tests := []struct {
		modelName    string
		expectedTool string
		shouldHaveCheckmark bool
	}{
		{
			modelName:    "llama3.1:8b",
			expectedTool: "Excellent ✓",
			shouldHaveCheckmark: true,
		},
		{
			modelName:    "qwen2.5:7b", 
			expectedTool: "Excellent ✓",
			shouldHaveCheckmark: true,
		},
		{
			modelName:    "mistral:7b",
			expectedTool: "Good ✓",
			shouldHaveCheckmark: true,
		},
		{
			modelName:    "gemma:7b",
			expectedTool: "None",
			shouldHaveCheckmark: false,
		},
		{
			modelName:    "unknown-model:latest",
			expectedTool: "Unknown",
			shouldHaveCheckmark: false,
		},
	}

	for _, test := range tests {
		t.Run(test.modelName, func(t *testing.T) {
			modelInfo := models.GetModelInfo(test.modelName)
			toolsSupport := modelInfo.ToolCompatibility.String()
			if modelInfo.RecommendedForTools {
				toolsSupport += " ✓"
			}

			if toolsSupport != test.expectedTool {
				t.Errorf("Expected %q for model %q, got %q", 
					test.expectedTool, test.modelName, toolsSupport)
			}

			if modelInfo.RecommendedForTools != test.shouldHaveCheckmark {
				t.Errorf("Expected RecommendedForTools=%v for model %q, got %v",
					test.shouldHaveCheckmark, test.modelName, modelInfo.RecommendedForTools)
			}
		})
	}
}