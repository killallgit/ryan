package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ConfigHierarchy implements Claude CLI's configuration hierarchy:
// Environment Variables → Project Configuration → Global Configuration → System Defaults
type ConfigHierarchy struct {
	contextManager *ContextManager
}

// NewConfigHierarchy creates a new configuration hierarchy resolver
func NewConfigHierarchy(contextManager *ContextManager) *ConfigHierarchy {
	return &ConfigHierarchy{
		contextManager: contextManager,
	}
}

// ResolveEffectiveConfig resolves the effective configuration by applying the hierarchy
func (ch *ConfigHierarchy) ResolveEffectiveConfig() (*Config, error) {
	// Load base configurations
	globalConfig, err := ch.contextManager.LoadGlobalConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	projectConfig, err := ch.contextManager.GetProjectConfig("")
	if err != nil {
		return nil, fmt.Errorf("failed to load project config: %w", err)
	}

	// Start with system defaults (lowest priority)
	effectiveConfig := ch.getSystemDefaults()

	// Apply global configuration overrides
	ch.applyGlobalConfigOverrides(effectiveConfig, globalConfig)

	// Apply project configuration overrides
	ch.applyProjectConfigOverrides(effectiveConfig, projectConfig)

	// Apply environment variable overrides (highest priority)
	ch.applyEnvironmentOverrides(effectiveConfig)

	return effectiveConfig, nil
}

// getSystemDefaults returns the system default configuration
func (ch *ConfigHierarchy) getSystemDefaults() *Config {
	return &Config{
		Logging: LoggingConfig{
			LogFile:  "./.ryan/system.log",
			Preserve: false,
			Level:    "info",
		},
		Context: ContextConfig{
			Directory:        "./.ryan/contexts",
			MaxFileSize:      "10MB",
			PersistLangChain: true,
		},
		ShowThinking: true,
		Streaming:    true,
		Ollama: OllamaConfig{
			URL:          "https://ollama.kitty-tetra.ts.net",
			Model:        "qwen3:latest",
			SystemPrompt: "",
			PollInterval: 10,
			Timeout:      90 * time.Second,
		},
		Tools: ToolsConfig{
			Enabled:        true,
			TruncateOutput: true,
			Models:         []string{},
			Bash: BashToolConfig{
				Enabled:         true,
				Timeout:         90 * time.Second,
				AllowedPaths:    []string{},
				SkipPermissions: false,
			},
			FileRead: FileReadConfig{
				Enabled:           true,
				MaxFileSize:       "10MB",
				AllowedExtensions: []string{},
			},
			Search: SearchConfig{
				Enabled: true,
				Timeout: 10 * time.Second,
			},
		},
		LangChain: LangChainConfig{
			Tools: LangChainToolsConfig{
				MaxIterations:       5,
				AutonomousReasoning: true,
				UseReActPattern:     true,
				VerboseLogging:      false,
			},
			Memory: LangChainMemoryConfig{
				Type:             "buffer",
				WindowSize:       10,
				MaxTokens:        4000,
				SummaryThreshold: 1000,
			},
			Prompts: LangChainPromptConfig{
				ContextInjection: true,
			},
		},
		VectorStore: VectorStoreConfig{
			Enabled:           true,
			Provider:          "chromem",
			PersistenceDir:    "./.ryan/vectorstore",
			EnablePersistence: true,
			Embedder: VectorStoreEmbedderConfig{
				Provider: "ollama",
				Model:    "nomic-embed-text",
				BaseURL:  "https://ollama.kitty-tetra.ts.net",
				APIKey:   "",
			},
			Collections: []VectorStoreCollectionConfig{},
			Indexer: VectorStoreIndexerConfig{
				ChunkSize:    1000,
				ChunkOverlap: 200,
				AutoIndex:    false,
			},
		},
		SelfConfigPath: "./.ryan/self.yaml",
	}
}

// applyGlobalConfigOverrides applies global configuration settings
func (ch *ConfigHierarchy) applyGlobalConfigOverrides(config *Config, globalConfig *GlobalConfig) {
	// Apply global configuration settings that map to Config struct
	if globalConfig.Verbose {
		config.Logging.Level = "debug"
	}

	// Apply theme and editor mode settings if they affect configuration
	// Note: Some global settings like theme might be handled by the UI layer

	// Apply environment variable overrides from global config
	for key, value := range globalConfig.Env {
		os.Setenv(key, value)
	}
}

