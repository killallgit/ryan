package orchestrator

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 10, config.MaxIterations)
	assert.True(t, config.ShowRoutingInfo)
	assert.Equal(t, 60, config.DefaultTimeout)
	assert.Equal(t, 2, config.RetryAttempts)

	// Check enabled agents
	assert.Contains(t, config.EnabledAgents, "tool_caller")
	assert.Contains(t, config.EnabledAgents, "reasoner")
	assert.Contains(t, config.EnabledAgents, "code_gen")

	// Check agent configs
	toolCallerConfig, exists := config.AgentConfigs["tool_caller"]
	assert.True(t, exists)
	assert.True(t, toolCallerConfig.Enabled)
	assert.Equal(t, 10, toolCallerConfig.Priority)
	assert.Equal(t, 0.2, toolCallerConfig.Temperature)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				MaxIterations:  5,
				DefaultTimeout: 30,
				RetryAttempts:  1,
				AgentConfigs:   map[string]AgentConfig{},
			},
			wantError: false,
		},
		{
			name: "invalid max iterations",
			config: &Config{
				MaxIterations:  0,
				DefaultTimeout: 30,
				RetryAttempts:  1,
			},
			wantError: true,
			errorMsg:  "max_iterations must be at least 1",
		},
		{
			name: "invalid timeout",
			config: &Config{
				MaxIterations:  5,
				DefaultTimeout: 0,
				RetryAttempts:  1,
			},
			wantError: true,
			errorMsg:  "default_timeout must be at least 1 second",
		},
		{
			name: "negative retry attempts",
			config: &Config{
				MaxIterations:  5,
				DefaultTimeout: 30,
				RetryAttempts:  -1,
			},
			wantError: true,
			errorMsg:  "retry_attempts cannot be negative",
		},
		{
			name: "invalid agent priority",
			config: &Config{
				MaxIterations:  5,
				DefaultTimeout: 30,
				RetryAttempts:  1,
				AgentConfigs: map[string]AgentConfig{
					"test_agent": {
						Enabled:  true,
						Priority: -1,
					},
				},
			},
			wantError: true,
			errorMsg:  "agent test_agent has invalid priority",
		},
		{
			name: "invalid temperature",
			config: &Config{
				MaxIterations:  5,
				DefaultTimeout: 30,
				RetryAttempts:  1,
				AgentConfigs: map[string]AgentConfig{
					"test_agent": {
						Enabled:     true,
						Priority:    5,
						Temperature: 3.0,
					},
				},
			},
			wantError: true,
			errorMsg:  "agent test_agent has invalid temperature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsAgentEnabled(t *testing.T) {
	config := &Config{
		EnabledAgents: []string{"tool_caller", "reasoner"},
		AgentConfigs: map[string]AgentConfig{
			"tool_caller": {Enabled: true},
			"reasoner":    {Enabled: false}, // Disabled in agent config
			"code_gen":    {Enabled: true},  // Not in enabled list
		},
	}

	assert.True(t, config.IsAgentEnabled("tool_caller"))
	assert.False(t, config.IsAgentEnabled("reasoner")) // Disabled in agent config
	assert.False(t, config.IsAgentEnabled("code_gen")) // Not in enabled list
	assert.False(t, config.IsAgentEnabled("unknown"))
}

func TestGetAgentConfig(t *testing.T) {
	config := &Config{
		AgentConfigs: map[string]AgentConfig{
			"tool_caller": {
				Enabled:     true,
				Priority:    10,
				Temperature: 0.5,
			},
		},
	}

	// Existing agent
	agentConfig, exists := config.GetAgentConfig("tool_caller")
	assert.True(t, exists)
	assert.Equal(t, 10, agentConfig.Priority)
	assert.Equal(t, 0.5, agentConfig.Temperature)

	// Non-existing agent
	_, exists = config.GetAgentConfig("unknown")
	assert.False(t, exists)
}

func TestGetAgentPriority(t *testing.T) {
	config := &Config{
		AgentConfigs: map[string]AgentConfig{
			"tool_caller": {Priority: 10},
			"reasoner":    {Priority: 5},
		},
	}

	assert.Equal(t, 10, config.GetAgentPriority("tool_caller"))
	assert.Equal(t, 5, config.GetAgentPriority("reasoner"))
	assert.Equal(t, 0, config.GetAgentPriority("unknown"))
}

func TestSelectBestAgent(t *testing.T) {
	config := &Config{
		EnabledAgents: []string{"tool_caller", "reasoner", "searcher"},
		AgentConfigs: map[string]AgentConfig{
			"tool_caller": {
				Enabled:      true,
				Priority:     10,
				Capabilities: []string{"bash_commands", "file_operations"},
			},
			"reasoner": {
				Enabled:      true,
				Priority:     8,
				Capabilities: []string{"analysis", "problem_solving"},
			},
			"searcher": {
				Enabled:      true,
				Priority:     6,
				Capabilities: []string{"file_search", "code_search"},
			},
		},
	}

	tests := []struct {
		name                 string
		requiredCapabilities []string
		availableAgents      []string
		expectedAgent        string
		expectError          bool
	}{
		{
			name:                 "select by capability match",
			requiredCapabilities: []string{"bash_commands"},
			availableAgents:      []string{"tool_caller", "reasoner"},
			expectedAgent:        "tool_caller",
			expectError:          false,
		},
		{
			name:                 "select by priority when equal match",
			requiredCapabilities: []string{"general"},
			availableAgents:      []string{"tool_caller", "reasoner"},
			expectedAgent:        "tool_caller", // Higher priority
			expectError:          false,
		},
		{
			name:                 "select searcher for search capability",
			requiredCapabilities: []string{"file_search"},
			availableAgents:      []string{"tool_caller", "reasoner", "searcher"},
			expectedAgent:        "searcher",
			expectError:          false,
		},
		{
			name:                 "no available agents",
			requiredCapabilities: []string{"unknown_capability"},
			availableAgents:      []string{},
			expectedAgent:        "",
			expectError:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := config.SelectBestAgent(tt.requiredCapabilities, tt.availableAgents)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAgent, agent)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Test with no config
	viper.Reset()
	config, err := LoadConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, DefaultConfig(), config)

	// Test with partial config
	viper.Reset()
	viper.Set("orchestrator.max_iterations", 20)
	viper.Set("orchestrator.show_routing_info", false)

	config, err = LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, 20, config.MaxIterations)
	assert.False(t, config.ShowRoutingInfo)

	// Test with invalid config
	viper.Reset()
	viper.Set("orchestrator.max_iterations", -1)

	config, err = LoadConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid orchestrator configuration")
}
