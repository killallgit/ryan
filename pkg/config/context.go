package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// ContextManager implements Claude CLI's two-tier configuration system
// with Global (system-wide) + Project (project-specific) configurations
type ContextManager struct {
	mu             sync.RWMutex
	globalConfig   *GlobalConfig
	projectConfigs map[string]*ProjectConfig
	cache          *LRUCache
	projectRoot    string
	deltaStorage   *GlobalConfigDeltaStorage
}

// GlobalConfig represents system-wide configuration
type GlobalConfig struct {
	// Core system settings
	NumStartups              int    `json:"numStartups"`
	InstallMethod            string `json:"installMethod,omitempty"`
	AutoUpdates              *bool  `json:"autoUpdates,omitempty"`
	Theme                    string `json:"theme"`
	PreferredNotifChannel    string `json:"preferredNotifChannel"`
	Verbose                  bool   `json:"verbose"`
	EditorMode               string `json:"editorMode"`
	AutoCompactEnabled       bool   `json:"autoCompactEnabled"`
	HasSeenTasksHint         bool   `json:"hasSeenTasksHint"`
	QueuedCommandUpHintCount int    `json:"queuedCommandUpHintCount"`
	DiffTool                 string `json:"diffTool"`

	// Custom API key management
	CustomAPIKeyResponses struct {
		Approved []string `json:"approved"`
		Rejected []string `json:"rejected"`
	} `json:"customApiKeyResponses"`

	// Environment variable overrides
	Env map[string]string `json:"env"`

	// Tips and onboarding state
	TipsHistory map[string]interface{} `json:"tipsHistory"`

	// Project-specific configurations stored within global config
	Projects map[string]*ProjectConfig `json:"projects"`

	// Configuration metadata
	LastModified time.Time `json:"lastModified"`
	Version      string    `json:"version"`
}

// ProjectConfig represents project-specific configuration
type ProjectConfig struct {
	// Tool permissions and allowed tools
	AllowedTools []string `json:"allowedTools"`

	// Conversation history for this project
	History []ConversationMessage `json:"history"`

	// MCP (Model Context Protocol) configuration
	MCPContextUris     []string               `json:"mcpContextUris"`
	MCPServers         map[string]interface{} `json:"mcpServers"`
	EnabledMCPServers  []string               `json:"enabledMcpjsonServers"`
	DisabledMCPServers []string               `json:"disabledMcpjsonServers"`

	// Security and trust settings
	HasTrustDialogAccepted bool `json:"hasTrustDialogAccepted"`

	// File and pattern management
	IgnorePatterns []string `json:"ignorePatterns"`

	// Onboarding and UX state
	ProjectOnboardingSeenCount              int  `json:"projectOnboardingSeenCount"`
	HasClaudeMdExternalIncludesApproved     bool `json:"hasClaudeMdExternalIncludesApproved"`
	HasClaudeMdExternalIncludesWarningShown bool `json:"hasClaudeMdExternalIncludesWarningShown"`

	// Project metadata
	ProjectRoot  string    `json:"projectRoot"`
	LastModified time.Time `json:"lastModified"`
}

// ConversationMessage represents a message in the conversation history
type ConversationMessage struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"` // "user", "assistant", "system"
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LRUCache implements a simple LRU cache with a maximum size of 50 items
type LRUCache struct {
	mu      sync.RWMutex
	maxSize int
	items   map[string]*cacheItem
	order   []string // Maintain access order
}

type cacheItem struct {
	value      interface{}
	lastAccess time.Time
}

// NewLRUCache creates a new LRU cache with the specified maximum size
func NewLRUCache(maxSize int) *LRUCache {
	if maxSize <= 0 {
		maxSize = 50 // Default cache size matching Claude CLI
	}

	return &LRUCache{
		maxSize: maxSize,
		items:   make(map[string]*cacheItem),
		order:   make([]string, 0, maxSize),
	}
}

// Get retrieves an item from the cache and updates its access time
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Update access time and move to end (most recently used)
	item.lastAccess = time.Now()
	c.moveToEnd(key)

	return item.value, true
}

// Set adds or updates an item in the cache
func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If item already exists, update it
	if item, exists := c.items[key]; exists {
		item.value = value
		item.lastAccess = time.Now()
		c.moveToEnd(key)
		return
	}

	// If cache is full, remove least recently used item
	if len(c.items) >= c.maxSize {
		c.removeLRU()
	}

	// Add new item
	c.items[key] = &cacheItem{
		value:      value,
		lastAccess: time.Now(),
	}
	c.order = append(c.order, key)
}

