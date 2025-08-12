package agent

import (
	"testing"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnifiedAgentSystem tests the unified agent system functionality
func TestUnifiedAgentSystem(t *testing.T) {
	// Set up test configuration
	viper.Reset()
	viper.Set("continue", false)
	viper.Set("vectorstore.enabled", false)
	viper.Set("agent.max_iterations", 5)
	viper.Set("ollama.default_model", "test-model")

	// Initialize config
	require.NoError(t, config.Init(""))
	require.NoError(t, config.Load())

	t.Run("mrkl_agent_creation", func(t *testing.T) {
		// Create mock LLM using existing MockLLM
		mockLLM := NewMockLLM([]string{"Final Answer: MRKL test response"})

		// Create MRKL agent
		agent, err := NewMRKLAgent(mockLLM, false, true)
		require.NoError(t, err)
		require.NotNil(t, agent)
		defer agent.Close()

		// Test that agent implements Agent interface
		var _ Agent = agent

		// Test that agent implements CustomizableAgent interface
		var _ CustomizableAgent = agent
	})

	t.Run("mrkl_agent_basic_functionality", func(t *testing.T) {
		mockLLM := NewMockLLM([]string{"Final Answer: Execution test response"})
		agent, err := NewMRKLAgent(mockLLM, false, true)
		require.NoError(t, err)
		defer agent.Close()

		// Test basic methods work without panicking
		sent, recv := agent.GetTokenStats()
		assert.GreaterOrEqual(t, sent, 0)
		assert.GreaterOrEqual(t, recv, 0)

		err = agent.ClearMemory()
		assert.NoError(t, err)
	})

	t.Run("mrkl_agent_custom_prompt", func(t *testing.T) {
		mockLLM := NewMockLLM([]string{"Final Answer: Custom prompt response"})
		agent, err := NewMRKLAgent(mockLLM, false, true)
		require.NoError(t, err)
		defer agent.Close()

		// Test custom prompt setting (should not panic)
		customPrompt := "You are a helpful test assistant."
		agent.SetCustomPrompt(customPrompt)

		// Verify the agent still functions after customization
		sent, recv := agent.GetTokenStats()
		assert.GreaterOrEqual(t, sent, 0)
		assert.GreaterOrEqual(t, recv, 0)
	})

	t.Run("react_agent_creation", func(t *testing.T) {
		mockLLM := NewMockLLM([]string{"Final Answer: React test response"})

		// Create React agent
		agent, err := NewReactAgent(mockLLM, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, agent)
		defer agent.Close()

		// Test that agent implements Agent interface
		var _ Agent = agent

		// Test that agent implements CustomizableAgent interface
		var _ CustomizableAgent = agent
	})

	t.Run("react_agent_basic_functionality", func(t *testing.T) {
		mockLLM := NewMockLLM([]string{"Final Answer: React execution response"})
		agent, err := NewReactAgent(mockLLM, nil, nil, nil)
		require.NoError(t, err)
		defer agent.Close()

		// Test basic methods work without panicking
		sent, recv := agent.GetTokenStats()
		assert.Equal(t, 0, sent) // React agent returns 0,0 currently
		assert.Equal(t, 0, recv)

		err = agent.ClearMemory()
		assert.NoError(t, err)
	})

	t.Run("react_agent_custom_prompt", func(t *testing.T) {
		mockLLM := NewMockLLM([]string{"Final Answer: React custom response"})
		agent, err := NewReactAgent(mockLLM, nil, nil, nil)
		require.NoError(t, err)
		defer agent.Close()

		// Test custom prompt setting (should not panic)
		customPrompt := "You are a React test assistant."
		agent.SetCustomPrompt(customPrompt)

		// Verify the agent still functions after customization
		sent, recv := agent.GetTokenStats()
		assert.Equal(t, 0, sent)
		assert.Equal(t, 0, recv)
	})
}

// TestPromptCustomizationFlow tests the CLI-style prompt customization flow
func TestPromptCustomizationFlow(t *testing.T) {
	// Set up test configuration
	viper.Reset()
	viper.Set("continue", false)
	viper.Set("vectorstore.enabled", false)
	viper.Set("agent.max_iterations", 5)
	viper.Set("ollama.default_model", "test-model")

	require.NoError(t, config.Init(""))
	require.NoError(t, config.Load())

	t.Run("cli_style_prompt_customization", func(t *testing.T) {
		mockLLM := NewMockLLM([]string{"Final Answer: CLI customization response"})
		agent, err := NewMRKLAgent(mockLLM, false, true)
		require.NoError(t, err)
		defer agent.Close()

		// Simulate CLI flag processing like in root.go
		customPrompt := "You are a specialized assistant."
		appendPrompt := "Always be helpful."
		planningBias := true

		// Build final prompt like applyPromptCustomizations does
		finalPrompt := customPrompt
		if appendPrompt != "" {
			if finalPrompt != "" {
				finalPrompt += "\n\n" + appendPrompt
			} else {
				finalPrompt = appendPrompt
			}
		}

		if planningBias {
			planBias := "IMPORTANT: For complex or ambiguous tasks, always plan first before executing. Ask for user confirmation before proceeding with multi-step operations."
			if finalPrompt != "" {
				finalPrompt += "\n\n" + planBias
			} else {
				finalPrompt = planBias
			}
		}

		// Apply the combined prompt
		if finalPrompt != "" {
			agent.SetCustomPrompt(finalPrompt)
		}

		// Verify the agent functions with the combined prompt
		sent, recv := agent.GetTokenStats()
		assert.GreaterOrEqual(t, sent, 0)
		assert.GreaterOrEqual(t, recv, 0)
	})
}

// TestAgentMemoryAndTokens tests memory and token functionality
func TestAgentMemoryAndTokens(t *testing.T) {
	// Set up test configuration
	viper.Reset()
	viper.Set("continue", false)
	viper.Set("vectorstore.enabled", false)
	viper.Set("agent.max_iterations", 5)
	viper.Set("ollama.default_model", "test-model")

	require.NoError(t, config.Init(""))
	require.NoError(t, config.Load())

	t.Run("mrkl_agent_memory_operations", func(t *testing.T) {
		mockLLM := NewMockLLM([]string{"Final Answer: Memory test"})
		agent, err := NewMRKLAgent(mockLLM, false, true)
		require.NoError(t, err)
		defer agent.Close()

		// Test memory clearing
		err = agent.ClearMemory()
		assert.NoError(t, err)
	})

	t.Run("mrkl_agent_token_stats", func(t *testing.T) {
		mockLLM := NewMockLLM([]string{"Final Answer: Token test"})
		agent, err := NewMRKLAgent(mockLLM, false, true)
		require.NoError(t, err)
		defer agent.Close()

		// Test token stats (should not panic)
		sent, recv := agent.GetTokenStats()
		assert.GreaterOrEqual(t, sent, 0)
		assert.GreaterOrEqual(t, recv, 0)
	})

	t.Run("react_agent_memory_operations", func(t *testing.T) {
		mockLLM := NewMockLLM([]string{"Final Answer: React memory test"})
		agent, err := NewReactAgent(mockLLM, nil, nil, nil)
		require.NoError(t, err)
		defer agent.Close()

		// Test memory clearing (should not error even with nil memory)
		err = agent.ClearMemory()
		assert.NoError(t, err)
	})

	t.Run("react_agent_token_stats", func(t *testing.T) {
		mockLLM := NewMockLLM([]string{"Final Answer: React token test"})
		agent, err := NewReactAgent(mockLLM, nil, nil, nil)
		require.NoError(t, err)
		defer agent.Close()

		// Test token stats (currently returns 0,0 for React agent)
		sent, recv := agent.GetTokenStats()
		assert.Equal(t, 0, sent)
		assert.Equal(t, 0, recv)
	})
}

// TestAgentResourceCleanup tests proper resource cleanup
func TestAgentResourceCleanup(t *testing.T) {
	// Set up test configuration
	viper.Reset()
	viper.Set("continue", false)
	viper.Set("vectorstore.enabled", false)
	viper.Set("agent.max_iterations", 5)
	viper.Set("ollama.default_model", "test-model")

	require.NoError(t, config.Init(""))
	require.NoError(t, config.Load())

	t.Run("mrkl_agent_cleanup", func(t *testing.T) {
		mockLLM := NewMockLLM([]string{"Final Answer: Cleanup test"})
		agent, err := NewMRKLAgent(mockLLM, false, true)
		require.NoError(t, err)

		// Close should not error
		err = agent.Close()
		assert.NoError(t, err)

		// Closing again should not error
		err = agent.Close()
		assert.NoError(t, err)
	})

	t.Run("react_agent_cleanup", func(t *testing.T) {
		mockLLM := NewMockLLM([]string{"Final Answer: React cleanup test"})
		agent, err := NewReactAgent(mockLLM, nil, nil, nil)
		require.NoError(t, err)

		// Close should not error
		err = agent.Close()
		assert.NoError(t, err)

		// Closing again should not error
		err = agent.Close()
		assert.NoError(t, err)
	})
}
