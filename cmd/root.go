package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/models"
	"github.com/killallgit/ryan/pkg/testing"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/killallgit/ryan/pkg/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "ryan",
	Short: "Claude's friend",
	Long:  `Open source Claude Code alternative.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize logger first
		if err := logger.InitLogger(); err != nil {
			fmt.Printf("Failed to initialize logger: %v\n", err)
			return
		}

		log := logger.WithComponent("main")
		log.Info("Application starting")

		model, _ := cmd.Flags().GetString("model")
		if model == "" {
			model = viper.GetString("ollama.model")
			if model == "" {
				model = "qwen2.5:7b" // Default to tool-compatible model
			}
		}

		systemPrompt, _ := cmd.Flags().GetString("ollama.system_prompt")

		log.Debug("Configuration loaded",
			"ollama_url", viper.GetString("ollama.url"),
			"model", model,
			"has_system_prompt", systemPrompt != "",
			"config_file", viper.ConfigFileUsed(),
		)

		ollamaURL := viper.GetString("ollama.url")
		client := chat.NewClient(ollamaURL)

		// Check Ollama server version and model compatibility before initializing tools
		tester := testing.NewModelCompatibilityTester(ollamaURL)
		version, versionSupported, err := tester.CheckOllamaVersion()

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
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .ryan/settings.yaml)")

	viper.SetDefault("ollama.url", "https://ollama.kitty-tetra.ts.net")
	viper.SetDefault("ollama.model", "qwen2.5:7b")
	viper.SetDefault("ollama.system_prompt", "")
	viper.SetDefault("show_thinking", true)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome == "" {
			xdgConfigHome = filepath.Join(home, ".config")
		}
		ryanCfgHome := filepath.Join(xdgConfigHome, ".ryan")
		viper.AddConfigPath("./.ryan")   // Check project directory first
		viper.AddConfigPath(ryanCfgHome) // Then check XDG config location
		viper.SetConfigType("yaml")
		viper.SetConfigName("settings.yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
