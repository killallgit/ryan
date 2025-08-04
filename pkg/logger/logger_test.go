package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLoggerWithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	t.Run("should create log file with correct level", func(t *testing.T) {
		err := InitLoggerWithConfig(logFile, false, "debug")
		require.NoError(t, err)

		// Test that file was created
		_, err = os.Stat(logFile)
		assert.NoError(t, err)

		// Clean up
		err = Close()
		require.NoError(t, err)
	})

	t.Run("should respect preserve flag", func(t *testing.T) {
		// Write initial content
		initialContent := "initial log content\n"
		err := os.WriteFile(logFile, []byte(initialContent), 0644)
		require.NoError(t, err)

		// Initialize with preserve=true
		err = InitLoggerWithConfig(logFile, true, "info")
		require.NoError(t, err)

		// Log a message
		Info("test message")

		// Close and read
		err = Close()
		require.NoError(t, err)

		content, err := os.ReadFile(logFile)
		require.NoError(t, err)

		// Should contain both initial and new content
		assert.Contains(t, string(content), "initial log content")
		assert.Contains(t, string(content), "test message")
	})
}
