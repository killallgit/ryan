package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/killallgit/ryan/pkg/orchestrator"
	"github.com/killallgit/ryan/pkg/orchestrator/agents"
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
		// Get CLI arguments directly from cobra flags
		promptValue, _ := cmd.Flags().GetString("prompt")
		headlessMode, _ := cmd.Flags().GetBool("headless")
		continueHistory, _ := cmd.Flags().GetBool("continue")
		skipPermissions, _ := cmd.Flags().GetBool("skip-permissions")

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

		// Always use orchestrator mode
		runOrchestratorMode(llm, headlessMode, promptValue, continueHistory, skipPermissions)
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

// runOrchestratorMode handles orchestrator-based execution
func runOrchestratorMode(llm llms.Model, headlessMode bool, prompt string, continueHistory bool, skipPermissions bool) {
	logger.Info("Running in orchestrator mode")

	// Create orchestrator with debugging options
	orch, err := orchestrator.New(llm, orchestrator.WithMaxIterations(10))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating orchestrator: %v\n", err)
		os.Exit(1)
	}

	// Register real agents
	if err := agents.RegisterRealAgents(orch, llm, skipPermissions); err != nil {
		fmt.Fprintf(os.Stderr, "Error registering agents: %v\n", err)
		os.Exit(1)
	}

	if headlessMode {
		runOrchestratorHeadless(orch, prompt)
	} else {
		runOrchestratorTUI(orch, continueHistory)
	}
}

// runOrchestratorHeadless runs orchestrator in headless mode
func runOrchestratorHeadless(orch *orchestrator.Orchestrator, prompt string) {
	if prompt == "" {
		prompt = "hello"
	}

	logger.Info("Executing orchestrator query: %s", prompt)

	ctx := context.Background()
	result, err := orch.Execute(ctx, prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Orchestrator execution failed: %v\n", err)
		os.Exit(1)
	}

	// Save to history for future sessions
	historyManager, err := orchestrator.NewHistoryManager("orchestrator_session")
	if err != nil {
		logger.Warn("Could not initialize history manager: %v", err)
	} else {
		if err := historyManager.SaveTaskExecution(result); err != nil {
			logger.Warn("Could not save task execution to history: %v", err)
		}
		defer historyManager.Close()
	}

	// Print only the result
	fmt.Println(result.Result)
}

// runOrchestratorTUI runs orchestrator with TUI
func runOrchestratorTUI(orch *orchestrator.Orchestrator, continueHistory bool) {
	// Create agent wrapper to make orchestrator compatible with TUI
	agentWrapper, err := orchestrator.NewAgentWrapper(orch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating orchestrator wrapper: %v\n", err)
		os.Exit(1)
	}
	defer agentWrapper.Close()

	// Run TUI with the orchestrator wrapper
	if err := tui.RunTUIWithOptions(agentWrapper, continueHistory); err != nil {
		fmt.Printf("Error: %v\n", err)
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

	// CLI-only flags (not stored in configuration)
	rootCmd.PersistentFlags().Bool("continue", false, "continue from previous chat history instead of starting fresh")
	rootCmd.PersistentFlags().StringP("prompt", "p", "", "execute a prompt directly without entering TUI")
	rootCmd.PersistentFlags().BoolP("headless", "H", false, "run without TUI (requires --prompt)")
	rootCmd.PersistentFlags().Bool("skip-permissions", false, "skip all ACL permission checks for tools")
}

func initConfig() {
	// Initialize the config package
	if err := config.Init(cfgFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}
}
