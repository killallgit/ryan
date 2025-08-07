package models

import (
	"context"
	"strings"
	"sync"
)

// ToolCompatibility represents the tool calling support level for a model
type ToolCompatibility int

const (
	// ToolCompatibilityUnknown indicates we haven't tested this model
	ToolCompatibilityUnknown ToolCompatibility = iota
	// ToolCompatibilityNone indicates the model doesn't support tool calling
	ToolCompatibilityNone
	// ToolCompatibilityBasic indicates basic tool calling support
	ToolCompatibilityBasic
	// ToolCompatibilityGood indicates good tool calling support with most features
	ToolCompatibilityGood
	// ToolCompatibilityExcellent indicates excellent tool calling support
	ToolCompatibilityExcellent
)

func (tc ToolCompatibility) String() string {
	switch tc {
	case ToolCompatibilityNone:
		return "None"
	case ToolCompatibilityBasic:
		return "Basic"
	case ToolCompatibilityGood:
		return "Good"
	case ToolCompatibilityExcellent:
		return "Excellent"
	default:
		return "Unknown"
	}
}

// ModelInfo contains information about a model's capabilities
type ModelInfo struct {
	Name                string
	ToolCompatibility   ToolCompatibility
	RecommendedForTools bool
	Notes               string
}

// Global model provider instance
var (
	defaultProvider ModelProvider
	providerMutex   sync.RWMutex
)

// SetDefaultProvider sets the default model provider
func SetDefaultProvider(provider ModelProvider) {
	providerMutex.Lock()
	defer providerMutex.Unlock()
	defaultProvider = provider
}

// GetDefaultProvider returns the default model provider
func GetDefaultProvider() ModelProvider {
	providerMutex.RLock()
	defer providerMutex.RUnlock()
	return defaultProvider
}

// GetModelInfo returns information about a model's capabilities
// This function maintains backward compatibility with existing code
func GetModelInfo(modelName string) ModelInfo {
	// If we have a provider, use it
	provider := GetDefaultProvider()
	if provider != nil {
		ctx := context.Background()
		info, err := provider.GetModelInfo(ctx, modelName)
		if err == nil && info != nil {
			return *info
		}
	}

	// Fallback to inference if no provider or error
	return ModelInfo{
		Name:                modelName,
		ToolCompatibility:   InferToolCompatibility(modelName),
		RecommendedForTools: InferToolCompatibility(modelName) >= ToolCompatibilityGood,
		Notes:               "Compatibility inferred from model name",
	}
}

// IsToolCompatible checks if a model is compatible with tool calling
func IsToolCompatible(modelName string) bool {
	modelInfo := GetModelInfo(modelName)
	return modelInfo.ToolCompatibility >= ToolCompatibilityBasic
}

// IsRecommendedForTools checks if a model is recommended for tool calling
func IsRecommendedForTools(modelName string) bool {
	modelInfo := GetModelInfo(modelName)
	return modelInfo.RecommendedForTools
}

// GetRecommendedModels returns a list of models recommended for tool calling
func GetRecommendedModels(ctx context.Context) ([]ModelInfo, error) {
	provider := GetDefaultProvider()
	if provider == nil {
		// Return empty list if no provider
		return []ModelInfo{}, nil
	}

	allModels, err := provider.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	// Filter for recommended models
	recommended := make([]ModelInfo, 0)
	for _, model := range allModels {
		if model.RecommendedForTools {
			recommended = append(recommended, model)
		}
	}

	return recommended, nil
}

// GetToolCompatibleModels returns all models that support tool calling
func GetToolCompatibleModels(ctx context.Context) ([]ModelInfo, error) {
	provider := GetDefaultProvider()
	if provider == nil {
		// Return empty list if no provider
		return []ModelInfo{}, nil
	}

	allModels, err := provider.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	// Filter for tool-compatible models
	compatible := make([]ModelInfo, 0)
	for _, model := range allModels {
		if model.ToolCompatibility >= ToolCompatibilityBasic {
			compatible = append(compatible, model)
		}
	}

	return compatible, nil
}

// RefreshModelCache refreshes the model cache if a provider is set
func RefreshModelCache(ctx context.Context) error {
	provider := GetDefaultProvider()
	if provider == nil {
		return nil // No provider, nothing to refresh
	}

	return provider.RefreshCache(ctx)
}

// NormalizeModelName normalizes a model name for comparison
func NormalizeModelName(modelName string) string {
	// Trim spaces and convert to lowercase
	normalized := strings.TrimSpace(strings.ToLower(modelName))

	// Remove common tag suffixes if present
	normalized = strings.TrimSuffix(normalized, ":latest")

	return normalized
}

// CompareModelNames checks if two model names refer to the same model
func CompareModelNames(name1, name2 string) bool {
	return NormalizeModelName(name1) == NormalizeModelName(name2)
}
