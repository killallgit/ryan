package agents

import (
	"fmt"
	"sync"
	
	"github.com/killallgit/ryan/pkg/logger"
)

// AgentFactoryFunc is a function that creates an agent
type AgentFactoryFunc func(config AgentConfig) (LangchainAgent, error)

// AgentFactory manages the creation and registration of agents
type AgentFactory struct {
	registry map[string]AgentFactoryFunc
	mu       sync.RWMutex
	log      *logger.Logger
}

// NewAgentFactory creates a new agent factory
func NewAgentFactory() *AgentFactory {
	factory := &AgentFactory{
		registry: make(map[string]AgentFactoryFunc),
		log:      logger.WithComponent("agent_factory"),
	}
	
	// Register default agents
	factory.RegisterDefaults()
	
	return factory
}

// RegisterDefaults registers all default agent types
func (f *AgentFactory) RegisterDefaults() {
	f.Register("conversational", NewConversationalAgent)
	f.Register("ollama-functions", NewOllamaFunctionsAgent)
	f.Register("openai-functions", NewOpenAIFunctionsAgent)
	f.Register("react", NewConversationalAgent) // Alias for conversational
	
	f.log.Debug("Registered default agents",
		"count", len(f.registry),
		"types", f.GetRegisteredTypes())
}

// Register registers a new agent factory function
func (f *AgentFactory) Register(agentType string, factoryFunc AgentFactoryFunc) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if _, exists := f.registry[agentType]; exists {
		return fmt.Errorf("agent type %s is already registered", agentType)
	}
	
	f.registry[agentType] = factoryFunc
	f.log.Debug("Registered agent type", "type", agentType)
	return nil
}

// Create creates an agent of the specified type
func (f *AgentFactory) Create(agentType string, config AgentConfig) (LangchainAgent, error) {
	f.mu.RLock()
	factoryFunc, exists := f.registry[agentType]
	f.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
	
	agent, err := factoryFunc(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent %s: %w", agentType, err)
	}
	
	f.log.Debug("Created agent",
		"type", agentType,
		"model", config.Model)
	
	return agent, nil
}

// GetRegisteredTypes returns all registered agent types
func (f *AgentFactory) GetRegisteredTypes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	types := make([]string, 0, len(f.registry))
	for agentType := range f.registry {
		types = append(types, agentType)
	}
	return types
}

// IsRegistered checks if an agent type is registered
func (f *AgentFactory) IsRegistered(agentType string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	_, exists := f.registry[agentType]
	return exists
}

// CreateBestAgent creates the best agent for the given configuration and request
func (f *AgentFactory) CreateBestAgent(config AgentConfig, request string) (LangchainAgent, error) {
	// Try to create agents in order of preference
	preferences := []string{"ollama-functions", "openai-functions", "conversational"}
	
	for _, agentType := range preferences {
		agent, err := f.Create(agentType, config)
		if err != nil {
			f.log.Debug("Failed to create agent type",
				"type", agentType,
				"error", err)
			continue
		}
		
		// Check if agent can handle the request
		canHandle, confidence := agent.CanHandle(request)
		if canHandle && confidence > 0.5 {
			f.log.Debug("Selected agent",
				"type", agentType,
				"confidence", confidence)
			return agent, nil
		}
	}
	
	// Fallback to conversational agent
	return f.Create("conversational", config)
}

// GlobalFactory is the global agent factory instance
var GlobalFactory = NewAgentFactory()