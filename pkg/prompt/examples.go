package prompt

import "github.com/tmc/langchaingo/prompts"

// Common prompt templates that can be registered and reused

func init() {
	// Register built-in templates
	registerBuiltInTemplates()
}

func registerBuiltInTemplates() {
	// Simple Q&A template
	MustRegister("simple_qa", NewPromptTemplate(
		"Answer the following question: {{.question}}",
		[]string{"question"},
	))

	// Summarization template
	MustRegister("summarize", NewPromptTemplate(
		`Summarize the following text in {{.style}} style:

{{.text}}

Summary:`,
		[]string{"text", "style"},
	))

	// Code generation template
	MustRegister("code_gen", NewPromptTemplate(
		`Generate {{.language}} code that {{.description}}.

Requirements:
{{.requirements}}

Code:`,
		[]string{"language", "description", "requirements"},
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

	// Simple RAG template
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

// GetSimpleAssistantTemplate returns a simple chat assistant template
func GetSimpleAssistantTemplate() ChatTemplate {
	return MustGet("simple_assistant").(ChatTemplate)
}

// GetSimpleRAGTemplate returns a simple RAG chat template
func GetSimpleRAGTemplate() ChatTemplate {
	return MustGet("simple_rag").(ChatTemplate)
}
