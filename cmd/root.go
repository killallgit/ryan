package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/headless"
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

		// Refresh config (this will clear and restore transient values)
		refreshConfig(promptValue, headlessMode, continueHistory)

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
		if headlessMode {
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
	provider := viper.GetString("provider")

	switch provider {
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
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}
}

func runHeadless(executorAgent agent.Agent) {
	// Get the prompt from config
	prompt := viper.GetString("prompt")
	if prompt == "" {
		prompt = "hello"
	}

	// Simply run the headless mode - no terminal manipulation needed
	if err := headless.RunHeadless(executorAgent, prompt); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func refreshConfig(promptValue string, headlessMode bool, continueHistory bool) {
	// Clear transient flags that shouldn't be persisted
	viper.Set("prompt", "")
	viper.Set("headless", false)
	viper.Set("continue", false)

	// Ensure config directory exists
	dirFromCfgFile := filepath.Dir(cfgFile)
	if _, err := os.Stat(dirFromCfgFile); os.IsNotExist(err) {
		os.Mkdir(dirFromCfgFile, 0755)
	}

	// Write config without transient values
	if err := viper.WriteConfigAs(cfgFile); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
	}

	// Restore transient values for use in this session
	viper.Set("prompt", promptValue)
	viper.Set("headless", headlessMode)
	viper.Set("continue", continueHistory)
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

	rootCmd.PersistentFlags().Bool("continue", false, "continue from previous chat history instead of starting fresh")
	viper.BindPFlag("continue", rootCmd.PersistentFlags().Lookup("continue"))

	rootCmd.PersistentFlags().StringP("prompt", "p", "", "execute a prompt directly without entering TUI")
	viper.BindPFlag("prompt", rootCmd.PersistentFlags().Lookup("prompt"))

	rootCmd.PersistentFlags().BoolP("headless", "H", false, "run without TUI (requires --prompt)")
	viper.BindPFlag("headless", rootCmd.PersistentFlags().Lookup("headless"))

	viper.SetDefault("provider", "ollama")
	viper.SetDefault("show_thinking", true)

	viper.SetDefault("ollama.url", "http://localhost:11434")
	viper.SetDefault("ollama.default_model", "qwen3:latest")
	viper.SetDefault("ollama.timeout", 90)

	viper.SetDefault("logging.log_file", "./.ryan/system.log")
	viper.SetDefault("logging.preserve", true)
	viper.SetDefault("logging.level", "info")

	viper.SetDefault("vectorstore.enabled", true)
	viper.SetDefault("vectorstore.provider", "chromem")
	viper.SetDefault("vectorstore.persistence_dir", "./.ryan/vectorstore")
	viper.SetDefault("vectorstore.enable_persistence", true)
	viper.SetDefault("vectorstore.embedder.provider", "ollama")
	viper.SetDefault("vectorstore.embedder.model", "nomic-embed-text")
	viper.SetDefault("vectorstore.embedder.base_url", "http://localhost:11434")
	viper.SetDefault("vectorstore.embedder.api_key", "")

	viper.SetDefault("vectorstore.indexer.chunk_size", 1000)
	viper.SetDefault("vectorstore.indexer.chunk_overlap", 200)
	viper.SetDefault("vectorstore.indexer.auto_index", false)

	viper.SetDefault("langchain.memory_type", "window")
	viper.SetDefault("langchain.memory_window_size", 10)
	viper.SetDefault("langchain.tools.max_iterations", 10)
	viper.SetDefault("langchain.tools.max_retries", 3)

}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("./.ryan")
		viper.SetConfigType("yaml")
		viper.SetConfigName("settings")
	}

	viper.AutomaticEnv()

	// Override ollama.url with OLLAMA_HOST if set
	if ollamaHost := os.Getenv("OLLAMA_HOST"); ollamaHost != "" {
		viper.Set("ollama.url", ollamaHost)
	}

	// Override ollama.default_model with OLLAMA_DEFAULT_MODEL if set
	if ollamaModel := os.Getenv("OLLAMA_DEFAULT_MODEL"); ollamaModel != "" {
		viper.Set("ollama.default_model", ollamaModel)
	}

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
