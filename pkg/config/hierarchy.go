package config

import (
	"fmt"
	"reflect"
	"strings"

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

// getSystemDefaults returns the system default configuration from Viper
func (ch *ConfigHierarchy) getSystemDefaults() *Config {
	// Create a new viper instance with only defaults
	defaultsViper := viper.New()

	// Set all defaults - reuse the existing setDefaults function
	// but call it on the new viper instance
	for key, value := range getAllDefaults() {
		defaultsViper.SetDefault(key, value)
	}

	// Create config and unmarshal from defaults
	config := &Config{}
	if err := defaultsViper.Unmarshal(config); err != nil {
		// If unmarshal fails, build config manually from defaults
		return buildConfigFromViper(defaultsViper)
	}

	return config
}

// buildConfigFromViper builds a config from a viper instance
func buildConfigFromViper(v *viper.Viper) *Config {
	return &Config{
		Logging: LoggingConfig{
			LogFile:  v.GetString("logging.log_file"),
			Preserve: v.GetBool("logging.preserve"),
			Level:    v.GetString("logging.level"),
		},
		Context: ContextConfig{
			Directory:        v.GetString("context.directory"),
			MaxFileSize:      v.GetString("context.max_file_size"),
			PersistLangChain: v.GetBool("context.persist_langchain"),
		},
		ShowThinking: v.GetBool("show_thinking"),
		Streaming:    v.GetBool("streaming"),
		Provider:     v.GetString("provider"),
		Ollama: OllamaConfig{
			URL:          v.GetString("ollama.url"),
			Model:        v.GetString("ollama.model"),
			SystemPrompt: v.GetString("ollama.system_prompt"),
			PollInterval: v.GetInt("ollama.poll_interval"),
			Timeout:      v.GetDuration("ollama.timeout"),
		},
		OpenAI: OpenAIConfig{
			APIKey:       v.GetString("openai.api_key"),
			Model:        v.GetString("openai.model"),
			SystemPrompt: v.GetString("openai.system_prompt"),
			Timeout:      v.GetDuration("openai.timeout"),
			BaseURL:      v.GetString("openai.base_url"),
		},
		Tools: ToolsConfig{
			Enabled:        v.GetBool("tools.enabled"),
			TruncateOutput: v.GetBool("tools.truncate_output"),
			Models:         v.GetStringSlice("tools.models"),
			Bash: BashToolConfig{
				Enabled:         v.GetBool("tools.bash.enabled"),
				Timeout:         v.GetDuration("tools.bash.timeout"),
				AllowedPaths:    v.GetStringSlice("tools.bash.allowed_paths"),
				SkipPermissions: v.GetBool("tools.bash.skip_permissions"),
			},
			FileRead: FileReadConfig{
				Enabled:           v.GetBool("tools.file_read.enabled"),
				MaxFileSize:       v.GetString("tools.file_read.max_file_size"),
				AllowedExtensions: v.GetStringSlice("tools.file_read.allowed_extensions"),
			},
			Search: SearchConfig{
				Enabled: v.GetBool("tools.search.enabled"),
				Timeout: v.GetDuration("tools.search.timeout"),
			},
		},
		LangChain: LangChainConfig{
			Tools: LangChainToolsConfig{
				MaxIterations:       v.GetInt("langchain.tools.max_iterations"),
				AutonomousReasoning: v.GetBool("langchain.tools.autonomous_reasoning"),
				UseReActPattern:     v.GetBool("langchain.tools.use_react_pattern"),
				VerboseLogging:      v.GetBool("langchain.tools.verbose_logging"),
			},
			Memory: LangChainMemoryConfig{
				Type:             v.GetString("langchain.memory.type"),
				WindowSize:       v.GetInt("langchain.memory.window_size"),
				MaxTokens:        v.GetInt("langchain.memory.max_tokens"),
				SummaryThreshold: v.GetInt("langchain.memory.summary_threshold"),
			},
			Prompts: LangChainPromptConfig{
				ContextInjection: v.GetBool("langchain.prompts.context_injection"),
			},
		},
		VectorStore: VectorStoreConfig{
			Enabled:           v.GetBool("vectorstore.enabled"),
			Provider:          v.GetString("vectorstore.provider"),
			PersistenceDir:    v.GetString("vectorstore.persistence_dir"),
			EnablePersistence: v.GetBool("vectorstore.enable_persistence"),
			Embedder: VectorStoreEmbedderConfig{
				Provider: v.GetString("vectorstore.embedder.provider"),
				Model:    v.GetString("vectorstore.embedder.model"),
				BaseURL:  v.GetString("vectorstore.embedder.base_url"),
				APIKey:   v.GetString("vectorstore.embedder.api_key"),
			},
			Collections: []VectorStoreCollectionConfig{},
			Indexer: VectorStoreIndexerConfig{
				ChunkSize:    v.GetInt("vectorstore.indexer.chunk_size"),
				ChunkOverlap: v.GetInt("vectorstore.indexer.chunk_overlap"),
				AutoIndex:    v.GetBool("vectorstore.indexer.auto_index"),
			},
		},
		SelfConfigPath: v.GetString("self_config_path"),
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

	// Set environment variables from global config into Viper
	// These will be picked up by Viper's AutomaticEnv
	for key, value := range globalConfig.Env {
		// Convert key to Viper format and set it
		viperKey := strings.ToLower(strings.ReplaceAll(key, "_", "."))
		if strings.HasPrefix(key, "RYAN_") {
			viperKey = strings.TrimPrefix(viperKey, "ryan.")
		}
		viper.Set(viperKey, value)
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
	// Viper's AutomaticEnv automatically picks up all environment variables
	// when they match the config keys (with RYAN_ prefix and dots replaced by underscores)
	// So RYAN_LOGGING_LEVEL automatically maps to logging.level
	// We just need to rebuild the config from the current Viper state
	*config = *buildConfigFromViper(viper.GetViper())
}

// No longer needed - Viper handles boolean parsing

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
