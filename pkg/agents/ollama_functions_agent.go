package agents

import (
	"context"
	"fmt"
	"strings"
	
	"github.com/killallgit/ryan/pkg/langchain"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/models"
)

// OllamaFunctionsAgent implements native Ollama function calling
type OllamaFunctionsAgent struct {
	BaseLangchainAgent
	client *langchain.Client
	log    *logger.Logger
}

// NewOllamaFunctionsAgent creates a new Ollama functions agent
func NewOllamaFunctionsAgent(config AgentConfig) (LangchainAgent, error) {
	// Verify the model supports Ollama functions
	if !isOllamaCompatibleModel(config.Model) {
		return nil, fmt.Errorf("model %s is not compatible with Ollama functions", config.Model)
	}
	
	client, err := langchain.NewClient(config.BaseURL, config.Model, config.ToolRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create langchain client: %w", err)
	}
	
	agent := &OllamaFunctionsAgent{
		BaseLangchainAgent: BaseLangchainAgent{
			name:         "ollama-functions",
			description:  "Native Ollama function calling for efficient tool usage",
			chainType:    ChainTypeOllamaFunctions,
			toolRegistry: config.ToolRegistry,
			model:        config.Model,
			requirements: ModelRequirements{
				MinToolCompatibility: models.ToolCompatibilityExcellent,
				RequiredFeatures:     []string{"function_calling", "ollama_compatible"},
				PreferredModels:      []string{"llama3.1", "qwen2.5", "mistral", "deepseek", "command-r"},
			},
		},
		client: client,
		log:    logger.WithComponent("ollama_functions_agent"),
	}
	
	return agent, nil
}

// CanHandle determines if this agent can handle the request
func (o *OllamaFunctionsAgent) CanHandle(request string) (bool, float64) {
	// Check if model is Ollama compatible
	if !isOllamaCompatibleModel(o.model) {
		return false, 0.0
	}
	
	// High confidence for tool-heavy tasks
	lowerRequest := strings.ToLower(request)
	
	toolKeywords := []string{
		"run", "execute", "create", "delete", "list", "show",
		"grep", "search", "find", "fetch", "download",
	}
	
	keywordCount := 0
	for _, keyword := range toolKeywords {
		if strings.Contains(lowerRequest, keyword) {
			keywordCount++
		}
	}
	
	if keywordCount > 0 {
		confidence := 0.7 + float64(keywordCount)*0.1
		if confidence > 1.0 {
			confidence = 1.0
		}
		return true, confidence
	}
	
	// Medium confidence if tools are available
	if o.toolRegistry != nil && o.toolRegistry.HasTools() {
		return true, 0.6
	}
	
	return false, 0.0
}

// Execute performs the agent's task using Ollama function calling
func (o *OllamaFunctionsAgent) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	o.log.Debug("Executing Ollama functions agent",
		"prompt", request.Prompt,
		"model", o.model,
		"tools_available", o.toolRegistry != nil)
	
	// Configure client for Ollama function mode
	if o.client != nil {
		// This would set the agent type in the client
		// The client would handle the specific Ollama function calling protocol
	}
	
	response, err := o.client.SendMessage(ctx, request.Prompt)
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: "Failed to execute Ollama functions agent",
			Details: err.Error(),
		}, err
	}
	
	return AgentResult{
		Success: true,
		Summary: "Ollama functions agent completed successfully",
		Details: response,
		Metadata: AgentMetadata{
			AgentName: o.name,
		},
	}, nil
}

// GetToolCompatibility returns tools this agent can use
func (o *OllamaFunctionsAgent) GetToolCompatibility() []string {
	if o.toolRegistry == nil {
		return []string{}
	}
	
	tools := o.toolRegistry.GetTools()
	toolNames := make([]string, 0, len(tools))
	for _, tool := range tools {
		toolNames = append(toolNames, tool.Name())
	}
	return toolNames
}

// isOllamaCompatibleModel checks if a model supports Ollama function calling
func isOllamaCompatibleModel(model string) bool {
	ollamaModels := []string{
		"llama", "qwen", "mistral", "deepseek", "command-r",
		"granite", "gemma2", "phi3",
	}
	
	modelLower := strings.ToLower(model)
	for _, compatible := range ollamaModels {
		if strings.Contains(modelLower, compatible) {
			return true
		}
	}
	
	return false
}