package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Settings holds all configuration values
type Settings struct {
	// Provider configuration
	Provider string

	// Display settings
	ShowThinking bool

	// Ollama configuration
	Ollama struct {
		DefaultModel string
		Timeout      int
		Host         string
	}

	// Logging configuration
	Logging struct {
		LogFile string
		Persist bool
		Level   string
	}

	// LangChain configuration
	LangChain struct {
		MemoryType       string
		MemoryWindowSize int
		Tools            struct {
			MaxIterations int
			MaxRetries    int
		}
	}

	// Tools configuration
	Tools struct {
		Enabled bool
		File    struct {
			Read  struct{ Enabled bool }
			Write struct{ Enabled bool }
		}
		Git    struct{ Enabled bool }
		Search struct{ Enabled bool }
		Web    struct{ Enabled bool }
		Bash   struct {
			Enabled bool
			Timeout int
		}
	}

	// Vector store configuration
	VectorStore struct {
		Enabled    bool
		Provider   string
		Collection struct {
			Name string
		}
		Persistence struct {
			Enabled bool
			Path    string
		}
		Embedding struct {
			Provider string
			Model    string
			Endpoint string
			APIKey   string
		}
		Retrieval struct {
			K              int
			ScoreThreshold float32
		}
	}

	// ConfigFile stores the path to the config file used
	ConfigFile string
}

// Global settings instance
var Global *Settings

// Init initializes the configuration system
func Init(cfgFile string) error {
	Global = &Settings{}

	// Set config file
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		Global.ConfigFile = cfgFile
	} else {
		viper.AddConfigPath("./.ryan")
		viper.SetConfigType("yaml")
		viper.SetConfigName("settings")
		Global.ConfigFile = ".ryan/settings.yaml"
	}

	// Set all defaults
	setDefaults()

	// Enable environment variable support
	viper.AutomaticEnv()

	// Bind specific environment variables to config keys
	// This allows OLLAMA_HOST to map to ollama.host
	viper.BindEnv("ollama.host", "OLLAMA_HOST")
	viper.BindEnv("ollama.default_model", "OLLAMA_DEFAULT_MODEL")
	viper.BindEnv("vectorstore.embedding.model", "OLLAMA_EMBEDDING_MODEL")
	viper.BindEnv("vectorstore.embedding.endpoint", "OLLAMA_HOST") // Reuse OLLAMA_HOST for embedding endpoint

	// Read config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	// Load settings into global struct
	return Load()
}

// setDefaults sets all default configuration values
func setDefaults() {
	// Provider defaults
	viper.SetDefault("provider", "ollama")
	viper.SetDefault("show_thinking", true)

	// Ollama defaults
	viper.SetDefault("ollama.default_model", "qwen3:latest")
	viper.SetDefault("ollama.timeout", 90)

	// Logging defaults
	viper.SetDefault("logging.log_file", "system.log")
	viper.SetDefault("logging.persist", false)
	viper.SetDefault("logging.level", "info")

	// LangChain defaults
	viper.SetDefault("langchain.memory_type", "window")
	viper.SetDefault("langchain.memory_window_size", 10)
	viper.SetDefault("langchain.tools.max_iterations", 10)
	viper.SetDefault("langchain.tools.max_retries", 3)

	// Tool configuration defaults
	viper.SetDefault("tools.enabled", true)
	viper.SetDefault("tools.file.read.enabled", true)
	viper.SetDefault("tools.file.write.enabled", true)
	viper.SetDefault("tools.git.enabled", true)
	viper.SetDefault("tools.search.enabled", true)
	viper.SetDefault("tools.web.enabled", true)
	viper.SetDefault("tools.bash.enabled", true)
	viper.SetDefault("tools.bash.timeout", 30)

	// Vector store defaults
	viper.SetDefault("vectorstore.enabled", false)
	viper.SetDefault("vectorstore.provider", "chromem")
	viper.SetDefault("vectorstore.collection.name", "default")
	viper.SetDefault("vectorstore.persistence.enabled", false)
	viper.SetDefault("vectorstore.persistence.path", "./data/vectors")
	viper.SetDefault("vectorstore.embedding.provider", "ollama")
	viper.SetDefault("vectorstore.embedding.model", "nomic-embed-text")
	viper.SetDefault("vectorstore.retrieval.k", 4)
	viper.SetDefault("vectorstore.retrieval.score_threshold", 0.0)
}

