package agents

import (
	"context"

	"github.com/killallgit/ryan/pkg/models"
	"github.com/killallgit/ryan/pkg/tools"
)

// ChainType represents the type of Langchain agent chain
type ChainType string

const (
	ChainTypeReAct           ChainType = "react"
	ChainTypeOllamaFunctions ChainType = "ollama-functions"
	ChainTypeOpenAIFunctions ChainType = "openai-functions"
	ChainTypeDirect          ChainType = "direct"
)

// ModelRequirements defines the minimum model capabilities needed for an agent
type ModelRequirements struct {
	MinToolCompatibility models.ToolCompatibility
	RequiredFeatures     []string
	PreferredModels      []string
}

// LangchainAgent extends the base Agent interface with Langchain-specific capabilities
type LangchainAgent interface {
	Agent

	// GetChainType returns the type of chain this agent uses
	GetChainType() ChainType

	// GetToolCompatibility returns the list of tools this agent can use
	GetToolCompatibility() []string

	// GetModelRequirements returns the minimum model requirements
	GetModelRequirements() ModelRequirements

	// SetToolRegistry sets the tool registry for this agent
	SetToolRegistry(registry *tools.Registry)

	// SetModel sets the model to use for this agent
	SetModel(model string) error

	// SupportsStreaming indicates if this agent supports streaming responses
	SupportsStreaming() bool
}

// BaseLangchainAgent provides common functionality for all Langchain agents
type BaseLangchainAgent struct {
	name         string
	description  string
	chainType    ChainType
	toolRegistry *tools.Registry
	model        string
	requirements ModelRequirements
}

// Name returns the agent's name
func (b *BaseLangchainAgent) Name() string {
	return b.name
}

// Description returns the agent's description
func (b *BaseLangchainAgent) Description() string {
	return b.description
}

// GetChainType returns the chain type
func (b *BaseLangchainAgent) GetChainType() ChainType {
	return b.chainType
}

// GetModelRequirements returns the model requirements
func (b *BaseLangchainAgent) GetModelRequirements() ModelRequirements {
	return b.requirements
}

// SetToolRegistry sets the tool registry
func (b *BaseLangchainAgent) SetToolRegistry(registry *tools.Registry) {
	b.toolRegistry = registry
}

// SetModel sets the model
func (b *BaseLangchainAgent) SetModel(model string) error {
	b.model = model
	return nil
}

// SupportsStreaming returns whether streaming is supported (default: true)
func (b *BaseLangchainAgent) SupportsStreaming() bool {
	return true
}

// AgentConfig contains configuration for creating agents
type AgentConfig struct {
	BaseURL      string
	Model        string
	ToolRegistry *tools.Registry
	SystemPrompt string
	Options      map[string]interface{}
}

// AgentCapability represents a specific capability an agent has
type AgentCapability string

const (
	CapabilityToolCalling     AgentCapability = "tool_calling"
	CapabilityStreaming       AgentCapability = "streaming"
	CapabilityContextWindow   AgentCapability = "context_window"
	CapabilityFunctionCalling AgentCapability = "function_calling"
	CapabilityCodeExecution   AgentCapability = "code_execution"
	CapabilityWebSearch       AgentCapability = "web_search"
	CapabilityFileOperations  AgentCapability = "file_operations"
)

// AgentMetrics tracks performance metrics for agents
type AgentMetrics struct {
	SuccessRate      float64
	AverageLatency   float64
	TokensPerRequest float64
	ErrorRate        float64
}

// AgentSelector helps select the best agent for a task
type AgentSelectorStrategy interface {
	// SelectAgent chooses the best agent for the given request
	SelectAgent(ctx context.Context, request string, available []LangchainAgent) (LangchainAgent, error)

	// RankAgents ranks agents by suitability for the request
	RankAgents(request string, available []LangchainAgent) []LangchainAgent
}
