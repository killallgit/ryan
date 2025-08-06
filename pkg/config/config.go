package config

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// LangChainConfig holds LangChain-specific configuration
type LangChainConfig struct {
	Tools   LangChainToolsConfig  `mapstructure:"tools"`
	Memory  LangChainMemoryConfig `mapstructure:"memory"`
	Prompts LangChainPromptConfig `mapstructure:"prompts"`
}

// LangChainToolsConfig holds tool integration configuration
type LangChainToolsConfig struct {
	MaxIterations       int  `mapstructure:"max_iterations"`
	AutonomousReasoning bool `mapstructure:"autonomous_reasoning"`
	UseReActPattern     bool `mapstructure:"use_react_pattern"`
	VerboseLogging      bool `mapstructure:"verbose_logging"`
}

// LangChainMemoryConfig holds memory configuration
type LangChainMemoryConfig struct {
	Type             string `mapstructure:"type"`
	WindowSize       int    `mapstructure:"window_size"`
	MaxTokens        int    `mapstructure:"max_tokens"`
	SummaryThreshold int    `mapstructure:"summary_threshold"`
}

// LangChainPromptConfig holds prompt template configuration
type LangChainPromptConfig struct {
	ContextInjection bool `mapstructure:"context_injection"`
}

// AgentsConfig holds agent-related configuration
type AgentsConfig struct {
	Preferred     string   `mapstructure:"preferred"`
	FallbackChain []string `mapstructure:"fallback_chain"`
	AutoSelect    bool     `mapstructure:"auto_select"`
	ShowSelection bool     `mapstructure:"show_selection"`
}

// Config represents the application configuration
type Config struct {
	Logging        LoggingConfig     `mapstructure:"logging"`
	Context        ContextConfig     `mapstructure:"context"`
	ShowThinking   bool              `mapstructure:"show_thinking"`
	Streaming      bool              `mapstructure:"streaming"`
	Provider       string            `mapstructure:"provider"` // Selected provider: ollama, openai, etc.
	Ollama         OllamaConfig      `mapstructure:"ollama"`
	OpenAI         OpenAIConfig      `mapstructure:"openai"` // For future implementation
	Tools          ToolsConfig       `mapstructure:"tools"`
	LangChain      LangChainConfig   `mapstructure:"langchain"`
	VectorStore    VectorStoreConfig `mapstructure:"vectorstore"`
	SelfConfigPath string            `mapstructure:"self_config_path"`
	Agents         AgentsConfig      `mapstructure:"agents"`
}

// ContextConfig holds context persistence configuration
type ContextConfig struct {
	Directory        string `mapstructure:"directory"`
	MaxFileSize      string `mapstructure:"max_file_size"`
	PersistLangChain bool   `mapstructure:"persist_langchain"`
}

// LoggingConfig holds logging-related configuration
type LoggingConfig struct {
	LogFile  string `mapstructure:"log_file"`
	Preserve bool   `mapstructure:"preserve"`
	Level    string `mapstructure:"level"`
}

// OllamaConfig holds Ollama-specific configuration
type OllamaConfig struct {
	URL          string        `mapstructure:"url"`
	Model        string        `mapstructure:"model"`
	SystemPrompt string        `mapstructure:"system_prompt"`
	PollInterval int           `mapstructure:"poll_interval"`
	Timeout      time.Duration `mapstructure:"timeout"`
	TimeoutStr   string        `mapstructure:"timeout"` // For parsing string duration
}

// OpenAIConfig holds OpenAI-specific configuration (for future implementation)
type OpenAIConfig struct {
	APIKey       string        `mapstructure:"api_key"`
	Model        string        `mapstructure:"model"`
	SystemPrompt string        `mapstructure:"system_prompt"`
	Timeout      time.Duration `mapstructure:"timeout"`
	TimeoutStr   string        `mapstructure:"timeout"`  // For parsing string duration
	BaseURL      string        `mapstructure:"base_url"` // For Azure or custom endpoints
}

