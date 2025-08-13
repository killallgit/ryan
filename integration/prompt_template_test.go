package integration

import (
	"context"
	"os"
	"testing"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/killallgit/ryan/pkg/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/fake"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/prompts"
)

// TestPromptTemplatesWithLangChain verifies our templates integrate with LangChain-Go
func TestPromptTemplatesWithLangChain(t *testing.T) {
	t.Run("Templates implement FormatPrompter interface", func(t *testing.T) {
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
	})

	t.Run("Chat templates work with LangChain memory", func(t *testing.T) {
		// Create a fake LLM for testing
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

		// Execute interaction
		result, err := chain.Call(context.Background(), map[string]any{
			"expertise": "Go programming",
			"style":     "helpful and concise",
			"input":     "Tell me about goroutines",
		})
		require.NoError(t, err)
		assert.NotNil(t, result["text"])

		// Memory is maintained
		assert.NotNil(t, mem)
	})

	t.Run("RAG template works with chains", func(t *testing.T) {
		// Create a fake LLM
		fakeLLM := fake.NewFakeLLM([]string{
			"Based on the context, Paris is the capital of France.",
		})

		// Load a RAG template from built-in templates
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
	})

	t.Run("Generic template provides flexibility", func(t *testing.T) {
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

// TestPromptTemplatesWithAgent tests prompt templates integrated with our ReactAgent
func TestPromptTemplatesWithAgent(t *testing.T) {
	// Skip if no Ollama available
	if !isOllamaAvailable() {
		t.Skip("Skipping test: Ollama is not available")
	}

	t.Run("Agent uses custom prompt template", func(t *testing.T) {
		// Skip if using model incompatible with LangChain agents
		if !isLangChainCompatibleModel() {
			t.Skipf("Skipping agent test: model %s may not be compatible with LangChain agent parsing",
				os.Getenv("OLLAMA_DEFAULT_MODEL"))
		}

		// Setup viper configuration
		setupViperForTest(t)

		// Create Ollama client
		ollamaClient := ollama.NewClient()
		require.NotNil(t, ollamaClient)

		// Create agent
		testAgent, err := agent.NewReactAgent(ollamaClient)
		require.NoError(t, err)
		require.NotNil(t, testAgent)
		defer testAgent.Close()

		// Create and set a custom prompt template
		customTemplate := prompt.NewPromptTemplate(
			`You are a helpful assistant specialized in {{.domain}}.
Instructions: {{.instructions}}

Task: {{.input}}

Response:`,
			[]string{"input"},
		).WithPartialVariables(map[string]any{
			"domain":       "Go programming",
			"instructions": "Keep responses concise and practical",
		})

		testAgent.SetPromptTemplate(customTemplate)

		// Test that the template is being used
		ctx := context.Background()
		response, err := testAgent.Execute(ctx, "What is a goroutine in one sentence?")
		require.NoError(t, err)
		assert.NotEmpty(t, response)

		// Response should be concise (checking it's following the template instructions)
		assert.Less(t, len(response), 500, "Response should be concise as per template instructions")
	})

	t.Run("Agent formats prompts with template", func(t *testing.T) {
		// Setup viper configuration
		setupViperForTest(t)

		// Create Ollama client
		ollamaClient := ollama.NewClient()
		require.NotNil(t, ollamaClient)

		// Create agent
		testAgent, err := agent.NewReactAgent(ollamaClient)
		require.NoError(t, err)
		defer testAgent.Close()

		// Set the analysis template
		analysisTemplate := prompt.GetAnalysisTemplate()
		testAgent.SetPromptTemplate(analysisTemplate)

		// Format a prompt (without executing)
		formatted, err := testAgent.FormatPrompt("review this", map[string]any{
			"type":    "code",
			"content": "func main() {}",
			"focus":   "structure",
		})
		require.NoError(t, err)
		assert.Contains(t, formatted, "Analyze the following code")
		assert.Contains(t, formatted, "func main()")
		assert.Contains(t, formatted, "Focus on: structure")
	})
}

// TestLoadTemplatesFromFiles tests loading templates from pkg/templates directory
func TestLoadTemplatesFromFiles(t *testing.T) {
	t.Run("Load YAML templates", func(t *testing.T) {
		loader := prompt.NewFileLoader("../pkg/templates")

		// Try to load the generic template
		template, err := loader.Load("generic.yaml")
		require.NoError(t, err)
		assert.NotNil(t, template)

		// Format with the loaded template (provide all variables)
		result, err := template.Format(map[string]any{
			"task":          "Test task",
			"system_prompt": "",
			"context":       "",
			"instructions":  "",
			"examples":      "",
			"constraints":   "",
			"input":         "",
			"format":        "",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Task: Test task")
	})

	t.Run("Load JSON chat templates", func(t *testing.T) {
		loader := prompt.NewFileLoader("../pkg/templates")

		// Try to load the generic chat template
		template, err := loader.LoadChat("generic_chat.json")
		require.NoError(t, err)
		assert.NotNil(t, template)

		// Format messages with the loaded template
		messages, err := template.FormatMessages(map[string]any{
			"role":         "assistant",
			"capabilities": "",
			"instructions": "",
			"constraints":  "",
			"message":      "Hello",
			"context":      "",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, messages)
	})

	t.Run("Load agent template", func(t *testing.T) {
		loader := prompt.NewFileLoader("../pkg/templates")

		// Load the agent template
		template, err := loader.Load("agent.yaml")
		require.NoError(t, err)
		assert.NotNil(t, template)

		// Format with tool information
		result, err := template.Format(map[string]any{
			"task":        "Calculate 5+5",
			"tools":       "calculator: Performs math calculations",
			"context":     "",
			"history":     "",
			"constraints": "",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "You have access to the following tools")
		assert.Contains(t, result, "calculator")
		assert.Contains(t, result, "Calculate 5+5")
	})
}

// TestBuiltInTemplates verifies all built-in templates work correctly
func TestBuiltInTemplates(t *testing.T) {
	templates := []struct {
		name     string
		getter   func() prompt.Template
		vars     map[string]any
		expected []string
	}{
		{
			name:   "Generic",
			getter: prompt.GetGenericTemplate,
			vars: map[string]any{
				"task":         "Write a test",
				"instructions": "Use Go",
			},
			expected: []string{"Task: Write a test", "Instructions: Use Go"},
		},
		{
			name:   "QA",
			getter: prompt.GetSimpleQATemplate,
			vars: map[string]any{
				"question": "What is TDD?",
			},
			expected: []string{"What is TDD?"},
		},
		{
			name:   "Summarization",
			getter: prompt.GetSummarizationTemplate,
			vars: map[string]any{
				"text":  "Long text here...",
				"style": "concise",
			},
			expected: []string{"Long text here", "concise"},
		},
		{
			name:   "Code Generation",
			getter: prompt.GetCodeGenTemplate,
			vars: map[string]any{
				"language":     "Go",
				"description":  "sorts a slice",
				"requirements": "Use generics",
			},
			expected: []string{"Go", "sorts a slice", "Use generics"},
		},
		{
			name:   "Analysis",
			getter: prompt.GetAnalysisTemplate,
			vars: map[string]any{
				"type":    "function",
				"content": "func foo() {}",
				"focus":   "naming",
			},
			expected: []string{"function", "func foo()", "naming"},
		},
		{
			name:   "Chain of Thought",
			getter: prompt.GetChainOfThoughtTemplate,
			vars: map[string]any{
				"problem": "Find the median",
			},
			expected: []string{"Find the median", "step-by-step"},
		},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			template := tt.getter()
			result, err := template.Format(tt.vars)
			require.NoError(t, err)
			for _, exp := range tt.expected {
				assert.Contains(t, result, exp)
			}
		})
	}
}

// TestChatTemplates verifies chat templates work correctly
func TestChatTemplates(t *testing.T) {
	t.Run("Simple Assistant", func(t *testing.T) {
		template := prompt.GetSimpleAssistantTemplate()
		messages, err := template.FormatMessages(map[string]any{
			"instructions": "Be helpful",
			"query":        "How are you?",
		})
		require.NoError(t, err)
		assert.Len(t, messages, 2)

		systemMsg := messages[0].GetContent()
		assert.Contains(t, systemMsg, "helpful assistant")
		assert.Contains(t, systemMsg, "Be helpful")

		humanMsg := messages[1].GetContent()
		assert.Equal(t, "How are you?", humanMsg)
	})

	t.Run("Expert", func(t *testing.T) {
		template := prompt.GetExpertTemplate()
		messages, err := template.FormatMessages(map[string]any{
			"domain": "mathematics",
			"style":  "academic",
			"query":  "Explain calculus",
		})
		require.NoError(t, err)
		assert.Len(t, messages, 2)

		systemMsg := messages[0].GetContent()
		assert.Contains(t, systemMsg, "expert in mathematics")
		assert.Contains(t, systemMsg, "academic")
	})

	t.Run("RAG", func(t *testing.T) {
		template := prompt.GetSimpleRAGTemplate()
		messages, err := template.FormatMessages(map[string]any{
			"context":  "The sky is blue due to Rayleigh scattering.",
			"question": "Why is the sky blue?",
		})
		require.NoError(t, err)
		assert.Len(t, messages, 2)

		humanMsg := messages[1].GetContent()
		assert.Contains(t, humanMsg, "Context:")
		assert.Contains(t, humanMsg, "Rayleigh scattering")
		assert.Contains(t, humanMsg, "Why is the sky blue?")
	})
}