// moveToEnd moves a key to the end of the order slice (most recently used)
func (c *LRUCache) moveToEnd(key string) {
	// Find and remove the key from its current position
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	// Add to end
	c.order = append(c.order, key)
}

// removeLRU removes the least recently used item
func (c *LRUCache) removeLRU() {
	if len(c.order) == 0 {
		return
	}

	// Remove the first item (least recently used)
	key := c.order[0]
	delete(c.items, key)
	c.order = c.order[1:]
}

// NewContextManager creates a new context manager
func NewContextManager() *ContextManager {
	return &ContextManager{
		projectConfigs: make(map[string]*ProjectConfig),
		cache:          NewLRUCache(50),
		deltaStorage:   NewGlobalConfigDeltaStorage(),
	}
}

// GetProjectRoot detects the project root using git or falls back to cwd
func (cm *ContextManager) GetProjectRoot() (string, error) {
	// Try to use git to find repository root
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err == nil {
		root := strings.TrimSpace(string(output))
		if root != "" {
			return root, nil
		}
	}

	// Fallback to current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	return cwd, nil
}

// GetGlobalConfigPath returns the path to the global configuration file
func (cm *ContextManager) GetGlobalConfigPath() (string, error) {
	// Check for environment variable override (via Viper)
	if configDir := viper.GetString("RYAN_CONFIG_DIR"); configDir != "" {
		return filepath.Join(configDir, "config.json"), nil
	}

	// Use standard location
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check for new format first: ~/.ryan/config.json
	newPath := filepath.Join(home, ".ryan", "config.json")
	if _, err := os.Stat(newPath); err == nil {
		return newPath, nil
	}

	// Fallback to legacy format: ~/.ryan.json
	legacyPath := filepath.Join(home, ".ryan.json")
	return legacyPath, nil
}

// LoadGlobalConfig loads the global configuration with caching
func (cm *ContextManager) LoadGlobalConfig() (*GlobalConfig, error) {
	configPath, err := cm.GetGlobalConfigPath()
	if err != nil {
		return nil, err
	}

	// Check cache first
	if cached, found := cm.cache.Get("global_config"); found {
		if config, ok := cached.(*GlobalConfig); ok {
			return config, nil
		}
	}

	// Load from file
	config, err := cm.loadGlobalConfigFromFile(configPath)
	if err != nil {
		// Return default configuration if file doesn't exist
		if os.IsNotExist(err) {
			config = cm.getDefaultGlobalConfig()
		} else {
			return nil, err
		}
	}

	// Cache the result
	cm.cache.Set("global_config", config)

	cm.mu.Lock()
	cm.globalConfig = config
	cm.mu.Unlock()

	return config, nil
}

// loadGlobalConfigFromFile loads configuration from the specified file path
// Supports both delta storage format and legacy full configuration format
func (cm *ContextManager) loadGlobalConfigFromFile(configPath string) (*GlobalConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Try to detect if this is a delta format file
	var deltaTest map[string]interface{}
	if json.Unmarshal(data, &deltaTest) == nil {
		if _, hasDelta := deltaTest["delta"]; hasDelta {
			// This is a delta format file
			config, err := cm.deltaStorage.LoadFromDelta(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load delta configuration: %w", err)
			}

			// Initialize Projects map if nil
			if config.Projects == nil {
				config.Projects = make(map[string]*ProjectConfig)
			}

			return config, nil
		}
	}

	// Legacy format - full configuration
	var config GlobalConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}

	// Initialize Projects map if nil
	if config.Projects == nil {
		config.Projects = make(map[string]*ProjectConfig)
	}

	return &config, nil
}

// getDefaultGlobalConfig returns the default global configuration
func (cm *ContextManager) getDefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		NumStartups:              0,
		Theme:                    "dark",
		PreferredNotifChannel:    "auto",
		Verbose:                  false,
		EditorMode:               "normal",
		AutoCompactEnabled:       true,
		HasSeenTasksHint:         false,
		QueuedCommandUpHintCount: 0,
		DiffTool:                 "auto",
		CustomAPIKeyResponses: struct {
			Approved []string `json:"approved"`
			Rejected []string `json:"rejected"`
		}{
			Approved: []string{},
			Rejected: []string{},
		},
		Env:          make(map[string]string),
		TipsHistory:  make(map[string]interface{}),
		Projects:     make(map[string]*ProjectConfig),
		LastModified: time.Now(),
		Version:      "1.0.0",
	}
}

