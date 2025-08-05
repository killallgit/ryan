package config

import (
	"fmt"
	"sync"
	"time"
)

// ConfigurationBridge bridges the new Claude CLI-style context management
// with the existing Viper-based configuration system for backward compatibility
type ConfigurationBridge struct {
	contextManager *ContextManager
	hierarchy      *ConfigHierarchy
	legacy         *Config
	mu             sync.RWMutex
}

var (
	bridge     *ConfigurationBridge
	bridgeMu   sync.RWMutex
)

// InitializeBridge initializes the configuration bridge with both systems
func InitializeBridge(legacyConfig *Config) error {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	
	contextManager := NewContextManager()
	hierarchy := NewConfigHierarchy(contextManager)
	
	// Load context configurations
	if _, err := contextManager.LoadGlobalConfig(); err != nil {
		return fmt.Errorf("failed to load global context config: %w", err)
	}
	
	if _, err := contextManager.GetProjectConfig(""); err != nil {
		return fmt.Errorf("failed to load project context config: %w", err)
	}
	
	bridge = &ConfigurationBridge{
		contextManager: contextManager,
		hierarchy:      hierarchy,
		legacy:         legacyConfig,
	}
	
	return nil
}

// GetBridge returns the global configuration bridge
func GetBridge() *ConfigurationBridge {
	bridgeMu.RLock()
	defer bridgeMu.RUnlock()
	return bridge
}

// GetContextManager returns the context manager for Claude CLI-style operations
func (b *ConfigurationBridge) GetContextManager() *ContextManager {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.contextManager
}

// GetLegacyConfig returns the legacy configuration for backward compatibility
func (b *ConfigurationBridge) GetLegacyConfig() *Config {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.legacy
}

// GetGlobalConfig returns the Claude CLI-style global configuration
func (b *ConfigurationBridge) GetGlobalConfig() (*GlobalConfig, error) {
	b.mu.RLock()
	contextManager := b.contextManager
	b.mu.RUnlock()
	
	return contextManager.LoadGlobalConfig()
}

// GetProjectConfig returns the Claude CLI-style project configuration
func (b *ConfigurationBridge) GetProjectConfig() (*ProjectConfig, error) {
	b.mu.RLock()
	contextManager := b.contextManager
	b.mu.RUnlock()
	
	return contextManager.GetProjectConfig("")
}

// SaveProjectConfig saves project configuration changes
func (b *ConfigurationBridge) SaveProjectConfig(config *ProjectConfig) error {
	b.mu.RLock()
	contextManager := b.contextManager
	b.mu.RUnlock()
	
	projectRoot, err := contextManager.GetProjectRoot()
	if err != nil {
		return err
	}
	
	return contextManager.SaveProjectConfig(projectRoot, config)
}

// AddToHistory adds a conversation message to the project history
func (b *ConfigurationBridge) AddToHistory(role, content string, metadata map[string]interface{}) error {
	projectConfig, err := b.GetProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to get project config: %w", err)
	}
	
	message := ConversationMessage{
		ID:        generateMessageID(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
	
	projectConfig.History = append(projectConfig.History, message)
	
	return b.SaveProjectConfig(projectConfig)
}

// GetHistory returns the conversation history for the current project
func (b *ConfigurationBridge) GetHistory() ([]ConversationMessage, error) {
	projectConfig, err := b.GetProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get project config: %w", err)
	}
	
	return projectConfig.History, nil
}

// ClearHistory clears the conversation history for the current project
func (b *ConfigurationBridge) ClearHistory() error {
	projectConfig, err := b.GetProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to get project config: %w", err)
	}
	
	projectConfig.History = []ConversationMessage{}
	
	return b.SaveProjectConfig(projectConfig)
}

// IsToolAllowed checks if a tool is allowed in the current project
func (b *ConfigurationBridge) IsToolAllowed(toolName string) (bool, error) {
	projectConfig, err := b.GetProjectConfig()
	if err != nil {
		return false, fmt.Errorf("failed to get project config: %w", err)
	}
	
	// If no allowed tools are specified, allow all tools (default behavior)
	if len(projectConfig.AllowedTools) == 0 {
		return true, nil
	}
	
	// Check if tool is in allowed list
	for _, allowedTool := range projectConfig.AllowedTools {
		if allowedTool == toolName {
			return true, nil
		}
	}
	
	return false, nil
}

