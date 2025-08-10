package prompt_test

import (
	"context"
	"testing"

	"github.com/killallgit/ryan/pkg/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/fake"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/prompts"
)

// TestPromptTemplateImplementsFormatPrompter verifies our templates implement the FormatPrompter interface
func TestPromptTemplateImplementsFormatPrompter(t *testing.T) {
	// Create our custom prompt template
	template := prompt.NewPromptTemplate(
		"Answer this question: {{.question}}",
		[]string{"question"},
	)

	// Verify it implements FormatPrompter interface
	var _ prompts.FormatPrompter = template

	// Test the methods
	vars := template.GetInputVariables()
	assert.Equal(t, []string{"question"}, vars)

	promptValue, err := template.FormatPrompt(map[string]any{
		"question": "What is Go?",
	})
	require.NoError(t, err)
	assert.NotNil(t, promptValue)
}

// TestChatTemplateImplementsFormatPrompter verifies chat templates implement the interface
func TestChatTemplateImplementsFormatPrompter(t *testing.T) {
	// Create our custom chat template
	messages := []prompt.MessageDefinition{
		{Role: "system", Template: "You are a {{.role}}", Variables: []string{"role"}},
		{Role: "human", Template: "{{.question}}", Variables: []string{"question"}},
	}

	template, err := prompt.NewChatTemplateFromMessages(messages)
	require.NoError(t, err)

	// Verify it implements FormatPrompter interface
	var _ prompts.FormatPrompter = template

	// Test the methods
	vars := template.GetInputVariables()
	assert.ElementsMatch(t, []string{"role", "question"}, vars)

	promptValue, err := template.FormatPrompt(map[string]any{
		"role":     "helpful assistant",
		"question": "What is Go?",
	})
	require.NoError(t, err)
	assert.NotNil(t, promptValue)
}

// TestPromptTemplateWithLLMChain tests using our prompt template with LangChain's LLMChain
func TestPromptTemplateWithLLMChain(t *testing.T) {
	// Create a fake LLM for testing
	fakeLLM := fake.NewFakeLLM([]string{
		"Go is a programming language developed by Google.",
	})

	// Create our custom prompt template
	template := prompt.NewPromptTemplate(
		"Answer this question concisely: {{.question}}",
		[]string{"question"},
	)

	// Create an LLMChain with our template
	chain := chains.NewLLMChain(fakeLLM, template)

	// Test the chain
	result, err := chain.Call(context.Background(), map[string]any{
		"question": "What is Go?",
	})
	require.NoError(t, err)

	output, ok := result["text"].(string)
	require.True(t, ok)
	assert.Equal(t, "Go is a programming language developed by Google.", output)
}

// TestChatTemplateWithLLMChain tests using our chat template with LangChain's LLMChain
func TestChatTemplateWithLLMChain(t *testing.T) {
	// Create a fake LLM for testing
	fakeLLM := fake.NewFakeLLM([]string{
		"I am a helpful assistant. Go is a programming language.",
	})

	// Create our custom chat template
	messages := []prompt.MessageDefinition{
		{Role: "system", Template: "You are a {{.role}} assistant", Variables: []string{"role"}},
		{Role: "human", Template: "{{.question}}", Variables: []string{"question"}},
	}

	template, err := prompt.NewChatTemplateFromMessages(messages)
	require.NoError(t, err)

	// Create an LLMChain with our chat template
	chain := chains.NewLLMChain(fakeLLM, template)

	// Test the chain
	result, err := chain.Call(context.Background(), map[string]any{
		"role":     "helpful",
		"question": "What is Go?",
	})
	require.NoError(t, err)

	output, ok := result["text"].(string)
	require.True(t, ok)
	assert.Contains(t, output, "Go is a programming language")
}

