package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/langchain"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/models"
)

// OpenAIFunctionsAgent implements OpenAI-style function calling
type OpenAIFunctionsAgent struct {
	BaseLangchainAgent
	client *langchain.Client
	log    *logger.Logger
}

// NewOpenAIFunctionsAgent creates a new OpenAI functions agent
func NewOpenAIFunctionsAgent(config AgentConfig) (LangchainAgent, error) {
	// Verify the model supports OpenAI functions
	if !isOpenAICompatibleModel(config.Model) {
		return nil, fmt.Errorf("model %s is not compatible with OpenAI functions", config.Model)
	}

	client, err := langchain.NewClient(config.BaseURL, config.Model, config.ToolRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create langchain client: %w", err)
	}

	agent := &OpenAIFunctionsAgent{
		BaseLangchainAgent: BaseLangchainAgent{
			name:         "openai-functions",
			description:  "OpenAI-style function calling for structured tool usage",
			chainType:    ChainTypeOpenAIFunctions,
			toolRegistry: config.ToolRegistry,
			model:        config.Model,
			requirements: ModelRequirements{
				MinToolCompatibility: models.ToolCompatibilityExcellent,
				RequiredFeatures:     []string{"function_calling", "openai_compatible"},
				PreferredModels:      []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "claude-3"},
			},
		},
		client: client,
		log:    logger.WithComponent("openai_functions_agent"),
	}

	return agent, nil
}

// CanHandle determines if this agent can handle the request
func (o *OpenAIFunctionsAgent) CanHandle(request string) (bool, float64) {
	// Check if model is OpenAI compatible
	if !isOpenAICompatibleModel(o.model) {
		return false, 0.0
	}

	// High confidence for structured tasks
	lowerRequest := strings.ToLower(request)

	// OpenAI functions excel at structured operations
	if strings.Contains(lowerRequest, "api") ||
		strings.Contains(lowerRequest, "function") ||
		strings.Contains(lowerRequest, "structured") ||
		strings.Contains(lowerRequest, "json") {
		return true, 0.95
	}

	// Good for general tool usage
	if o.toolRegistry != nil && o.toolRegistry.HasTools() {
		return true, 0.8
	}

	return true, 0.5
}

// Execute performs the agent's task using OpenAI function calling
func (o *OpenAIFunctionsAgent) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	o.log.Debug("Executing OpenAI functions agent",
		"prompt", request.Prompt,
		"model", o.model,
		"tools_available", o.toolRegistry != nil)

	response, err := o.client.SendMessage(ctx, request.Prompt)
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: "Failed to execute OpenAI functions agent",
			Details: err.Error(),
		}, err
	}

	return AgentResult{
		Success: true,
		Summary: "OpenAI functions agent completed successfully",
		Details: response,
		Metadata: AgentMetadata{
			AgentName: o.name,
		},
	}, nil
}

// GetToolCompatibility returns tools this agent can use
func (o *OpenAIFunctionsAgent) GetToolCompatibility() []string {
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

// isOpenAICompatibleModel checks if a model supports OpenAI function calling
func isOpenAICompatibleModel(model string) bool {
	openAIModels := []string{
		"gpt-4", "gpt-3.5", "claude", "gemini",
	}

	modelLower := strings.ToLower(model)
	for _, compatible := range openAIModels {
		if strings.Contains(modelLower, compatible) {
			return true
		}
	}

	return false
}
