package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	// Reset viper
	viper.Reset()

	// Load config without a file
	cfg, err := Load("")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check defaults
	assert.Equal(t, "http://localhost:11434", cfg.Ollama.URL)
	assert.Equal(t, "qwen3:latest", cfg.Ollama.Model)
	assert.Equal(t, 90*time.Second, cfg.Ollama.Timeout)
	assert.Equal(t, 10, cfg.Ollama.PollInterval)
	assert.True(t, cfg.ShowThinking)
	assert.True(t, cfg.Streaming)
	assert.Equal(t, "./.ryan/logs/debug.log", cfg.Logging.File)
	assert.False(t, cfg.Logging.Preserve)

	// Context configuration
	assert.Equal(t, "./.ryan/contexts", cfg.Context.Directory)
	assert.Equal(t, "./.ryan/logs/debug.history", cfg.Context.HistoryFile)
	assert.Equal(t, "10MB", cfg.Context.MaxFileSize)
	assert.True(t, cfg.Context.PersistLangChain)

	assert.True(t, cfg.Tools.Enabled)
	assert.True(t, cfg.Tools.Bash.Enabled)
	assert.Equal(t, 90*time.Second, cfg.Tools.Bash.Timeout)
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-settings.yaml")

	configContent := `
ollama:
  url: http://test-ollama:11434
  model: test-model
  timeout: "2m"
  poll_interval: 5
show_thinking: false
logging:
  file: /tmp/test.log
  preserve: true
tools:
  bash:
    timeout: "30s"
    allowed_paths: ["/test", "/tmp"]
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset viper
	viper.Reset()

	// Load config from file
	cfg, err := Load(configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check loaded values
	assert.Equal(t, "http://test-ollama:11434", cfg.Ollama.URL)
	assert.Equal(t, "test-model", cfg.Ollama.Model)
	assert.Equal(t, 2*time.Minute, cfg.Ollama.Timeout)
	assert.Equal(t, 5, cfg.Ollama.PollInterval)
	assert.False(t, cfg.ShowThinking)
	assert.Equal(t, "/tmp/test.log", cfg.Logging.File)
	assert.True(t, cfg.Logging.Preserve)
	assert.Equal(t, 30*time.Second, cfg.Tools.Bash.Timeout)
	assert.Equal(t, []string{"/test", "/tmp"}, cfg.Tools.Bash.AllowedPaths)
}

func TestProcessDurations(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
	}{
		{
			name: "valid durations",
			config: &Config{
				Ollama: OllamaConfig{TimeoutStr: "1m30s"},
				Tools: ToolsConfig{
					Bash:   BashToolConfig{TimeoutStr: "2m"},
					Search: SearchConfig{TimeoutStr: "30s"},
				},
			},
			expectErr: false,
		},
		{
			name: "invalid ollama timeout",
			config: &Config{
				Ollama: OllamaConfig{TimeoutStr: "invalid"},
			},
			expectErr: true,
		},
		{
			name: "empty durations use defaults",
			config: &Config{
				Ollama: OllamaConfig{},
				Tools:  ToolsConfig{},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processDurations(tt.config)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Check defaults were applied if strings were empty
				if tt.config.Ollama.TimeoutStr == "" {
					assert.Equal(t, 90*time.Second, tt.config.Ollama.Timeout)
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	// Reset global config
	cfg = nil

	// Should panic if not initialized
	assert.Panics(t, func() {
		Get()
	})

	// Initialize config
	viper.Reset()
	_, err := Load("")
	require.NoError(t, err)

	// Now Get should work
	assert.NotPanics(t, func() {
		c := Get()
		assert.NotNil(t, c)
	})
}
