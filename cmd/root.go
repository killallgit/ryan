package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/headless"
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
		useOrchestrator, _ := cmd.Flags().GetBool("orchestrate")

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

		// Choose between orchestrator and simple agent modes
		if useOrchestrator {
			runOrchestratorMode(llm, headlessMode, promptValue, continueHistory, skipPermissions)
		} else {
			// Create the executor agent once, to be used by both modes
			// Pass skipPermissions to the agent creation
			executorAgent, err := createExecutorAgent(llm, continueHistory, skipPermissions)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating executor agent: %v\n", err)
				os.Exit(1)
			}
			defer executorAgent.Close()

			// Check if running in headless mode
			if headlessMode {
				runHeadless(executorAgent, promptValue, continueHistory)
			} else {
				runTUI(executorAgent, continueHistory)
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

// createExecutorAgent creates an executor agent with the given configuration
func createExecutorAgent(llm llms.Model, continueHistory, skipPermissions bool) (agent.Agent, error) {
	return agent.NewExecutorAgentWithOptions(llm, continueHistory, skipPermissions)
}

func runHeadless(executorAgent agent.Agent, prompt string, continueHistory bool) {
	// Use provided prompt or default
	if prompt == "" {
		prompt = "hello"
	}

	// Simply run the headless mode - no terminal manipulation needed
	if err := headless.RunHeadlessWithOptions(executorAgent, prompt, continueHistory); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runTUI(executorAgent agent.Agent, continueHistory bool) {
	if err := tui.RunTUIWithOptions(executorAgent, continueHistory); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
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
	fmt.Printf("🎯 Orchestrator Mode: %s\n\n", prompt)

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

	// Print detailed results with debugging info
	fmt.Printf("✅ Task completed with status: %s\n", result.Status)
	fmt.Printf("⏱️  Duration: %v\n", result.Duration)
	fmt.Printf("🔄 Agent interactions: %d\n\n", len(result.History))

	// Print execution flow
	fmt.Println("📋 Execution Flow:")
	for i, response := range result.History {
		fmt.Printf("  %d. %s: %s\n", i+1, response.AgentType, response.Status)
		if len(response.ToolsCalled) > 0 {
			for _, tool := range response.ToolsCalled {
				fmt.Printf("     🔧 Used tool: %s\n", tool.Name)
			}
		}
	}
	fmt.Println()

	// Print final result
	fmt.Println("📄 Final Result:")
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
	rootCmd.PersistentFlags().Bool("orchestrate", false, "use multi-agent orchestrator system for complex tasks")
}

func initConfig() {
	// Initialize the config package
	if err := config.Init(cfgFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}
}
