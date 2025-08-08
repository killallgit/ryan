package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/killallgit/ryan/pkg/headless"
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
		// Save transient values before refreshing config
		promptValue := viper.GetString("prompt")
		headlessMode := viper.GetBool("headless")
		continueHistory := viper.GetBool("continue")

		// Refresh config (this will clear and restore transient values)
		refreshConfig(promptValue, headlessMode, continueHistory)

		// Check if running in headless mode
		if headlessMode {
			runHeadless()
		} else {
			if err := tui.StartApp(); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

func runHeadless() {
	// Get the prompt from config, default to "hello" if not provided
	prompt := viper.GetString("prompt")
	if prompt == "" {
		prompt = "hello"
	}

	// Create and run the headless runner
	runner, err := headless.NewRunner()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing headless mode: %v\n", err)
		os.Exit(1)
	}

	// Run with context
	ctx := context.Background()
	if err := runner.Run(ctx, prompt); err != nil {
		fmt.Fprintf(os.Stderr, "Error running headless mode: %v\n", err)
		os.Exit(1)
	}

	// Cleanup
	if err := runner.Cleanup(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cleanup error: %v\n", err)
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

	viper.SetDefault("langchain.memory.type", "window")
	viper.SetDefault("langchain.memory.window_size", 10)
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

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
