package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentReasoningChain tests that the agent can use reasoning to answer system queries
func TestAgentReasoningChain(t *testing.T) {
	// Skip if no Ollama available
	if !isOllamaAvailable() {
		t.Skip("Skipping test: Ollama is not available")
	}

	// Skip if using model incompatible with LangChain agents
	if !isLangChainCompatibleModel() {
		t.Skipf("Skipping agent test: model %s may not be compatible with LangChain agent parsing",
			os.Getenv("OLLAMA_DEFAULT_MODEL"))
	}

	t.Run("Agent reasons about file counting", func(t *testing.T) {
		// Create a temporary directory with known number of files
		tempDir := t.TempDir()

		// Create exactly 5 test files
		for i := 1; i <= 5; i++ {
			testFile := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
			err := os.WriteFile(testFile, []byte(fmt.Sprintf("test content %d", i)), 0644)
			require.NoError(t, err)
		}

		// Also create a subdirectory to ensure we're counting correctly
		subDir := filepath.Join(tempDir, "subdir")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		// Setup viper configuration with bash tool enabled
		setupViperWithBashTool(t)

		// Create LLM
		ollamaClient := ollama.NewClient()

		// Create executor agent with bash tool enabled (skip permissions for testing)
		executorAgent, err := agent.NewReactAgentWithOptions(ollamaClient.LLM, false, true)
		require.NoError(t, err, "Should create executor agent")
		defer executorAgent.Close()

		// Ask the agent to count files in the directory
		ctx := context.Background()
		prompt := fmt.Sprintf("How many files are in the directory %s? Just count the files, not subdirectories.", tempDir)

		t.Logf("Asking agent: %s", prompt)
		response, err := executorAgent.Execute(ctx, prompt)
		require.NoError(t, err, "Should execute prompt")
		t.Logf("Agent response: %s", response)

		// Check that the response contains "5" - the agent should have reasoned to use bash tool
		// and executed something like "ls -1 /path | grep -v / | wc -l" or "find /path -maxdepth 1 -type f | wc -l"
		assert.True(t,
			strings.Contains(response, "5") || strings.Contains(response, "five"),
			"Response should indicate there are 5 files: %s", response)
	})

	t.Run("Agent uses bash to check current directory", func(t *testing.T) {
		setupViperWithBashTool(t)
		ollamaClient := ollama.NewClient()
		executorAgent, err := agent.NewReactAgentWithOptions(ollamaClient.LLM, false, true)
		require.NoError(t, err)
		defer executorAgent.Close()

		ctx := context.Background()
		response, err := executorAgent.Execute(ctx, "What is the current working directory? Use the pwd command.")
		require.NoError(t, err)
		t.Logf("Agent response: %s", response)

		// The response should contain a path (starting with /)
		assert.True(t,
			strings.Contains(response, "/"),
			"Response should contain a directory path: %s", response)
	})

	t.Run("Agent counts Go files in project", func(t *testing.T) {
		// Create test directory with Go files
		tempDir := t.TempDir()

		// Create some .go files
		for i := 1; i <= 3; i++ {
			goFile := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
			err := os.WriteFile(goFile, []byte("package main\n"), 0644)
			require.NoError(t, err)
		}

		// Create some non-.go files
		for i := 1; i <= 2; i++ {
			txtFile := filepath.Join(tempDir, fmt.Sprintf("readme%d.txt", i))
			err := os.WriteFile(txtFile, []byte("readme"), 0644)
			require.NoError(t, err)
		}

		setupViperWithBashTool(t)
		ollamaClient := ollama.NewClient()
		executorAgent, err := agent.NewReactAgentWithOptions(ollamaClient.LLM, false, true)
		require.NoError(t, err)
		defer executorAgent.Close()

		ctx := context.Background()
		prompt := fmt.Sprintf("How many .go files are in %s?", tempDir)

		response, err := executorAgent.Execute(ctx, prompt)
		require.NoError(t, err)
		t.Logf("Agent response for Go files: %s", response)

		// Should find 3 .go files
		assert.True(t,
			strings.Contains(response, "3") || strings.Contains(response, "three"),
			"Response should indicate there are 3 Go files: %s", response)
	})
}

// setupViperWithBashTool initializes viper configuration with bash tool enabled
func setupViperWithBashTool(t *testing.T) {
	// Initialize config package first
	if err := config.Init(""); err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	// Set test-specific overrides
	viper.Set("provider", "ollama")

	// Use OLLAMA_DEFAULT_MODEL environment variable if set, otherwise default to qwen3:latest
	testModel := os.Getenv("OLLAMA_DEFAULT_MODEL")
	if testModel == "" {
		testModel = "qwen3:latest"
	}
	viper.Set("ollama.default_model", testModel)
	viper.Set("ollama.timeout", 90)

	// Enable tools including bash
	viper.Set("tools.enabled", true)
	viper.Set("tools.bash.enabled", true)
	viper.Set("tools.bash.timeout", 30)
	viper.Set("tools.file.read.enabled", true)
	viper.Set("tools.file.write.enabled", false) // Disable write for safety in tests

	// LangChain settings for agent
	viper.Set("langchain.memory_type", "window")
	viper.Set("langchain.memory_window_size", 10)
	viper.Set("langchain.tools.max_iterations", 10)
	viper.Set("langchain.tools.max_retries", 3)

	// Reload config after setting test values
	if err := config.Load(); err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}
}

// TestBashToolDirectly tests the bash tool independently
func TestBashToolDirectly(t *testing.T) {
	// This test verifies the bash tool works correctly without the agent
	t.Run("Bash tool executes commands", func(t *testing.T) {
		// Initialize config
		if err := config.Init(""); err != nil {
			t.Fatalf("Failed to initialize config: %v", err)
		}

		// Create bash tool with permission bypass for testing
		bashTool := tools.NewBashToolWithBypass(true)

		ctx := context.Background()

		// Test pwd command
		result, err := bashTool.Call(ctx, "pwd")
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.True(t, strings.HasPrefix(result, "/"), "pwd should return absolute path")

		// Test ls with pipe to count
		result, err = bashTool.Call(ctx, "ls -1 | wc -l")
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		// Result should be a number
		assert.Regexp(t, `^\d+`, strings.TrimSpace(result))

		// Test echo command
		result, err = bashTool.Call(ctx, "echo 'Hello from bash'")
		require.NoError(t, err)
		assert.Equal(t, "Hello from bash", strings.TrimSpace(result))
	})
}
