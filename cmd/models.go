/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models",
	Long:  `List all models that have been downloaded from Ollama`,
	Run: func(cmd *cobra.Command, args []string) {
		client := ollama.NewClient(viper.GetString("ollama.url"))
		controller := controllers.NewModelsController(client)
		
		if err := controller.ListModels(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error listing models: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(modelsCmd)
	viper.SetDefault("ollama.url", "https://ollama.kitty-tetra.ts.net")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// modelsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// modelsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
