package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/killallgit/ryan/pkg/agents"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/models"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/killallgit/ryan/pkg/tui"
	"github.com/spf13/cobra"
)

// LangChainControllerAdapter adapts LangChainController to both ChatControllerInterface and TUI ControllerInterface
type LangChainControllerAdapter struct {
	*controllers.LangChainController
}

// Implement any missing methods needed by the interface
func (lca *LangChainControllerAdapter) StartStreaming(ctx context.Context, content string) (<-chan controllers.StreamingUpdate, error) {
	return lca.LangChainController.StartStreaming(ctx, content)
}

// SetOllamaClient accepts any type to satisfy tui.ControllerInterface
func (lca *LangChainControllerAdapter) SetOllamaClient(client any) {
	// LangChainController also expects any type, so we can pass it directly
	lca.LangChainController.SetOllamaClient(client)
}

func (lca *LangChainControllerAdapter) ValidateModel(model string) error {
	return lca.LangChainController.ValidateModel(model)
}

func (lca *LangChainControllerAdapter) GetTokenUsage() (promptTokens, responseTokens int) {
	return lca.LangChainController.GetTokenUsage()
}

func (lca *LangChainControllerAdapter) CleanThinkingBlocks() {
	lca.LangChainController.CleanThinkingBlocks()
}

// GetLastAssistantMessage returns the last assistant message from the conversation
func (lca *LangChainControllerAdapter) GetLastAssistantMessage() (chat.Message, bool) {
	history := lca.GetHistory()
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			return history[i], true
		}
	}
	return chat.Message{}, false
}

// GetLastUserMessage returns the last user message from the conversation
func (lca *LangChainControllerAdapter) GetLastUserMessage() (chat.Message, bool) {
	history := lca.GetHistory()
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "human" {
			return history[i], true
		}
	}
	return chat.Message{}, false
}

// GetMessageCount returns the number of messages in the conversation
func (lca *LangChainControllerAdapter) GetMessageCount() int {
	return len(lca.GetHistory())
}

// HasSystemMessage returns true if the conversation has a system message
func (lca *LangChainControllerAdapter) HasSystemMessage() bool {
	history := lca.GetHistory()
	for _, msg := range history {
		if msg.Role == "system" {
			return true
		}
	}
	return false
}

// SetModelWithValidation sets the model after validating it
func (lca *LangChainControllerAdapter) SetModelWithValidation(model string) error {
	if err := lca.ValidateModel(model); err != nil {
		return err
	}
	lca.SetModel(model)
	return nil
}

// SetToolRegistry sets the tool registry
func (lca *LangChainControllerAdapter) SetToolRegistry(registry *tools.Registry) {
	// LangChainController doesn't have SetToolRegistry method
	// The tool registry is set during construction via NewLangChainController
	// This is a no-op for compatibility with the interface
}

var cfgFile string
var directPrompt string
var noTUI bool
var agentType string
var fallbackAgents string