// GetProjectConfig loads project-specific configuration
func (cm *ContextManager) GetProjectConfig(projectRoot string) (*ProjectConfig, error) {
	if projectRoot == "" {
		var err error
		projectRoot, err = cm.GetProjectRoot()
		if err != nil {
			return nil, err
		}
	}

	// Check cache first
	cacheKey := fmt.Sprintf("project_config_%s", projectRoot)
	if cached, found := cm.cache.Get(cacheKey); found {
		if config, ok := cached.(*ProjectConfig); ok {
			return config, nil
		}
	}

	// Load global config to access project configurations
	globalConfig, err := cm.LoadGlobalConfig()
	if err != nil {
		return nil, err
	}

	// Get project config from global config
	projectConfig, exists := globalConfig.Projects[projectRoot]
	if !exists {
		// Create default project configuration
		projectConfig = cm.getDefaultProjectConfig(projectRoot)
		globalConfig.Projects[projectRoot] = projectConfig
	}

	// Cache the result
	cm.cache.Set(cacheKey, projectConfig)

	cm.mu.Lock()
	cm.projectConfigs[projectRoot] = projectConfig
	cm.projectRoot = projectRoot
	cm.mu.Unlock()

	return projectConfig, nil
}

// getDefaultProjectConfig returns the default project configuration
func (cm *ContextManager) getDefaultProjectConfig(projectRoot string) *ProjectConfig {
	return &ProjectConfig{
		AllowedTools:                            []string{},
		History:                                 []ConversationMessage{},
		MCPContextUris:                          []string{},
		MCPServers:                              make(map[string]interface{}),
		EnabledMCPServers:                       []string{},
		DisabledMCPServers:                      []string{},
		HasTrustDialogAccepted:                  false,
		IgnorePatterns:                          []string{},
		ProjectOnboardingSeenCount:              0,
		HasClaudeMdExternalIncludesApproved:     false,
		HasClaudeMdExternalIncludesWarningShown: false,
		ProjectRoot:                             projectRoot,
		LastModified:                            time.Now(),
	}
}

// GetEffectiveConfig returns the effective configuration by merging global and project settings
// following the hierarchy: Environment Variables → Project → Global → System Defaults
func (cm *ContextManager) GetEffectiveConfig() (*Config, error) {
	// Load both global and project configurations
	globalConfig, err := cm.LoadGlobalConfig()
	if err != nil {
		return nil, err
	}

	projectRoot, err := cm.GetProjectRoot()
	if err != nil {
		return nil, err
	}

	projectConfig, err := cm.GetProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	// Create the effective configuration by merging
	// This will be used to populate the existing Config struct
	// For now, return the existing configuration with context awareness
	existingConfig := Get()

	// Add context information to the existing config
	// This is a bridge to maintain compatibility while we migrate
	if existingConfig != nil {
		// Store project root and context information
		cm.mu.Lock()
		cm.projectRoot = projectRoot
		cm.globalConfig = globalConfig
		cm.projectConfigs[projectRoot] = projectConfig
		cm.mu.Unlock()
	}

	return existingConfig, nil
}

// SaveProjectConfig saves the project configuration with atomic operations
func (cm *ContextManager) SaveProjectConfig(projectRoot string, config *ProjectConfig) error {
	if projectRoot == "" {
		var err error
		projectRoot, err = cm.GetProjectRoot()
		if err != nil {
			return err
		}
	}

	// Load global config
	globalConfig, err := cm.LoadGlobalConfig()
	if err != nil {
		return err
	}

	// Update project config in global config
	config.LastModified = time.Now()
	globalConfig.Projects[projectRoot] = config
	globalConfig.LastModified = time.Now()

	// Save global config with atomic operations
	if err := cm.saveGlobalConfigAtomic(globalConfig); err != nil {
		return err
	}

	// Update cache
	cm.cache.Set("global_config", globalConfig)
	cacheKey := fmt.Sprintf("project_config_%s", projectRoot)
	cm.cache.Set(cacheKey, config)

	// Update in-memory state
	cm.mu.Lock()
	cm.globalConfig = globalConfig
	cm.projectConfigs[projectRoot] = config
	cm.mu.Unlock()

	return nil
}

// saveGlobalConfigAtomic saves the global configuration using atomic write operations with file locking and delta storage
func (cm *ContextManager) saveGlobalConfigAtomic(config *GlobalConfig) error {
	configPath, err := cm.GetGlobalConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Use delta storage for efficient persistence
	return cm.deltaStorage.SaveDelta(config, configPath)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}

// GetCurrentProjectRoot returns the currently detected project root
func (cm *ContextManager) GetCurrentProjectRoot() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.projectRoot
}

// InvalidateCache invalidates the configuration cache
func (cm *ContextManager) InvalidateCache() {
	cm.cache = NewLRUCache(50)
}
