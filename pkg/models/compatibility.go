package models

import (
	"strings"
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

// KnownModels contains our tested model compatibility data
var KnownModels = map[string]ModelInfo{
	// Llama models - generally excellent tool calling support
	"llama3.1:8b": {
		Name:                "llama3.1:8b",
		ToolCompatibility:   ToolCompatibilityExcellent,
		RecommendedForTools: true,
		Notes:               "Mature tool calling implementation, reliable for production",
	},
	"llama3.1:70b": {
		Name:                "llama3.1:70b",
		ToolCompatibility:   ToolCompatibilityExcellent,
		RecommendedForTools: true,
		Notes:               "High-quality tool calling, resource intensive",
	},
	"llama3.2:1b": {
		Name:                "llama3.2:1b",
		ToolCompatibility:   ToolCompatibilityGood,
		RecommendedForTools: true,
		Notes:               "Lightweight option with solid tool support",
	},
	"llama3.2:3b": {
		Name:                "llama3.2:3b",
		ToolCompatibility:   ToolCompatibilityGood,
		RecommendedForTools: true,
		Notes:               "Good balance of size and tool calling capability",
	},
	"llama3.3:70b": {
		Name:                "llama3.3:70b",
		ToolCompatibility:   ToolCompatibilityExcellent,
		RecommendedForTools: true,
		Notes:               "Latest Llama with enhanced tool calling",
	},

	// Qwen models - excellent for coding and math tasks
	"qwen2.5:0.5b": {
		Name:                "qwen2.5:0.5b",
		ToolCompatibility:   ToolCompatibilityBasic,
		RecommendedForTools: false,
		Notes:               "Very lightweight, limited tool calling accuracy",
	},
	"qwen2.5:1.5b": {
		Name:                "qwen2.5:1.5b",
		ToolCompatibility:   ToolCompatibilityGood,
		RecommendedForTools: true,
		Notes:               "Compact model with reasonable tool support",
	},
	"qwen2.5:3b": {
		Name:                "qwen2.5:3b",
		ToolCompatibility:   ToolCompatibilityGood,
		RecommendedForTools: true,
		Notes:               "Good tool calling with improved reasoning",
	},
	"qwen2.5:7b": {
		Name:                "qwen2.5:7b",
		ToolCompatibility:   ToolCompatibilityExcellent,
		RecommendedForTools: true,
		Notes:               "Excellent for coding tasks and tool usage",
	},
	"qwen2.5:14b": {
		Name:                "qwen2.5:14b",
		ToolCompatibility:   ToolCompatibilityExcellent,
		RecommendedForTools: true,
		Notes:               "High-quality tool calling with strong reasoning",
	},
	"qwen2.5:32b": {
		Name:                "qwen2.5:32b",
		ToolCompatibility:   ToolCompatibilityExcellent,
		RecommendedForTools: true,
		Notes:               "Top-tier tool calling performance",
	},
	"qwen2.5-coder:1.5b": {
		Name:                "qwen2.5-coder:1.5b",
		ToolCompatibility:   ToolCompatibilityGood,
		RecommendedForTools: true,
		Notes:               "Optimized for coding tasks, good tool integration",
	},
	"qwen2.5-coder:7b": {
		Name:                "qwen2.5-coder:7b",
		ToolCompatibility:   ToolCompatibilityExcellent,
		RecommendedForTools: true,
		Notes:               "Excellent for development workflows with tools",
	},
	"qwen3:8b": {
		Name:                "qwen3:8b",
		ToolCompatibility:   ToolCompatibilityExcellent,
		RecommendedForTools: true,
		Notes:               "Latest Qwen with enhanced tool calling capabilities",
	},

	// Mistral models - reliable tool calling
	"mistral:7b": {
		Name:                "mistral:7b",
		ToolCompatibility:   ToolCompatibilityGood,
		RecommendedForTools: true,
		Notes:               "Solid tool calling implementation",
	},
	"mistral-nemo": {
		Name:                "mistral-nemo",
		ToolCompatibility:   ToolCompatibilityGood,
		RecommendedForTools: true,
		Notes:               "Efficient model with good tool support",
	},
	"mistral-small": {
		Name:                "mistral-small",
		ToolCompatibility:   ToolCompatibilityGood,
		RecommendedForTools: true,
		Notes:               "Compact with reliable tool calling",
	},

	// DeepSeek models
	"deepseek-r1": {
		Name:                "deepseek-r1",
		ToolCompatibility:   ToolCompatibilityGood,
		RecommendedForTools: true,
		Notes:               "Reasoning-focused model with tool support",
	},

	// Command-R models
	"command-r": {
		Name:                "command-r",
		ToolCompatibility:   ToolCompatibilityGood,
		RecommendedForTools: true,
		Notes:               "Enterprise-focused with solid tool integration",
	},
	"command-r-plus": {
		Name:                "command-r-plus",
		ToolCompatibility:   ToolCompatibilityExcellent,
		RecommendedForTools: true,
		Notes:               "Enhanced version with excellent tool calling",
	},

	// Granite models (IBM)
	"granite3.2:8b": {
		Name:                "granite3.2:8b",
		ToolCompatibility:   ToolCompatibilityGood,
		RecommendedForTools: true,
		Notes:               "IBM model with reliable tool support",
	},

	// Models known to have limited or no tool support
	"gemma:2b": {
		Name:                "gemma:2b",
		ToolCompatibility:   ToolCompatibilityNone,
		RecommendedForTools: false,
		Notes:               "No native tool calling support",
	},
	"gemma:7b": {
		Name:                "gemma:7b",
		ToolCompatibility:   ToolCompatibilityNone,
		RecommendedForTools: false,
		Notes:               "No native tool calling support",
	},
}

// GetModelInfo returns information about a model's tool calling capabilities
func GetModelInfo(modelName string) ModelInfo {
	// Normalize model name (remove version tags if present)
	normalizedName := normalizeModelName(modelName)

	// Check exact match first
	if info, exists := KnownModels[normalizedName]; exists {
		return info
	}

	// Check partial matches for versioned models
	for knownModel, info := range KnownModels {
		if strings.HasPrefix(normalizedName, extractBaseModelName(knownModel)) {
			// Return info but update the name to match what was requested
			info.Name = modelName
			return info
		}
	}

	// Try to infer from model family
	return inferModelCapabilities(modelName)
}

// IsToolCompatible returns true if the model supports tool calling
func IsToolCompatible(modelName string) bool {
	info := GetModelInfo(modelName)
	return info.ToolCompatibility != ToolCompatibilityNone &&
		info.ToolCompatibility != ToolCompatibilityUnknown
}

// IsRecommendedForTools returns true if the model is recommended for tool usage
func IsRecommendedForTools(modelName string) bool {
	info := GetModelInfo(modelName)
	return info.RecommendedForTools
}

// GetRecommendedModels returns a list of models recommended for tool calling
func GetRecommendedModels() []string {
	var recommended []string
	for modelName, info := range KnownModels {
		if info.RecommendedForTools {
			recommended = append(recommended, modelName)
		}
	}
	return recommended
}

// GetModelsByCompatibility returns models grouped by compatibility level
func GetModelsByCompatibility() map[ToolCompatibility][]string {
	result := make(map[ToolCompatibility][]string)

	for modelName, info := range KnownModels {
		result[info.ToolCompatibility] = append(result[info.ToolCompatibility], modelName)
	}

	return result
}

// normalizeModelName removes common variations and standardizes the model name
func normalizeModelName(modelName string) string {
	// Remove common prefixes/suffixes that don't affect tool compatibility
	name := strings.ToLower(strings.TrimSpace(modelName))

	// Remove trailing version descriptors like "-base", "-instruct", etc.
	suffixesToRemove := []string{"-base", "-instruct", "-chat"}
	for _, suffix := range suffixesToRemove {
		if strings.HasSuffix(name, suffix) {
			name = strings.TrimSuffix(name, suffix)
		}
	}

	return name
}

// extractBaseModelName extracts the base model name without size specifiers
func extractBaseModelName(modelName string) string {
	parts := strings.Split(modelName, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return modelName
}

// inferModelCapabilities attempts to infer tool calling capabilities from model name
func inferModelCapabilities(modelName string) ModelInfo {
	name := strings.ToLower(modelName)

	// Model families known to support tools
	if strings.Contains(name, "llama3") {
		return ModelInfo{
			Name:                modelName,
			ToolCompatibility:   ToolCompatibilityGood,
			RecommendedForTools: true,
			Notes:               "Llama 3 family generally supports tool calling",
		}
	}

	if strings.Contains(name, "qwen") {
		return ModelInfo{
			Name:                modelName,
			ToolCompatibility:   ToolCompatibilityGood,
			RecommendedForTools: true,
			Notes:               "Qwen family generally supports tool calling",
		}
	}

	if strings.Contains(name, "mistral") {
		return ModelInfo{
			Name:                modelName,
			ToolCompatibility:   ToolCompatibilityGood,
			RecommendedForTools: true,
			Notes:               "Mistral family generally supports tool calling",
		}
	}

	if strings.Contains(name, "command-r") {
		return ModelInfo{
			Name:                modelName,
			ToolCompatibility:   ToolCompatibilityGood,
			RecommendedForTools: true,
			Notes:               "Command-R family supports tool calling",
		}
	}

	// Model families known to have limited/no support
	if strings.Contains(name, "gemma") || strings.Contains(name, "phi") {
		return ModelInfo{
			Name:                modelName,
			ToolCompatibility:   ToolCompatibilityNone,
			RecommendedForTools: false,
			Notes:               "This model family typically lacks tool calling support",
		}
	}

	// Unknown model
	return ModelInfo{
		Name:                modelName,
		ToolCompatibility:   ToolCompatibilityUnknown,
		RecommendedForTools: false,
		Notes:               "Tool calling compatibility unknown - test recommended",
	}
}