var rootCmd = &cobra.Command{
	Use:   "ryan",
	Short: "Claude's friend",
	Long:  `Open source Claude Code alternative.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize default configuration if needed
		if err := config.InitializeDefaults(); err != nil {
			fmt.Printf("Failed to initialize default configuration: %v\n", err)
			return
		}

		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			fmt.Printf("Failed to load configuration: %v\n", err)
			return
		}

		// Initialize logger with config
		if err := logger.InitLoggerWithConfig(cfg.Logging.LogFile, cfg.Logging.Preserve, cfg.Logging.Level); err != nil {
			fmt.Printf("Failed to initialize logger: %v\n", err)
			return
		}

		log := logger.WithComponent("main")
		log.Info("Application starting")

		model, _ := cmd.Flags().GetString("model")
		if model == "" {
			model = cfg.Ollama.Model
		}

		systemPromptPath, _ := cmd.Flags().GetString("ollama.system_prompt")
		if systemPromptPath == "" {
			systemPromptPath = cfg.Ollama.SystemPrompt
		}

		// Check if we should continue from previous chat history
		continueHistory, _ := cmd.Flags().GetBool("continue")

		// Read system prompt from file if specified
		var systemPrompt string
		if systemPromptPath != "" {
			// Handle relative paths
			if !filepath.IsAbs(systemPromptPath) {
				systemPromptPath = filepath.Join(".", systemPromptPath)
			}

			content, err := os.ReadFile(systemPromptPath)
			if err != nil {
				log.Warn("Failed to read system prompt file", "path", systemPromptPath, "error", err)
				fmt.Printf("Warning: Could not read system prompt file '%s': %v\n", systemPromptPath, err)
				systemPrompt = "" // Continue without system prompt
			} else {
				systemPrompt = strings.TrimSpace(string(content))
				log.Debug("Loaded system prompt from file", "path", systemPromptPath, "length", len(systemPrompt))
			}
		}

		log.Debug("Configuration loaded",
			"ollama_url", cfg.Ollama.URL,
			"model", model,
			"has_system_prompt", systemPrompt != "",
			"system_prompt_file", systemPromptPath,
			"config_file", config.GetConfigFileUsed(),
		)

		// Create LangChain agent client
		log.Debug("Creating LangChain agent client", "base_url", cfg.Ollama.URL, "model", model)

		// Check Ollama server version and model compatibility before initializing tools
		version, versionSupported, err := models.CheckOllamaVersion(cfg.Ollama.URL)

		var toolRegistry *tools.Registry
		if err != nil {
			log.Warn("Could not check Ollama server version", "error", err)
			fmt.Printf("Warning: Could not verify Ollama server compatibility: %v\n", err)
			// Continue without tools for now
			toolRegistry = nil
		} else if !versionSupported {
			log.Warn("Ollama server version does not support tool calling",
				"version", version, "minimum_required", "0.4.0")
			fmt.Printf("Warning: Ollama server v%s does not support tool calling (requires v0.4.0+)\n", version)
			fmt.Printf("Tool functionality will be disabled. Consider upgrading your Ollama server.\n")
			toolRegistry = nil
		} else {
			log.Info("Ollama server supports tool calling", "version", version)

			// Check if the selected model supports tool calling
			if !models.IsToolCompatible(model) {
				modelInfo := models.GetModelInfo(model)
				log.Warn("Selected model may not support tool calling",
					"model", model, "compatibility", modelInfo.ToolCompatibility.String())
				fmt.Printf("Warning: Model '%s' has %s tool calling support\n", model, modelInfo.ToolCompatibility.String())
				if modelInfo.Notes != "" {
					fmt.Printf("Note: %s\n", modelInfo.Notes)
				}

				// Suggest better alternatives
				recommended := models.GetRecommendedModels()
				if len(recommended) > 0 {
					fmt.Printf("Recommended tool-compatible models: %v\n", recommended[:3]) // Show first 3
				}
			}

			// Initialize tool registry with built-in tools
			toolRegistry = tools.NewRegistry()
			if err := toolRegistry.RegisterBuiltinTools(); err != nil {
				log.Error("Failed to register built-in tools", "error", err)
				fmt.Printf("Failed to register built-in tools: %v\n", err)
				return
			}
			log.Debug("Initialized tool registry with built-in tools")
		}

		// Create agent orchestrator if agent type is specified or configured
		var agentOrchestrator *agents.LangchainOrchestrator

		// Determine preferred agent (CLI flag takes precedence over config)
		finalAgentType := agentType
		if finalAgentType == "" && cfg.Agents.Preferred != "" {
			finalAgentType = cfg.Agents.Preferred
		}

		// Determine fallback chain (CLI flag takes precedence over config)
		var finalFallbackChain []string
		if fallbackAgents != "" {
			finalFallbackChain = strings.Split(fallbackAgents, ",")
		} else if len(cfg.Agents.FallbackChain) > 0 {
			finalFallbackChain = cfg.Agents.FallbackChain
		}

		if finalAgentType != "" || len(finalFallbackChain) > 0 || cfg.Agents.AutoSelect {
			agentOrchestrator = agents.NewLangchainOrchestrator(toolRegistry)

			// Set preferred agent if specified
			if finalAgentType != "" {
				if err := agentOrchestrator.SetPreferredAgent(finalAgentType); err != nil {
					log.Warn("Invalid agent type specified", "agent", finalAgentType, "error", err)
					fmt.Printf("Warning: Invalid agent type '%s': %v\n", finalAgentType, err)
				} else {
					log.Info("Set preferred agent", "type", finalAgentType)
					if cfg.Agents.ShowSelection {
						fmt.Printf("Using preferred agent: %s\n", finalAgentType)
					}
				}
			}

			// Set fallback chain if specified
			if len(finalFallbackChain) > 0 {
				if err := agentOrchestrator.SetFallbackChain(finalFallbackChain); err != nil {
					log.Warn("Invalid fallback chain", "chain", finalFallbackChain, "error", err)
					fmt.Printf("Warning: Invalid fallback chain: %v\n", err)
				} else {
					log.Info("Set fallback chain", "agents", finalFallbackChain)
					if cfg.Agents.ShowSelection {
						fmt.Printf("Fallback chain: %v\n", finalFallbackChain)
					}
				}
			}
		}

		// Create LangChain controller with agent framework
		var langchainController *controllers.LangChainController

		if systemPrompt != "" {
			langchainController, err = controllers.NewLangChainControllerWithSystem(cfg.Ollama.URL, model, systemPrompt, toolRegistry)
			if err != nil {
				log.Error("Failed to create LangChain controller with system prompt", "error", err)
				fmt.Printf("Failed to create LangChain controller: %v\n", err)
				return
			}
			if toolRegistry != nil {
				log.Debug("Created LangChain controller with system prompt and tools")
			} else {
				log.Debug("Created LangChain controller with system prompt but without tools")
			}
		} else {
			langchainController, err = controllers.NewLangChainController(cfg.Ollama.URL, model, toolRegistry)
			if err != nil {
				log.Error("Failed to create LangChain controller", "error", err)
				fmt.Printf("Failed to create LangChain controller: %v\n", err)
				return
			}
			if toolRegistry != nil {
				log.Debug("Created LangChain controller without system prompt but with tools")
			} else {
				log.Debug("Created LangChain controller without system prompt or tools")
			}
		}

		// Set the orchestrator in the controller if available
		if agentOrchestrator != nil {
			langchainController.SetAgentOrchestrator(agentOrchestrator)
		}

		// Clear chat history if not continuing from previous session
		if !continueHistory {
			log.Debug("Starting fresh chat session - clearing history")
			langchainController.Reset()
		} else {
			log.Debug("Continuing from previous chat history")
		}

		// Check if we should execute a direct prompt
		if directPrompt != "" || noTUI {
			if directPrompt == "" {
				fmt.Printf("Error: --prompt is required when using --no-tui\n")
				return
			}

			log.Info("Executing direct prompt", "prompt", directPrompt)

			// Create orchestrator with new API
			orchestrator := agents.NewOrchestrator()

			// Register built-in agents with tool registry
			if err := orchestrator.RegisterBuiltinAgents(toolRegistry); err != nil {
				log.Error("Failed to register built-in agents", "error", err)
				fmt.Printf("Failed to register agents: %v\n", err)
				return
			}

			// Execute the prompt
			ctx := context.Background()
			result, err := orchestrator.Execute(ctx, directPrompt, nil)
			if err != nil {
				log.Error("Failed to execute prompt", "error", err)
				fmt.Printf("Error: %v\n", err)
				return
			}

			// Output results
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
			if len(result.Metadata.ToolsUsed) > 0 {
				fmt.Printf("Tools Used: %v\n", result.Metadata.ToolsUsed)
			}
			if len(result.Metadata.FilesProcessed) > 0 {
				fmt.Printf("Files Processed: %d\n", len(result.Metadata.FilesProcessed))
			}

			// Close log files and exit
			if err := logger.Close(); err != nil {
				fmt.Printf("Failed to close log files: %v\n", err)
			}

			// Exit after direct prompt execution
			return
		}

		log.Info("Creating TUI application")

		// Create tview-based TUI
		app, err := tui.NewApp(&LangChainControllerAdapter{langchainController})
		if err != nil {
			log.Error("Failed to create TUI application", "error", err)
			fmt.Printf("Failed to create TUI application: %v\n", err)
			return
		}

		log.Info("Starting TUI application")
		if err := app.Run(); err != nil {
			log.Error("Application error", "error", err)
			fmt.Printf("TUI application error: %v\n", err)
		}

		log.Info("Application shutting down")

		// Close log files
		if err := logger.Close(); err != nil {
			fmt.Printf("Failed to close log files: %v\n", err)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .ryan/settings.yaml)")
	rootCmd.PersistentFlags().String("model", "", "model to use (overrides config)")
	rootCmd.PersistentFlags().String("ollama.system_prompt", "", "system prompt to use (overrides config)")
	rootCmd.PersistentFlags().Bool("continue", false, "continue from previous chat history instead of starting fresh")
	rootCmd.PersistentFlags().StringVarP(&directPrompt, "prompt", "p", "", "execute a prompt directly without entering TUI")
	rootCmd.PersistentFlags().BoolVar(&noTUI, "no-tui", false, "run without TUI (requires --prompt)")
	rootCmd.PersistentFlags().StringVar(&agentType, "agent", "", "preferred agent type (conversational, ollama-functions, openai-functions)")
	rootCmd.PersistentFlags().StringVar(&fallbackAgents, "fallback-agents", "", "comma-separated list of fallback agents")
}
