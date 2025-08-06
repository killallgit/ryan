package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/killallgit/ryan/pkg/agents"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
)

// InitConfig contains configuration for controller initialization
type InitConfig struct {
	Config           *config.Config
	Model            string
	SystemPromptPath string
	ToolRegistry     *tools.Registry
	ContinueHistory  bool
	AgentType        string
	FallbackAgents   []string
}

// InitializeLangChainController creates and configures a new LangChain controller
func InitializeLangChainController(cfg *InitConfig) (*LangChainController, error) {
	log := logger.WithComponent("controller_init")

	// Determine the model to use
	model := cfg.Model
	if model == "" && cfg.Config != nil {
		model = cfg.Config.Ollama.Model
	}

	// Determine system prompt path
	systemPromptPath := cfg.SystemPromptPath
	if systemPromptPath == "" && cfg.Config != nil {
		systemPromptPath = cfg.Config.Ollama.SystemPrompt
	}

	// Load system prompt if specified
	var systemPrompt string
	if systemPromptPath != "" {
		systemPrompt = loadSystemPrompt(systemPromptPath)
	}

	// Get Ollama URL from config
	ollamaURL := ""
	if cfg.Config != nil {
		ollamaURL = cfg.Config.Ollama.URL
	}

	// Create LangChain controller
	var controller *LangChainController
	var err error

	if systemPrompt != "" {
		controller, err = NewLangChainControllerWithSystem(ollamaURL, model, systemPrompt, cfg.ToolRegistry)
		if err != nil {
			return nil, fmt.Errorf("failed to create LangChain controller with system prompt: %w", err)
		}
		log.Debug("Created LangChain controller with system prompt",
			"has_tools", cfg.ToolRegistry != nil)
	} else {
		controller, err = NewLangChainController(ollamaURL, model, cfg.ToolRegistry)
		if err != nil {
			return nil, fmt.Errorf("failed to create LangChain controller: %w", err)
		}
		log.Debug("Created LangChain controller without system prompt",
			"has_tools", cfg.ToolRegistry != nil)
	}

	// Set up agent orchestrator if needed
	if shouldConfigureAgents(cfg) {
		orchestrator, err := setupAgentOrchestrator(cfg)
		if err != nil {
			log.Warn("Failed to set up agent orchestrator", "error", err)
		} else {
			controller.SetAgentOrchestrator(orchestrator)
			log.Info("Configured agent orchestrator for controller")
		}
	}

	// Clear history if not continuing
	if !cfg.ContinueHistory {
		log.Debug("Starting fresh chat session - clearing history")
		controller.Reset()
	} else {
		log.Debug("Continuing from previous chat history")
	}

	return controller, nil
}

// loadSystemPrompt loads a system prompt from a file
func loadSystemPrompt(path string) string {
	log := logger.WithComponent("controller_init")

	// Handle relative paths
	if !filepath.IsAbs(path) {
		path = filepath.Join(".", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		log.Warn("Failed to read system prompt file", "path", path, "error", err)
		return ""
	}

	systemPrompt := strings.TrimSpace(string(content))
	log.Debug("Loaded system prompt from file", "path", path, "length", len(systemPrompt))
	return systemPrompt
}

// shouldConfigureAgents determines if agent orchestrator should be configured
func shouldConfigureAgents(cfg *InitConfig) bool {
	if cfg.Config == nil {
		return false
	}

	return cfg.AgentType != "" ||
		len(cfg.FallbackAgents) > 0 ||
		cfg.Config.Agents.Preferred != "" ||
		len(cfg.Config.Agents.FallbackChain) > 0 ||
		cfg.Config.Agents.AutoSelect
}

// setupAgentOrchestrator creates and configures the agent orchestrator
func setupAgentOrchestrator(cfg *InitConfig) (*agents.LangchainOrchestrator, error) {
	log := logger.WithComponent("controller_init")

	orchestrator := agents.NewLangchainOrchestrator(cfg.ToolRegistry)

	// Determine preferred agent (CLI flag takes precedence)
	preferredAgent := cfg.AgentType
	if preferredAgent == "" && cfg.Config != nil && cfg.Config.Agents.Preferred != "" {
		preferredAgent = cfg.Config.Agents.Preferred
	}

	if preferredAgent != "" {
		if err := orchestrator.SetPreferredAgent(preferredAgent); err != nil {
			log.Warn("Invalid agent type specified", "agent", preferredAgent, "error", err)
		} else {
			log.Info("Set preferred agent", "type", preferredAgent)
			if cfg.Config != nil && cfg.Config.Agents.ShowSelection {
				fmt.Printf("Using preferred agent: %s\n", preferredAgent)
			}
		}
	}

	// Determine fallback chain (CLI flag takes precedence)
	fallbackChain := cfg.FallbackAgents
	if len(fallbackChain) == 0 && cfg.Config != nil && len(cfg.Config.Agents.FallbackChain) > 0 {
		fallbackChain = cfg.Config.Agents.FallbackChain
	}

	if len(fallbackChain) > 0 {
		if err := orchestrator.SetFallbackChain(fallbackChain); err != nil {
			log.Warn("Invalid fallback chain", "chain", fallbackChain, "error", err)
		} else {
			log.Info("Set fallback chain", "agents", fallbackChain)
			if cfg.Config != nil && cfg.Config.Agents.ShowSelection {
				fmt.Printf("Fallback chain: %v\n", fallbackChain)
			}
		}
	}

	return orchestrator, nil
}
