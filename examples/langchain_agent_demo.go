package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/langchain"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
)

// This example demonstrates the enhanced LangChain integration
// Run with: go run examples/langchain_agent_demo.go

func main() {
	fmt.Println("🚀 LangChain Agent Demo - Docker Tool Calling")
	fmt.Println("====================================================")

	// Initialize minimal config for demo
	cfg := &config.Config{
		LangChain: config.LangChainConfig{
			Enabled: true,
			Tools: config.LangChainToolsConfig{
				UseAgentFramework:   true,
				AutonomousExecution: true,
				MaxIterations:       5,
			},
			Memory: config.LangChainMemoryConfig{
				Type:       "buffer",
				WindowSize: 10,
			},
			Streaming: config.LangChainStreamConfig{
				UseLangChain:         true,
				ProviderOptimization: true,
			},
		},
	}

	// Initialize logger
	if err := logger.InitLoggerWithConfig("./.ryan/demo.log", false, "debug"); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	// Create tool registry with our existing tools
	toolRegistry := tools.NewRegistry()
	if err := toolRegistry.RegisterBuiltinTools(); err != nil {
		log.Fatal("Failed to register tools:", err)
	}

	// Create enhanced LangChain client
	client, err := langchain.NewEnhancedClient(
		"http://localhost:11434", // Ollama URL
		"qwen3:latest",           // Model
		toolRegistry,             // Our tool registry
	)
	if err != nil {
		log.Fatal("Failed to create enhanced client:", err)
	}

	fmt.Printf("✅ Enhanced LangChain client initialized\n")
	fmt.Printf("📊 Available tools: %v\n", getToolNames(toolRegistry))
	fmt.Printf("🤖 Agent framework: %v\n", cfg.LangChain.Tools.UseAgentFramework)
	fmt.Println()

	// Demo scenarios
	scenarios := []struct {
		name        string
		query       string
		description string
	}{
		{
			name:        "Docker System Info",
			query:       "How many docker images are on the system?",
			description: "Agent should automatically call execute_bash with 'docker images | wc -l'",
		},
		{
			name:        "File Reading",
			query:       "What's in the README.md file?",
			description: "Agent should automatically call read_file with './README.md'",
		},
		{
			name:        "Multi-step Analysis", 
			query:       "Check how many docker images I have and also read the main.go file",
			description: "Agent should make multiple tool calls autonomously",
		},
		{
			name:        "Conversational",
			query:       "Hello! What can you help me with?",
			description: "Should respond conversationally without tools",
		},
	}

	ctx := context.Background()

	for i, scenario := range scenarios {
		fmt.Printf("🧪 Scenario %d: %s\n", i+1, scenario.name)
		fmt.Printf("❓ Query: %s\n", scenario.query)
		fmt.Printf("📝 Expected: %s\n", scenario.description)
		fmt.Println("🔄 Processing...")

		start := time.Now()

		// Use the agent to process the query
		response, err := client.SendMessage(ctx, scenario.query)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("✅ Response: %s\n", response)
		}

		duration := time.Since(start)
		fmt.Printf("⏱️  Execution time: %v\n", duration)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		// Small delay between scenarios
		time.Sleep(1 * time.Second)
	}

	// Demo streaming
	fmt.Println("🌊 Streaming Demo")
	fmt.Println("================================")

	outputChan := make(chan string, 100)
	
	// Start streaming in a goroutine
	go func() {
		defer close(outputChan)
		err := client.StreamMessage(ctx, "Explain what Docker is in a brief paragraph", outputChan)
		if err != nil {
			fmt.Printf("Streaming error: %v\n", err)
		}
	}()

	// Print streaming output
	fmt.Print("🤖 Streaming response: ")
	for chunk := range outputChan {
		fmt.Print(chunk)
	}
	fmt.Println()

	// Demo memory
	fmt.Println("\n🧠 Memory Demo")
	fmt.Println("=====================")

	memory := client.GetMemory()
	if memory != nil {
		memVars, err := memory.LoadMemoryVariables(ctx, map[string]any{})
		if err != nil {
			fmt.Printf("Memory error: %v\n", err)
		} else {
			fmt.Printf("💭 Memory state: %+v\n", memVars)
		}
	}

	fmt.Println("\n🎉 Demo completed!")
	fmt.Println("🔍 Check ./.ryan/demo.log for detailed logs")
}

// getToolNames returns list of tool names from registry
func getToolNames(registry *tools.Registry) []string {
	return registry.List()
}