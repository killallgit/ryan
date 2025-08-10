package prompt

import "github.com/tmc/langchaingo/prompts"

// Common prompt templates that can be registered and reused.
// These templates serve as both practical tools and self-documenting examples
// of how to use the prompt template system with LangChain-Go.

func init() {
	// Register built-in templates
	registerBuiltInTemplates()
}

func registerBuiltInTemplates() {
	// Generic template - a flexible, all-purpose template
	// This demonstrates conditional sections and multiple optional variables
	MustRegister("generic", NewPromptTemplate(
		`{{if .context}}Context: {{.context}}

{{end}}{{if .instructions}}Instructions: {{.instructions}}

{{end}}Task: {{.task}}{{if .format}}

Expected Format: {{.format}}{{end}}

Response:`,
		[]string{"task"},
	).WithPartialVariables(map[string]any{
		"format":       "Provide a clear, concise response",
		"context":      "",
		"instructions": "",
	}))

	// Simple Q&A template
	MustRegister("simple_qa", NewPromptTemplate(
		"Answer the following question: {{.question}}",
		[]string{"question"},
	))

	// Summarization template with style control
	MustRegister("summarize", NewPromptTemplate(
		`Summarize the following text in {{.style}} style:

{{.text}}

Summary:`,
		[]string{"text", "style"},
	))

	// Code generation template with requirements
	MustRegister("code_gen", NewPromptTemplate(
		`Generate {{.language}} code that {{.description}}.

Requirements:
{{.requirements}}

Code:`,
		[]string{"language", "description", "requirements"},
	))

	// Analysis template for code or text analysis
	MustRegister("analyze", NewPromptTemplate(
		`Analyze the following {{.type}}:

{{.content}}

Focus on: {{.focus}}

Analysis:`,
		[]string{"type", "content", "focus"},
	))

	// Chain-of-thought reasoning template
	MustRegister("chain_of_thought", NewPromptTemplate(
		`Problem: {{.problem}}

Let's approach this step-by-step:
1. First, identify what we know
2. Then, determine what we need to find
3. Next, work through the solution
4. Finally, verify our answer

Solution:`,
		[]string{"problem"},
	))

	// Simple chat template for assistant
	chatMessages := []prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate(
			"You are a helpful assistant. {{.instructions}}",
			[]string{"instructions"},
		),
		prompts.NewHumanMessagePromptTemplate(
			"{{.query}}",
			[]string{"query"},
		),
	}

	MustRegister("simple_assistant", NewChatTemplate(chatMessages))

	// Expert chat template with domain specialization
	// We need to provide all variables that the template references
	expertMessages := []prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate(
			`You are an expert in {{.domain}}.{{if .style}} Your communication style is {{.style}}.{{end}}{{if .constraints}} Constraints: {{.constraints}}{{end}}`,
			[]string{"domain", "style", "constraints"},
		),
		prompts.NewHumanMessagePromptTemplate(
			"{{.query}}",
			[]string{"query"},
		),
	}

	MustRegister("expert", NewChatTemplate(expertMessages).WithPartialVariables(map[string]any{
		"style":       "",
		"constraints": "",
	}))

	// Simple RAG (Retrieval-Augmented Generation) template
	ragMessages := []prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate(
			`You are a knowledgeable assistant. Use the provided context to answer questions accurately.
If the context doesn't contain relevant information, say so.`,
			[]string{},
		),
		prompts.NewHumanMessagePromptTemplate(
			`Context:
{{.context}}

Question: {{.question}}`,
			[]string{"context", "question"},
		),
	}

	MustRegister("simple_rag", NewChatTemplate(ragMessages))
}

// GetGenericTemplate returns a flexible, general-purpose template
// that can be customized for various tasks
func GetGenericTemplate() Template {
	return MustGet("generic")
}

// GetSimpleQATemplate returns a simple Q&A prompt template
func GetSimpleQATemplate() Template {
	return MustGet("simple_qa")
}

// GetSummarizationTemplate returns a summarization template
func GetSummarizationTemplate() Template {
	return MustGet("summarize")
}

// GetCodeGenTemplate returns a code generation template
func GetCodeGenTemplate() Template {
	return MustGet("code_gen")
}

// GetAnalysisTemplate returns a template for analyzing code or text
func GetAnalysisTemplate() Template {
	return MustGet("analyze")
}

// GetChainOfThoughtTemplate returns a template for step-by-step reasoning
func GetChainOfThoughtTemplate() Template {
	return MustGet("chain_of_thought")
}

// GetSimpleAssistantTemplate returns a simple chat assistant template
func GetSimpleAssistantTemplate() ChatTemplate {
	return MustGet("simple_assistant").(ChatTemplate)
}

// GetExpertTemplate returns a chat template for domain experts
func GetExpertTemplate() ChatTemplate {
	return MustGet("expert").(ChatTemplate)
}

// GetSimpleRAGTemplate returns a simple RAG chat template
func GetSimpleRAGTemplate() ChatTemplate {
	return MustGet("simple_rag").(ChatTemplate)
}
