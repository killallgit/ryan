package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/killallgit/ryan/pkg/tools"
)

func main() {
	// Create a new tool registry
	registry := tools.NewRegistry()

	// Register the builtin tools
	if err := registry.RegisterBuiltinTools(); err != nil {
		log.Fatalf("Failed to register builtin tools: %v", err)
	}

	fmt.Println("=== Ryan Tool System Demo ===\n")

	// List available tools
	fmt.Println("Available tools:")
	for _, name := range registry.List() {
		fmt.Printf("- %s\n", name)
	}
	fmt.Println()

	// Demo 1: File Read Tool
	fmt.Println("Demo 1: Reading this demo file...")
	demoReadFile(registry)
	fmt.Println()

	// Demo 2: Bash Tool
	fmt.Println("Demo 2: Running bash commands...")
	demoBashTool(registry)
	fmt.Println()

	// Demo 3: Provider Format Conversion
	fmt.Println("Demo 3: Provider format conversion...")
	demoProviderFormats(registry)
	fmt.Println()

	fmt.Println("Demo completed!")
}

func demoReadFile(registry *tools.Registry) {
	// Get current file path
	wd, _ := os.Getwd()
	filePath := wd + "/examples/tool_demo.go"

	params := map[string]interface{}{
		"path":       filePath,
		"start_line": float64(1),
		"end_line":   float64(10), // First 10 lines
	}

	req := tools.ToolRequest{
		Name:       "read_file",
		Parameters: params,
		Context:    context.Background(),
	}

	result, err := registry.Execute(context.Background(), req)
	if err != nil {
		fmt.Printf("Error executing file read: %v\n", err)
		return
	}

	if result.Success {
		fmt.Printf("✅ Successfully read file (first 10 lines):\n")
		fmt.Printf("---\n%s\n---\n", result.Content)
	} else {
		fmt.Printf("❌ File read failed: %s\n", result.Error)
	}

	fmt.Printf("Execution time: %v\n", result.Metadata.ExecutionTime)
}

func demoBashTool(registry *tools.Registry) {
	commands := []map[string]interface{}{
		{
			"command": "echo 'Hello from Ryan tool system!'",
		},
		{
			"command": "pwd",
		},
		{
			"command": "ls -la | head -5",
		},
		{
			"command": "date",
		},
	}

	for i, params := range commands {
		fmt.Printf("Command %d: %s\n", i+1, params["command"])

		req := tools.ToolRequest{
			Name:       "execute_bash",
			Parameters: params,
			Context:    context.Background(),
		}

		result, err := registry.Execute(context.Background(), req)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n", err)
			continue
		}

		if result.Success {
			fmt.Printf("  ✅ Output: %s", result.Content)
			if result.Content != "" && result.Content[len(result.Content)-1] != '\n' {
				fmt.Println()
			}
		} else {
			fmt.Printf("  ❌ Failed: %s\n", result.Error)
		}

		fmt.Printf("  ⏱️  Time: %v\n\n", result.Metadata.ExecutionTime)
	}
}

func demoProviderFormats(registry *tools.Registry) {
	// Get all tools
	toolsMap := registry.GetTools()

	providers := []string{"openai", "anthropic", "mcp"}

	for _, provider := range providers {
		fmt.Printf("=== %s Format ===\n", provider)

		for _, tool := range toolsMap {
			definition, err := tools.ConvertToProvider(tool, provider)
			if err != nil {
				fmt.Printf("Error converting %s for %s: %v\n", tool.Name(), provider, err)
				continue
			}

			// Pretty print the JSON
			jsonData, err := json.MarshalIndent(definition, "", "  ")
			if err != nil {
				fmt.Printf("Error marshaling JSON: %v\n", err)
				continue
			}

			fmt.Printf("Tool: %s\n", tool.Name())
			fmt.Printf("%s\n\n", jsonData)
		}
	}
}