// Load loads configuration from viper into the Settings struct
func Load() error {
	// Provider settings
	Global.Provider = viper.GetString("provider")
	Global.ShowThinking = viper.GetBool("show_thinking")

	// Ollama settings
	Global.Ollama.DefaultModel = viper.GetString("ollama.default_model")
	Global.Ollama.Timeout = viper.GetInt("ollama.timeout")
	Global.Ollama.Host = viper.GetString("ollama.host")

	// Logging settings
	Global.Logging.LogFile = viper.GetString("logging.log_file")
	Global.Logging.Persist = viper.GetBool("logging.persist")
	Global.Logging.Level = viper.GetString("logging.level")

	// LangChain settings
	Global.LangChain.MemoryType = viper.GetString("langchain.memory_type")
	Global.LangChain.MemoryWindowSize = viper.GetInt("langchain.memory_window_size")
	Global.LangChain.Tools.MaxIterations = viper.GetInt("langchain.tools.max_iterations")
	Global.LangChain.Tools.MaxRetries = viper.GetInt("langchain.tools.max_retries")

	// Tools settings
	Global.Tools.Enabled = viper.GetBool("tools.enabled")
	Global.Tools.File.Read.Enabled = viper.GetBool("tools.file.read.enabled")
	Global.Tools.File.Write.Enabled = viper.GetBool("tools.file.write.enabled")
	Global.Tools.Git.Enabled = viper.GetBool("tools.git.enabled")
	Global.Tools.Search.Enabled = viper.GetBool("tools.search.enabled")
	Global.Tools.Web.Enabled = viper.GetBool("tools.web.enabled")
	Global.Tools.Bash.Enabled = viper.GetBool("tools.bash.enabled")
	Global.Tools.Bash.Timeout = viper.GetInt("tools.bash.timeout")

	// Vector store settings
	Global.VectorStore.Enabled = viper.GetBool("vectorstore.enabled")
	Global.VectorStore.Provider = viper.GetString("vectorstore.provider")
	Global.VectorStore.Collection.Name = viper.GetString("vectorstore.collection.name")
	Global.VectorStore.Persistence.Enabled = viper.GetBool("vectorstore.persistence.enabled")
	Global.VectorStore.Persistence.Path = viper.GetString("vectorstore.persistence.path")
	Global.VectorStore.Embedding.Provider = viper.GetString("vectorstore.embedding.provider")
	Global.VectorStore.Embedding.Model = viper.GetString("vectorstore.embedding.model")
	Global.VectorStore.Embedding.Endpoint = viper.GetString("vectorstore.embedding.endpoint")
	Global.VectorStore.Embedding.APIKey = viper.GetString("vectorstore.embedding.api_key")
	Global.VectorStore.Retrieval.K = viper.GetInt("vectorstore.retrieval.k")
	Global.VectorStore.Retrieval.ScoreThreshold = float32(viper.GetFloat64("vectorstore.retrieval.score_threshold"))

	return nil
}

// WriteDefaultConfig writes default configuration values to disk, preserving existing settings
func WriteDefaultConfig() error {
	if Global.ConfigFile == "" {
		return fmt.Errorf("config file path not set")
	}

	// Ensure config directory exists
	configDir := filepath.Dir(Global.ConfigFile)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Write current configuration to file (preserves existing + adds defaults)
	if err := viper.WriteConfigAs(Global.ConfigFile); err != nil {
		return fmt.Errorf("error writing config: %w", err)
	}

	return nil
}

// Get returns the global settings instance
func Get() *Settings {
	if Global == nil {
		panic("config not initialized - call Init() first")
	}
	return Global
}
