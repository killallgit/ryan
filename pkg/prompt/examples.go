package prompt

import "github.com/tmc/langchaingo/prompts"

// Common prompt templates that can be registered and reused

func init() {
	// Register some common templates
	registerCommonTemplates()
}

func registerCommonTemplates() {
	// Simple Q&A template
	MustRegister("qa", NewPromptTemplate(
		"Answer the following question: {{.question}}",
		[]string{"question"},
	))

	// Context-based Q&A template
	MustRegister("qa_with_context", NewPromptTemplate(
		`Use the following context to answer the question:

Context: {{.context}}

Question: {{.question}}

Answer:`,
		[]string{"context", "question"},
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

	// Chat template for assistant
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

	MustRegister("assistant", NewChatTemplate(chatMessages))

	// RAG template
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

	MustRegister("rag", NewChatTemplate(ragMessages))
}

// GetQATemplate returns a Q&A prompt template
func GetQATemplate() Template {
	return MustGet("qa")
}

// GetQAWithContextTemplate returns a context-based Q&A template
func GetQAWithContextTemplate() Template {
	return MustGet("qa_with_context")
}

// GetSummarizationTemplate returns a summarization template
func GetSummarizationTemplate() Template {
	return MustGet("summarize")
}

// GetCodeGenTemplate returns a code generation template
func GetCodeGenTemplate() Template {
	return MustGet("code_gen")
}

// GetAssistantTemplate returns a chat assistant template
func GetAssistantTemplate() ChatTemplate {
	return MustGet("assistant").(ChatTemplate)
}

// GetRAGTemplate returns a RAG chat template
func GetRAGTemplate() ChatTemplate {
	return MustGet("rag").(ChatTemplate)
}
