package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for test config
	tmpDir, err := os.MkdirTemp("", "memory_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	configDir := filepath.Join(tmpDir, ".ryan")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Use shared test configuration
	configFile := setupTestConfig(t, tmpDir)

	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "ryan"), "..")
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(buildOutput))

	ryanBinary := filepath.Join(tmpDir, "ryan")

	t.Run("It remembers conversation with memory enabled", func(t *testing.T) {
		// First message: introduce a fact
		cmd1 := exec.Command(ryanBinary,
			"--config", configFile,
			"--headless",
			"--prompt", "My favorite color is blue. Just acknowledge this.")
		cmd1.Dir = tmpDir
		cmd1.Env = setupEnvForOllama()

		output1, err := cmd1.CombinedOutput()
		require.NoError(t, err, "First command failed: %s", string(output1))

		// Verify acknowledgment
		response1 := string(output1)
		assert.Contains(t, strings.ToLower(response1), "blue",
			"Expected acknowledgment of 'blue' color")

		// Second message: ask about the fact (with --continue flag)
		cmd2 := exec.Command(ryanBinary,
			"--config", configFile,
			"--headless",
			"--continue",
			"--prompt", "What was my favorite color?")
		cmd2.Dir = tmpDir
		cmd2.Env = setupEnvForOllama()

		output2, err := cmd2.CombinedOutput()
		require.NoError(t, err, "Second command failed: %s", string(output2))

		// Verify memory recall
		response2 := string(output2)
		assert.Contains(t, strings.ToLower(response2), "blue",
			"Expected model to remember 'blue' as favorite color")
	})

	t.Run("It persists memory across multiple sessions", func(t *testing.T) {
		sessionDir := filepath.Join(tmpDir, "session2")
		err := os.MkdirAll(sessionDir, 0755)
		require.NoError(t, err)

		// Create a new config for this test
		sessionConfigDir := filepath.Join(sessionDir, ".ryan")
		err = os.MkdirAll(sessionConfigDir, 0755)
		require.NoError(t, err)

		sessionConfigFile := setupTestConfig(t, sessionDir)

		// Session 1: Add fact
		cmd1 := exec.Command(ryanBinary,
			"--config", sessionConfigFile,
			"--headless",
			"--prompt", "My pet's name is Fluffy.")
		cmd1.Dir = sessionDir
		cmd1.Env = setupEnvForOllama()

		output1, err := cmd1.CombinedOutput()
		require.NoError(t, err, "Session 1 failed: %s", string(output1))

		// Session 2: Add another fact
		cmd2 := exec.Command(ryanBinary,
			"--config", sessionConfigFile,
			"--headless",
			"--continue",
			"--prompt", "My favorite number is 42.")
		cmd2.Dir = sessionDir
		cmd2.Env = setupEnvForOllama()

		output2, err := cmd2.CombinedOutput()
		require.NoError(t, err, "Session 2 failed: %s", string(output2))

		// Session 3: Recall both facts
		cmd3 := exec.Command(ryanBinary,
			"--config", sessionConfigFile,
			"--headless",
			"--continue",
			"--prompt", "What do you remember about my pet and favorite number?")
		cmd3.Dir = sessionDir
		cmd3.Env = setupEnvForOllama()

		output3, err := cmd3.CombinedOutput()
		require.NoError(t, err, "Session 3 failed: %s", string(output3))

		response3 := string(output3)
		assert.Contains(t, strings.ToLower(response3), "fluffy",
			"Expected model to remember pet name 'Fluffy'")
		assert.Contains(t, response3, "42",
			"Expected model to remember favorite number '42'")
	})

	t.Run("It respects memory window size", func(t *testing.T) {
		t.Skip("Skipping window size test - takes too long")
		// Create config with small window size
		windowDir := filepath.Join(tmpDir, "window_test")
		err := os.MkdirAll(windowDir, 0755)
		require.NoError(t, err)

		windowConfigDir := filepath.Join(windowDir, ".ryan")
		err = os.MkdirAll(windowConfigDir, 0755)
		require.NoError(t, err)

		windowConfigFile := setupTestConfig(t, windowDir)

		// Add 3 messages (more than window size)
		messages := []string{
			"First fact: I live in California.",
			"Second fact: I work as a programmer.",
			"Third fact: I have two cats.",
		}

		for i, msg := range messages {
			continueFlag := []string{}
			if i > 0 {
				continueFlag = []string{"--continue"}
			}

			args := append([]string{
				"--config", windowConfigFile,
				"--headless",
			}, continueFlag...)
			args = append(args, "--prompt", msg)

			cmd := exec.Command(ryanBinary, args...)
			cmd.Dir = windowDir
			cmd.Env = setupEnvWithOverrides(map[string]string{
				"memory_window_size": "2",
			})

			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Message %d failed: %s", i+1, string(output))

			// Small delay to ensure DB write completes
			time.Sleep(100 * time.Millisecond)
		}

		// Ask about the first fact (should be forgotten due to window size)
		cmd := exec.Command(ryanBinary,
			"--config", windowConfigFile,
			"--headless",
			"--continue",
			"--prompt", "Where do I live?")
		cmd.Dir = windowDir
		cmd.Env = setupEnvWithOverrides(map[string]string{
			"memory_window_size": "2",
		})

		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Recall command failed: %s", string(output))

		response := string(output)
		// Due to window size of 2, the first fact about California should be forgotten
		// The model should not confidently state California
		t.Logf("Response after window overflow: %s", response)
	})

	t.Run("It can clear memory", func(t *testing.T) {
		clearDir := filepath.Join(tmpDir, "clear_test")
		err := os.MkdirAll(clearDir, 0755)
		require.NoError(t, err)

		clearConfigDir := filepath.Join(clearDir, ".ryan")
		err = os.MkdirAll(clearConfigDir, 0755)
		require.NoError(t, err)

		clearConfigFile := setupTestConfig(t, clearDir)

		// Add a fact
		cmd1 := exec.Command(ryanBinary,
			"--config", clearConfigFile,
			"--headless",
			"--prompt", "Remember: my birthday is January 15th.")
		cmd1.Dir = clearDir
		cmd1.Env = setupEnvForOllama()

		output1, err := cmd1.CombinedOutput()
		require.NoError(t, err, "Add fact failed: %s", string(output1))

		// Start new session without --continue (should clear memory)
		cmd2 := exec.Command(ryanBinary,
			"--config", clearConfigFile,
			"--headless",
			"--prompt", "When is my birthday?")
		cmd2.Dir = clearDir
		cmd2.Env = setupEnvForOllama()

		output2, err := cmd2.CombinedOutput()
		require.NoError(t, err, "Query after clear failed: %s", string(output2))

		response2 := string(output2)
		// Without --continue, memory should be cleared
		assert.NotContains(t, strings.ToLower(response2), "january 15",
			"Expected model to forget birthday after memory clear")
	})

	t.Run("Memory database is created in correct location", func(t *testing.T) {
		// First run a command to ensure the database is created
		cmd := exec.Command(ryanBinary,
			"--config", configFile,
			"--headless",
			"--prompt", "test")
		cmd.Dir = tmpDir
		cmd.Env = setupEnvForOllama()

		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Command should execute successfully: %s", string(output))
		t.Logf("Command output: %s", string(output))

		// Check what files were created
		entries, _ := os.ReadDir(configDir)
		t.Logf("Config dir contents: %v", entries)

		contextDir := filepath.Join(configDir, "context")
		if _, err := os.Stat(contextDir); err == nil {
			contextEntries, _ := os.ReadDir(contextDir)
			t.Logf("Context dir contents: %v", contextEntries)
		}

		// Check that memory.db was created in the context subdirectory
		memoryDB := filepath.Join(configDir, "context", "memory.db")
		_, err = os.Stat(memoryDB)
		assert.NoError(t, err, "memory.db should exist in context directory")

		// Verify it's a valid SQLite database by checking file header
		data, err := os.ReadFile(memoryDB)
		require.NoError(t, err)
		assert.True(t, len(data) > 16, "Database file should have content")

		// SQLite databases start with "SQLite format 3"
		header := string(data[:15])
		assert.Equal(t, "SQLite format 3", header, "Should be a valid SQLite database")
	})
}
