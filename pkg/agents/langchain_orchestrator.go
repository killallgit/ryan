package agents

import (
	"context"
	"fmt"
	"sort"
	"sync"
	
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
)

// LangchainOrchestrator orchestrates multiple Langchain agents
type LangchainOrchestrator struct {
	agents       map[string]LangchainAgent
	factory      *AgentFactory
	toolRegistry *tools.Registry
	log          *logger.Logger
	mu           sync.RWMutex
	
	// User preferences
	preferredAgent string
	fallbackChain  []string
}

// NewLangchainOrchestrator creates a new orchestrator
func NewLangchainOrchestrator(toolRegistry *tools.Registry) *LangchainOrchestrator {
	return &LangchainOrchestrator{
		agents:       make(map[string]LangchainAgent),
		factory:      GlobalFactory,
		toolRegistry: toolRegistry,
		log:          logger.WithComponent("langchain_orchestrator"),
		fallbackChain: []string{"ollama-functions", "openai-functions", "conversational"},
	}
}

// SetPreferredAgent sets the preferred agent type
func (o *LangchainOrchestrator) SetPreferredAgent(agentType string) error {
	if !o.factory.IsRegistered(agentType) {
		return fmt.Errorf("unknown agent type: %s", agentType)
	}
	o.preferredAgent = agentType
	o.log.Debug("Set preferred agent", "type", agentType)
	return nil
}

// SetFallbackChain sets the fallback chain for agent selection
func (o *LangchainOrchestrator) SetFallbackChain(chain []string) error {
	for _, agentType := range chain {
		if !o.factory.IsRegistered(agentType) {
			return fmt.Errorf("unknown agent type in fallback chain: %s", agentType)
		}
	}
	o.fallbackChain = chain
	return nil
}

// RegisterAgent registers a pre-created agent
func (o *LangchainOrchestrator) RegisterAgent(agent LangchainAgent) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	name := agent.Name()
	if _, exists := o.agents[name]; exists {
		return fmt.Errorf("agent %s is already registered", name)
	}
	
	o.agents[name] = agent
	o.log.Debug("Registered agent", "name", name, "type", agent.GetChainType())
	return nil
}

// Execute routes a request to the best available agent
func (o *LangchainOrchestrator) Execute(ctx context.Context, request string, options map[string]interface{}) (AgentResult, error) {
	// Check if a specific agent is requested in options
	if agentName, ok := options["agent"].(string); ok {
		return o.ExecuteWithAgent(ctx, agentName, AgentRequest{
			Prompt:  request,
			Options: options,
		})
	}
	
	// Try to select the best agent
	agent, err := o.selectBestAgent(request, options)
	if err != nil {
		return AgentResult{}, fmt.Errorf("failed to select agent: %w", err)
	}
	
	return agent.Execute(ctx, AgentRequest{
		Prompt:  request,
		Options: options,
	})
}

// ExecuteWithAgent executes with a specific agent
func (o *LangchainOrchestrator) ExecuteWithAgent(ctx context.Context, agentName string, request AgentRequest) (AgentResult, error) {
	o.mu.RLock()
	agent, exists := o.agents[agentName]
	o.mu.RUnlock()
	
	if !exists {
		// Try to create the agent on-demand
		config := AgentConfig{
			ToolRegistry: o.toolRegistry,
			Options:      request.Options,
		}
		
		// Extract model from options if available
		if model, ok := request.Options["model"].(string); ok {
			config.Model = model
		}
		
		newAgent, err := o.factory.Create(agentName, config)
		if err != nil {
			return AgentResult{}, fmt.Errorf("agent %s not found and could not be created: %w", agentName, err)
		}
		
		// Register the newly created agent
		if err := o.RegisterAgent(newAgent); err != nil {
			o.log.Warn("Failed to register newly created agent", "error", err)
		}
		
		agent = newAgent
	}
	
	return agent.Execute(ctx, request)
}

// ListAgents returns all registered agents
func (o *LangchainOrchestrator) ListAgents() []Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	agents := make([]Agent, 0, len(o.agents))
	for _, agent := range o.agents {
		agents = append(agents, agent)
	}
	return agents
}

