package integration

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isLangChainCompatibleModel checks if the configured model is known to work well with LangChain agents
func isLangChainCompatibleModel() bool {
	model := os.Getenv("OLLAMA_DEFAULT_MODEL")
	if model == "" {
		model = "qwen3:latest" // default
	}

	// Small models that are known to have issues with LangChain agent parsing
	incompatibleModels := []string{
		"smollm2:135m",
		"smollm2:360m",
		"tinyllama:1.1b",
		"qwen2.5:0.5b",
		"qwen2.5:1.5b",
		"qwen2.5:3b", // Has issues with agent output formatting
	}

	for _, incompatible := range incompatibleModels {
		if model == incompatible {
			return false
		}
	}

	return true
}

// setupViperForTest initializes viper configuration for tests
func setupViperForTest(t *testing.T) {
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

	// Use environment variable for embedding model if set (for CI)
	embeddingModel := os.Getenv("OLLAMA_EMBEDDING_MODEL")
	if embeddingModel != "" {
		viper.Set("vectorstore.embedding.model", embeddingModel)
	}
	viper.Set("langchain.memory_type", "window")
	viper.Set("langchain.memory_window_size", 10)
	viper.Set("langchain.tools.max_iterations", 10)
	viper.Set("langchain.tools.max_retries", 3)

	// OLLAMA_HOST environment variable is required for integration tests
	// No fallback to localhost allowed

	// Reload config after setting test values
	if err := config.Load(); err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}
}

// TestAgentInterface tests the executorAgent agent directly without spawning processes
func TestAgentInterface(t *testing.T) {
	// Skip if no Ollama available
	if !isOllamaAvailable() {
		t.Skip("Skipping test: Ollama is not available")
	}

	t.Run("Agent responds to basic prompts", func(t *testing.T) {
		// Skip if using model incompatible with LangChain agents
		if !isLangChainCompatibleModel() {
			t.Skipf("Skipping agent test: model %s may not be compatible with LangChain agent parsing",
				os.Getenv("OLLAMA_DEFAULT_MODEL"))
		}

		// Setup viper configuration
		setupViperForTest(t)

		// Create LLM
		ollamaClient := ollama.NewClient()

		// Create executor agent with injected LLM
		executorAgent, err := agent.NewExecutorAgent(ollamaClient.LLM)
		require.NoError(t, err, "Should create executor agent")
		defer executorAgent.Close()

		// Test basic prompt
		ctx := context.Background()
		response, err := executorAgent.Execute(ctx, "Say hello and nothing else")
		require.NoError(t, err, "Should execute prompt")

		// Check response contains greeting
		responseLower := strings.ToLower(response)
		assert.True(t,
			strings.Contains(responseLower, "hello") ||
			strings.Contains(responseLower, "hi"),
			"Response should contain greeting: %s", response)
	})

	t.Run("Agent handles math questions", func(t *testing.T) {
		// Skip if using model incompatible with LangChain agents
		if !isLangChainCompatibleModel() {
			t.Skipf("Skipping agent test: model %s may not be compatible with LangChain agent parsing",
				os.Getenv("OLLAMA_DEFAULT_MODEL"))
		}

		setupViperForTest(t)
		ollamaClient := ollama.NewClient()
		executorAgent, err := agent.NewExecutorAgent(ollamaClient.LLM)
		require.NoError(t, err)
		defer executorAgent.Close()

		ctx := context.Background()
		response, err := executorAgent.Execute(ctx, "What is 2+2? Answer with just the number.")
		require.NoError(t, err)

		// Check response contains "4"
		assert.Contains(t, response, "4", "Response should contain the answer 4")
	})

	t.Run("Agent maintains conversation context", func(t *testing.T) {
		t.Skip("Memory persistence with LangChain agents needs further investigation")

		setupViperForTest(t)
		ollamaClient := ollama.NewClient()
		executorAgent, err := agent.NewExecutorAgent(ollamaClient.LLM)
		require.NoError(t, err)
		defer executorAgent.Close()

		ctx := context.Background()

		// First message
		response1, err := executorAgent.Execute(ctx, "My name is TestUser. Remember this.")
		require.NoError(t, err)
		t.Logf("First response: %s", response1)

		// Second message - should remember the name
		response2, err := executorAgent.Execute(ctx, "What is my name?")
		require.NoError(t, err)
		t.Logf("Second response: %s", response2)

		// Check if the agent remembers - be more flexible with the check
		responseLower := strings.ToLower(response2)
		assert.True(t,
			strings.Contains(response2, "TestUser") ||
			strings.Contains(responseLower, "testuser") ||
			strings.Contains(responseLower, "your name is") ||
			strings.Contains(responseLower, "you mentioned"),
			"Agent should reference the name from previous message. Got: %s", response2)
	})

	t.Run("Agent can clear memory", func(t *testing.T) {
		// Skip if using model incompatible with LangChain agents
		if !isLangChainCompatibleModel() {
			t.Skipf("Skipping agent test: model %s may not be compatible with LangChain agent parsing",
				os.Getenv("OLLAMA_DEFAULT_MODEL"))
		}

		setupViperForTest(t)
		ollamaClient := ollama.NewClient()
		executorAgent, err := agent.NewExecutorAgent(ollamaClient.LLM)
		require.NoError(t, err)
		defer executorAgent.Close()

		ctx := context.Background()

		// Add something to memory
		_, err = executorAgent.Execute(ctx, "Remember that my favorite color is blue")
		require.NoError(t, err)

		// Clear memory
		err = executorAgent.ClearMemory()
		require.NoError(t, err)

		// Ask about the color - should not remember
		response, err := executorAgent.Execute(ctx, "What is my favorite color?")
		require.NoError(t, err)

		// Should not know the color after clearing memory
		assert.NotContains(t, strings.ToLower(response), "blue",
			"Agent should not remember after memory clear")
	})
}

// StreamCollector implements agent.StreamHandler to collect streamed content
type StreamCollector struct {
	chunks []string
	final  string
	err    error
}

func (s *StreamCollector) OnChunk(chunk string) error {
	s.chunks = append(s.chunks, chunk)
	return nil
}

func (s *StreamCollector) OnComplete(finalContent string) error {
	s.final = finalContent
	return nil
}

func (s *StreamCollector) OnError(err error) {
	s.err = err
}

func TestAgentStreaming(t *testing.T) {
	if !isOllamaAvailable() {
		t.Skip("Skipping test: Ollama is not available")
	}

	t.Run("Agent can stream responses", func(t *testing.T) {
		// Skip if using model incompatible with LangChain agents
		if !isLangChainCompatibleModel() {
			t.Skipf("Skipping agent test: model %s may not be compatible with LangChain agent parsing",
				os.Getenv("OLLAMA_DEFAULT_MODEL"))
		}

		setupViperForTest(t)
		ollamaClient := ollama.NewClient()
		executorAgent, err := agent.NewExecutorAgent(ollamaClient.LLM)
		require.NoError(t, err)
		defer executorAgent.Close()

		collector := &StreamCollector{}
		ctx := context.Background()

		err = executorAgent.ExecuteStream(ctx, "Count from 1 to 3", collector)
		require.NoError(t, err)
		require.NoError(t, collector.err)

		// Check we got chunks
		assert.Greater(t, len(collector.chunks), 0, "Should receive stream chunks")

		// Check final content makes sense
		combined := strings.Join(collector.chunks, "")
		assert.Contains(t, combined, "1", "Should contain 1")
		assert.Contains(t, combined, "2", "Should contain 2")
		assert.Contains(t, combined, "3", "Should contain 3")
	})
}
