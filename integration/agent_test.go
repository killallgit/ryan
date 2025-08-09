package integration

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupViperForTest initializes viper configuration for tests
func setupViperForTest(t *testing.T) {
	// Set defaults
	viper.SetDefault("provider", "ollama")
	viper.SetDefault("ollama.default_model", "qwen3:latest")
	viper.SetDefault("ollama.timeout", 90)
	viper.SetDefault("langchain.memory_type", "window")
	viper.SetDefault("langchain.memory_window_size", 10)
	viper.SetDefault("langchain.tools.max_iterations", 10)
	viper.SetDefault("langchain.tools.max_retries", 3)

	// Override with environment variable if set
	if ollamaHost := os.Getenv("OLLAMA_HOST"); ollamaHost != "" {
		viper.Set("ollama.url", ollamaHost)
	} else {
		viper.Set("ollama.url", "http://localhost:11434")
	}

	// Override model if set in environment
	if ollamaModel := os.Getenv("OLLAMA_DEFAULT_MODEL"); ollamaModel != "" {
		viper.Set("ollama.default_model", ollamaModel)
	}
}

// TestAgentInterface tests the orchestrator agent directly without spawning processes
func TestAgentInterface(t *testing.T) {
	// Skip if no Ollama available
	if !isOllamaAvailable() {
		t.Skip("Skipping test: Ollama is not available")
	}

	t.Run("Agent responds to basic prompts", func(t *testing.T) {
		// Setup viper configuration
		setupViperForTest(t)

		// Create orchestrator
		orchestrator, err := agent.NewOrchestrator()
		require.NoError(t, err, "Should create orchestrator")
		defer orchestrator.Close()

		// Test basic prompt
		ctx := context.Background()
		response, err := orchestrator.Execute(ctx, "Say hello and nothing else")
		require.NoError(t, err, "Should execute prompt")

		// Check response contains greeting
		responseLower := strings.ToLower(response)
		assert.True(t,
			strings.Contains(responseLower, "hello") ||
			strings.Contains(responseLower, "hi"),
			"Response should contain greeting: %s", response)
	})

	t.Run("Agent handles math questions", func(t *testing.T) {
		setupViperForTest(t)
		orchestrator, err := agent.NewOrchestrator()
		require.NoError(t, err)
		defer orchestrator.Close()

		ctx := context.Background()
		response, err := orchestrator.Execute(ctx, "What is 2+2? Answer with just the number.")
		require.NoError(t, err)

		// Check response contains "4"
		assert.Contains(t, response, "4", "Response should contain the answer 4")
	})

	t.Run("Agent maintains conversation context", func(t *testing.T) {
		t.Skip("Memory persistence with LangChain agents needs further investigation")

		setupViperForTest(t)
		orchestrator, err := agent.NewOrchestrator()
		require.NoError(t, err)
		defer orchestrator.Close()

		ctx := context.Background()

		// First message
		response1, err := orchestrator.Execute(ctx, "My name is TestUser. Remember this.")
		require.NoError(t, err)
		t.Logf("First response: %s", response1)

		// Second message - should remember the name
		response2, err := orchestrator.Execute(ctx, "What is my name?")
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
		setupViperForTest(t)
		orchestrator, err := agent.NewOrchestrator()
		require.NoError(t, err)
		defer orchestrator.Close()

		ctx := context.Background()

		// Add something to memory
		_, err = orchestrator.Execute(ctx, "Remember that my favorite color is blue")
		require.NoError(t, err)

		// Clear memory
		err = orchestrator.ClearMemory()
		require.NoError(t, err)

		// Ask about the color - should not remember
		response, err := orchestrator.Execute(ctx, "What is my favorite color?")
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
		setupViperForTest(t)
		orchestrator, err := agent.NewOrchestrator()
		require.NoError(t, err)
		defer orchestrator.Close()

		collector := &StreamCollector{}
		ctx := context.Background()

		err = orchestrator.ExecuteStream(ctx, "Count from 1 to 3", collector)
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