// TestPromptTemplateWithAgent tests using our templates with LangChain agents
func TestPromptTemplateWithAgent(t *testing.T) {
	// Create a fake LLM
	fakeLLM := fake.NewFakeLLM([]string{
		"Final Answer: Go is a programming language.",
	})

	// Create our custom prompt template
	template := prompt.NewPromptTemplate(
		"You are an agent. Answer: {{.input}}",
		[]string{"input"},
	)

	// Create a simple executor with our template
	// Note: This is a simplified test - real agent usage would be more complex
	chain := chains.NewLLMChain(fakeLLM, template)

	result, err := chain.Call(context.Background(), map[string]any{
		"input": "What is Go?",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestDirectLangChainPromptUsage shows we can also use LangChain prompts directly
func TestDirectLangChainPromptUsage(t *testing.T) {
	// We can still use LangChain's native prompts
	langchainTemplate := prompts.NewPromptTemplate(
		"Answer: {{.question}}",
		[]string{"question"},
	)

	// And wrap them if needed
	var _ prompts.FormatPrompter = langchainTemplate

	// Our templates are fully compatible
	ourTemplate := prompt.NewPromptTemplate(
		"Answer: {{.question}}",
		[]string{"question"},
	)

	var _ prompts.FormatPrompter = ourTemplate

	// Both work the same way
	lcResult, err1 := langchainTemplate.FormatPrompt(map[string]any{"question": "test"})
	ourResult, err2 := ourTemplate.FormatPrompt(map[string]any{"question": "test"})

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotNil(t, lcResult)
	assert.NotNil(t, ourResult)
}

// TestConversationalAgentWithPromptTemplate tests a more complex agent setup
func TestConversationalAgentWithPromptTemplate(t *testing.T) {
	// Create a fake LLM
	fakeLLM := fake.NewFakeLLM([]string{
		"I understand. The answer is 42.",
	})

	// Create our conversational prompt template
	messages := []prompt.MessageDefinition{
		{
			Role:      "system",
			Template:  "You are a helpful AI assistant with knowledge about {{.topic}}.",
			Variables: []string{"topic"},
		},
		{
			Role:      "human",
			Template:  "{{.input}}",
			Variables: []string{"input"},
		},
	}

	template, err := prompt.NewChatTemplateFromMessages(messages)
	require.NoError(t, err)

	// Create chain with our template
	chain := chains.NewLLMChain(fakeLLM, template)

	// Execute with both template variables
	result, err := chain.Call(context.Background(), map[string]any{
		"topic": "mathematics",
		"input": "What is 6 times 7?",
	})
	require.NoError(t, err)

	output, ok := result["text"].(string)
	require.True(t, ok)
	assert.Contains(t, output, "42")
}

// TestChatTemplateWithMemory demonstrates chat templates with conversation memory
func TestChatTemplateWithMemory(t *testing.T) {
	// Create a fake LLM
	fakeLLM := fake.NewFakeLLM([]string{
		"I understand your request about Go programming.",
	})

	// Create a conversational chat template
	messages := []prompt.MessageDefinition{
		{
			Role:      "system",
			Template:  "You are an expert in {{.expertise}}. Always be {{.style}}.",
			Variables: []string{"expertise", "style"},
		},
		{
			Role:      "human",
			Template:  "{{.input}}",
			Variables: []string{"input"},
		},
	}

	chatTemplate, err := prompt.NewChatTemplateFromMessages(messages)
	require.NoError(t, err)

	// Create a chain with memory for conversational context
	mem := memory.NewConversationBuffer()
	chain := chains.NewLLMChain(fakeLLM, chatTemplate)
	chain.Memory = mem

	// First interaction
	result1, err := chain.Call(context.Background(), map[string]any{
		"expertise": "Go programming",
		"style":     "helpful and concise",
		"input":     "Tell me about goroutines",
	})
	require.NoError(t, err)
	assert.NotNil(t, result1["text"])

	// The memory is being maintained internally
	assert.NotNil(t, mem)
}

// TestRAGTemplate tests RAG (Retrieval-Augmented Generation) templates
func TestRAGTemplate(t *testing.T) {
	// Create a fake LLM
	fakeLLM := fake.NewFakeLLM([]string{
		"Based on the context, Paris is the capital of France.",
	})

	// Load a RAG template from our built-in templates
	ragTemplate := prompt.GetSimpleRAGTemplate()

	// Verify it implements FormatPrompter
	var _ prompts.FormatPrompter = ragTemplate

	// Create a chain with the RAG template
	chain := chains.NewLLMChain(fakeLLM, ragTemplate)

	// Execute with context and question
	result, err := chain.Call(context.Background(), map[string]any{
		"context":  "Paris is the capital and largest city of France.",
		"question": "What is the capital of France?",
	})
	require.NoError(t, err)

	output, ok := result["text"].(string)
	require.True(t, ok)
	assert.Contains(t, output, "Paris")
	assert.Contains(t, output, "capital")
}

// TestTemplateRegistryWithChains tests loading templates from registry for use in chains
func TestTemplateRegistryWithChains(t *testing.T) {
	// Register a template
	prompt.DefaultRegistry.Clear()
	template := prompt.NewPromptTemplate(
		"Solve this problem: {{.problem}}",
		[]string{"problem"},
	)
	err := prompt.DefaultRegistry.Register("math_solver", template)
	require.NoError(t, err)

	// Load it from registry
	loaded, err := prompt.DefaultRegistry.Get("math_solver")
	require.NoError(t, err)

	// Use it as a FormatPrompter in a chain
	fakeLLM := fake.NewFakeLLM([]string{
		"The solution is 10.",
	})

	chain := chains.NewLLMChain(fakeLLM, loaded.(prompts.FormatPrompter))

	result, err := chain.Call(context.Background(), map[string]any{
		"problem": "5 + 5",
	})
	require.NoError(t, err)
	assert.NotNil(t, result["text"])
}
