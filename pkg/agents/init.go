package agents

import (
	"context"
	"fmt"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/models"
	"github.com/killallgit/ryan/pkg/tools"
)

// OrchestratorConfig contains configuration for orchestrator initialization
type OrchestratorConfig struct {
	ToolRegistry *tools.Registry
	Config       *config.Config
	Model        string
}

// InitializeOrchestrator creates and configures a new orchestrator with all built-in agents
func InitializeOrchestrator(cfg *OrchestratorConfig) (*Orchestrator, error) {
	log := logger.WithComponent("agent_init")
	log.Info("Initializing orchestrator")

	// Create orchestrator
	orchestrator := NewOrchestrator()

	// Register built-in agents with tool registry
	if cfg.ToolRegistry != nil {
		if err := orchestrator.RegisterBuiltinAgents(cfg.ToolRegistry); err != nil {
			return nil, fmt.Errorf("failed to register built-in agents: %w", err)
		}
		log.Info("Registered built-in agents with orchestrator")
	}

	return orchestrator, nil
}

// InitializeLangChainOrchestrator creates and configures a LangChain orchestrator
func InitializeLangChainOrchestrator(cfg *OrchestratorConfig) (*LangchainOrchestrator, error) {
	log := logger.WithComponent("langchain_init")
	log.Info("Initializing LangChain orchestrator")

	if cfg.ToolRegistry == nil {
		return nil, fmt.Errorf("tool registry is required for LangChain orchestrator")
	}

	orchestrator := NewLangchainOrchestrator(cfg.ToolRegistry)

	// Set preferred agent if specified in config
	if cfg.Config != nil && cfg.Config.Agents.Preferred != "" {
		if err := orchestrator.SetPreferredAgent(cfg.Config.Agents.Preferred); err != nil {
			log.Warn("Invalid preferred agent type", "agent", cfg.Config.Agents.Preferred, "error", err)
		} else {
			log.Info("Set preferred agent", "type", cfg.Config.Agents.Preferred)
		}
	}

	// Set fallback chain if specified
	if cfg.Config != nil && len(cfg.Config.Agents.FallbackChain) > 0 {
		if err := orchestrator.SetFallbackChain(cfg.Config.Agents.FallbackChain); err != nil {
			log.Warn("Invalid fallback chain", "chain", cfg.Config.Agents.FallbackChain, "error", err)
		} else {
			log.Info("Set fallback chain", "agents", cfg.Config.Agents.FallbackChain)
		}
	}

	return orchestrator, nil
}

// ExecuteDirectPrompt handles direct prompt execution with the orchestrator
func ExecuteDirectPrompt(ctx context.Context, prompt string, orchestrator *Orchestrator) (*AgentResult, error) {
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	log := logger.WithComponent("direct_prompt")
	log.Info("Executing direct prompt", "prompt_preview", truncateString(prompt, 100))

	// Execute the prompt
	result, err := orchestrator.Execute(ctx, prompt, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute prompt: %w", err)
	}

	return &result, nil
}

// InitializeToolRegistry creates and configures the tool registry based on Ollama compatibility
func InitializeToolRegistry(ollamaURL string, model string) (*tools.Registry, error) {
	log := logger.WithComponent("tools_init")

	// Check Ollama server version and model compatibility
	version, versionSupported, err := models.CheckOllamaVersion(ollamaURL)
	if err != nil {
		log.Warn("Could not check Ollama server version", "error", err)
		return nil, fmt.Errorf("could not verify Ollama server compatibility: %w", err)
	}

	if !versionSupported {
		log.Warn("Ollama server version does not support tool calling",
			"version", version, "minimum_required", "0.4.0")
		return nil, fmt.Errorf("Ollama server v%s does not support tool calling (requires v0.4.0+)", version)
	}

	log.Info("Ollama server supports tool calling", "version", version)

	// Check if the selected model supports tool calling
	if !models.IsToolCompatible(model) {
		modelInfo := models.GetModelInfo(model)
		log.Warn("Selected model may not support tool calling",
			"model", model, "compatibility", modelInfo.ToolCompatibility.String())
		// Continue anyway, but log the warning
	}

	// Initialize tool registry with built-in tools
	toolRegistry := tools.NewRegistry()
	if err := toolRegistry.RegisterBuiltinTools(); err != nil {
		return nil, fmt.Errorf("failed to register built-in tools: %w", err)
	}

	log.Debug("Initialized tool registry with built-in tools")
	return toolRegistry, nil
}
