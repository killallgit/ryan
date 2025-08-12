package orchestrator

import (
	"fmt"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/spf13/viper"
)

// Config represents the orchestrator configuration
type Config struct {
	// EnabledAgents specifies which agents are enabled
	EnabledAgents []string `mapstructure:"enabled_agents"`

	// MaxIterations is the maximum number of feedback loop iterations
	MaxIterations int `mapstructure:"max_iterations"`

	// ShowRoutingInfo determines if routing decisions are shown in output
	ShowRoutingInfo bool `mapstructure:"show_routing_info"`

	// AgentConfigs holds agent-specific configurations
	AgentConfigs map[string]AgentConfig `mapstructure:"agents"`

	// DefaultTimeout for agent operations in seconds
	DefaultTimeout int `mapstructure:"default_timeout"`

	// RetryAttempts for failed agent operations
	RetryAttempts int `mapstructure:"retry_attempts"`
}

// AgentConfig represents configuration for a specific agent
type AgentConfig struct {
	// Enabled indicates if this agent is enabled
	Enabled bool `mapstructure:"enabled"`

	// Priority determines agent selection priority (higher = preferred)
	Priority int `mapstructure:"priority"`

	// Timeout override for this specific agent (seconds)
	Timeout int `mapstructure:"timeout"`

	// MaxTokens limit for LLM calls
	MaxTokens int `mapstructure:"max_tokens"`

	// Temperature for LLM calls
	Temperature float64 `mapstructure:"temperature"`

	// CustomPrompt override for the agent
	CustomPrompt string `mapstructure:"custom_prompt"`

	// Capabilities that this agent can handle
	Capabilities []string `mapstructure:"capabilities"`
}

// DefaultConfig returns the default orchestrator configuration
func DefaultConfig() *Config {
	return &Config{
		EnabledAgents: []string{
			"tool_caller",
			"reasoner",
			"code_gen",
			"searcher",
			"planner",
		},
		MaxIterations:   10,
		ShowRoutingInfo: true,
		DefaultTimeout:  60,
		RetryAttempts:   2,
		AgentConfigs: map[string]AgentConfig{
			"tool_caller": {
				Enabled:     true,
				Priority:    10,
				Timeout:     30,
				MaxTokens:   2000,
				Temperature: 0.2,
				Capabilities: []string{
					"bash_commands",
					"file_operations",
					"git_operations",
					"search",
				},
			},
			"reasoner": {
				Enabled:     true,
				Priority:    8,
				Timeout:     45,
				MaxTokens:   4000,
				Temperature: 0.7,
				Capabilities: []string{
					"analysis",
					"problem_solving",
					"logic",
					"mathematical_reasoning",
				},
			},
			"code_gen": {
				Enabled:     true,
				Priority:    9,
				Timeout:     60,
				MaxTokens:   8000,
				Temperature: 0.3,
				Capabilities: []string{
					"code_generation",
					"refactoring",
					"debugging",
					"documentation",
				},
			},
			"searcher": {
				Enabled:     true,
				Priority:    7,
				Timeout:     30,
				MaxTokens:   2000,
				Temperature: 0.5,
				Capabilities: []string{
					"file_search",
					"code_search",
					"documentation_search",
				},
			},
			"planner": {
				Enabled:     true,
				Priority:    6,
				Timeout:     45,
				MaxTokens:   3000,
				Temperature: 0.6,
				Capabilities: []string{
					"task_decomposition",
					"strategy_planning",
					"workflow_design",
				},
			},
		},
	}
}

// LoadConfig loads orchestrator configuration from viper
func LoadConfig() (*Config, error) {
	config := DefaultConfig()

	// Set config prefix for orchestrator settings
	sub := viper.Sub("orchestrator")
	if sub == nil {
		logger.Debug("No orchestrator configuration found, using defaults")
		return config, nil
	}

	// Unmarshal configuration
	if err := sub.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal orchestrator config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid orchestrator configuration: %w", err)
	}

	logger.Info("Loaded orchestrator configuration: %d agents enabled", len(config.EnabledAgents))
	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.MaxIterations < 1 {
		return fmt.Errorf("max_iterations must be at least 1")
	}

	if c.DefaultTimeout < 1 {
		return fmt.Errorf("default_timeout must be at least 1 second")
	}

	if c.RetryAttempts < 0 {
		return fmt.Errorf("retry_attempts cannot be negative")
	}

	// Validate agent configs
	for name, agentConfig := range c.AgentConfigs {
		if agentConfig.Enabled && agentConfig.Priority < 0 {
			return fmt.Errorf("agent %s has invalid priority: %d", name, agentConfig.Priority)
		}

		if agentConfig.Temperature < 0 || agentConfig.Temperature > 2 {
			return fmt.Errorf("agent %s has invalid temperature: %f (must be 0-2)", name, agentConfig.Temperature)
		}

		if agentConfig.MaxTokens < 0 {
			return fmt.Errorf("agent %s has invalid max_tokens: %d", name, agentConfig.MaxTokens)
		}
	}

	return nil
}

// IsAgentEnabled checks if a specific agent type is enabled
func (c *Config) IsAgentEnabled(agentType string) bool {
	// Check in enabled agents list
	for _, enabled := range c.EnabledAgents {
		if enabled == agentType {
			// Also check agent-specific config
			if agentConfig, exists := c.AgentConfigs[agentType]; exists {
				return agentConfig.Enabled
			}
			return true
		}
	}
	return false
}

// GetAgentConfig returns configuration for a specific agent
func (c *Config) GetAgentConfig(agentType string) (AgentConfig, bool) {
	config, exists := c.AgentConfigs[agentType]
	return config, exists
}

// GetAgentPriority returns the priority for a specific agent
func (c *Config) GetAgentPriority(agentType string) int {
	if config, exists := c.AgentConfigs[agentType]; exists {
		return config.Priority
	}
	return 0
}

// SelectBestAgent selects the best agent based on capabilities and priority
func (c *Config) SelectBestAgent(requiredCapabilities []string, availableAgents []string) (string, error) {
	var bestAgent string
	bestPriority := -1
	bestCapabilityMatch := 0

	for _, agentType := range availableAgents {
		if !c.IsAgentEnabled(agentType) {
			continue
		}

		agentConfig, exists := c.GetAgentConfig(agentType)
		if !exists || !agentConfig.Enabled {
			continue
		}

		// Count capability matches
		capabilityMatch := 0
		for _, required := range requiredCapabilities {
			for _, capability := range agentConfig.Capabilities {
				if capability == required {
					capabilityMatch++
					break
				}
			}
		}

		// Select agent with best capability match and highest priority
		if capabilityMatch > bestCapabilityMatch ||
			(capabilityMatch == bestCapabilityMatch && agentConfig.Priority > bestPriority) {
			bestAgent = agentType
			bestPriority = agentConfig.Priority
			bestCapabilityMatch = capabilityMatch
		}
	}

	if bestAgent == "" {
		return "", fmt.Errorf("no suitable agent found for capabilities: %v", requiredCapabilities)
	}

	logger.Debug("Selected agent %s with priority %d and %d capability matches",
		bestAgent, bestPriority, bestCapabilityMatch)

	return bestAgent, nil
}