// ToolsConfig holds tool-related configuration
type ToolsConfig struct {
	Enabled        bool           `mapstructure:"enabled"`
	TruncateOutput bool           `mapstructure:"truncate_output"`
	Models         []string       `mapstructure:"models"`
	Bash           BashToolConfig `mapstructure:"bash"`
	FileRead       FileReadConfig `mapstructure:"file_read"`
	Search         SearchConfig   `mapstructure:"search"`
}

// BashToolConfig holds bash tool configuration
type BashToolConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Timeout         time.Duration `mapstructure:"timeout"`
	TimeoutStr      string        `mapstructure:"timeout"` // For parsing string duration
	AllowedPaths    []string      `mapstructure:"allowed_paths"`
	SkipPermissions bool          `mapstructure:"skip_permissions"`
}

// FileReadConfig holds file read tool configuration
type FileReadConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	MaxFileSize       string   `mapstructure:"max_file_size"`
	AllowedExtensions []string `mapstructure:"allowed_extensions"`
}

// SearchConfig holds search tool configuration
type SearchConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	Timeout    time.Duration `mapstructure:"timeout"`
	TimeoutStr string        `mapstructure:"timeout"` // For parsing string duration
}

// VectorStoreConfig holds vector store configuration
type VectorStoreConfig struct {
	Enabled           bool                          `mapstructure:"enabled"`
	Provider          string                        `mapstructure:"provider"`
	PersistenceDir    string                        `mapstructure:"persistence_dir"`
	EnablePersistence bool                          `mapstructure:"enable_persistence"`
	Embedder          VectorStoreEmbedderConfig     `mapstructure:"embedder"`
	Collections       []VectorStoreCollectionConfig `mapstructure:"collections"`
	Indexer           VectorStoreIndexerConfig      `mapstructure:"indexer"`
}

// VectorStoreEmbedderConfig holds embedder configuration
type VectorStoreEmbedderConfig struct {
	Provider string `mapstructure:"provider"` // ollama, openai, mock
	Model    string `mapstructure:"model"`
	BaseURL  string `mapstructure:"base_url"`
	APIKey   string `mapstructure:"api_key"`
}

// VectorStoreCollectionConfig holds collection configuration
type VectorStoreCollectionConfig struct {
	Name     string         `mapstructure:"name"`
	Metadata map[string]any `mapstructure:"metadata"`
}

// VectorStoreIndexerConfig holds document indexer configuration
type VectorStoreIndexerConfig struct {
	ChunkSize    int  `mapstructure:"chunk_size"`
	ChunkOverlap int  `mapstructure:"chunk_overlap"`
	AutoIndex    bool `mapstructure:"auto_index"`
}

// SelfConfig represents the self.yaml configuration
type SelfConfig struct {
	Role   string      `mapstructure:"role"`
	Traits []SelfTrait `mapstructure:"traits"`
}

// SelfTrait represents a trait in the self.yaml configuration
type SelfTrait struct {
	Name         string `mapstructure:"name"`
	SystemPrompt string `mapstructure:"system_prompt"`
}

var (
	// Global config instance
	cfg     *Config
	selfCfg *SelfConfig
)

// Get returns the global config instance
func Get() *Config {
	if cfg == nil {
		panic("config not initialized")
	}
	return cfg
}

// GetSelf returns the global self config instance
func GetSelf() *SelfConfig {
	return selfCfg // Allow nil return if self config is optional
}

