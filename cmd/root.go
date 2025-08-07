package cmd

import (
	"os"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/spf13/cobra"
)

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
			logger.Error("Failed to initialize default configuration: %v\n", err)
			return
		}

		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			logger.Error("Failed to load configuration: %v\n", err)
			return
		}

		// Initialize logger with config
		if err := logger.InitLoggerWithConfig(cfg.Logging.LogFile, cfg.Logging.Preserve, cfg.Logging.Level); err != nil {
			logger.Error("Failed to initialize logger: %v\n", err)
			return
		}
		defer func() {
			// Close log files
			if err := logger.Close(); err != nil {
				logger.Error("Failed to close log files: %v\n", err)
			}
		}()

		// Get command flags
		// Check for provider override
		provider, _ := cmd.Flags().GetString("provider")
		if provider != "" {
			cfg.Provider = provider
		}

		// Get model based on provider
		model := ""
		ollamaModel, _ := cmd.Flags().GetString("ollama.model")
		openaiModel, _ := cmd.Flags().GetString("openai.model")

		switch cfg.GetActiveProvider() {
		case "openai":
			if openaiModel != "" {
				model = openaiModel
			} else {
				model = cfg.GetActiveProviderModel()
			}
		case "ollama":
			fallthrough
		default:
			if ollamaModel != "" {
				model = ollamaModel
			} else {
				model = cfg.GetActiveProviderModel()
			}
		}

		// Get system prompt based on provider
		systemPromptPath := ""
		ollamaSystemPrompt, _ := cmd.Flags().GetString("ollama.system_prompt")
		openaiSystemPrompt, _ := cmd.Flags().GetString("openai.system_prompt")

		switch cfg.GetActiveProvider() {
		case "openai":
			if openaiSystemPrompt != "" {
				systemPromptPath = openaiSystemPrompt
			} else {
				systemPromptPath = cfg.GetActiveProviderSystemPrompt()
			}
		case "ollama":
			fallthrough
		default:
			if ollamaSystemPrompt != "" {
				systemPromptPath = ollamaSystemPrompt
			} else {
				systemPromptPath = cfg.GetActiveProviderSystemPrompt()
			}
		}

		continueHistory, _ := cmd.Flags().GetBool("continue")

		// Create application config
		appConfig := &AppConfig{
			Config:           cfg,
			Model:            model,
			SystemPromptPath: systemPromptPath,
			ContinueHistory:  continueHistory,
			DirectPrompt:     directPrompt,
			NoTUI:            noTUI,
			AgentType:        agentType,
			FallbackAgents:   fallbackAgents,
		}

		// Run the application
		if err := RunApplication(appConfig); err != nil {
			logger.Error("Error: %v\n", err)
			os.Exit(1)
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
	rootCmd.PersistentFlags().Bool("continue", false, "continue from previous chat history instead of starting fresh")
	rootCmd.PersistentFlags().StringVarP(&directPrompt, "prompt", "p", "", "execute a prompt directly without entering TUI")
	rootCmd.PersistentFlags().BoolVar(&noTUI, "no-tui", false, "run without TUI (requires --prompt)")
}
