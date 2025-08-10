package prompt_test

import (
	"context"
	"testing"

	"github.com/killallgit/ryan/pkg/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/fake"
)

// TestGenericTemplate tests the generic template functionality
func TestGenericTemplate(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		template := prompt.GetGenericTemplate()

		// Test with minimal required variables
		result, err := template.Format(map[string]any{
			"task": "Write a hello world program",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Task: Write a hello world program")
		assert.Contains(t, result, "Expected Format: Provide a clear, concise response")
	})

	t.Run("with all optional fields", func(t *testing.T) {
		template := prompt.GetGenericTemplate()

		result, err := template.Format(map[string]any{
			"task":         "Analyze this code",
			"context":      "Working on a Go project",
			"instructions": "Focus on performance",
			"format":       "Bullet points",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Context: Working on a Go project")
		assert.Contains(t, result, "Instructions: Focus on performance")
		assert.Contains(t, result, "Task: Analyze this code")
		assert.Contains(t, result, "Expected Format: Bullet points")
	})

	t.Run("with LangChain integration", func(t *testing.T) {
		fakeLLM := fake.NewFakeLLM([]string{
			"Hello, World! program in Go: fmt.Println(\"Hello, World!\")",
		})

		template := prompt.GetGenericTemplate()
		chain := chains.NewLLMChain(fakeLLM, template)

		result, err := chain.Call(context.Background(), map[string]any{
			"task":         "Write Hello World in Go",
			"instructions": "Keep it simple",
			"format":       "Single line of code",
		})
		require.NoError(t, err)
		assert.Contains(t, result["text"].(string), "Hello")
	})
}

// TestChainOfThoughtTemplate tests the chain-of-thought reasoning template
func TestChainOfThoughtTemplate(t *testing.T) {
	template := prompt.GetChainOfThoughtTemplate()

	result, err := template.Format(map[string]any{
		"problem": "Calculate the sum of first 10 natural numbers",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Problem: Calculate the sum of first 10 natural numbers")
	assert.Contains(t, result, "step-by-step")
	assert.Contains(t, result, "identify what we know")
	assert.Contains(t, result, "verify our answer")
}

// TestAnalysisTemplate tests the analysis template
func TestAnalysisTemplate(t *testing.T) {
	template := prompt.GetAnalysisTemplate()

	result, err := template.Format(map[string]any{
		"type":    "code",
		"content": "func add(a, b int) int { return a + b }",
		"focus":   "performance and readability",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Analyze the following code")
	assert.Contains(t, result, "func add")
	assert.Contains(t, result, "Focus on: performance and readability")
}

// TestExpertTemplate tests the expert chat template
func TestExpertTemplate(t *testing.T) {
	template := prompt.GetExpertTemplate()

	messages, err := template.FormatMessages(map[string]any{
		"domain": "Go programming",
		"style":  "concise and practical",
		"query":  "How do I handle errors in Go?",
	})
	require.NoError(t, err)
	require.Len(t, messages, 2)

	// Check system message
	systemMsg := messages[0].GetContent()
	assert.Contains(t, systemMsg, "expert in Go programming")
	assert.Contains(t, systemMsg, "concise and practical")

	// Check human message
	humanMsg := messages[1].GetContent()
	assert.Equal(t, "How do I handle errors in Go?", humanMsg)
}
