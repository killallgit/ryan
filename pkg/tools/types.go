package tools

import (
	"context"
	"time"
)

// Tool represents a function that can be called by an LLM
type Tool interface {
	// Name returns the unique identifier for this tool
	Name() string

	// Description returns a human/LLM-readable description of what this tool does
	Description() string

	// JSONSchema returns the JSON Schema for the tool's parameters
	// This is used to generate provider-specific tool definitions
	JSONSchema() map[string]any

	// Execute runs the tool with the given parameters
	Execute(ctx context.Context, params map[string]any) (ToolResult, error)
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	// Success indicates whether the tool execution was successful
	Success bool `json:"success"`

	// Content is the main result content
	Content string `json:"content"`

	// Data contains structured data from the tool execution (optional)
	Data map[string]any `json:"data,omitempty"`

	// Error contains error information if Success is false
	Error string `json:"error,omitempty"`

	// Metadata contains additional information about the execution
	Metadata ToolMetadata `json:"metadata"`
}

// ToolMetadata contains execution information
type ToolMetadata struct {
	// ExecutionTime is how long the tool took to execute
	ExecutionTime time.Duration `json:"execution_time"`

	// StartTime is when the tool execution began
	StartTime time.Time `json:"start_time"`

	// EndTime is when the tool execution completed
	EndTime time.Time `json:"end_time"`

	// ToolName is the name of the tool that was executed
	ToolName string `json:"tool_name"`

	// Parameters are the parameters that were passed to the tool
	Parameters map[string]any `json:"parameters"`
}

// ToolRequest represents a request to execute a tool
type ToolRequest struct {
	// Name is the name of the tool to execute
	Name string `json:"name"`

	// Parameters are the parameters to pass to the tool
	Parameters map[string]any `json:"parameters"`

	// Context for the execution (timeout, cancellation)
	Context context.Context `json:"-"`
}

// ToolDefinition represents a tool definition in provider-specific format
type ToolDefinition struct {
	// Provider is the target provider (openai, anthropic, ollama, mcp)
	Provider string `json:"provider"`

	// Definition is the provider-specific tool definition
	Definition map[string]any `json:"definition"`
}

// JSONSchemaProperty represents a property in a JSON Schema
type JSONSchemaProperty struct {
	Type        string                        `json:"type"`
	Description string                        `json:"description"`
	Properties  map[string]JSONSchemaProperty `json:"properties,omitempty"`
	Required    []string                      `json:"required,omitempty"`
	Items       *JSONSchemaProperty           `json:"items,omitempty"`
	Enum        []any                         `json:"enum,omitempty"`
	Default     any                           `json:"default,omitempty"`
}

// NewJSONSchema creates a basic JSON Schema structure
func NewJSONSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": make(map[string]any),
		"required":   []string{},
	}
}

// AddProperty adds a property to a JSON Schema
func AddProperty(schema map[string]any, name string, property JSONSchemaProperty) {
	if properties, ok := schema["properties"].(map[string]any); ok {
		properties[name] = property
	}
}

// AddRequired adds a required field to a JSON Schema
func AddRequired(schema map[string]any, field string) {
	if required, ok := schema["required"].([]string); ok {
		schema["required"] = append(required, field)
	}
}

// ToolError represents an error from tool execution
type ToolError struct {
	ToolName string
	Message  string
	Cause    error
}

func (e ToolError) Error() string {
	if e.Cause != nil {
		return e.ToolName + ": " + e.Message + ": " + e.Cause.Error()
	}
	return e.ToolName + ": " + e.Message
}

func (e ToolError) Unwrap() error {
	return e.Cause
}
