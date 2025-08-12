package registry

import (
	"github.com/killallgit/ryan/pkg/config"
	"github.com/tmc/langchaingo/tools"
)

// ToolFactory is a function that creates a tool instance
type ToolFactory func(skipPermissions bool) tools.Tool

// Registry manages tool registration and creation
type Registry interface {
	// Register registers a tool factory with a given name
	Register(name string, factory ToolFactory) error

	// Get retrieves a tool by name
	Get(name string, skipPermissions bool) (tools.Tool, error)

	// GetAll returns all registered tool names
	GetAll() []string

	// GetEnabled returns all enabled tools based on configuration
	GetEnabled(settings *config.Settings, skipPermissions bool) []tools.Tool

	// IsRegistered checks if a tool is registered
	IsRegistered(name string) bool

	// Clear removes all registered tools (useful for testing)
	Clear()
}

// ToolInfo contains metadata about a registered tool
type ToolInfo struct {
	Name        string
	Description string
	Factory     ToolFactory
}

// InitFunc is a function that initializes tools in the registry
type InitFunc func(r Registry) error