// applyProjectConfigOverrides applies project-specific configuration settings
func (ch *ConfigHierarchy) applyProjectConfigOverrides(config *Config, projectConfig *ProjectConfig) {
	// Project-specific overrides can be added here
	// For now, most project config is used directly by the bridge

	// Example: If project has specific tool restrictions
	if len(projectConfig.AllowedTools) > 0 {
		// This would be handled by the bridge's IsToolAllowed method
		// but we could also modify the config here if needed
	}

	// Apply ignore patterns to relevant tools
	if len(projectConfig.IgnorePatterns) > 0 {
		// This could affect file read tool or search tool behavior
	}
}

// applyEnvironmentOverrides applies environment variable overrides (highest priority)
func (ch *ConfigHierarchy) applyEnvironmentOverrides(config *Config) {
	// Define environment variable mappings
	envMappings := map[string]func(string){
		// Logging configuration
		"RYAN_LOG_FILE":     func(v string) { config.Logging.LogFile = v },
		"RYAN_LOG_LEVEL":    func(v string) { config.Logging.Level = v },
		"RYAN_LOG_PRESERVE": func(v string) { config.Logging.Preserve = parseBool(v) },

		// Ollama configuration
		"RYAN_OLLAMA_URL":           func(v string) { config.Ollama.URL = v },
		"RYAN_OLLAMA_MODEL":         func(v string) { config.Ollama.Model = v },
		"RYAN_OLLAMA_SYSTEM_PROMPT": func(v string) { config.Ollama.SystemPrompt = v },
		"RYAN_OLLAMA_TIMEOUT": func(v string) {
			if d, err := time.ParseDuration(v); err == nil {
				config.Ollama.Timeout = d
			}
		},

		// General configuration
		"RYAN_SHOW_THINKING": func(v string) { config.ShowThinking = parseBool(v) },
		"RYAN_STREAMING":     func(v string) { config.Streaming = parseBool(v) },

		// Tools configuration
		"RYAN_TOOLS_ENABLED": func(v string) { config.Tools.Enabled = parseBool(v) },
		"RYAN_BASH_ENABLED":  func(v string) { config.Tools.Bash.Enabled = parseBool(v) },
		"RYAN_BASH_TIMEOUT": func(v string) {
			if d, err := time.ParseDuration(v); err == nil {
				config.Tools.Bash.Timeout = d
			}
		},

		// Vector store configuration
		"RYAN_VECTORSTORE_ENABLED":         func(v string) { config.VectorStore.Enabled = parseBool(v) },
		"RYAN_VECTORSTORE_PROVIDER":        func(v string) { config.VectorStore.Provider = v },
		"RYAN_VECTORSTORE_PERSISTENCE_DIR": func(v string) { config.VectorStore.PersistenceDir = v },
		"RYAN_EMBEDDER_PROVIDER":           func(v string) { config.VectorStore.Embedder.Provider = v },
		"RYAN_EMBEDDER_MODEL":              func(v string) { config.VectorStore.Embedder.Model = v },
		"RYAN_EMBEDDER_BASE_URL":           func(v string) { config.VectorStore.Embedder.BaseURL = v },
		"RYAN_EMBEDDER_API_KEY":            func(v string) { config.VectorStore.Embedder.APIKey = v },

		// Configuration directory override
		"RYAN_CONFIG_DIR": func(v string) {
			// This affects where configurations are stored, handled by GetGlobalConfigPath
		},
	}

	// Apply environment variable overrides using Viper
	for envVar, setter := range envMappings {
		if value := viper.GetString(envVar); value != "" {
			setter(value)
		}
	}

	// Special handling for complex environment variables
	ch.applyComplexEnvironmentOverrides(config)
}

