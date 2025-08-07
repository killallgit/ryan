package models

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/ollama"
)

// OllamaModelProvider implements ModelProvider for Ollama
type OllamaModelProvider struct {
	client *ollama.Client

	// Cache
	cache      []ModelInfo
	cacheTime  time.Time
	cacheTTL   time.Duration
	cacheMutex sync.RWMutex

	logger *logger.Logger
}

// NewOllamaModelProvider creates a new Ollama model provider
func NewOllamaModelProvider(client *ollama.Client) *OllamaModelProvider {
	return &OllamaModelProvider{
		client:   client,
		cacheTTL: 5 * time.Minute, // Cache for 5 minutes
		logger:   logger.WithComponent("ollama_model_provider"),
	}
}

// ListModels returns a list of available models from Ollama
func (p *OllamaModelProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Check cache first
	p.cacheMutex.RLock()
	if time.Since(p.cacheTime) < p.cacheTTL && len(p.cache) > 0 {
		cached := p.cache
		p.cacheMutex.RUnlock()
		p.logger.Debug("Returning cached model list", "count", len(cached))
		return cached, nil
	}
	p.cacheMutex.RUnlock()

	// Fetch fresh data
	p.logger.Debug("Fetching model list from Ollama")
	tagsResp, err := p.client.Tags()
	if err != nil {
		return nil, fmt.Errorf("failed to get models from Ollama: %w", err)
	}

	models := make([]ModelInfo, 0, len(tagsResp.Models))
	for _, ollamaModel := range tagsResp.Models {
		modelInfo := p.convertOllamaModel(ollamaModel)
		models = append(models, modelInfo)
	}

	// Update cache
	p.cacheMutex.Lock()
	p.cache = models
	p.cacheTime = time.Now()
	p.cacheMutex.Unlock()

	p.logger.Info("Retrieved models from Ollama", "count", len(models))
	return models, nil
}

// GetModelInfo returns information about a specific model
func (p *OllamaModelProvider) GetModelInfo(ctx context.Context, modelName string) (*ModelInfo, error) {
	models, err := p.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	for _, model := range models {
		if model.Name == modelName {
			return &model, nil
		}
	}

	// Model not found in list, create basic info with inferred compatibility
	modelInfo := ModelInfo{
		Name:              modelName,
		ToolCompatibility: InferToolCompatibility(modelName),
		Notes:             "Model not found in local list, compatibility inferred from name",
	}

	// Check if it's recommended based on compatibility
	if modelInfo.ToolCompatibility >= ToolCompatibilityGood {
		modelInfo.RecommendedForTools = true
	}

	return &modelInfo, nil
}

// RefreshCache forces a refresh of the model list cache
func (p *OllamaModelProvider) RefreshCache(ctx context.Context) error {
	p.cacheMutex.Lock()
	p.cacheTime = time.Time{} // Reset cache time to force refresh
	p.cacheMutex.Unlock()

	_, err := p.ListModels(ctx)
	return err
}

// IsModelAvailable checks if a model is available in Ollama
func (p *OllamaModelProvider) IsModelAvailable(ctx context.Context, modelName string) (bool, error) {
	models, err := p.ListModels(ctx)
	if err != nil {
		return false, err
	}

	for _, model := range models {
		if model.Name == modelName {
			return true, nil
		}
	}

	return false, nil
}

// convertOllamaModel converts an Ollama model to our ModelInfo format
func (p *OllamaModelProvider) convertOllamaModel(ollamaModel ollama.Model) ModelInfo {
	// Infer tool compatibility from model name and metadata
	toolCompat := InferToolCompatibility(ollamaModel.Name)

	// Determine if recommended for tools
	recommendedForTools := toolCompat >= ToolCompatibilityGood

	// Build notes based on model characteristics
	notes := p.buildModelNotes(ollamaModel, toolCompat)

	return ModelInfo{
		Name:                ollamaModel.Name,
		ToolCompatibility:   toolCompat,
		RecommendedForTools: recommendedForTools,
		Notes:               notes,
	}
}

// buildModelNotes creates descriptive notes about the model
func (p *OllamaModelProvider) buildModelNotes(model ollama.Model, compat ToolCompatibility) string {
	var notes []string

	// Add size information
	if model.Size > 0 {
		notes = append(notes, fmt.Sprintf("Size: %s", FormatModelSize(model.Size)))
	}

	// Add parameter size if available
	if model.Details.ParameterSize != "" {
		notes = append(notes, fmt.Sprintf("Parameters: %s", model.Details.ParameterSize))
	}

	// Add quantization if available
	if model.Details.QuantizationLevel != "" {
		notes = append(notes, fmt.Sprintf("Quantization: %s", model.Details.QuantizationLevel))
	}

	// Add compatibility note
	switch compat {
	case ToolCompatibilityExcellent:
		notes = append(notes, "Excellent tool calling support")
	case ToolCompatibilityGood:
		notes = append(notes, "Good tool calling support")
	case ToolCompatibilityBasic:
		notes = append(notes, "Basic tool calling support")
	case ToolCompatibilityNone:
		notes = append(notes, "No tool calling support")
	default:
		notes = append(notes, "Tool calling compatibility unknown")
	}

	// Add family-specific notes
	family := InferModelFamily(model.Name)
	switch family {
	case "llama":
		notes = append(notes, "Llama family - versatile and well-tested")
	case "qwen":
		notes = append(notes, "Qwen family - excellent for coding and reasoning")
	case "mistral":
		notes = append(notes, "Mistral family - efficient and capable")
	case "deepseek":
		notes = append(notes, "DeepSeek family - strong reasoning capabilities")
	case "codellama", "starcoder", "wizardcoder":
		notes = append(notes, "Specialized for code generation")
	}

	return strings.Join(notes, ", ")
}

// GetExtendedModelInfo returns extended information about a model
func (p *OllamaModelProvider) GetExtendedModelInfo(ctx context.Context, modelName string) (*ExtendedModelInfo, error) {
	// Get basic info first
	basicInfo, err := p.GetModelInfo(ctx, modelName)
	if err != nil {
		return nil, err
	}

	// Try to get more detailed info from Ollama
	tagsResp, err := p.client.Tags()
	if err != nil {
		// Return basic info if we can't get detailed info
		return &ExtendedModelInfo{
			ModelInfo: *basicInfo,
			Family:    InferModelFamily(modelName),
		}, nil
	}

	// Find the model in the response
	for _, ollamaModel := range tagsResp.Models {
		if ollamaModel.Name == modelName {
			return &ExtendedModelInfo{
				ModelInfo:     *basicInfo,
				Size:          ollamaModel.Size,
				ParameterSize: ollamaModel.Details.ParameterSize,
				Quantization:  ollamaModel.Details.QuantizationLevel,
				Family:        InferModelFamily(modelName),
				Format:        ollamaModel.Details.Format,
				Capabilities:  InferCapabilities(modelName, ollamaModel.Details.ParameterSize),
			}, nil
		}
	}

	// Model not found, return basic extended info
	return &ExtendedModelInfo{
		ModelInfo:    *basicInfo,
		Family:       InferModelFamily(modelName),
		Capabilities: InferCapabilities(modelName, ""),
	}, nil
}

// SetCacheTTL sets the cache time-to-live duration
func (p *OllamaModelProvider) SetCacheTTL(ttl time.Duration) {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()
	p.cacheTTL = ttl
}