// ListLangchainAgents returns all registered Langchain agents
func (o *LangchainOrchestrator) ListLangchainAgents() []LangchainAgent {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	agents := make([]LangchainAgent, 0, len(o.agents))
	for _, agent := range o.agents {
		agents = append(agents, agent)
	}
	return agents
}

// selectBestAgent selects the best agent for the request
func (o *LangchainOrchestrator) selectBestAgent(request string, options map[string]interface{}) (LangchainAgent, error) {
	// If preferred agent is set, try it first
	if o.preferredAgent != "" {
		agent, err := o.getOrCreateAgent(o.preferredAgent, options)
		if err == nil {
			canHandle, confidence := agent.CanHandle(request)
			if canHandle && confidence > 0.3 { // Lower threshold for preferred
				o.log.Debug("Using preferred agent",
					"type", o.preferredAgent,
					"confidence", confidence)
				return agent, nil
			}
		}
	}
	
	// Rank all available agents
	rankings := o.rankAgents(request)
	
	// Try agents in order of ranking
	for _, ranking := range rankings {
		if ranking.confidence > 0.5 {
			agent, err := o.getOrCreateAgent(ranking.agentName, options)
			if err == nil {
				o.log.Debug("Selected agent",
					"name", ranking.agentName,
					"confidence", ranking.confidence)
				return agent, nil
			}
		}
	}
	
	// Fallback to the first agent in the fallback chain
	for _, agentType := range o.fallbackChain {
		agent, err := o.getOrCreateAgent(agentType, options)
		if err == nil {
			o.log.Debug("Using fallback agent", "type", agentType)
			return agent, nil
		}
	}
	
	return nil, fmt.Errorf("no suitable agent found for request")
}

// getOrCreateAgent gets an existing agent or creates a new one
func (o *LangchainOrchestrator) getOrCreateAgent(agentName string, options map[string]interface{}) (LangchainAgent, error) {
	o.mu.RLock()
	agent, exists := o.agents[agentName]
	o.mu.RUnlock()
	
	if exists {
		return agent, nil
	}
	
	// Create the agent
	config := AgentConfig{
		ToolRegistry: o.toolRegistry,
		Options:      options,
	}
	
	if model, ok := options["model"].(string); ok {
		config.Model = model
	}
	
	newAgent, err := o.factory.Create(agentName, config)
	if err != nil {
		return nil, err
	}
	
	// Register it for future use
	_ = o.RegisterAgent(newAgent)
	
	return newAgent, nil
}

// agentRanking represents an agent's suitability ranking
type agentRanking struct {
	agentName  string
	confidence float64
}

// rankAgents ranks agents by their suitability for the request
func (o *LangchainOrchestrator) rankAgents(request string) []agentRanking {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	rankings := make([]agentRanking, 0, len(o.agents))
	
	// Check existing agents
	for name, agent := range o.agents {
		canHandle, confidence := agent.CanHandle(request)
		if canHandle {
			rankings = append(rankings, agentRanking{
				agentName:  name,
				confidence: confidence,
			})
		}
	}
	
	// Also check factory-registered types that aren't instantiated yet
	for _, agentType := range o.factory.GetRegisteredTypes() {
		if _, exists := o.agents[agentType]; !exists {
			// Estimate confidence based on agent type and request
			// This is a simplified heuristic
			confidence := o.estimateConfidence(agentType, request)
			if confidence > 0 {
				rankings = append(rankings, agentRanking{
					agentName:  agentType,
					confidence: confidence,
				})
			}
		}
	}
	
	// Sort by confidence (highest first)
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].confidence > rankings[j].confidence
	})
	
	return rankings
}

// estimateConfidence estimates confidence for an uninstantiated agent type
func (o *LangchainOrchestrator) estimateConfidence(agentType, request string) float64 {
	// Simple heuristic based on agent type
	switch agentType {
	case "ollama-functions":
		if o.toolRegistry != nil && o.toolRegistry.HasTools() {
			return 0.7
		}
		return 0.3
	case "openai-functions":
		if o.toolRegistry != nil && o.toolRegistry.HasTools() {
			return 0.6
		}
		return 0.3
	case "conversational", "react":
		return 0.5 // Can handle most things reasonably
	default:
		return 0.2
	}
}