package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "ryan",
	Short: "Claude's friend",
	Long: `Open source Claude Code alternative.`,
	Run: func(cmd *cobra.Command, args []string) { 
		model, _ := cmd.Flags().GetString("model")
		if model == "" {
			model = viper.GetString("ollama.model")
			if model == "" {
				model = "qwen2.5-coder:1.5b-base"
			}
		}

		systemPrompt, _ := cmd.Flags().GetString("ollama.system_prompt")
		
		client := chat.NewClient(viper.GetString("ollama.url"))
		
		var controller *controllers.ChatController
		if systemPrompt != "" {
			controller = controllers.NewChatControllerWithSystem(client, model, systemPrompt)
		} else {
			controller = controllers.NewChatController(client, model)
		}

		app, err := tui.NewApp(controller)
		if err != nil {
			fmt.Printf("Failed to create TUI application: %v\n", err)
			return
		}

		if err := app.Run(); err != nil {
			fmt.Printf("TUI application error: %v\n", err)
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
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ryan.yaml)")

	
	viper.SetDefault("ollama.url", "https://ollama.kitty-tetra.ts.net")
	viper.SetDefault("ollama.model", "qwen2.5-coder:1.5b-base")
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
		viper.AddConfigPath(ryanCfgHome)
		viper.AddConfigPath("./.ryan")
		viper.SetConfigType("yaml")
		viper.SetConfigName("settings.yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