// Load loads configuration from file and environment
func Load(cfgFile string) (*Config, error) {
	// Set defaults first
	setDefaults()

	// Configure viper
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Set config search paths
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome == "" {
			xdgConfigHome = filepath.Join(home, ".config")
		}
		ryanCfgHome := filepath.Join(xdgConfigHome, ".ryan")

		viper.AddConfigPath("./.ryan")   // Check project directory first
		viper.AddConfigPath(ryanCfgHome) // Then check XDG config location
		viper.SetConfigType("yaml")
		viper.SetConfigName("settings.yaml")
	}

	// Enable environment variable support
	viper.AutomaticEnv()

	// Bind specific environment variables to Viper keys for explicit mapping
	bindEnvironmentVariables()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		// Log config file usage instead of printing to stderr
		// This prevents cluttering TUI output
	}

	// Create config instance
	cfg = &Config{}

	// Unmarshal into struct
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Post-process durations (viper doesn't handle time.Duration directly)
	if err := processDurations(cfg); err != nil {
		return nil, fmt.Errorf("failed to process durations: %w", err)
	}

	// Load self config if path is specified
	if cfg.SelfConfigPath != "" {
		if err := loadSelfConfig(cfg.SelfConfigPath); err != nil {
			// Don't fail if self config can't be loaded - it's optional
			// Log the error but continue
		}
	}

	return cfg, nil
}

// loadSelfConfig loads the self.yaml configuration using a separate viper instance
func loadSelfConfig(selfConfigPath string) error {
	// Create a separate viper instance for self config
	selfViper := viper.New()
	selfViper.SetConfigFile(selfConfigPath)
	selfViper.SetConfigType("yaml")

	// Try to read the self config file
	if err := selfViper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read self config file %s: %w", selfConfigPath, err)
	}

	// Create self config instance
	selfCfg = &SelfConfig{}

	// Unmarshal into struct
	if err := selfViper.Unmarshal(selfCfg); err != nil {
		return fmt.Errorf("failed to unmarshal self config: %w", err)
	}

	return nil
}

// setDefaults sets all default configuration values
func setDefaults() {
	// Provider defaults
	viper.SetDefault("provider", "ollama") // Default to ollama provider

	// Ollama defaults
	viper.SetDefault("ollama.url", "https://ollama.kitty-tetra.ts.net")
	viper.SetDefault("ollama.model", "qwen3:latest")
	viper.SetDefault("ollama.system_prompt", "")
	viper.SetDefault("ollama.timeout", "90s")
	viper.SetDefault("ollama.poll_interval", 10)

	// OpenAI defaults (for future implementation)
	viper.SetDefault("openai.api_key", "")
	viper.SetDefault("openai.model", "gpt-4-turbo-preview")
	viper.SetDefault("openai.system_prompt", "")
	viper.SetDefault("openai.timeout", "60s")
	viper.SetDefault("openai.base_url", "")

	// General defaults
	viper.SetDefault("show_thinking", true)
	viper.SetDefault("streaming", true)

	// Logging defaults
	viper.SetDefault("logging.log_file", "./.ryan/system.log")
	viper.SetDefault("logging.preserve", false)
	viper.SetDefault("logging.level", "info")

	// Context defaults
	viper.SetDefault("context.directory", "./.ryan/contexts")
	viper.SetDefault("context.max_file_size", "10MB")
	viper.SetDefault("context.persist_langchain", true)

	// Tools defaults
	viper.SetDefault("tools.enabled", true)
	viper.SetDefault("tools.truncate_output", true)
	viper.SetDefault("tools.bash.enabled", true)
	viper.SetDefault("tools.bash.timeout", "90s")
	viper.SetDefault("tools.file_read.enabled", true)
	viper.SetDefault("tools.search.enabled", true)
	viper.SetDefault("tools.search.timeout", "10s")

	// LangChain defaults
	viper.SetDefault("langchain.tools.max_iterations", 5)
	viper.SetDefault("langchain.tools.autonomous_reasoning", true)
	viper.SetDefault("langchain.tools.use_react_pattern", true)
	viper.SetDefault("langchain.tools.verbose_logging", false)
	viper.SetDefault("langchain.memory.type", "buffer")
	viper.SetDefault("langchain.memory.window_size", 10)
	viper.SetDefault("langchain.memory.max_tokens", 4000)
	viper.SetDefault("langchain.memory.summary_threshold", 1000)
	viper.SetDefault("langchain.prompts.context_injection", true)

	// Vector store defaults
	viper.SetDefault("vectorstore.enabled", true)
	viper.SetDefault("vectorstore.provider", "chromem")
	viper.SetDefault("vectorstore.persistence_dir", "./.ryan/vectorstore")
	viper.SetDefault("vectorstore.enable_persistence", true)
	viper.SetDefault("vectorstore.embedder.provider", "ollama")
	viper.SetDefault("vectorstore.embedder.model", "nomic-embed-text")
	viper.SetDefault("vectorstore.embedder.base_url", "https://ollama.kitty-tetra.ts.net")
	viper.SetDefault("vectorstore.embedder.api_key", "")
	viper.SetDefault("vectorstore.indexer.chunk_size", 1000)
	viper.SetDefault("vectorstore.indexer.chunk_overlap", 200)
	viper.SetDefault("vectorstore.indexer.auto_index", false)

	// Self config path default
	viper.SetDefault("self_config_path", "./.ryan/self.yaml")
}

