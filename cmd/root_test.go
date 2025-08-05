package cmd

import (
	"testing"

	"github.com/killallgit/ryan/pkg/agents"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestratorAPI(t *testing.T) {
	// This test verifies the orchestrator API is used correctly
	
	// Create orchestrator with new API
	orchestrator := agents.NewOrchestrator()
	assert.NotNil(t, orchestrator)
	
	// Create tool registry
	toolRegistry := tools.NewRegistry()
	err := toolRegistry.RegisterBuiltinTools()
	require.NoError(t, err)
	
	// Register built-in agents with tool registry
	err = orchestrator.RegisterBuiltinAgents(toolRegistry)
	assert.NoError(t, err)
	
	// Verify agents are registered
	agentList := orchestrator.ListAgents()
	assert.Greater(t, len(agentList), 0)
	
	// Should have at least the dispatcher agent
	found := false
	for _, agent := range agentList {
		if agent.Name() == "dispatcher" {
			found = true
			break
		}
	}
	assert.True(t, found, "Dispatcher agent should be registered")
}

func TestLangChainControllerAdapter(t *testing.T) {
	// Test that the adapter implements the required methods
	// Load config for testing
	_, err := config.Load("")
	require.NoError(t, err)
	
	// Create a minimal LangChain controller for testing
	controller, err := controllers.NewLangChainController("http://localhost:11434", "test-model", nil)
	require.NoError(t, err)
	
	adapter := &LangChainControllerAdapter{
		LangChainController: controller,
	}
	
	// These should compile and run without errors
	err = adapter.ValidateModel("test-model")
	assert.NoError(t, err)
	
	adapter.SetOllamaClient(nil)
	adapter.CleanThinkingBlocks()
}