// applyComplexEnvironmentOverrides handles more complex environment variable patterns
func (ch *ConfigHierarchy) applyComplexEnvironmentOverrides(config *Config) {
	// Handle RYAN_TOOLS_MODELS (comma-separated list)
	if modelsEnv := viper.GetString("RYAN_TOOLS_MODELS"); modelsEnv != "" {
		config.Tools.Models = strings.Split(modelsEnv, ",")
		// Trim whitespace
		for i, model := range config.Tools.Models {
			config.Tools.Models[i] = strings.TrimSpace(model)
		}
	}

	// Handle RYAN_BASH_ALLOWED_PATHS (comma-separated list)
	if pathsEnv := viper.GetString("RYAN_BASH_ALLOWED_PATHS"); pathsEnv != "" {
		config.Tools.Bash.AllowedPaths = strings.Split(pathsEnv, ",")
		for i, path := range config.Tools.Bash.AllowedPaths {
			config.Tools.Bash.AllowedPaths[i] = strings.TrimSpace(path)
		}
	}

	// Handle RYAN_FILE_READ_ALLOWED_EXTENSIONS (comma-separated list)
	if extEnv := viper.GetString("RYAN_FILE_READ_ALLOWED_EXTENSIONS"); extEnv != "" {
		config.Tools.FileRead.AllowedExtensions = strings.Split(extEnv, ",")
		for i, ext := range config.Tools.FileRead.AllowedExtensions {
			config.Tools.FileRead.AllowedExtensions[i] = strings.TrimSpace(ext)
		}
	}

	// Handle numeric environment variables
	if pollIntervalEnv := viper.GetString("RYAN_OLLAMA_POLL_INTERVAL"); pollIntervalEnv != "" {
		if interval, err := strconv.Atoi(pollIntervalEnv); err == nil {
			config.Ollama.PollInterval = interval
		}
	}

	if maxIterEnv := viper.GetString("RYAN_LANGCHAIN_MAX_ITERATIONS"); maxIterEnv != "" {
		if maxIter, err := strconv.Atoi(maxIterEnv); err == nil {
			config.LangChain.Tools.MaxIterations = maxIter
		}
	}
}

// parseBool parses a string to boolean with common true/false values
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on" || s == "enabled"
}

// GetConfigValue retrieves a configuration value using the hierarchy
func (ch *ConfigHierarchy) GetConfigValue(keyPath string) (interface{}, error) {
	config, err := ch.ResolveEffectiveConfig()
	if err != nil {
		return nil, err
	}

	return ch.getValueByPath(config, keyPath)
}

// getValueByPath extracts a value from the config struct using dot notation
func (ch *ConfigHierarchy) getValueByPath(config *Config, keyPath string) (interface{}, error) {
	parts := strings.Split(keyPath, ".")

	// Use reflection to navigate the struct
	value := reflect.ValueOf(config).Elem()

	for _, part := range parts {
		// Handle field names case-insensitively
		field := ch.findFieldByName(value, part)
		if !field.IsValid() {
			return nil, fmt.Errorf("field not found: %s", part)
		}
		value = field
	}

	return value.Interface(), nil
}

// findFieldByName finds a struct field by name (case-insensitive)
func (ch *ConfigHierarchy) findFieldByName(structValue reflect.Value, fieldName string) reflect.Value {
	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		field := structType.Field(i)
		if strings.EqualFold(field.Name, fieldName) {
			return structValue.Field(i)
		}
	}

	return reflect.Value{}
}

// SetConfigValue sets a configuration value and persists it appropriately
func (ch *ConfigHierarchy) SetConfigValue(keyPath string, value interface{}, scope string) error {
	switch scope {
	case "global":
		return ch.setGlobalConfigValue(keyPath, value)
	case "project":
		return ch.setProjectConfigValue(keyPath, value)
	default:
		return fmt.Errorf("invalid scope: %s (must be 'global' or 'project')", scope)
	}
}

// setGlobalConfigValue sets a value in the global configuration
func (ch *ConfigHierarchy) setGlobalConfigValue(keyPath string, value interface{}) error {
	globalConfig, err := ch.contextManager.LoadGlobalConfig()
	if err != nil {
		return err
	}

	// For now, handle common global settings
	switch keyPath {
	case "verbose":
		if b, ok := value.(bool); ok {
			globalConfig.Verbose = b
		}
	case "theme":
		if s, ok := value.(string); ok {
			globalConfig.Theme = s
		}
	case "editorMode":
		if s, ok := value.(string); ok {
			globalConfig.EditorMode = s
		}
	default:
		return fmt.Errorf("global setting not supported: %s", keyPath)
	}

	// Save global configuration
	return ch.contextManager.saveGlobalConfigAtomic(globalConfig)
}

// setProjectConfigValue sets a value in the project configuration
func (ch *ConfigHierarchy) setProjectConfigValue(keyPath string, value interface{}) error {
	projectConfig, err := ch.contextManager.GetProjectConfig("")
	if err != nil {
		return err
	}

	projectRoot, err := ch.contextManager.GetProjectRoot()
	if err != nil {
		return err
	}

	// Handle project-specific settings
	switch keyPath {
	case "trustDialogAccepted":
		if b, ok := value.(bool); ok {
			projectConfig.HasTrustDialogAccepted = b
		}
	default:
		return fmt.Errorf("project setting not supported: %s", keyPath)
	}

	// Save project configuration
	return ch.contextManager.SaveProjectConfig(projectRoot, projectConfig)
}