// AllowTool adds a tool to the allowed tools list for the current project
func (b *ConfigurationBridge) AllowTool(toolName string) error {
	projectConfig, err := b.GetProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to get project config: %w", err)
	}
	
	// Check if tool is already allowed
	for _, allowedTool := range projectConfig.AllowedTools {
		if allowedTool == toolName {
			return nil // Already allowed
		}
	}
	
	// Add to allowed tools
	projectConfig.AllowedTools = append(projectConfig.AllowedTools, toolName)
	
	return b.SaveProjectConfig(projectConfig)
}

// GetTrustStatus returns the trust dialog acceptance status for the current project
func (b *ConfigurationBridge) GetTrustStatus() (bool, error) {
	projectConfig, err := b.GetProjectConfig()
	if err != nil {
		return false, fmt.Errorf("failed to get project config: %w", err)
	}
	
	return projectConfig.HasTrustDialogAccepted, nil
}

// SetTrustStatus sets the trust dialog acceptance status for the current project
func (b *ConfigurationBridge) SetTrustStatus(accepted bool) error {
	projectConfig, err := b.GetProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to get project config: %w", err)
	}
	
	projectConfig.HasTrustDialogAccepted = accepted
	
	return b.SaveProjectConfig(projectConfig)
}

// GetIgnorePatterns returns the file ignore patterns for the current project
func (b *ConfigurationBridge) GetIgnorePatterns() ([]string, error) {
	projectConfig, err := b.GetProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get project config: %w", err)
	}
	
	return projectConfig.IgnorePatterns, nil
}

// AddIgnorePattern adds a file ignore pattern for the current project
func (b *ConfigurationBridge) AddIgnorePattern(pattern string) error {
	projectConfig, err := b.GetProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to get project config: %w", err)
	}
	
	// Check if pattern already exists
	for _, existingPattern := range projectConfig.IgnorePatterns {
		if existingPattern == pattern {
			return nil // Already exists
		}
	}
	
	// Add pattern
	projectConfig.IgnorePatterns = append(projectConfig.IgnorePatterns, pattern)
	
	return b.SaveProjectConfig(projectConfig)
}

// GetMCPServers returns the MCP server configuration for the current project
func (b *ConfigurationBridge) GetMCPServers() (map[string]interface{}, error) {
	projectConfig, err := b.GetProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get project config: %w", err)
	}
	
	return projectConfig.MCPServers, nil
}

// SetMCPServer sets MCP server configuration for the current project
func (b *ConfigurationBridge) SetMCPServer(serverName string, config interface{}) error {
	projectConfig, err := b.GetProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to get project config: %w", err)
	}
	
	if projectConfig.MCPServers == nil {
		projectConfig.MCPServers = make(map[string]interface{})
	}
	
	projectConfig.MCPServers[serverName] = config
	
	return b.SaveProjectConfig(projectConfig)
}

// generateMessageID generates a simple message ID
// In a production system, this might use UUIDs or other more robust ID generation
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// GetEffectiveConfiguration returns the effective configuration by merging
// all configuration sources according to the hierarchy
func (b *ConfigurationBridge) GetEffectiveConfiguration() (*Config, error) {
	b.mu.RLock()
	hierarchy := b.hierarchy
	legacy := b.legacy
	b.mu.RUnlock()
	
	// Use the hierarchy system to resolve effective configuration
	effectiveConfig, err := hierarchy.ResolveEffectiveConfig()
	if err != nil {
		// Fallback to legacy config if hierarchy resolution fails
		return legacy, nil
	}
	
	return effectiveConfig, nil
}

// GetConfigValue retrieves a configuration value using the hierarchy
func (b *ConfigurationBridge) GetConfigValue(keyPath string) (interface{}, error) {
	b.mu.RLock()
	hierarchy := b.hierarchy
	b.mu.RUnlock()
	
	return hierarchy.GetConfigValue(keyPath)
}

// SetConfigValue sets a configuration value in the specified scope
func (b *ConfigurationBridge) SetConfigValue(keyPath string, value interface{}, scope string) error {
	b.mu.RLock()
	hierarchy := b.hierarchy
	b.mu.RUnlock()
	
	return hierarchy.SetConfigValue(keyPath, value, scope)
}