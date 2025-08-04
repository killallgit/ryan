package config

import (
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
	assert.Equal(t, "https://ollama.kitty-tetra.ts.net", cfg.Ollama.URL)
	assert.Equal(t, "qwen3:latest", cfg.Ollama.Model)
	assert.Equal(t, 90*time.Second, cfg.Ollama.Timeout)
	assert.Equal(t, 10, cfg.Ollama.PollInterval)
	assert.True(t, cfg.ShowThinking)
	assert.True(t, cfg.Streaming)
	assert.Equal(t, "./.ryan/system.log", cfg.Logging.LogFile)
	assert.False(t, cfg.Logging.Preserve)

	// Context configuration
	assert.Equal(t, "./.ryan/contexts", cfg.Context.Directory)
	assert.Equal(t, "10MB", cfg.Context.MaxFileSize)
	assert.True(t, cfg.Context.PersistLangChain)

	assert.True(t, cfg.Tools.Enabled)
	assert.True(t, cfg.Tools.Bash.Enabled)
	assert.Equal(t, 90*time.Second, cfg.Tools.Bash.Timeout)
}

func TestLoadFromFile(t *testing.T) {
	// Reset viper
	viper.Reset()

	// Load config from test file
	cfg, err := Load("testdata/test-settings.yaml")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check loaded values
	assert.Equal(t, "http://test-ollama:11434", cfg.Ollama.URL)
	assert.Equal(t, "test-model", cfg.Ollama.Model)
	assert.Equal(t, 2*time.Minute, cfg.Ollama.Timeout)
	assert.Equal(t, 5, cfg.Ollama.PollInterval)
	assert.False(t, cfg.ShowThinking)
	assert.Equal(t, "/tmp/test.log", cfg.Logging.LogFile)
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

func TestLoadSelfConfig(t *testing.T) {
	// Reset viper and global state
	viper.Reset()
	cfg = nil
	selfCfg = nil

	// Load config with self config path
	loadedCfg, err := Load("testdata/settings-with-self.yaml")
	require.NoError(t, err)
	require.NotNil(t, loadedCfg)

	// Check main config
	assert.Equal(t, "http://test-ollama:11434", loadedCfg.Ollama.URL)
	assert.Equal(t, "./testdata/test-self.yaml", loadedCfg.SelfConfigPath)

	// Check self config was loaded
	selfConfig := GetSelf()
	require.NotNil(t, selfConfig)
	assert.Equal(t, "self", selfConfig.Role)
	assert.Len(t, selfConfig.Traits, 2)
	
	// Check traits
	assert.Equal(t, "explorer", selfConfig.Traits[0].Name)
	assert.Contains(t, selfConfig.Traits[0].SystemPrompt, "inner explorer")
	assert.Equal(t, "planner", selfConfig.Traits[1].Name)
	assert.Contains(t, selfConfig.Traits[1].SystemPrompt, "broad goals")
}

func TestLoadSelfConfigMissing(t *testing.T) {
	// Reset viper and global state
	viper.Reset()
	cfg = nil
	selfCfg = nil

	// Load config with missing self config path
	loadedCfg, err := Load("testdata/settings-with-missing-self.yaml")
	require.NoError(t, err)
	require.NotNil(t, loadedCfg)

	// Self config should be nil
	selfConfig := GetSelf()
	assert.Nil(t, selfConfig)
}

func TestGetSelf(t *testing.T) {
	// Reset global state
	selfCfg = nil

	// Should return nil if not loaded
	assert.Nil(t, GetSelf())

	// Set a test self config
	selfCfg = &SelfConfig{
		Role: "test",
		Traits: []SelfTrait{
			{Name: "test-trait", SystemPrompt: "test prompt"},
		},
	}

	// Should return the config
	retrieved := GetSelf()
	require.NotNil(t, retrieved)
	assert.Equal(t, "test", retrieved.Role)
	assert.Len(t, retrieved.Traits, 1)
}
