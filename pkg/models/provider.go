package models

import (
	"context"
	"fmt"
	"strings"
)

// ModelProvider defines the interface for discovering and retrieving model information
type ModelProvider interface {
	// ListModels returns a list of available models
	ListModels(ctx context.Context) ([]ModelInfo, error)

	// GetModelInfo returns information about a specific model
	GetModelInfo(ctx context.Context, modelName string) (*ModelInfo, error)

	// RefreshCache forces a refresh of the model list cache
	RefreshCache(ctx context.Context) error

	// IsModelAvailable checks if a model is available
	IsModelAvailable(ctx context.Context, modelName string) (bool, error)
}

// ModelCapability represents specific capabilities a model might have
type ModelCapability string

const (
	CapabilityToolCalling     ModelCapability = "tool_calling"
	CapabilityFunctionCalling ModelCapability = "function_calling"
	CapabilityCodeGeneration  ModelCapability = "code_generation"
	CapabilityReasoning       ModelCapability = "reasoning"
	CapabilityMultiModal      ModelCapability = "multi_modal"
	CapabilityStreaming       ModelCapability = "streaming"
)

// ExtendedModelInfo provides extended information about a model
type ExtendedModelInfo struct {
	ModelInfo

	// Size in bytes
	Size int64

	// Parameter count (e.g., "7B", "70B")
	ParameterSize string

	// Quantization level (e.g., "Q4_0", "Q8_0")
	Quantization string

	// Capabilities detected or inferred
	Capabilities []ModelCapability

	// Family (e.g., "llama", "qwen", "mistral")
	Family string

	// Version info
	Version string

	// Format (e.g., "gguf")
	Format string
}

// InferModelFamily attempts to infer the model family from the name
func InferModelFamily(modelName string) string {
	lowerName := strings.ToLower(modelName)

	// Check more specific patterns first
	switch {
	case strings.Contains(lowerName, "codellama"):
		return "codellama"
	case strings.Contains(lowerName, "wizardcoder"):
		return "wizardcoder"
	case strings.Contains(lowerName, "starcoder"):
		return "starcoder"
	case strings.Contains(lowerName, "mixtral"):
		return "mixtral"
	case strings.Contains(lowerName, "neural-chat"):
		return "neural-chat"
	case strings.Contains(lowerName, "llama"):
		return "llama"
	case strings.Contains(lowerName, "qwen"):
		return "qwen"
	case strings.Contains(lowerName, "mistral"):
		return "mistral"
	case strings.Contains(lowerName, "deepseek"):
		return "deepseek"
	case strings.Contains(lowerName, "gemma"):
		return "gemma"
	case strings.Contains(lowerName, "phi"):
		return "phi"
	case strings.Contains(lowerName, "vicuna"):
		return "vicuna"
	case strings.Contains(lowerName, "solar"):
		return "solar"
	case strings.Contains(lowerName, "yi"):
		return "yi"
	default:
		return "unknown"
	}
}

// InferToolCompatibility attempts to infer tool compatibility based on model family and version
func InferToolCompatibility(modelName string) ToolCompatibility {
	lowerName := strings.ToLower(modelName)
	family := InferModelFamily(modelName)

	// Check for known excellent tool support patterns
	if family == "llama" {
		// Llama 3.1+ has excellent tool support
		if strings.Contains(lowerName, "3.1") || strings.Contains(lowerName, "3.2") || strings.Contains(lowerName, "3.3") {
			return ToolCompatibilityExcellent
		}
		// Llama 3.0 has good support
		if strings.Contains(lowerName, "3") {
			return ToolCompatibilityGood
		}
		// Older Llama models have basic support
		return ToolCompatibilityBasic
	}

	if family == "qwen" {
		// Qwen2.5 and Qwen3 have excellent support
		if strings.Contains(lowerName, "qwen2.5") || strings.Contains(lowerName, "qwen3") {
			return ToolCompatibilityExcellent
		}
		// Qwen2 has good support
		if strings.Contains(lowerName, "qwen2") {
			return ToolCompatibilityGood
		}
		return ToolCompatibilityBasic
	}

	if family == "mistral" || family == "mixtral" {
		// Mixtral models have excellent support
		if family == "mixtral" {
			return ToolCompatibilityExcellent
		}
		// Regular Mistral models have good support
		return ToolCompatibilityGood
	}

	if family == "deepseek" {
		// DeepSeek R1 and coder models have good support
		if strings.Contains(lowerName, "r1") || strings.Contains(lowerName, "coder") {
			return ToolCompatibilityGood
		}
		return ToolCompatibilityBasic
	}

	// Code-specific models often have good tool support
	if strings.Contains(lowerName, "coder") || strings.Contains(lowerName, "code") ||
		family == "codellama" || family == "starcoder" || family == "wizardcoder" {
		return ToolCompatibilityGood
	}

	// Models known to have no tool support
	if family == "gemma" || family == "phi" {
		return ToolCompatibilityNone
	}

	// Default to unknown for unfamiliar models
	return ToolCompatibilityUnknown
}

// InferCapabilities attempts to infer model capabilities from its name and metadata
func InferCapabilities(modelName string, parameterSize string) []ModelCapability {
	caps := []ModelCapability{CapabilityStreaming} // Most models support streaming

	lowerName := strings.ToLower(modelName)
	family := InferModelFamily(modelName)

	// Tool calling capability
	compat := InferToolCompatibility(modelName)
	if compat >= ToolCompatibilityBasic {
		caps = append(caps, CapabilityToolCalling, CapabilityFunctionCalling)
	}

	// Code generation capability
	if strings.Contains(lowerName, "code") || strings.Contains(lowerName, "coder") ||
		family == "codellama" || family == "starcoder" || family == "wizardcoder" {
		caps = append(caps, CapabilityCodeGeneration)
	}

	// Reasoning capability (usually larger models or specific families)
	if strings.Contains(lowerName, "r1") || strings.Contains(lowerName, "reason") ||
		strings.Contains(parameterSize, "70") || strings.Contains(parameterSize, "405") {
		caps = append(caps, CapabilityReasoning)
	}

	// Multi-modal capability
	if strings.Contains(lowerName, "vision") || strings.Contains(lowerName, "llava") ||
		strings.Contains(lowerName, "bakllava") {
		caps = append(caps, CapabilityMultiModal)
	}

	return caps
}

// ParseParameterSize extracts parameter size from model details
func ParseParameterSize(details map[string]interface{}) string {
	// Check for parameter_size field
	if ps, ok := details["parameter_size"].(string); ok {
		return ps
	}

	// Try to infer from model name or other fields
	if family, ok := details["family"].(string); ok {
		// Extract size from family name (e.g., "llama2:7b")
		parts := strings.Split(family, ":")
		if len(parts) > 1 {
			return parts[1]
		}
	}

	return "unknown"
}

// ParseQuantization extracts quantization level from model details
func ParseQuantization(details map[string]interface{}) string {
	// Check for quantization_level field
	if ql, ok := details["quantization_level"].(string); ok {
		return ql
	}

	// Check in details sub-object
	if detailsMap, ok := details["details"].(map[string]interface{}); ok {
		if ql, ok := detailsMap["quantization_level"].(string); ok {
			return ql
		}
	}

	return "unknown"
}

// FormatModelSize formats size in bytes to human-readable format
func FormatModelSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
