package prompt

import (
	"github.com/tmc/langchaingo/llms"
)

// Template represents a generic prompt template interface
type Template interface {
	// Format formats the template with the given variables
	Format(values map[string]any) (string, error)

	// FormatPrompt formats the template as a prompt value
	FormatPrompt(values map[string]any) (llms.PromptValue, error)

	// GetInputVariables returns the list of input variable names
	GetInputVariables() []string

	// WithPartialVariables creates a new template with partial variables set
	WithPartialVariables(partials map[string]any) Template
}

// ChatTemplate represents a chat-based prompt template
type ChatTemplate interface {
	Template

	// FormatMessages formats the template as chat messages
	FormatMessages(values map[string]any) ([]llms.ChatMessage, error)
}

// Loader loads templates from various sources
type Loader interface {
	// Load loads a template by name/path
	Load(name string) (Template, error)

	// LoadChat loads a chat template by name/path
	LoadChat(name string) (ChatTemplate, error)
}

// Registry manages prompt templates
type Registry interface {
	// Register registers a template with a name
	Register(name string, template Template) error

	// Get retrieves a template by name
	Get(name string) (Template, error)

	// List returns all registered template names
	List() []string

	// Clear removes all registered templates
	Clear()
}

// Variable represents a template variable with metadata
type Variable struct {
	Name        string
	Type        string // string, int, float, bool, any
	Required    bool
	Default     any
	Description string
	Validator   func(value any) error
}
