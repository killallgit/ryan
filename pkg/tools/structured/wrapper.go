package structured

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/tools"
)

// Parameter defines a tool parameter with its schema
type Parameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// Tool wraps a function with structured parameters
type Tool struct {
	name        string
	description string
	parameters  []Parameter
	executor    func(context.Context, map[string]interface{}) (string, error)
}

// New creates a new structured tool
func New(name, description string, params []Parameter, exec func(context.Context, map[string]interface{}) (string, error)) *Tool {
	return &Tool{
		name:        name,
		description: description,
		parameters:  params,
		executor:    exec,
	}
}

// Name returns the tool name
func (t *Tool) Name() string {
	return t.name
}

// Description returns the tool description with parameter info
func (t *Tool) Description() string {
	desc := t.description
	if len(t.parameters) > 0 {
		desc += "\nParameters (JSON format):\n"
		for _, p := range t.parameters {
			req := ""
			if p.Required {
				req = " (required)"
			}
			desc += fmt.Sprintf("  - %s: %s (%s)%s\n", p.Name, p.Description, p.Type, req)
		}
	}
	return desc
}

// Call implements tools.Tool interface
func (t *Tool) Call(ctx context.Context, input string) (string, error) {
	// Try to parse as JSON first
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		// Fallback: treat as single string parameter if tool expects it
		if len(t.parameters) == 1 && t.parameters[0].Type == "string" {
			params = map[string]interface{}{
				t.parameters[0].Name: input,
			}
		} else {
			return "", fmt.Errorf("invalid input format: expected JSON object, got: %s", input)
		}
	}

	// Validate required parameters
	for _, p := range t.parameters {
		if p.Required {
			if _, exists := params[p.Name]; !exists {
				if p.Default != nil {
					params[p.Name] = p.Default
				} else {
					return "", fmt.Errorf("missing required parameter: %s", p.Name)
				}
			}
		}
	}

	// Execute the tool
	return t.executor(ctx, params)
}

// Ensure we implement the interface
var _ tools.Tool = (*Tool)(nil)

// Builder provides a fluent interface for creating tools
type Builder struct {
	tool *Tool
}

// NewBuilder creates a new tool builder
func NewBuilder(name, description string) *Builder {
	return &Builder{
		tool: &Tool{
			name:        name,
			description: description,
			parameters:  []Parameter{},
		},
	}
}

// WithParameter adds a parameter to the tool
func (b *Builder) WithParameter(name, paramType, description string, required bool) *Builder {
	b.tool.parameters = append(b.tool.parameters, Parameter{
		Name:        name,
		Type:        paramType,
		Description: description,
		Required:    required,
	})
	return b
}

// WithOptionalParameter adds an optional parameter with a default value
func (b *Builder) WithOptionalParameter(name, paramType, description string, defaultValue interface{}) *Builder {
	b.tool.parameters = append(b.tool.parameters, Parameter{
		Name:        name,
		Type:        paramType,
		Description: description,
		Required:    false,
		Default:     defaultValue,
	})
	return b
}

// WithExecutor sets the executor function
func (b *Builder) WithExecutor(exec func(context.Context, map[string]interface{}) (string, error)) *Builder {
	b.tool.executor = exec
	return b
}

// Build returns the constructed tool
func (b *Builder) Build() *Tool {
	return b.tool
}
