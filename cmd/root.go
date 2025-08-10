package cmd

import (
	"fmt"
	"os"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/headless"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/killallgit/ryan/pkg/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/llms"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "ryan",
	Short: "Claude's friend",
	Long:  `Open source Claude Code alternative.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Save transient values before refreshing config
		promptValue := viper.GetString("prompt")
		headlessMode := viper.GetBool("headless")
		continueHistory := viper.GetBool("continue")
		skipPermissions := viper.GetBool("skip_permissions")

		// Set transient values in config
		config.SetTransientValues(promptValue, headlessMode, continueHistory, skipPermissions)

		// Refresh config (this will clear and restore transient values)
		if err := config.RefreshConfig(promptValue, headlessMode, continueHistory); err != nil {
			fmt.Fprintf(os.Stderr, "Error refreshing config: %v\n", err)
			os.Exit(1)
		}

		// Initialize logger
		if err := logger.Init(); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing logger: %v\n", err)
			os.Exit(1)
		}
		defer logger.Close()

		// Create the LLM based on provider configuration
		llm, err := createLLM()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating LLM: %v\n", err)
			os.Exit(1)
		}

		// Create the executor agent once, to be used by both modes
		executorAgent, err := agent.NewExecutorAgent(llm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating executor agent: %v\n", err)
			os.Exit(1)
		}
		defer executorAgent.Close()

		// Check if running in headless mode
		if config.Global.Headless {
			runHeadless(executorAgent)
		} else {
			if err := tui.RunTUI(executorAgent); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

// createLLM creates an LLM instance based on the configured provider
func createLLM() (llms.Model, error) {
	switch config.Global.Provider {
	case "ollama":
		// Create Ollama LLM
		ollamaClient := ollama.NewClient()
		return ollamaClient.LLM, nil

	// Future providers can be added here
	// case "openai":
	//     return createOpenAILLM()
	// case "anthropic":
	//     return createAnthropicLLM()

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.Global.Provider)
	}
}

func runHeadless(executorAgent agent.Agent) {
	// Get the prompt from config
	prompt := config.Global.Prompt
	if prompt == "" {
		prompt = "hello"
	}

	// Simply run the headless mode - no terminal manipulation needed
	if err := headless.RunHeadless(executorAgent, prompt); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", ".ryan/settings.yaml", "config file (default is .ryan/settings.yaml)")

	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "log level")
	viper.BindPFlag("logging.level", rootCmd.PersistentFlags().Lookup("log-level"))

	rootCmd.PersistentFlags().Bool("persist", false, "persist system logs across sessions")
	viper.BindPFlag("logging.persist", rootCmd.PersistentFlags().Lookup("persist"))

	rootCmd.PersistentFlags().Bool("continue", false, "continue from previous chat history instead of starting fresh")
	viper.BindPFlag("continue", rootCmd.PersistentFlags().Lookup("continue"))

	rootCmd.PersistentFlags().StringP("prompt", "p", "", "execute a prompt directly without entering TUI")
	viper.BindPFlag("prompt", rootCmd.PersistentFlags().Lookup("prompt"))

	rootCmd.PersistentFlags().BoolP("headless", "H", false, "run without TUI (requires --prompt)")
	viper.BindPFlag("headless", rootCmd.PersistentFlags().Lookup("headless"))

	rootCmd.PersistentFlags().Bool("skip-permissions", false, "skip all ACL permission checks for tools")
	viper.BindPFlag("skip_permissions", rootCmd.PersistentFlags().Lookup("skip-permissions"))
}

func initConfig() {
	// Initialize the config package
	if err := config.Init(cfgFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}
}
