package cmd

import (
	"fmt"
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
		defer func() {
			// Close log files
			if err := logger.Close(); err != nil {
				fmt.Printf("Failed to close log files: %v\n", err)
			}
		}()

		// Get command flags
		model, _ := cmd.Flags().GetString("model")
		if model == "" {
			model = cfg.Ollama.Model
		}

		systemPromptPath, _ := cmd.Flags().GetString("ollama.system_prompt")
		if systemPromptPath == "" {
			systemPromptPath = cfg.Ollama.SystemPrompt
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
			fmt.Printf("Error: %v\n", err)
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
	rootCmd.PersistentFlags().String("model", "", "model to use (overrides config)")
	rootCmd.PersistentFlags().String("ollama.system_prompt", "", "system prompt to use (overrides config)")
	rootCmd.PersistentFlags().Bool("continue", false, "continue from previous chat history instead of starting fresh")
	rootCmd.PersistentFlags().StringVarP(&directPrompt, "prompt", "p", "", "execute a prompt directly without entering TUI")
	rootCmd.PersistentFlags().BoolVar(&noTUI, "no-tui", false, "run without TUI (requires --prompt)")
	rootCmd.PersistentFlags().StringVar(&agentType, "agent", "", "preferred agent type (conversational, ollama-functions, openai-functions)")
	rootCmd.PersistentFlags().StringVar(&fallbackAgents, "fallback-agents", "", "comma-separated list of fallback agents")
}
