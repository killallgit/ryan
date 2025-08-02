package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Logging      LoggingConfig `mapstructure:"logging"`
	ShowThinking bool          `mapstructure:"show_thinking"`
	Streaming    bool          `mapstructure:"streaming"`
	Ollama       OllamaConfig  `mapstructure:"ollama"`
	Tools        ToolsConfig   `mapstructure:"tools"`
}

// LoggingConfig holds logging-related configuration
type LoggingConfig struct {
	File     string `mapstructure:"file"`
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

// ToolsConfig holds tool-related configuration
type ToolsConfig struct {
	Enabled         bool             `mapstructure:"enabled"`
	TruncateOutput  bool             `mapstructure:"truncate_output"`
	Models          []string         `mapstructure:"models"`
	Bash            BashToolConfig   `mapstructure:"bash"`
	FileRead        FileReadConfig   `mapstructure:"file_read"`
	Search          SearchConfig     `mapstructure:"search"`
}

// BashToolConfig holds bash tool configuration
type BashToolConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	Timeout      time.Duration `mapstructure:"timeout"`
	TimeoutStr   string        `mapstructure:"timeout"` // For parsing string duration
	AllowedPaths []string      `mapstructure:"allowed_paths"`
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

var (
	// Global config instance
	cfg *Config
)

// Get returns the global config instance
func Get() *Config {
	if cfg == nil {
		panic("config not initialized")
	}
	return cfg
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

	// Read config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
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

	return cfg, nil
}

// setDefaults sets all default configuration values
func setDefaults() {
	// Ollama defaults
	viper.SetDefault("ollama.url", "https://ollama.kitty-tetra.ts.net")
	viper.SetDefault("ollama.model", "qwen2.5:7b")
	viper.SetDefault("ollama.system_prompt", "")
	viper.SetDefault("ollama.timeout", "90s")
	viper.SetDefault("ollama.poll_interval", 10)

	// General defaults
	viper.SetDefault("show_thinking", true)
	viper.SetDefault("streaming", true)

	// Logging defaults
	viper.SetDefault("logging.file", "./.ryan/debug.log")
	viper.SetDefault("logging.preserve", false)
	viper.SetDefault("logging.level", "info")

	// Tools defaults
	viper.SetDefault("tools.enabled", true)
	viper.SetDefault("tools.truncate_output", true)
	viper.SetDefault("tools.bash.enabled", true)
	viper.SetDefault("tools.bash.timeout", "90s")
	viper.SetDefault("tools.file_read.enabled", true)
	viper.SetDefault("tools.search.enabled", true)
	viper.SetDefault("tools.search.timeout", "10s")
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