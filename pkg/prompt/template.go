package prompt

import (
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
)

// PromptTemplate is a concrete implementation of the Template interface
// that wraps langchaingo's PromptTemplate
type PromptTemplate struct {
	template         prompts.PromptTemplate
	partialVariables map[string]any
	metadata         map[string]*Variable
}

// NewPromptTemplate creates a new prompt template
func NewPromptTemplate(template string, inputVars []string) *PromptTemplate {
	pt := prompts.NewPromptTemplate(template, inputVars)
	return &PromptTemplate{
		template:         pt,
		partialVariables: make(map[string]any),
		metadata:         make(map[string]*Variable),
	}
}

// NewPromptTemplateWithOptions creates a new prompt template with options
func NewPromptTemplateWithOptions(template string, inputVars []string, options ...PromptOption) (*PromptTemplate, error) {
	pt := &PromptTemplate{
		template:         prompts.NewPromptTemplate(template, inputVars),
		partialVariables: make(map[string]any),
		metadata:         make(map[string]*Variable),
	}

	for _, opt := range options {
		if err := opt(pt); err != nil {
			return nil, err
		}
	}

	return pt, nil
}

// Format formats the template with the given values
func (p *PromptTemplate) Format(values map[string]any) (string, error) {
	// Merge partial variables with provided values
	merged := p.mergeValues(values)

	// Validate required variables
	if err := p.validateVariables(merged); err != nil {
		return "", err
	}

	return p.template.Format(merged)
}

// FormatPrompt formats the template as a prompt value
func (p *PromptTemplate) FormatPrompt(values map[string]any) (llms.PromptValue, error) {
	merged := p.mergeValues(values)

	if err := p.validateVariables(merged); err != nil {
		return nil, err
	}

	return p.template.FormatPrompt(merged)
}

// GetInputVariables returns the list of input variable names
func (p *PromptTemplate) GetInputVariables() []string {
	return p.template.InputVariables
}

// WithPartialVariables creates a new template with partial variables set
func (p *PromptTemplate) WithPartialVariables(partials map[string]any) Template {
	newTemplate := &PromptTemplate{
		template:         p.template,
		partialVariables: make(map[string]any),
		metadata:         p.metadata,
	}

	// Copy existing partials
	for k, v := range p.partialVariables {
		newTemplate.partialVariables[k] = v
	}

	// Add new partials
	for k, v := range partials {
		newTemplate.partialVariables[k] = v
	}

	return newTemplate
}

// SetVariableMetadata sets metadata for a variable
func (p *PromptTemplate) SetVariableMetadata(variable *Variable) {
	p.metadata[variable.Name] = variable
}

// mergeValues merges partial variables with provided values
func (p *PromptTemplate) mergeValues(values map[string]any) map[string]any {
	merged := make(map[string]any)

	// Start with partial variables
	for k, v := range p.partialVariables {
		merged[k] = v
	}

	// Override with provided values
	for k, v := range values {
		merged[k] = v
	}

	// Apply defaults for missing variables
	for _, varName := range p.template.InputVariables {
		if _, exists := merged[varName]; !exists {
			if meta, ok := p.metadata[varName]; ok && meta.Default != nil {
				merged[varName] = meta.Default
			}
		}
	}

	return merged
}

// validateVariables validates that all required variables are present
func (p *PromptTemplate) validateVariables(values map[string]any) error {
	var missing []string

	for _, varName := range p.template.InputVariables {
		meta, hasMeta := p.metadata[varName]
		value, exists := values[varName]

		// Check if required variable is missing
		if !exists && hasMeta && meta.Required {
			missing = append(missing, varName)
			continue
		}

		// Run validator if present
		if exists && hasMeta && meta.Validator != nil {
			if err := meta.Validator(value); err != nil {
				return fmt.Errorf("validation failed for variable %s: %w", varName, err)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// PromptOption is a functional option for configuring a PromptTemplate
type PromptOption func(*PromptTemplate) error

// WithPartials sets partial variables
func WithPartials(partials map[string]any) PromptOption {
	return func(pt *PromptTemplate) error {
		pt.partialVariables = partials
		return nil
	}
}

// WithVariableMetadata sets metadata for variables
func WithVariableMetadata(variables ...*Variable) PromptOption {
	return func(pt *PromptTemplate) error {
		for _, v := range variables {
			pt.metadata[v.Name] = v
		}
		return nil
	}
}
