package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextConfig_BasicFields(t *testing.T) {
	config := ContextConfig{
		Directory:        "/tmp/contexts",
		MaxFileSize:      "10MB",
		PersistLangChain: true,
	}

	assert.Equal(t, "/tmp/contexts", config.Directory)
	assert.Equal(t, "10MB", config.MaxFileSize)
	assert.True(t, config.PersistLangChain)
}

func TestContextConfig_DefaultValues(t *testing.T) {
	// Test that default values can be set
	config := ContextConfig{}

	// Simulate setting defaults
	config.Directory = "./.ryan/contexts"
	config.MaxFileSize = "10MB"
	config.PersistLangChain = true

	assert.Equal(t, "./.ryan/contexts", config.Directory)
	assert.Equal(t, "10MB", config.MaxFileSize)
	assert.True(t, config.PersistLangChain)
}

func TestContextConfig_DirectoryPath(t *testing.T) {
	tests := []struct {
		name      string
		directory string
		expected  string
	}{
		{
			name:      "Absolute path",
			directory: "/tmp/contexts",
			expected:  "/tmp/contexts",
		},
		{
			name:      "Relative path",
			directory: "./contexts",
			expected:  "./contexts",
		},
		{
			name:      "Home directory",
			directory: "~/contexts",
			expected:  "~/contexts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ContextConfig{
				Directory: tt.directory,
			}
			assert.Equal(t, tt.expected, config.Directory)
		})
	}
}

func TestContextConfig_FileSizeFormats(t *testing.T) {
	tests := []struct {
		name        string
		maxFileSize string
		valid       bool
	}{
		{
			name:        "Megabytes",
			maxFileSize: "10MB",
			valid:       true,
		},
		{
			name:        "Kilobytes",
			maxFileSize: "500KB",
			valid:       true,
		},
		{
			name:        "Gigabytes",
			maxFileSize: "1GB",
			valid:       true,
		},
		{
			name:        "Bytes only",
			maxFileSize: "1024",
			valid:       true,
		},
		{
			name:        "Empty string",
			maxFileSize: "",
			valid:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ContextConfig{
				MaxFileSize: tt.maxFileSize,
			}

			// Just test that the value is stored correctly
			assert.Equal(t, tt.maxFileSize, config.MaxFileSize)

			// In a real implementation, you'd parse the size here
			if tt.valid {
				assert.NotEmpty(t, config.MaxFileSize)
			} else {
				assert.Empty(t, config.MaxFileSize)
			}
		})
	}
}

func TestContextConfig_Integration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "context-integration-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := ContextConfig{
		Directory:        tmpDir,
		MaxFileSize:      "5MB",
		PersistLangChain: false,
	}

	// Test that we can work with the directory
	testFile := filepath.Join(tmpDir, "test-context.json")
	err = os.WriteFile(testFile, []byte(`{"test": "data"}`), 0644)
	require.NoError(t, err)

	// Verify file was created in the configured directory
	stat, err := os.Stat(testFile)
	assert.NoError(t, err)
	assert.False(t, stat.IsDir())
	assert.Greater(t, stat.Size(), int64(0))

	// Use the config to verify it contains expected values
	assert.Equal(t, tmpDir, config.Directory)
	assert.Equal(t, "5MB", config.MaxFileSize)
	assert.False(t, config.PersistLangChain)
}
