package prompt

import (
	"path/filepath"
	"runtime"

	"github.com/tmc/langchaingo/prompts"
)

const (
	// TemplatesDir is the fixed directory for prompt templates relative to this package
	templatesDir = "templates"
)

// getTemplatesPath returns the absolute path to the templates directory
func getTemplatesPath() string {
	// Get the directory of this source file
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "..", templatesDir)
}

// LoadDefaultTemplate loads a template from the pkg/templates directory
func LoadDefaultTemplate(name string) (Template, error) {
	// Try to get from registry first
	if template, err := DefaultRegistry.Get(name); err == nil {
		return template, nil
	}

	// Load from pkg/templates directory
	loader := NewFileLoader(getTemplatesPath())
	return loader.Load(name)
}

// LoadDefaultChatTemplate loads a chat template from the pkg/templates directory
func LoadDefaultChatTemplate(name string) (ChatTemplate, error) {
	// Try to get from registry first
	if template, err := DefaultRegistry.Get(name); err == nil {
		if chatTemplate, ok := template.(ChatTemplate); ok {
			return chatTemplate, nil
		}
	}

	// Load from pkg/templates directory
	loader := NewFileLoader(getTemplatesPath())
	return loader.LoadChat(name)
}

// LoadAllTemplates loads all templates from the pkg/templates directory into the registry
func LoadAllTemplates() error {
	templatesPath := getTemplatesPath()
	loader := NewFileLoader(templatesPath)

	// Load all YAML templates
	pattern := filepath.Join(templatesPath, "*.yaml")
	matches, _ := filepath.Glob(pattern)
	for _, file := range matches {
		name := filepath.Base(file)
		name = name[:len(name)-5] // Remove .yaml extension

		if template, err := loader.Load(filepath.Base(file)); err == nil {
			// Use the name without extension
			DefaultRegistry.Register(name, template)
		}
	}

	// Load all JSON templates
	pattern = filepath.Join(templatesPath, "*.json")
	matches, _ = filepath.Glob(pattern)
	for _, file := range matches {
		name := filepath.Base(file)
		name = name[:len(name)-5] // Remove .json extension

		// Try to load as chat template first
		if chatTemplate, err := loader.LoadChat(filepath.Base(file)); err == nil {
			DefaultRegistry.Register(name, chatTemplate)
		} else if template, err := loader.Load(filepath.Base(file)); err == nil {
			DefaultRegistry.Register(name, template)
		}
	}

	return nil
}

// CreateSystemPrompt creates a system prompt for the agent
func CreateSystemPrompt(role string, instructions string) prompts.MessageFormatter {
	template := "You are " + role + ". " + instructions
	return prompts.NewSystemMessagePromptTemplate(template, []string{})
}

// CreateUserPrompt creates a user prompt
func CreateUserPrompt(template string, variables []string) prompts.MessageFormatter {
	return prompts.NewHumanMessagePromptTemplate(template, variables)
}

// CreateAssistantPrompt creates an assistant prompt
func CreateAssistantPrompt(template string, variables []string) prompts.MessageFormatter {
	return prompts.NewAIMessagePromptTemplate(template, variables)
}
