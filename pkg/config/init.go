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

	// Runtime flags (transient, not persisted)
	Prompt          string
	Headless        bool
	Continue        bool
	SkipPermissions bool
	ConfigFile      string
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

	// Override with environment variables
	applyEnvironmentOverrides()

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

// applyEnvironmentOverrides applies environment variable overrides
func applyEnvironmentOverrides() {
	// Override ollama.default_model with OLLAMA_DEFAULT_MODEL if set
	if ollamaModel := os.Getenv("OLLAMA_DEFAULT_MODEL"); ollamaModel != "" {
		viper.Set("ollama.default_model", ollamaModel)
	}

	// Override ollama.host with OLLAMA_HOST if set
	if ollamaHost := os.Getenv("OLLAMA_HOST"); ollamaHost != "" {
		viper.Set("ollama.host", ollamaHost)
		// Also set for vectorstore embedding endpoint
		viper.SetDefault("vectorstore.embedding.endpoint", ollamaHost)
	}
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

	// Runtime flags
	Global.Prompt = viper.GetString("prompt")
	Global.Headless = viper.GetBool("headless")
	Global.Continue = viper.GetBool("continue")
	Global.SkipPermissions = viper.GetBool("skip_permissions")

	return nil
}

// RefreshConfig refreshes the configuration, clearing transient values
func RefreshConfig(promptValue string, headlessMode bool, continueHistory bool) error {
	// Clear transient flags that shouldn't be persisted
	viper.Set("prompt", "")
	viper.Set("headless", false)
	viper.Set("continue", false)

	// Ensure config directory exists
	dirFromCfgFile := filepath.Dir(Global.ConfigFile)
	if _, err := os.Stat(dirFromCfgFile); os.IsNotExist(err) {
		if err := os.MkdirAll(dirFromCfgFile, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Write config without transient values
	if err := viper.WriteConfigAs(Global.ConfigFile); err != nil {
		return fmt.Errorf("error writing config: %w", err)
	}

	// Only restore prompt value if running in headless mode
	// In TUI mode, prompt should not be used
	if headlessMode {
		viper.Set("prompt", promptValue)
		Global.Prompt = promptValue
	}
	viper.Set("headless", headlessMode)
	viper.Set("continue", continueHistory)

	Global.Headless = headlessMode
	Global.Continue = continueHistory

	return nil
}

// SetTransientValues sets transient runtime values
func SetTransientValues(prompt string, headless, continueHistory, skipPermissions bool) {
	Global.Prompt = prompt
	Global.Headless = headless
	Global.Continue = continueHistory
	Global.SkipPermissions = skipPermissions
}

// Get returns the global settings instance
func Get() *Settings {
	if Global == nil {
		panic("config not initialized - call Init() first")
	}
	return Global
}
