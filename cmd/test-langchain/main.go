package main

import (
	"context"
	"fmt"
	"log"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/langchain"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
)

func main() {
	fmt.Println("🧪 Testing Enhanced LangChain Agent Tool Execution")
	fmt.Println("=================================================")

	// Initialize minimal configuration
	if err := config.InitializeDefaults(); err != nil {
		log.Fatal("Failed to initialize config:", err)
	}

	cfg, err := config.Load("")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Configure LangChain
	cfg.LangChain.Tools.MaxIterations = 5

	// Initialize logger
	if err := logger.InitLoggerWithConfig("./.ryan/logs/test.log", false, "debug"); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	// Create tool registry
	toolRegistry := tools.NewRegistry()
	if err := toolRegistry.RegisterBuiltinTools(); err != nil {
		log.Fatal("Failed to register tools:", err)
	}

	fmt.Printf("✅ Tools registered: %v\n", toolRegistry.List())

	// Create LangChain client
	client, err := langchain.NewClient(
		cfg.Ollama.URL,   // http://localhost:11434
		cfg.Ollama.Model, // Use configured model
		toolRegistry,
	)
	if err != nil {
		log.Fatal("Failed to create LangChain client:", err)
	}

	fmt.Printf("✅ LangChain client created\n")
	fmt.Printf("🔧 Available tools: %v\n", client.GetTools())

	// Test scenarios
	scenarios := []struct {
		name  string
		query string
	}{
		{
			name:  "Docker Images Count",
			query: "How many docker images are on this system? Use the execute_bash tool to run 'docker images | wc -l'",
		},
		{
			name:  "List Files",
			query: "What files are in the current directory? Use bash to run 'ls -la'",
		},
		{
			name:  "System Info",
			query: "What's the current date and time? Use bash to run 'date'",
		},
	}

	ctx := context.Background()

	for i, scenario := range scenarios {
		fmt.Printf("\n🧪 Test %d: %s\n", i+1, scenario.name)
		fmt.Printf("❓ Query: %s\n", scenario.query)
		fmt.Println("🔄 Processing...")

		response, err := client.SendMessage(ctx, scenario.query)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("✅ Response: %s\n", response)
		}

		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}

	// Test simple conversational query (should not use tools)
	fmt.Printf("\n🧪 Test: Conversational Query\n")
	fmt.Printf("❓ Query: Hello! What can you help me with?\n")
	fmt.Println("🔄 Processing...")

	response, err := client.SendMessage(ctx, "Hello! What can you help me with?")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Response: %s\n", response)
	}

	fmt.Println("\n🎉 Testing completed!")
	fmt.Println("📊 Check ./.ryan/logs/test.log for detailed logs")
}
