package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/killallgit/ryan/pkg/models"
)

func TestToolCompatibilityDisplay(t *testing.T) {
	tests := []struct {
		modelName           string
		expectedTool        string
		shouldHaveCheckmark bool
	}{
		{
			modelName:           "llama3.1:8b",
			expectedTool:        "Excellent ✓",
			shouldHaveCheckmark: true,
		},
		{
			modelName:           "qwen2.5:7b",
			expectedTool:        "Excellent ✓",
			shouldHaveCheckmark: true,
		},
		{
			modelName:           "mistral:7b",
			expectedTool:        "Good ✓",
			shouldHaveCheckmark: true,
		},
		{
			modelName:           "gemma:7b",
			expectedTool:        "None",
			shouldHaveCheckmark: false,
		},
		{
			modelName:           "unknown-model:latest",
			expectedTool:        "Unknown",
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

func TestProgressBarFormatting(t *testing.T) {
	tests := []struct {
		progress     float64
		expectedBar  string
		expectedChar string
	}{
		{
			progress:     0.0,
			expectedChar: "░",
		},
		{
			progress:     50.0,
			expectedChar: "█",
		},
		{
			progress:     100.0,
			expectedChar: "█",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%.1f%%", test.progress), func(t *testing.T) {
			// Test progress bar calculation
			barWidth := 40
			filledWidth := int(test.progress / 100.0 * float64(barWidth))
			emptyWidth := barWidth - filledWidth

			bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", emptyWidth)

			if test.progress == 0.0 && !strings.Contains(bar, test.expectedChar) {
				t.Errorf("Expected progress bar to contain %q for 0%% progress", test.expectedChar)
			}

			if test.progress > 0.0 && test.progress < 100.0 {
				if !strings.Contains(bar, "█") || !strings.Contains(bar, "░") {
					t.Errorf("Expected progress bar to contain both filled and empty characters for %.1f%% progress", test.progress)
				}
			}

			if test.progress == 100.0 && strings.Contains(bar, "░") {
				t.Errorf("Expected progress bar to not contain empty characters for 100%% progress")
			}
		})
	}
}

func TestModelValidation(t *testing.T) {
	tests := []struct {
		modelName string
		valid     bool
	}{
		{"llama3.1:8b", true},
		{"", false},
		{"   ", false},
		{"model-with-spaces", true},
	}

	for _, test := range tests {
		t.Run(test.modelName, func(t *testing.T) {
			trimmed := strings.TrimSpace(test.modelName)
			isValid := trimmed != ""

			if isValid != test.valid {
				t.Errorf("Expected validation of %q to be %v, got %v", test.modelName, test.valid, isValid)
			}
		})
	}
}
