package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/spf13/viper"
)

// ServerDiscoveryManager implements MCP server discovery and registration
type ServerDiscoveryManager struct {
	// Server registry
	servers map[string]ServerConfig
	mu      sync.RWMutex

	// Configuration storage
	cfg *config.Config

	// Discovery sources
	discoveryPaths []string

	// Logging
	logger *logger.Logger
}

// NewServerDiscoveryManager creates a new server discovery manager
func NewServerDiscoveryManager(cfg *config.Config) *ServerDiscoveryManager {
	manager := &ServerDiscoveryManager{
		servers: make(map[string]ServerConfig),
		cfg:     cfg,
		logger:  logger.WithComponent("mcp-discovery"),
		discoveryPaths: []string{
			"./.ryan/mcp-servers.json",   // Project-specific servers
			"~/.ryan/mcp-servers.json",   // User-global servers
			"/etc/ryan/mcp-servers.json", // System-wide servers
		},
	}

	// Load existing server configurations
	if err := manager.loadServerConfigurations(); err != nil {
		manager.logger.Warn("Failed to load server configurations", "error", err)
	}

	return manager
}

// DiscoverServers discovers MCP servers from various sources
func (sdm *ServerDiscoveryManager) DiscoverServers(ctx context.Context) ([]ServerConfig, error) {
	sdm.mu.Lock()
	defer sdm.mu.Unlock()

	var discoveredServers []ServerConfig

	// Discovery from configuration files
	fileServers, err := sdm.discoverFromFiles(ctx)
	if err != nil {
		sdm.logger.Warn("Failed to discover servers from files", "error", err)
	} else {
		discoveredServers = append(discoveredServers, fileServers...)
	}

	// Discovery from environment variables
	envServers := sdm.discoverFromEnvironment()
	discoveredServers = append(discoveredServers, envServers...)

	// Discovery from project configuration
	// TODO: Implement project-specific MCP server configuration using Viper
	// For now, skip project-specific server discovery

	// Update registry with discovered servers
	for _, server := range discoveredServers {
		sdm.servers[server.Name] = server
	}

	sdm.logger.Info("Discovered MCP servers", "count", len(discoveredServers))
	return discoveredServers, nil
}

// discoverFromFiles discovers servers from configuration files
func (sdm *ServerDiscoveryManager) discoverFromFiles(ctx context.Context) ([]ServerConfig, error) {
	var allServers []ServerConfig

	for _, path := range sdm.discoveryPaths {
		// Expand home directory
		if path[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			path = filepath.Join(home, path[1:])
		}

		// Check if file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		// Load servers from file
		servers, err := sdm.loadServersFromFile(path)
		if err != nil {
			sdm.logger.Warn("Failed to load servers from file", "file", path, "error", err)
			continue
		}

		allServers = append(allServers, servers...)
		sdm.logger.Debug("Loaded servers from file", "file", path, "count", len(servers))
	}

	return allServers, nil
}

// loadServersFromFile loads server configurations from a JSON file
func (sdm *ServerDiscoveryManager) loadServersFromFile(filePath string) ([]ServerConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var serverFile struct {
		Servers []ServerConfig `json:"servers"`
		Version string         `json:"version,omitempty"`
	}

	if err := json.Unmarshal(data, &serverFile); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Set default values for servers
	for i := range serverFile.Servers {
		server := &serverFile.Servers[i]
		sdm.setServerDefaults(server)
	}

	return serverFile.Servers, nil
}

