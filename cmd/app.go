package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/agents"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/models"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/killallgit/ryan/pkg/tui"
)

// AppConfig contains all configuration needed to run the application
type AppConfig struct {
	Config           *config.Config
	Model            string
	SystemPromptPath string
	ContinueHistory  bool
	DirectPrompt     string
	NoTUI            bool
	AgentType        string
	FallbackAgents   string // comma-separated list
}

// RunApplication is the main entry point for the application logic
func RunApplication(appCfg *AppConfig) error {
	log := logger.WithComponent("app")
	log.Info("Application starting")

	// Initialize tools if provider supports them
	var toolRegistry *tools.Registry
	if appCfg.Config != nil {
		// Get provider URL based on active provider
		providerURL := appCfg.Config.GetActiveProviderURL()
		registry, err := agents.InitializeToolRegistry(providerURL, appCfg.Model)
		if err != nil {
			// Log warning but continue without tools
			log.Warn("Tool functionality disabled", "reason", err)
			fmt.Printf("Warning: %v\n", err)
			fmt.Printf("Tool functionality will be disabled.\n")

			// Suggest upgrading if it's a version issue
			if strings.Contains(err.Error(), "v0.4.0") {
				fmt.Printf("Consider upgrading your Ollama server.\n")
			}
		} else {
			toolRegistry = registry

			// Check model compatibility
			if !models.IsToolCompatible(appCfg.Model) {
				modelInfo := models.GetModelInfo(appCfg.Model)
				fmt.Printf("Warning: Model '%s' has %s tool calling support\n",
					appCfg.Model, modelInfo.ToolCompatibility.String())
				if modelInfo.Notes != "" {
					fmt.Printf("Note: %s\n", modelInfo.Notes)
				}

				// Suggest alternatives
				recommended := models.GetRecommendedModels()
				if len(recommended) > 0 {
					fmt.Printf("Recommended tool-compatible models: %v\n", recommended[:3])
				}
			}
		}
	}

	// Handle direct prompt execution
	if appCfg.DirectPrompt != "" || appCfg.NoTUI {
		return runDirectPrompt(appCfg, toolRegistry)
	}

	// Run TUI application
	return runTUI(appCfg, toolRegistry)
}

// runDirectPrompt handles non-interactive prompt execution
func runDirectPrompt(appCfg *AppConfig, toolRegistry *tools.Registry) error {
	if appCfg.DirectPrompt == "" {
		return fmt.Errorf("--prompt is required when using --no-tui")
	}

	log := logger.WithComponent("direct_prompt")
	log.Info("Executing direct prompt", "prompt_preview", truncateString(appCfg.DirectPrompt, 100))

	// Initialize orchestrator
	orchestratorCfg := &agents.OrchestratorConfig{
		ToolRegistry:    toolRegistry,
		Config:          appCfg.Config,
		Model:           appCfg.Model,
		OllamaURL:       appCfg.Config.Ollama.URL,
		EnableLLMIntent: appCfg.Config.Ollama.URL != "", // Enable LLM intent if Ollama is configured
	}

	orchestrator, err := agents.InitializeOrchestrator(orchestratorCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize orchestrator: %w", err)
	}

	// Execute the prompt
	ctx := context.Background()
	result, err := agents.ExecuteDirectPrompt(ctx, appCfg.DirectPrompt, orchestrator)
	if err != nil {
		return fmt.Errorf("failed to execute prompt: %w", err)
	}

	// Output results
	printAgentResult(result)

	return nil
}

// runTUI starts the terminal user interface
func runTUI(appCfg *AppConfig, toolRegistry *tools.Registry) error {
	log := logger.WithComponent("tui")

	// Parse fallback agents
	var fallbackAgents []string
	if appCfg.FallbackAgents != "" {
		fallbackAgents = strings.Split(appCfg.FallbackAgents, ",")
	}

	// Initialize controller
	controllerCfg := &controllers.InitConfig{
		Config:           appCfg.Config,
		Model:            appCfg.Model,
		SystemPromptPath: appCfg.SystemPromptPath,
		ToolRegistry:     toolRegistry,
		ContinueHistory:  appCfg.ContinueHistory,
		AgentType:        appCfg.AgentType,
		FallbackAgents:   fallbackAgents,
		UseNative:        true, // Enable native orchestrator
	}

	controller, err := controllers.InitializeNativeController(controllerCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize controller: %w", err)
	}

	// Create TUI application with native controller adapter
	log.Info("Creating TUI application")
	app, err := tui.NewApp(&NativeControllerAdapter{controller})
	if err != nil {
		return fmt.Errorf("failed to create TUI application: %w", err)
	}

	log.Info("Starting TUI application with health check")
	if err := app.StartWithHealthCheck(); err != nil {
		return fmt.Errorf("TUI application error: %w", err)
	}

	log.Info("Application shutting down")
	return nil
}

// printAgentResult formats and prints the agent execution result
func printAgentResult(result *agents.AgentResult) {
	fmt.Printf("\n=== Agent: %s ===\n", result.Metadata.AgentName)
	fmt.Printf("Status: %s\n", map[bool]string{true: "Success", false: "Failed"}[result.Success])
	fmt.Printf("Summary: %s\n\n", result.Summary)

	if result.Details != "" {
		fmt.Printf("=== Details ===\n%s\n", result.Details)
	}

	if len(result.Artifacts) > 0 {
		fmt.Printf("\n=== Artifacts ===\n")
		for key, value := range result.Artifacts {
			fmt.Printf("%s: %v\n", key, value)
		}
	}

	fmt.Printf("\n=== Execution Info ===\n")
	fmt.Printf("Duration: %v\n", result.Metadata.Duration)
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
