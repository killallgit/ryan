package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
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
				model = "qwen2.5-coder:1.5b-base"
			}
		}

		systemPrompt, _ := cmd.Flags().GetString("ollama.system_prompt")

		log.Debug("Configuration loaded",
			"ollama_url", viper.GetString("ollama.url"),
			"model", model,
			"has_system_prompt", systemPrompt != "",
			"config_file", viper.ConfigFileUsed(),
		)

		client := chat.NewClient(viper.GetString("ollama.url"))

		var controller *controllers.ChatController
		if systemPrompt != "" {
			controller = controllers.NewChatControllerWithSystem(client, model, systemPrompt)
			log.Debug("Created chat controller with system prompt")
		} else {
			controller = controllers.NewChatController(client, model)
			log.Debug("Created chat controller without system prompt")
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ryan.yaml)")

	viper.SetDefault("ollama.url", "https://ollama.kitty-tetra.ts.net")
	viper.SetDefault("ollama.model", "qwen2.5-coder:1.5b-base")
	viper.SetDefault("ollama.system_prompt", "")
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