// discoverFromEnvironment discovers servers from environment variables
func (sdm *ServerDiscoveryManager) discoverFromEnvironment() []ServerConfig {
	var servers []ServerConfig

	// Check for RYAN_MCP_SERVERS environment variable (via Viper)
	serversEnv := viper.GetString("RYAN_MCP_SERVERS")
	if serversEnv != "" {
		var envServers []ServerConfig
		if err := json.Unmarshal([]byte(serversEnv), &envServers); err == nil {
			for i := range envServers {
				sdm.setServerDefaults(&envServers[i])
			}
			servers = append(servers, envServers...)
			sdm.logger.Debug("Loaded servers from environment", "count", len(envServers))
		}
	}

	// Check for individual server environment variables
	// Format: RYAN_MCP_SERVER_<NAME>_URL, RYAN_MCP_SERVER_<NAME>_AUTH, etc.
	envVars := os.Environ()
	serverEnvs := make(map[string]map[string]string)

	for _, env := range envVars {
		if !strings.HasPrefix(env, "RYAN_MCP_SERVER_") {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Parse server name and property
		keyParts := strings.Split(key, "_")
		if len(keyParts) < 5 {
			continue
		}

		serverName := strings.ToLower(keyParts[3])
		property := strings.ToLower(keyParts[4])

		if serverEnvs[serverName] == nil {
			serverEnvs[serverName] = make(map[string]string)
		}
		serverEnvs[serverName][property] = value
	}

	// Convert environment variables to server configs
	for name, props := range serverEnvs {
		if url, exists := props["url"]; exists {
			server := ServerConfig{
				Name:    name,
				URL:     url,
				Enabled: true,
			}

			if authType, exists := props["auth"]; exists {
				server.AuthType = authType
				server.Credentials = make(map[string]string)

				// Look for credential properties
				for prop, value := range props {
					if strings.HasPrefix(prop, "cred_") {
						credKey := strings.TrimPrefix(prop, "cred_")
						server.Credentials[credKey] = value
					}
				}
			}

			sdm.setServerDefaults(&server)
			servers = append(servers, server)
		}
	}

	return servers
}

// discoverFromProject discovers servers from project configuration
func (sdm *ServerDiscoveryManager) discoverFromProject() ([]ServerConfig, error) {
	// TODO: Implement project-specific MCP server configuration using Viper
	// For now, return empty list
	return []ServerConfig{}, nil
}

// discoverFromProjectOld was the old implementation - preserved for reference
func (sdm *ServerDiscoveryManager) discoverFromProjectOld() ([]ServerConfig, error) {
	var servers []ServerConfig
	/* Old implementation that used ProjectConfig.MCPServers
	for serverName, serverConfig := range projectConfig.MCPServers {
		if configMap, ok := serverConfig.(map[string]interface{}); ok {
			server := ServerConfig{
				Name:    serverName,
				Enabled: true,
			}

			// Extract configuration from map
			if url, ok := configMap["url"].(string); ok {
				server.URL = url
			}

			if authType, ok := configMap["authType"].(string); ok {
				server.AuthType = authType
			}

			if credentials, ok := configMap["credentials"].(map[string]interface{}); ok {
				server.Credentials = make(map[string]string)
				for key, value := range credentials {
					if strValue, ok := value.(string); ok {
						server.Credentials[key] = strValue
					}
				}
			}

			if enabled, ok := configMap["enabled"].(bool); ok {
				server.Enabled = enabled
			}

			sdm.setServerDefaults(&server)
			servers = append(servers, server)
		}
	}
	*/
	return servers, nil
}

// setServerDefaults sets default values for server configuration
func (sdm *ServerDiscoveryManager) setServerDefaults(server *ServerConfig) {
	if server.Timeout == 0 {
		server.Timeout = 30 * time.Second
	}

	if server.RetryAttempts == 0 {
		server.RetryAttempts = 3
	}

	if server.RetryDelay == 0 {
		server.RetryDelay = 1 * time.Second
	}

	if server.MaxConnections == 0 {
		server.MaxConnections = 10
	}

	if server.Metadata == nil {
		server.Metadata = make(map[string]interface{})
	}
}

// RegisterServer registers a new MCP server
func (sdm *ServerDiscoveryManager) RegisterServer(config ServerConfig) error {
	sdm.mu.Lock()
	defer sdm.mu.Unlock()

	// Set defaults
	sdm.setServerDefaults(&config)

	// Store in registry
	sdm.servers[config.Name] = config

	// TODO: Persist to project configuration using Viper
	// For now, just log that we would persist
	sdm.logger.Debug("Would persist server configuration", "server", config.Name)

	sdm.logger.Info("Registered MCP server", "server", config.Name, "url", config.URL)
	return nil
}

// UnregisterServer removes a server from the registry
func (sdm *ServerDiscoveryManager) UnregisterServer(serverName string) error {
	sdm.mu.Lock()
	defer sdm.mu.Unlock()

	// Remove from registry
	delete(sdm.servers, serverName)

	// TODO: Remove from project configuration using Viper
	// For now, just log that we would remove
	sdm.logger.Debug("Would remove server from configuration", "server", serverName)

	sdm.logger.Info("Unregistered MCP server", "server", serverName)
	return nil
}

// GetServerConfig retrieves a server configuration by name
func (sdm *ServerDiscoveryManager) GetServerConfig(serverName string) (*ServerConfig, error) {
	sdm.mu.RLock()
	defer sdm.mu.RUnlock()

	config, exists := sdm.servers[serverName]
	if !exists {
		return nil, fmt.Errorf("server not found: %s", serverName)
	}

	return &config, nil
}

// ListRegisteredServers returns all registered servers
func (sdm *ServerDiscoveryManager) ListRegisteredServers() []ServerConfig {
	sdm.mu.RLock()
	defer sdm.mu.RUnlock()

	servers := make([]ServerConfig, 0, len(sdm.servers))
	for _, server := range sdm.servers {
		servers = append(servers, server)
	}

	return servers
}

// loadServerConfigurations loads server configurations from various sources
func (sdm *ServerDiscoveryManager) loadServerConfigurations() error {
	ctx := context.Background()
	_, err := sdm.DiscoverServers(ctx)
	return err
}

// SaveServerConfiguration saves server configurations to file
func (sdm *ServerDiscoveryManager) SaveServerConfiguration(filePath string) error {
	sdm.mu.RLock()
	defer sdm.mu.RUnlock()

	servers := make([]ServerConfig, 0, len(sdm.servers))
	for _, server := range sdm.servers {
		servers = append(servers, server)
	}

	serverFile := struct {
		Version string         `json:"version"`
		Servers []ServerConfig `json:"servers"`
	}{
		Version: "1.0",
		Servers: servers,
	}

	data, err := json.MarshalIndent(serverFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal server configuration: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	sdm.logger.Info("Saved server configuration", "file", filePath, "count", len(servers))
	return nil
}

// CreateDefaultServerConfiguration creates a default server configuration file
func (sdm *ServerDiscoveryManager) CreateDefaultServerConfiguration(filePath string) error {
	defaultServers := []ServerConfig{
		{
			Name:              "ryan-builtin",
			URL:               "http://localhost:8000/mcp",
			Enabled:           true,
			Timeout:           30 * time.Second,
			RetryAttempts:     3,
			RetryDelay:        1 * time.Second,
			KeepAlive:         true,
			MaxConnections:    10,
			SupportedFeatures: []string{"tools", "resources", "prompts"},
			Metadata: map[string]interface{}{
				"description": "Built-in Ryan MCP server with core tools",
				"type":        "builtin",
			},
		},
		{
			Name:              "filesystem",
			URL:               "http://localhost:8001/mcp",
			Enabled:           false, // Disabled by default for security
			Timeout:           15 * time.Second,
			RetryAttempts:     2,
			RetryDelay:        500 * time.Millisecond,
			KeepAlive:         true,
			MaxConnections:    5,
			SupportedFeatures: []string{"tools"},
			Metadata: map[string]interface{}{
				"description": "Filesystem operations MCP server",
				"type":        "filesystem",
				"security":    "requires-permission",
			},
		},
	}

	serverFile := struct {
		Version     string         `json:"version"`
		Description string         `json:"description"`
		Servers     []ServerConfig `json:"servers"`
	}{
		Version:     "1.0",
		Description: "Ryan MCP Server Configuration",
		Servers:     defaultServers,
	}

	data, err := json.MarshalIndent(serverFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal default configuration: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	sdm.logger.Info("Created default server configuration", "file", filePath)
	return nil
}
