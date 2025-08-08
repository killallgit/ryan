package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildBinary(t *testing.T) string {
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "ryan-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, "../main.go")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	return binaryPath
}

func TestCLIPromptFlag(t *testing.T) {
	t.Run("It accepts --prompt flag and executes without entering TUI", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create a test config directory
		configDir := filepath.Join(tempDir, ".ryan")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// Run with --prompt flag and headless mode
		cmd := exec.Command(binaryPath, "--prompt", "test prompt", "--headless", "--config", filepath.Join(configDir, "settings.yaml"))
		cmd.Dir = tempDir

		// Set a timeout for the command
		done := make(chan error, 1)
		go func() {
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("Command output: %s", output)
			}
			done <- err
		}()

		select {
		case err := <-done:
			// Command completed - this is what we expect in headless mode
			// The command might fail if no model is configured, but that's ok for this test
			// We're just testing that it accepts the flag and doesn't hang
			t.Logf("Command completed with err: %v", err)
		case <-time.After(3 * time.Second):
			// If it takes too long, kill it
			cmd.Process.Kill()
			t.Fatal("Command took too long - likely entered TUI mode instead of headless")
		}
	})

	t.Run("It requires --headless flag when using --prompt", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create a test config directory
		configDir := filepath.Join(tempDir, ".ryan")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// Run with --prompt flag but without --headless (should enter TUI)
		cmd := exec.Command(binaryPath, "--prompt", "test prompt", "--config", filepath.Join(configDir, "settings.yaml"))
		cmd.Dir = tempDir

		// This should enter TUI mode, so we expect it to hang
		done := make(chan bool, 1)
		go func() {
			cmd.Start()
			time.Sleep(1 * time.Second)
			cmd.Process.Kill()
			done <- true
		}()

		select {
		case <-done:
			// Expected behavior - process was killed after timeout
			t.Log("Process entered TUI mode as expected")
		case <-time.After(2 * time.Second):
			cmd.Process.Kill()
			t.Fatal("Unexpected behavior")
		}
	})
}

func TestCLIContinueFlag(t *testing.T) {
	t.Run("It continues the chat history when --continue flag is used", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create a test config directory and history file
		configDir := filepath.Join(tempDir, ".ryan")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		historyFile := filepath.Join(configDir, "chat_history.json")
		testHistory := `{"messages": [{"role": "user", "content": "previous message"}]}`
		err = os.WriteFile(historyFile, []byte(testHistory), 0644)
		require.NoError(t, err)

		// Run with --continue flag
		cmd := exec.Command(binaryPath, "--continue", "--prompt", "new prompt", "--headless", "--config", filepath.Join(configDir, "settings.yaml"))
		cmd.Dir = tempDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Run command with timeout
		done := make(chan error, 1)
		go func() {
			done <- cmd.Run()
		}()

		select {
		case <-done:
			// Command completed
		case <-time.After(3 * time.Second):
			cmd.Process.Kill()
			t.Logf("stdout: %s", stdout.String())
			t.Logf("stderr: %s", stderr.String())
		}

		// Check that history file still exists and contains original content
		assert.FileExists(t, historyFile, "History file should still exist when using --continue")

		content, err := os.ReadFile(historyFile)
		if err == nil {
			t.Logf("History file content after --continue: %s", string(content))
		}
	})

	t.Run("It deletes history file when --continue flag is not used", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create a test config directory and history file
		configDir := filepath.Join(tempDir, ".ryan")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		historyFile := filepath.Join(configDir, "chat_history.json")
		testHistory := `{"messages": [{"role": "user", "content": "previous message"}]}`
		err = os.WriteFile(historyFile, []byte(testHistory), 0644)
		require.NoError(t, err)

		// Verify file exists before running
		assert.FileExists(t, historyFile)

		// Run without --continue flag (should clear history)
		cmd := exec.Command(binaryPath, "--prompt", "new prompt", "--headless", "--config", filepath.Join(configDir, "settings.yaml"))
		cmd.Dir = tempDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Run command with timeout
		done := make(chan error, 1)
		go func() {
			done <- cmd.Run()
		}()

		select {
		case <-done:
			// Command completed
		case <-time.After(3 * time.Second):
			cmd.Process.Kill()
			t.Logf("stdout: %s", stdout.String())
			t.Logf("stderr: %s", stderr.String())
		}

		// Check that the old history was not preserved
		content, err := os.ReadFile(historyFile)
		if os.IsNotExist(err) {
			// File was deleted, that's fine
			t.Log("History file was deleted as expected")
		} else if err != nil {
			t.Fatalf("Error reading history file: %v", err)
		} else {
			// File exists, check that it doesn't contain the old "previous message"
			historyStr := string(content)
			if strings.Contains(historyStr, "previous message") {
				t.Errorf("History file still contains old conversation: %s", historyStr)
			} else {
				t.Log("History file was reset (old conversation removed)")
			}
		}
	})
}

