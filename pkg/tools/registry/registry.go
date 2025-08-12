package registry

import (
	"fmt"
	"sync"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/tmc/langchaingo/tools"
)

// toolRegistry is the default implementation of Registry
type toolRegistry struct {
	mu        sync.RWMutex
	factories map[string]ToolFactory
}

// global is the global tool registry instance
var global Registry

// init initializes the global registry
func init() {
	global = New()
}

// New creates a new tool registry
func New() Registry {
	return &toolRegistry{
		factories: make(map[string]ToolFactory),
	}
}

// Global returns the global registry instance
func Global() Registry {
	return global
}

// Register registers a tool factory with a given name
func (r *toolRegistry) Register(name string, factory ToolFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if factory == nil {
		return fmt.Errorf("factory cannot be nil")
	}

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("tool %s is already registered", name)
	}

	r.factories[name] = factory
	logger.Debug("Registered tool: %s", name)
	return nil
}

// Get retrieves a tool by name
func (r *toolRegistry) Get(name string, skipPermissions bool) (tools.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return factory(skipPermissions), nil
}

// GetAll returns all registered tool names
func (r *toolRegistry) GetAll() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// GetEnabled returns all enabled tools based on configuration
func (r *toolRegistry) GetEnabled(settings *config.Settings, skipPermissions bool) []tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var enabledTools []tools.Tool

	// Only add tools if enabled in config
	if !settings.Tools.Enabled {
		logger.Info("Tools are disabled in configuration")
		return enabledTools
	}

	logger.Debug("Tools are enabled, initializing available tools")

	// Check each tool type and add if enabled
	toolConfigs := map[string]bool{
		"file_read":  settings.Tools.File.Read.Enabled,
		"file_write": settings.Tools.File.Write.Enabled,
		"git":        settings.Tools.Git.Enabled,
		"ripgrep":    settings.Tools.Search.Enabled,
		"webfetch":   settings.Tools.Web.Enabled,
		"bash":       settings.Tools.Bash.Enabled,
	}

	for toolName, isEnabled := range toolConfigs {
		if !isEnabled {
			continue
		}

		factory, exists := r.factories[toolName]
		if !exists {
			logger.Warn("Tool %s is enabled but not registered", toolName)
			continue
		}

		tool := factory(skipPermissions)
		enabledTools = append(enabledTools, tool)
		logger.Debug("Added %s tool", toolName)
	}

	logger.Info("Initialized %d tools", len(enabledTools))
	return enabledTools
}

// IsRegistered checks if a tool is registered
func (r *toolRegistry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.factories[name]
	return exists
}

// Clear removes all registered tools (useful for testing)
func (r *toolRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.factories = make(map[string]ToolFactory)
}
