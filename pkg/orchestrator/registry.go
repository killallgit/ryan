package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/killallgit/ryan/pkg/logger"
)

// Agent defines the interface for all specialized agents
type Agent interface {
	// Execute processes a routing decision and returns a response
	Execute(ctx context.Context, decision *RouteDecision, state *TaskState) (*AgentResponse, error)

	// GetCapabilities returns the agent's capabilities
	GetCapabilities() []string

	// GetType returns the agent type
	GetType() AgentType
}

// AgentRegistry manages registered agents
type AgentRegistry struct {
	agents map[AgentType]Agent
	models map[AgentType]string // Maps agent types to preferred models
	mu     sync.RWMutex
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[AgentType]Agent),
		models: make(map[AgentType]string),
	}
}

// Register adds an agent to the registry
func (r *AgentRegistry) Register(agentType AgentType, agent Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[agentType]; exists {
		return fmt.Errorf("agent type %s already registered", agentType)
	}

	r.agents[agentType] = agent
	logger.Info("Registered agent: %s", agentType)
	return nil
}

// RegisterWithModel adds an agent with a preferred model
func (r *AgentRegistry) RegisterWithModel(agentType AgentType, agent Agent, model string) error {
	if err := r.Register(agentType, agent); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[agentType] = model
	logger.Info("Registered model %s for agent %s", model, agentType)
	return nil
}

// GetAgent retrieves an agent by type
func (r *AgentRegistry) GetAgent(agentType AgentType) (Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[agentType]
	if !exists {
		return nil, fmt.Errorf("agent type %s not registered", agentType)
	}
	return agent, nil
}

// GetModel retrieves the preferred model for an agent type
func (r *AgentRegistry) GetModel(agentType AgentType) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	model, exists := r.models[agentType]
	return model, exists
}

// HasAgent checks if an agent type is registered
func (r *AgentRegistry) HasAgent(agentType AgentType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.agents[agentType]
	return exists
}

// ListAgents returns all registered agent types
func (r *AgentRegistry) ListAgents() []AgentType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]AgentType, 0, len(r.agents))
	for agentType := range r.agents {
		types = append(types, agentType)
	}
	return types
}

// Unregister removes an agent from the registry
func (r *AgentRegistry) Unregister(agentType AgentType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[agentType]; !exists {
		return fmt.Errorf("agent type %s not registered", agentType)
	}

	delete(r.agents, agentType)
	delete(r.models, agentType)
	logger.Info("Unregistered agent: %s", agentType)
	return nil
}

// Clear removes all agents from the registry
func (r *AgentRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents = make(map[AgentType]Agent)
	r.models = make(map[AgentType]string)
	logger.Info("Cleared agent registry")
}

// DefaultModelMapping returns the default model mapping for agent types
func DefaultModelMapping() map[AgentType]string {
	return map[AgentType]string{
		AgentOrchestrator: "llama3.1:8b",             // Good general reasoning
		AgentToolCaller:   "llama3-groq-tool-use:8b", // Optimized for tool calling
		AgentCodeGen:      "qwen2.5-coder:7b",        // Specialized for code generation
		AgentReasoner:     "llama3.1:8b",             // General reasoning
		AgentSearcher:     "llama3.1:8b",             // Search and analysis
		AgentPlanner:      "llama3.1:8b",             // Planning and decomposition
	}
}
