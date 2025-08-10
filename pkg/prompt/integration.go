package prompt

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/prompts"
)

// LoadDefaultTemplate loads the default prompt template based on configuration
func LoadDefaultTemplate() (Template, error) {
	// Check if a custom template is configured
	templateName := viper.GetString("prompt.template")
	if templateName != "" {
		// Try to get from registry first
		if template, err := DefaultRegistry.Get(templateName); err == nil {
			return template, nil
		}

		// Try to load from file
		templateDir := viper.GetString("prompt.template_dir")
		if templateDir == "" {
			templateDir = "./prompts"
		}

		loader := NewFileLoader(templateDir)
		if template, err := loader.Load(templateName); err == nil {
			return template, nil
		}
	}

	// Check for inline template configuration
	if inlineTemplate := viper.GetString("prompt.inline_template"); inlineTemplate != "" {
		vars := viper.GetStringSlice("prompt.variables")
		return NewPromptTemplate(inlineTemplate, vars), nil
	}

	// Use a sensible default based on mode
	if viper.GetBool("vectorstore.enabled") {
		// Use RAG template if vectorstore is enabled
		return GetRAGTemplate(), nil
	}

	// Use basic assistant template
	return GetAssistantTemplate(), nil
}

// LoadChatTemplate loads a chat template from configuration or defaults
func LoadChatTemplate() (ChatTemplate, error) {
	// Check if a custom chat template is configured
	templateName := viper.GetString("prompt.chat_template")
	if templateName != "" {
		// Try to get from registry
		if template, err := DefaultRegistry.Get(templateName); err == nil {
			if chatTemplate, ok := template.(ChatTemplate); ok {
				return chatTemplate, nil
			}
		}

		// Try to load from file
		templateDir := viper.GetString("prompt.template_dir")
		if templateDir == "" {
			templateDir = "./prompts"
		}

		loader := NewFileLoader(templateDir)
		return loader.LoadChat(templateName)
	}

	// Create from configured messages
	if viper.IsSet("prompt.messages") {
		var messages []MessageDefinition
		if err := viper.UnmarshalKey("prompt.messages", &messages); err == nil {
			return NewChatTemplateFromMessages(messages)
		}
	}

	// Default to assistant template
	return GetAssistantTemplate(), nil
}

// ConfigureFromViper configures the prompt system from Viper settings
func ConfigureFromViper() error {
	// Load and register templates from directory if configured
	if templateDir := viper.GetString("prompt.template_dir"); templateDir != "" {
		loader := NewFileLoader(templateDir)

		// Load all template files
		pattern := filepath.Join(templateDir, "*.yaml")
		matches, _ := filepath.Glob(pattern)
		for _, file := range matches {
			name := filepath.Base(file)
			name = name[:len(name)-5] // Remove .yaml extension

			if template, err := loader.Load(file); err == nil {
				DefaultRegistry.Register(name, template)
			}
		}

		// Also check for JSON files
		pattern = filepath.Join(templateDir, "*.json")
		matches, _ = filepath.Glob(pattern)
		for _, file := range matches {
			name := filepath.Base(file)
			name = name[:len(name)-5] // Remove .json extension

			if template, err := loader.Load(file); err == nil {
				DefaultRegistry.Register(name, template)
			}
		}
	}

	// Register any inline templates from config
	if viper.IsSet("prompt.templates") {
		templates := viper.GetStringMap("prompt.templates")
		for name, tmpl := range templates {
			if tmplStr, ok := tmpl.(string); ok {
				vars := extractVariables(tmplStr)
				template := NewPromptTemplate(tmplStr, vars)
				DefaultRegistry.Register(name, template)
			}
		}
	}

	return nil
}

// CreateSystemPrompt creates a system prompt for the agent
func CreateSystemPrompt(role string, instructions string) prompts.MessageFormatter {
	template := fmt.Sprintf("You are %s. %s", role, instructions)
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
