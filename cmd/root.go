package cmd

import (
	"fmt"
	"os"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/models"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/killallgit/ryan/pkg/tui"
	"github.com/spf13/cobra"
)

var cfgFile string

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
		if err := logger.InitLoggerWithConfig(cfg.Logging.File, cfg.Logging.Preserve, cfg.Logging.Level); err != nil {
			fmt.Printf("Failed to initialize logger: %v\n", err)
			return
		}

		log := logger.WithComponent("main")
		log.Info("Application starting")

		model, _ := cmd.Flags().GetString("model")
		if model == "" {
			model = cfg.Ollama.Model
		}

		systemPrompt, _ := cmd.Flags().GetString("ollama.system_prompt")
		if systemPrompt == "" {
			systemPrompt = cfg.Ollama.SystemPrompt
		}

		log.Debug("Configuration loaded",
			"ollama_url", cfg.Ollama.URL,
			"model", model,
			"has_system_prompt", systemPrompt != "",
			"config_file", config.GetConfigFileUsed(),
		)

		client := chat.NewStreamingClientWithTimeout(cfg.Ollama.URL, cfg.Ollama.Timeout)

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

		var controller *controllers.ChatController
		if systemPrompt != "" {
			controller = controllers.NewChatControllerWithSystem(client, model, systemPrompt, toolRegistry)
			if toolRegistry != nil {
				log.Debug("Created chat controller with system prompt and tools")
			} else {
				log.Debug("Created chat controller with system prompt but without tools")
			}
		} else {
			controller = controllers.NewChatController(client, model, toolRegistry)
			if toolRegistry != nil {
				log.Debug("Created chat controller without system prompt but with tools")
			} else {
				log.Debug("Created chat controller without system prompt or tools")
			}
		}

		log.Info("Creating TUI application")
		app, err := tui.NewApp(controller)
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
}
