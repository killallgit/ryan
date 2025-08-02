/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models",
	Long:  `List all models that have been downloaded from Ollama`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			fmt.Printf("Failed to load configuration: %v\n", err)
			return
		}

		client := ollama.NewClient(cfg.Ollama.URL)
		controller := controllers.NewModelsController(client)

		if err := controller.ListModels(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error listing models: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}
