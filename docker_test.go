package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/killallgit/ryan/pkg/tools"
)

// TestDockerIntegration tests that our bash tool can execute Docker commands
func TestDockerIntegration(t *testing.T) {
	// Initialize tool registry
	registry := tools.NewRegistry()
	if err := registry.RegisterBuiltinTools(); err != nil {
		t.Fatalf("Failed to register tools: %v", err)
	}

	// Get the bash tool
	bashTool, exists := registry.Get("execute_bash")
	if !exists {
		t.Fatal("Bash tool not found in registry")
	}

	// Test Docker version command
	ctx := context.Background()
	params := map[string]interface{}{
		"command": "docker --version",
	}

	result, err := bashTool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Failed to execute docker --version: %v", err)
	}

	if !result.Success {
		t.Fatalf("Docker version command failed: %s", result.Error)
	}

	if !strings.Contains(result.Content, "Docker version") {
		t.Fatalf("Expected 'Docker version' in output, got: %s", result.Content)
	}

	fmt.Printf("✓ Docker version: %s\n", strings.TrimSpace(result.Content))

	// Test Docker images count command
	params = map[string]interface{}{
		"command": "docker images --quiet | wc -l",
	}

	result, err = bashTool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Failed to execute docker images count: %v", err)
	}

	if !result.Success {
		t.Fatalf("Docker images count failed: %s", result.Error)
	}

	imageCount := strings.TrimSpace(result.Content)
	fmt.Printf("✓ Docker images count: %s\n", imageCount)

	// Test the actual scenario: "How many docker images are on the system?"
	params = map[string]interface{}{
		"command": "docker images",
	}

	result, err = bashTool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Failed to execute docker images: %v", err)
	}

	if !result.Success {
		t.Fatalf("Docker images command failed: %s", result.Error)
	}

	// Count lines (excluding header)
	lines := strings.Split(strings.TrimSpace(result.Content), "\n")
	actualImageCount := len(lines) - 1 // Subtract 1 for header line

	fmt.Printf("✓ Docker images list retrieved successfully\n")
	fmt.Printf("✓ Found %d Docker images on the system\n", actualImageCount)

	// Verify the count matches
	if imageCount != fmt.Sprintf("%d", actualImageCount) {
		t.Logf("Warning: wc -l count (%s) doesn't match actual parsed count (%d)", imageCount, actualImageCount)
	}

	// Display first few images as example
	fmt.Println("✓ Sample images:")
	for i, line := range lines {
		if i == 0 || i > 3 { // Skip header and show only first 3 data lines
			continue
		}
		fmt.Printf("  %s\n", line)
	}
}