// bindEnvironmentVariables binds specific environment variables to Viper keys
func bindEnvironmentVariables() {
	// OpenAI API key for vector store embeddings
	viper.BindEnv("OPENAI_API_KEY")
	viper.BindEnv("vectorstore.embedder.api_key", "OPENAI_API_KEY")

	// MCP server configuration
	viper.BindEnv("RYAN_MCP_SERVERS")
	viper.BindEnv("mcp.servers", "RYAN_MCP_SERVERS")

	// Configuration directory override
	viper.BindEnv("RYAN_CONFIG_DIR")
	viper.BindEnv("config.directory", "RYAN_CONFIG_DIR")

	// All RYAN_ prefixed environment variables used in hierarchy
	viper.BindEnv("RYAN_LOG_FILE")
	viper.BindEnv("RYAN_LOG_LEVEL")
	viper.BindEnv("RYAN_LOG_PRESERVE")
	viper.BindEnv("RYAN_OLLAMA_URL")
	viper.BindEnv("RYAN_OLLAMA_MODEL")
	viper.BindEnv("RYAN_OLLAMA_SYSTEM_PROMPT")
	viper.BindEnv("RYAN_OLLAMA_TIMEOUT")
	viper.BindEnv("RYAN_OLLAMA_POLL_INTERVAL")
	viper.BindEnv("RYAN_SHOW_THINKING")
	viper.BindEnv("RYAN_STREAMING")
	viper.BindEnv("RYAN_TOOLS_ENABLED")
	viper.BindEnv("RYAN_TOOLS_MODELS")
	viper.BindEnv("RYAN_BASH_ENABLED")
	viper.BindEnv("RYAN_BASH_TIMEOUT")
	viper.BindEnv("RYAN_BASH_ALLOWED_PATHS")
	viper.BindEnv("RYAN_FILE_READ_ALLOWED_EXTENSIONS")
	viper.BindEnv("RYAN_VECTORSTORE_ENABLED")
	viper.BindEnv("RYAN_VECTORSTORE_PROVIDER")
	viper.BindEnv("RYAN_VECTORSTORE_PERSISTENCE_DIR")
	viper.BindEnv("RYAN_EMBEDDER_PROVIDER")
	viper.BindEnv("RYAN_EMBEDDER_MODEL")
	viper.BindEnv("RYAN_EMBEDDER_BASE_URL")
	viper.BindEnv("RYAN_EMBEDDER_API_KEY")
	viper.BindEnv("RYAN_LANGCHAIN_MAX_ITERATIONS")
}

