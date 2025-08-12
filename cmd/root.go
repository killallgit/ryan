package cmd

import (
	"fmt"
	"os"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/headless"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/memory"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/killallgit/ryan/pkg/stream"
	"github.com/killallgit/ryan/pkg/stream/providers/react"
	"github.com/killallgit/ryan/pkg/tools/structured"
	"github.com/killallgit/ryan/pkg/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
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

		// Get operating mode from flags (default to ExecuteMode)
		planMode, _ := cmd.Flags().GetBool("plan")
		mode := agent.ReactExecuteMode
		if planMode {
			mode = agent.ReactPlanMode
		}

		// Create the ReAct agent with the specified mode
		reactAgent, err := createReactAgent(llm, mode, continueHistory, skipPermissions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating ReAct agent: %v\n", err)
			os.Exit(1)
		}

		// Check if running in headless mode
		if headlessMode {
			runHeadless(reactAgent, promptValue, continueHistory)
		} else {
			runTUI(reactAgent, continueHistory)
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

// createReactAgent creates a ReAct agent with the given configuration
func createReactAgent(llm llms.Model, mode agent.ReactMode, continueHistory, skipPermissions bool) (agent.Agent, error) {
	// Create memory if continuing history
	var mem *memory.Memory
	if continueHistory {
		m, err := memory.New("default")
		if err != nil {
			return nil, fmt.Errorf("failed to create memory: %w", err)
		}
		mem = m
	}

	// Get tools with structured parameters
	tools := getStructuredTools(skipPermissions)

	// Create streaming handler with ReAct interceptor
	baseHandler := stream.NewConsoleHandler()
	reactHandler := react.NewInterceptor(baseHandler)

	// Create the ReAct agent
	reactAgent, err := agent.NewReactAgent(llm, tools, mem, reactHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to create ReAct agent: %w", err)
	}

	// Set the operating mode
	if err := reactAgent.SetMode(mode); err != nil {
		return nil, fmt.Errorf("failed to set mode: %w", err)
	}

	return reactAgent, nil
}

// createExecutorAgent creates an executor agent with the given configuration (kept for compatibility)
func createExecutorAgent(llm llms.Model, continueHistory, skipPermissions bool) (agent.Agent, error) {
	return agent.NewExecutorAgentWithOptions(llm, continueHistory, skipPermissions)
}

// getStructuredTools returns tools with structured parameters
func getStructuredTools(skipPermissions bool) []tools.Tool {
	settings := config.Get()
	var structuredTools []tools.Tool

	// Migrate enabled tools to structured format
	if settings.Tools.File.Read.Enabled {
		structuredTools = append(structuredTools, structured.MigrateFileRead(skipPermissions))
	}
	if settings.Tools.File.Write.Enabled {
		structuredTools = append(structuredTools, structured.MigrateFileWrite(skipPermissions))
	}
	if settings.Tools.Bash.Enabled {
		structuredTools = append(structuredTools, structured.MigrateBash(skipPermissions))
	}
	if settings.Tools.Search.Enabled {
		structuredTools = append(structuredTools, structured.MigrateRipgrep(skipPermissions))
	}
	if settings.Tools.Web.Enabled {
		structuredTools = append(structuredTools, structured.MigrateWebFetch(skipPermissions))
	}
	if settings.Tools.Git.Enabled {
		structuredTools = append(structuredTools, structured.MigrateGit(skipPermissions))
	}

	return structuredTools
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
	rootCmd.PersistentFlags().Bool("plan", false, "use plan mode instead of execute mode")
}

func initConfig() {
	// Initialize the config package
	if err := config.Init(cfgFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}
}
