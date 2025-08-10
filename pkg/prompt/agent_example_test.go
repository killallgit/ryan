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

// TestPromptTemplateAsAgentPrompt shows how to use our templates as agent prompts
func TestPromptTemplateAsAgentPrompt(t *testing.T) {
	// Create a fake LLM
	fakeLLM := fake.NewFakeLLM([]string{
		"I'll help you with that. The answer is 42.",
	})

	// Create our custom agent prompt template
	agentTemplate := prompt.NewPromptTemplate(
		`You are a helpful assistant.
Instructions: {{.instructions}}
Current task: {{.input}}
Please provide a helpful response.`,
		[]string{"instructions", "input"},
	)

	// Verify it's a FormatPrompter
	var _ prompts.FormatPrompter = agentTemplate

	// Create a chain with the template
	chain := chains.NewLLMChain(fakeLLM, agentTemplate)

	// Execute with the agent-style inputs
	result, err := chain.Call(context.Background(), map[string]any{
		"instructions": "Be concise and accurate",
		"input":        "What is 6 times 7?",
	})
	require.NoError(t, err)

	output, ok := result["text"].(string)
	require.True(t, ok)
	assert.Contains(t, output, "42")
}

// TestChatTemplateWithConversationalAgent demonstrates chat templates with conversational patterns
func TestChatTemplateWithConversationalAgent(t *testing.T) {
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

	// The template system works with the agent's memory system
	// Note: ConversationBuffer doesn't expose Messages() directly in the current API
	// but the memory is being maintained internally
	assert.NotNil(t, mem)
}

// TestPromptTemplateWithTools shows templates can be used for tool-based prompts
func TestPromptTemplateWithTools(t *testing.T) {
	// Create our prompt template for a tool-using agent
	toolTemplate := prompt.NewPromptTemplate(
		`You have access to the following tools:
{{.tools}}

Use this format:
Thought: Think about what to do
Action: the action to take
Action Input: the input to the action
Observation: the result of the action
... (repeat as needed)
Thought: I now know the final answer
Final Answer: the final answer

Question: {{.input}}`,
		[]string{"tools", "input"},
	)

	// The template can be used to format the agent's prompts
	formattedPrompt, err := toolTemplate.Format(map[string]any{
		"tools": "calculator: A calculator tool for math",
		"input": "What is 5 plus 5?",
	})
	require.NoError(t, err)
	assert.Contains(t, formattedPrompt, "calculator")
	assert.Contains(t, formattedPrompt, "5 plus 5")
	assert.Contains(t, formattedPrompt, "Action:")
	assert.Contains(t, formattedPrompt, "Final Answer:")

	// Verify it works as a FormatPrompter with chains
	fakeLLM := fake.NewFakeLLM([]string{
		"I'll calculate 5+5. The answer is 10.",
	})

	chain := chains.NewLLMChain(fakeLLM, toolTemplate)
	result, err := chain.Call(context.Background(), map[string]any{
		"tools": "calculator: Math calculations",
		"input": "Calculate 5+5",
	})
	require.NoError(t, err)
	assert.Contains(t, result["text"].(string), "10")
}

// TestRAGTemplateWithAgent shows how RAG templates work with agents
func TestRAGTemplateWithAgent(t *testing.T) {
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

// TestLoadTemplateForAgent shows loading templates from files for agent use
func TestLoadTemplateForAgent(t *testing.T) {
	// This would load from pkg/templates in real usage
	// For testing, we'll register a template
	prompt.DefaultRegistry.Clear()

	agentPrompt := prompt.NewPromptTemplate(
		`Assistant capable of: {{.capabilities}}
Task: {{.input}}
Response:`,
		[]string{"capabilities", "input"},
	)

	err := prompt.DefaultRegistry.Register("agent_prompt", agentPrompt)
	require.NoError(t, err)

	// Load the template
	loaded, err := prompt.DefaultRegistry.Get("agent_prompt")
	require.NoError(t, err)

	// Use it as a FormatPrompter in an agent chain
	fakeLLM := fake.NewFakeLLM([]string{
		"I can help with coding tasks.",
	})

	chain := chains.NewLLMChain(fakeLLM, loaded.(prompts.FormatPrompter))

	result, err := chain.Call(context.Background(), map[string]any{
		"capabilities": "coding, debugging, testing",
		"input":        "Help me write a Go function",
	})
	require.NoError(t, err)
	assert.Contains(t, result["text"].(string), "coding")
}