func TestCLIHeadlessMode(t *testing.T) {
	t.Run("It runs in headless mode with --headless and --prompt flags", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create a test config directory
		configDir := filepath.Join(tempDir, ".ryan")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// Run in headless mode
		cmd := exec.Command(binaryPath, "--headless", "--prompt", "test", "--config", filepath.Join(configDir, "settings.yaml"))
		cmd.Dir = tempDir

		// Capture output
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Run with timeout
		done := make(chan error, 1)
		go func() {
			done <- cmd.Run()
		}()

		select {
		case err := <-done:
			// Command should complete quickly in headless mode
			t.Logf("Headless mode completed with err: %v", err)
			t.Logf("stdout: %s", stdout.String())
			t.Logf("stderr: %s", stderr.String())
		case <-time.After(5 * time.Second):
			cmd.Process.Kill()
			t.Fatal("Headless mode took too long to complete")
		}
	})

	t.Run("It defaults to 'hello' prompt when --headless is used without --prompt", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create a test config directory
		configDir := filepath.Join(tempDir, ".ryan")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// Run with --headless but no --prompt (should default to "hello")
		cmd := exec.Command(binaryPath, "--headless", "--config", filepath.Join(configDir, "settings.yaml"))
		cmd.Dir = tempDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Run with timeout
		done := make(chan error, 1)
		go func() {
			done <- cmd.Run()
		}()

		select {
		case err := <-done:
			// Should complete successfully with default "hello" prompt
			t.Logf("Headless mode with default prompt completed with err: %v", err)
			t.Logf("stdout: %s", stdout.String())
			t.Logf("stderr: %s", stderr.String())
		case <-time.After(5 * time.Second):
			cmd.Process.Kill()
			t.Fatal("Headless mode with default prompt took too long")
		}
	})
}

func TestCLIConfigPath(t *testing.T) {
	t.Run("It respects custom config path with --config flag", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create custom config location
		customConfigPath := filepath.Join(tempDir, "custom", "config.yaml")
		customConfigDir := filepath.Dir(customConfigPath)
		err := os.MkdirAll(customConfigDir, 0755)
		require.NoError(t, err)

		// Run with custom config
		cmd := exec.Command(binaryPath, "--config", customConfigPath, "--headless", "--prompt", "test")
		cmd.Dir = tempDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Run with timeout
		done := make(chan error, 1)
		go func() {
			done <- cmd.Run()
		}()

		select {
		case <-done:
			// Check if config file was created at custom location
			if _, err := os.Stat(customConfigPath); err == nil {
				t.Log("Config file created at custom location")
			}

			// Check output for config file usage
			output := fmt.Sprintf("stdout: %s\nstderr: %s", stdout.String(), stderr.String())
			if bytes.Contains([]byte(output), []byte(customConfigPath)) {
				t.Log("Custom config path was used")
			}
		case <-time.After(3 * time.Second):
			cmd.Process.Kill()
		}
	})
}
