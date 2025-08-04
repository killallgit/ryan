package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitHistoryFile(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "test.history")

	t.Run("should create new history file when continue is false", func(t *testing.T) {
		// Write some initial content
		err := os.WriteFile(historyPath, []byte("existing content\n"), 0644)
		require.NoError(t, err)

		// Initialize with continue=false (should truncate)
		err = InitHistoryFile(historyPath, false)
		require.NoError(t, err)

		// Close the file so we can read it
		err = Close()
		require.NoError(t, err)

		// Read the file
		content, err := os.ReadFile(historyPath)
		require.NoError(t, err)

		// Should contain only the new session marker
		assert.Contains(t, string(content), "Ryan Chat Session Started")
		assert.NotContains(t, string(content), "existing content")
	})

	t.Run("should append to existing history when continue is true", func(t *testing.T) {
		// Write some initial content
		initialContent := "=== Previous Session ===\nOld chat history\n"
		err := os.WriteFile(historyPath, []byte(initialContent), 0644)
		require.NoError(t, err)

		// Initialize with continue=true (should append)
		err = InitHistoryFile(historyPath, true)
		require.NoError(t, err)

		// Close the file so we can read it
		err = Close()
		require.NoError(t, err)

		// Read the file
		content, err := os.ReadFile(historyPath)
		require.NoError(t, err)

		// Should contain both old and new content
		assert.Contains(t, string(content), "Previous Session")
		assert.Contains(t, string(content), "Old chat history")
		assert.Contains(t, string(content), "Ryan Chat Session Continued")
	})

	t.Run("should create directory if it doesn't exist", func(t *testing.T) {
		// Use a nested path that doesn't exist
		nestedPath := filepath.Join(tmpDir, "nested", "dir", "test.history")

		err := InitHistoryFile(nestedPath, false)
		require.NoError(t, err)

		// Close the file
		err = Close()
		require.NoError(t, err)

		// Check that the file was created
		_, err = os.Stat(nestedPath)
		assert.NoError(t, err)
	})

	t.Run("should handle empty path with default", func(t *testing.T) {
		// Temporarily change working directory to temp dir
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(oldWd)

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Initialize with empty path
		err = InitHistoryFile("", false)
		require.NoError(t, err)

		// Close the file
		err = Close()
		require.NoError(t, err)

		// Check that default path was used
		_, err = os.Stat(".ryan/logs/debug.history")
		assert.NoError(t, err)
	})
}

func TestLogChatHistory(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "test.history")

	// Initialize the history file first
	err := InitHistoryFile(historyPath, false)
	require.NoError(t, err)

	t.Run("should log chat messages", func(t *testing.T) {
		// Log some messages
		err := LogChatHistory("user", "Hello, how are you?")
		require.NoError(t, err)

		err = LogChatHistory("assistant", "I'm doing well, thank you!")
		require.NoError(t, err)

		// Close and read the file
		err = Close()
		require.NoError(t, err)

		content, err := os.ReadFile(historyPath)
		require.NoError(t, err)

		// Check content
		lines := strings.Split(string(content), "\n")
		foundUser := false
		foundAssistant := false

		for _, line := range lines {
			if strings.Contains(line, "user:") && strings.Contains(line, "Hello, how are you?") {
				foundUser = true
			}
			if strings.Contains(line, "assistant:") && strings.Contains(line, "I'm doing well, thank you!") {
				foundAssistant = true
			}
		}

		assert.True(t, foundUser, "User message not found in history")
		assert.True(t, foundAssistant, "Assistant message not found in history")
	})
}
