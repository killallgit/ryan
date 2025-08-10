// Package prompt provides a comprehensive prompt template system for Ryan,
// built on top of LangChain-Go's prompt functionality.
//
// This package offers:
//   - Template creation and management
//   - Variable substitution and validation
//   - Chat-based prompt templates
//   - Multiple template loaders (file, string, embedded)
//   - A global registry for template reuse
//   - Common pre-built templates
//
// Basic Usage:
//
//	// Create a simple template
//	template := prompt.NewPromptTemplate(
//	    "Hello {{.name}}, welcome to {{.place}}!",
//	    []string{"name", "place"},
//	)
//
//	// Format the template
//	result, err := template.Format(map[string]any{
//	    "name":  "Alice",
//	    "place": "Wonderland",
//	})
//
// Chat Templates:
//
//	// Create a chat template
//	messages := []prompt.MessageDefinition{
//	    {Role: "system", Template: "You are a {{.role}}", Variables: []string{"role"}},
//	    {Role: "human", Template: "{{.query}}", Variables: []string{"query"}},
//	}
//	chatTemplate, _ := prompt.NewChatTemplateFromMessages(messages)
//
// Loading Templates:
//
//	// From files
//	loader := prompt.NewFileLoader("./prompts")
//	template, _ := loader.Load("my-template.txt")
//
//	// From strings
//	stringLoader := prompt.NewStringLoader()
//	stringLoader.AddTemplate("greeting", "Hello {{.name}}!", []string{"name"})
//
// Registry Usage:
//
//	// Register a template globally
//	prompt.MustRegister("my-template", template)
//
//	// Retrieve it later
//	template = prompt.MustGet("my-template")
//
// The package integrates seamlessly with LangChain-Go and provides
// a decoupled, reusable prompt management system for AI agents.
package prompt