// processDurations converts string durations to time.Duration
func processDurations(cfg *Config) error {
	// Process Ollama timeout
	if cfg.Ollama.TimeoutStr != "" {
		d, err := time.ParseDuration(cfg.Ollama.TimeoutStr)
		if err != nil {
			return fmt.Errorf("invalid ollama.timeout: %w", err)
		}
		cfg.Ollama.Timeout = d
	} else if cfg.Ollama.Timeout == 0 {
		// Use default if not set
		cfg.Ollama.Timeout = 90 * time.Second
	}

	// Process Bash timeout
	if cfg.Tools.Bash.TimeoutStr != "" {
		d, err := time.ParseDuration(cfg.Tools.Bash.TimeoutStr)
		if err != nil {
			return fmt.Errorf("invalid tools.bash.timeout: %w", err)
		}
		cfg.Tools.Bash.Timeout = d
	} else if cfg.Tools.Bash.Timeout == 0 {
		// Use default if not set
		cfg.Tools.Bash.Timeout = 90 * time.Second
	}

	// Process Search timeout
	if cfg.Tools.Search.TimeoutStr != "" {
		d, err := time.ParseDuration(cfg.Tools.Search.TimeoutStr)
		if err != nil {
			return fmt.Errorf("invalid tools.search.timeout: %w", err)
		}
		cfg.Tools.Search.Timeout = d
	} else if cfg.Tools.Search.Timeout == 0 {
		// Use default if not set
		cfg.Tools.Search.Timeout = 10 * time.Second
	}

	return nil
}

// GetConfigFileUsed returns the path to the config file being used
func GetConfigFileUsed() string {
	return viper.ConfigFileUsed()
}

// InitializeDefaults creates a default .ryan/settings.yaml file if it doesn't exist
func InitializeDefaults() error {
	// Check if .ryan/settings.yaml already exists
	if _, err := os.Stat(".ryan/settings.yaml"); err == nil {
		// File exists, nothing to do
		return nil
	}

	// Prompt user to create settings file
	if !promptUserForSettingsCreation() {
		return nil // User declined, continue without creating file
	}

	// Create .ryan directory structure if it doesn't exist
	if err := os.MkdirAll(".ryan/logs", 0755); err != nil {
		return fmt.Errorf("failed to create .ryan/logs directory: %w", err)
	}
	if err := os.MkdirAll(".ryan/contexts", 0755); err != nil {
		return fmt.Errorf("failed to create .ryan/contexts directory: %w", err)
	}
	if err := os.MkdirAll(".ryan/vectorstore", 0755); err != nil {
		return fmt.Errorf("failed to create .ryan/vectorstore directory: %w", err)
	}

	// Create a new viper instance for writing defaults
	v := viper.New()
	v.SetConfigType("yaml")

	// Set all the defaults
	v.SetDefault("logging.log_file", "./.ryan/system.log")
	v.SetDefault("logging.preserve", false)
	v.SetDefault("logging.level", "info")

	v.SetDefault("context.directory", "./.ryan/contexts")
	v.SetDefault("context.max_file_size", "10MB")
	v.SetDefault("context.persist_langchain", true)

	v.SetDefault("show_thinking", true)
	v.SetDefault("streaming", true)

	v.SetDefault("provider", "ollama")

	v.SetDefault("ollama.url", "https://ollama.kitty-tetra.ts.net")
	v.SetDefault("ollama.model", "qwen3:latest")
	v.SetDefault("ollama.system_prompt", "")
	v.SetDefault("ollama.timeout", "90s")
	v.SetDefault("ollama.poll_interval", 10)

	v.SetDefault("openai.api_key", "")
	v.SetDefault("openai.model", "gpt-4-turbo-preview")
	v.SetDefault("openai.system_prompt", "")
	v.SetDefault("openai.timeout", "60s")
	v.SetDefault("openai.base_url", "")

	v.SetDefault("tools.enabled", true)
	v.SetDefault("tools.truncate_output", true)
	v.SetDefault("tools.bash.enabled", true)
	v.SetDefault("tools.bash.timeout", "90s")
	v.SetDefault("tools.file_read.enabled", true)
	v.SetDefault("tools.search.enabled", true)
	v.SetDefault("tools.search.timeout", "10s")

	// LangChain autonomous agent defaults
	v.SetDefault("langchain.tools.max_iterations", 5)
	v.SetDefault("langchain.tools.autonomous_reasoning", true)
	v.SetDefault("langchain.tools.use_react_pattern", true)
	v.SetDefault("langchain.tools.verbose_logging", false)
	v.SetDefault("langchain.memory.type", "buffer")
	v.SetDefault("langchain.memory.window_size", 10)
	v.SetDefault("langchain.memory.max_tokens", 4000)
	v.SetDefault("langchain.memory.summary_threshold", 1000)
	v.SetDefault("langchain.prompts.context_injection", true)

	// Vector store defaults
	v.SetDefault("vectorstore.enabled", true)
	v.SetDefault("vectorstore.provider", "chromem")
	v.SetDefault("vectorstore.persistence_dir", "./.ryan/vectorstore")
	v.SetDefault("vectorstore.enable_persistence", true)
	v.SetDefault("vectorstore.embedder.provider", "ollama")
	v.SetDefault("vectorstore.embedder.model", "nomic-embed-text")
	v.SetDefault("vectorstore.embedder.base_url", "https://ollama.kitty-tetra.ts.net")
	v.SetDefault("vectorstore.embedder.api_key", "")
	v.SetDefault("vectorstore.indexer.chunk_size", 1000)
	v.SetDefault("vectorstore.indexer.chunk_overlap", 200)
	v.SetDefault("vectorstore.indexer.auto_index", false)

	// Self config path default
	v.SetDefault("self_config_path", "./.ryan/self.yaml")

	// Write the default configuration to .ryan/settings.yaml
	if err := v.SafeWriteConfigAs(".ryan/settings.yaml"); err != nil {
		return fmt.Errorf("failed to write default configuration: %w", err)
	}

	fmt.Printf("Created default settings file at .ryan/settings.yaml\n")
	return nil
}

