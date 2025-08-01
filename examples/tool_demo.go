package main

import (
	"context"
	"fmt"
	"log"

	"github.com/killallgit/ryan/pkg/tools"
)

func main() {
	registry := tools.NewRegistry()
	if err := registry.RegisterBuiltinTools(); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}

	fmt.Println("=== Ryan Tool System ===")
	fmt.Printf("Available tools: %v\n\n", registry.List())

	// Demo bash tool
	fmt.Println("1. Bash Tool Demo:")
	demoBashTool(registry)

	// Demo file read tool  
	fmt.Println("\n2. File Read Tool Demo:")
	demoReadFile(registry)

	// Demo provider formats
	fmt.Println("\n3. Provider Format Demo:")
	demoProviderFormats(registry)
}

func demoReadFile(registry *tools.Registry) {
	req := tools.ToolRequest{
		Name: "read_file",
		Parameters: map[string]interface{}{
			"path":       "README.md",
			"start_line": float64(1),
			"end_line":   float64(5),
		},
	}

	result, err := registry.Execute(context.Background(), req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if result.Success {
		fmt.Printf("✅ Read README.md (lines 1-5):\n%s\n", result.Content)
	} else {
		fmt.Printf("❌ Failed: %s\n", result.Error)
	}
}

func demoBashTool(registry *tools.Registry) {
	req := tools.ToolRequest{
		Name: "execute_bash",
		Parameters: map[string]interface{}{
			"command": "echo 'Hello from Ryan tools!'",
		},
	}

	result, err := registry.Execute(context.Background(), req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if result.Success {
		fmt.Printf("✅ Command output: %s", result.Content)
	} else {
		fmt.Printf("❌ Failed: %s\n", result.Error)
	}
}

func demoProviderFormats(registry *tools.Registry) {
	providers := []string{"openai", "anthropic", "mcp"}
	
	for _, provider := range providers {
		definitions, err := registry.GetDefinitions(provider)
		if err != nil {
			fmt.Printf("Error getting %s definitions: %v\n", provider, err)
			continue
		}
		fmt.Printf("✅ %s format: %d tools converted\n", provider, len(definitions))
	}
}