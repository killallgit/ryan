package prompt

import (
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
)

// ChatPromptTemplate is a concrete implementation of ChatTemplate
type ChatPromptTemplate struct {
	messages         []prompts.MessageFormatter
	partialVariables map[string]any
	metadata         map[string]*Variable
}

// NewChatTemplate creates a new chat prompt template
func NewChatTemplate(messages []prompts.MessageFormatter) *ChatPromptTemplate {
	return &ChatPromptTemplate{
		messages:         messages,
		partialVariables: make(map[string]any),
		metadata:         make(map[string]*Variable),
	}
}

// NewChatTemplateFromMessages creates a chat template from message definitions
func NewChatTemplateFromMessages(messages []MessageDefinition) (*ChatPromptTemplate, error) {
	formatters := make([]prompts.MessageFormatter, 0, len(messages))

	for _, msg := range messages {
		formatter, err := createMessageFormatter(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to create message formatter: %w", err)
		}
		formatters = append(formatters, formatter)
	}

	return NewChatTemplate(formatters), nil
}

// Format formats the template with the given values as a string
func (c *ChatPromptTemplate) Format(values map[string]any) (string, error) {
	merged := c.mergeValues(values)

	if err := c.validateVariables(merged); err != nil {
		return "", err
	}

	template := prompts.ChatPromptTemplate{
		Messages:         c.messages,
		PartialVariables: c.partialVariables,
	}

	return template.Format(merged)
}

// FormatPrompt formats the template as a prompt value
func (c *ChatPromptTemplate) FormatPrompt(values map[string]any) (llms.PromptValue, error) {
	merged := c.mergeValues(values)

	if err := c.validateVariables(merged); err != nil {
		return nil, err
	}

	template := prompts.ChatPromptTemplate{
		Messages:         c.messages,
		PartialVariables: c.partialVariables,
	}

	return template.FormatPrompt(merged)
}

// FormatMessages formats the template as chat messages
func (c *ChatPromptTemplate) FormatMessages(values map[string]any) ([]llms.ChatMessage, error) {
	merged := c.mergeValues(values)

	if err := c.validateVariables(merged); err != nil {
		return nil, err
	}

	template := prompts.ChatPromptTemplate{
		Messages:         c.messages,
		PartialVariables: c.partialVariables,
	}

	return template.FormatMessages(merged)
}

// GetInputVariables returns the list of input variable names
func (c *ChatPromptTemplate) GetInputVariables() []string {
	varMap := make(map[string]bool)

	for _, msg := range c.messages {
		for _, v := range msg.GetInputVariables() {
			varMap[v] = true
		}
	}

	// Remove partial variables from the list
	for k := range c.partialVariables {
		delete(varMap, k)
	}

	vars := make([]string, 0, len(varMap))
	for v := range varMap {
		vars = append(vars, v)
	}

	return vars
}

// WithPartialVariables creates a new template with partial variables set
func (c *ChatPromptTemplate) WithPartialVariables(partials map[string]any) Template {
	newTemplate := &ChatPromptTemplate{
		messages:         c.messages,
		partialVariables: make(map[string]any),
		metadata:         c.metadata,
	}

	// Copy existing partials
	for k, v := range c.partialVariables {
		newTemplate.partialVariables[k] = v
	}

	// Add new partials
	for k, v := range partials {
		newTemplate.partialVariables[k] = v
	}

	return newTemplate
}

// mergeValues merges partial variables with provided values
func (c *ChatPromptTemplate) mergeValues(values map[string]any) map[string]any {
	merged := make(map[string]any)

	// Start with partial variables
	for k, v := range c.partialVariables {
		merged[k] = v
	}

	// Override with provided values
	for k, v := range values {
		merged[k] = v
	}

	// Apply defaults for missing variables
	for _, varName := range c.GetInputVariables() {
		if _, exists := merged[varName]; !exists {
			if meta, ok := c.metadata[varName]; ok && meta.Default != nil {
				merged[varName] = meta.Default
			}
		}
	}

	return merged
}

// validateVariables validates that all required variables are present
func (c *ChatPromptTemplate) validateVariables(values map[string]any) error {
	for _, varName := range c.GetInputVariables() {
		meta, hasMeta := c.metadata[varName]
		value, exists := values[varName]

		// Check if required variable is missing
		if !exists && hasMeta && meta.Required {
			return fmt.Errorf("missing required variable: %s", varName)
		}

		// Run validator if present
		if exists && hasMeta && meta.Validator != nil {
			if err := meta.Validator(value); err != nil {
				return fmt.Errorf("validation failed for variable %s: %w", varName, err)
			}
		}
	}

	return nil
}

// MessageDefinition defines a message in a chat template
type MessageDefinition struct {
	Role      string // system, human, ai, generic
	Template  string
	Variables []string
}

// createMessageFormatter creates a message formatter from a definition
func createMessageFormatter(def MessageDefinition) (prompts.MessageFormatter, error) {
	switch def.Role {
	case "system":
		return prompts.NewSystemMessagePromptTemplate(def.Template, def.Variables), nil
	case "human":
		return prompts.NewHumanMessagePromptTemplate(def.Template, def.Variables), nil
	case "ai":
		return prompts.NewAIMessagePromptTemplate(def.Template, def.Variables), nil
	case "generic":
		return prompts.NewGenericMessagePromptTemplate(def.Role, def.Template, def.Variables), nil
	default:
		return nil, fmt.Errorf("unknown message role: %s", def.Role)
	}
}
