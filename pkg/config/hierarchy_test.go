package config

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigHierarchy(t *testing.T) {
	t.Run("getSystemDefaults uses Viper defaults", func(t *testing.T) {
		// Create a new ConfigHierarchy
		cm := NewContextManager()
		ch := NewConfigHierarchy(cm)

		// Get system defaults
		config := ch.getSystemDefaults()

		// Verify defaults are correctly set
		assert.Equal(t, "ollama", config.Provider)
		assert.Equal(t, "https://ollama.kitty-tetra.ts.net", config.Ollama.URL)
		assert.Equal(t, "qwen3:latest", config.Ollama.Model)
		assert.Equal(t, true, config.ShowThinking)
		assert.Equal(t, true, config.Streaming)
		assert.Equal(t, "./.ryan/system.log", config.Logging.LogFile)
		assert.Equal(t, "info", config.Logging.Level)
		assert.Equal(t, false, config.Logging.Preserve)
	})

	t.Run("environment variables override defaults via Viper", func(t *testing.T) {
		// Set environment variables
		os.Setenv("RYAN_OLLAMA_MODEL", "test-model-env")
		os.Setenv("RYAN_LOGGING_LEVEL", "debug")
		os.Setenv("RYAN_SHOW_THINKING", "false")
		defer func() {
			os.Unsetenv("RYAN_OLLAMA_MODEL")
			os.Unsetenv("RYAN_LOGGING_LEVEL")
			os.Unsetenv("RYAN_SHOW_THINKING")
		}()

		// Create a new viper instance for testing
		v := viper.New()
		v.SetEnvPrefix("RYAN")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()

		// Set defaults
		for key, value := range getAllDefaults() {
			v.SetDefault(key, value)
		}

		// Build config from viper
		config := buildConfigFromViper(v)

		// Verify environment overrides work
		assert.Equal(t, "test-model-env", config.Ollama.Model)
		assert.Equal(t, "debug", config.Logging.Level)
		assert.Equal(t, false, config.ShowThinking)
	})

	t.Run("getAllDefaults returns complete defaults map", func(t *testing.T) {
		defaults := getAllDefaults()

		// Check all major sections have defaults
		require.Contains(t, defaults, "provider")
		require.Contains(t, defaults, "ollama.url")
		require.Contains(t, defaults, "logging.log_file")
		require.Contains(t, defaults, "tools.enabled")
		require.Contains(t, defaults, "langchain.tools.max_iterations")
		require.Contains(t, defaults, "vectorstore.enabled")

		// Verify specific values
		assert.Equal(t, "ollama", defaults["provider"])
		assert.Equal(t, true, defaults["show_thinking"])
		assert.Equal(t, 5, defaults["langchain.tools.max_iterations"])
		assert.Equal(t, "chromem", defaults["vectorstore.provider"])
	})

	t.Run("applyEnvironmentOverrides uses Viper state", func(t *testing.T) {
		// Set environment variables
		os.Setenv("RYAN_TOOLS_ENABLED", "false")
		os.Setenv("RYAN_VECTORSTORE_PROVIDER", "test-provider")
		defer func() {
			os.Unsetenv("RYAN_TOOLS_ENABLED")
			os.Unsetenv("RYAN_VECTORSTORE_PROVIDER")
		}()

		// Setup viper with environment support
		viper.Reset()
		viper.SetEnvPrefix("RYAN")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AutomaticEnv()
		setDefaults()

		// Create config hierarchy
		cm := NewContextManager()
		ch := NewConfigHierarchy(cm)

		// Start with defaults
		config := ch.getSystemDefaults()

		// Apply environment overrides
		ch.applyEnvironmentOverrides(config)

		// Verify overrides were applied
		assert.Equal(t, false, config.Tools.Enabled)
		assert.Equal(t, "test-provider", config.VectorStore.Provider)
	})
}