// promptUserForSettingsCreation prompts the user to create a settings file
func promptUserForSettingsCreation() bool {
	// Skip interactive prompt during tests
	if isTestEnvironment() {
		return false
	}

	fmt.Print("No .ryan/settings.yaml file found. Would you like to create one with default settings? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// isTestEnvironment checks if we're running in a test environment
func isTestEnvironment() bool {
	// Check if we're running under go test
	if flag.CommandLine.Lookup("test.v") != nil {
		return true
	}

	// Check for common test environment variables
	if os.Getenv("GO_TEST") == "1" || os.Getenv("TESTING") == "1" {
		return true
	}

	// Check if any test flags are present in the command line
	for _, arg := range os.Args {
		if strings.Contains(arg, "test") && strings.Contains(arg, ".test") {
			return true
		}
	}

	return false
}

// GetActiveProviderModel returns the model name for the currently active provider
func (c *Config) GetActiveProviderModel() string {
	switch c.Provider {
	case "openai":
		return c.OpenAI.Model
	case "ollama", "":
		// Default to ollama if provider not specified
		return c.Ollama.Model
	default:
		return c.Ollama.Model
	}
}

// GetActiveProviderURL returns the URL for the currently active provider
func (c *Config) GetActiveProviderURL() string {
	switch c.Provider {
	case "openai":
		if c.OpenAI.BaseURL != "" {
			return c.OpenAI.BaseURL
		}
		return "https://api.openai.com/v1"
	case "ollama", "":
		// Default to ollama if provider not specified
		return c.Ollama.URL
	default:
		return c.Ollama.URL
	}
}

// GetActiveProviderSystemPrompt returns the system prompt for the currently active provider
func (c *Config) GetActiveProviderSystemPrompt() string {
	switch c.Provider {
	case "openai":
		return c.OpenAI.SystemPrompt
	case "ollama", "":
		// Default to ollama if provider not specified
		return c.Ollama.SystemPrompt
	default:
		return c.Ollama.SystemPrompt
	}
}

// GetActiveProvider returns the currently active provider name
func (c *Config) GetActiveProvider() string {
	if c.Provider == "" {
		return "ollama" // Default provider
	}
	return c.Provider
}
