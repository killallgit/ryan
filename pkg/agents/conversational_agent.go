package agents

import (
	"context"
	"fmt"
	"strings"
	
	"github.com/killallgit/ryan/pkg/langchain"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/models"
)

// ConversationalAgent implements a ReAct-style conversational agent
type ConversationalAgent struct {
	BaseLangchainAgent
	client *langchain.Client
	log    *logger.Logger
}

// NewConversationalAgent creates a new conversational agent
func NewConversationalAgent(config AgentConfig) (LangchainAgent, error) {
	client, err := langchain.NewClient(config.BaseURL, config.Model, config.ToolRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create langchain client: %w", err)
	}
	
	agent := &ConversationalAgent{
		BaseLangchainAgent: BaseLangchainAgent{
			name:         "conversational",
			description:  "ReAct-style conversational agent with tool usage through natural language",
			chainType:    ChainTypeReAct,
			toolRegistry: config.ToolRegistry,
			model:        config.Model,
			requirements: ModelRequirements{
				MinToolCompatibility: models.ToolCompatibilityGood,
				RequiredFeatures:     []string{"reasoning", "tool_description"},
				PreferredModels:      []string{"claude", "gpt-4", "llama3"},
			},
		},
		client: client,
		log:    logger.WithComponent("conversational_agent"),
	}
	
	return agent, nil
}

// CanHandle determines if this agent can handle the request
func (c *ConversationalAgent) CanHandle(request string) (bool, float64) {
	// Conversational agent can handle most requests but with varying confidence
	lowerRequest := strings.ToLower(request)
	
	// High confidence for conversational and reasoning tasks
	if strings.Contains(lowerRequest, "explain") ||
		strings.Contains(lowerRequest, "think") ||
		strings.Contains(lowerRequest, "reason") ||
		strings.Contains(lowerRequest, "analyze") {
		return true, 0.9
	}
	
	// Medium confidence for tool usage
	if c.toolRegistry != nil && c.toolRegistry.HasTools() {
		return true, 0.7
	}
	
	// Low confidence as fallback
	return true, 0.5
}

// Execute performs the agent's task
func (c *ConversationalAgent) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	c.log.Debug("Executing conversational agent",
		"prompt", request.Prompt,
		"model", c.model)
	
	// Send message through the langchain client
	response, err := c.client.SendMessage(ctx, request.Prompt)
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: "Failed to execute conversational agent",
			Details: err.Error(),
		}, err
	}
	
	return AgentResult{
		Success: true,
		Summary: "Conversational agent completed successfully",
		Details: response,
		Metadata: AgentMetadata{
			AgentName: c.name,
		},
	}, nil
}

// GetToolCompatibility returns tools this agent can use
func (c *ConversationalAgent) GetToolCompatibility() []string {
	if c.toolRegistry == nil {
		return []string{}
	}
	
	tools := c.toolRegistry.GetTools()
	toolNames := make([]string, 0, len(tools))
	for _, tool := range tools {
		toolNames = append(toolNames, tool.Name())
	}
	return toolNames
}