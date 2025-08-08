package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAgentResponses(t *testing.T) {
	t.Run("It responds to a basic prompt in headless mode", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create a test config directory
		configDir := filepath.Join(tempDir, ".ryan")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// Run with a simple prompt
		cmd := exec.Command(binaryPath,
			"--headless",
			"--prompt", "Say hello and nothing else",
			"--config", filepath.Join(configDir, "settings.yaml"))
		cmd.Dir = tempDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Run with timeout (agent responses can take time)
		done := make(chan error, 1)
		go func() {
			done <- cmd.Run()
		}()

		select {
		case err := <-done:
			require.NoError(t, err, "Command should complete successfully")

			output := stdout.String()
			t.Logf("Agent response: %s", output)

			// Should have some response
			assert.NotEmpty(t, output, "Agent should produce output")

			// Should contain something related to hello
			outputLower := strings.ToLower(output)
			assert.True(t,
				strings.Contains(outputLower, "hello") ||
				strings.Contains(outputLower, "hi") ||
				strings.Contains(outputLower, "greet"),
				"Response should be related to the prompt")

		case <-time.After(30 * time.Second):
			cmd.Process.Kill()
			t.Fatal("Agent took too long to respond")
		}
	})

	t.Run("It outputs response to stdout", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create a test config directory
		configDir := filepath.Join(tempDir, ".ryan")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// Run with a math prompt for predictable output
		cmd := exec.Command(binaryPath,
			"--headless",
			"--prompt", "What is 2+2? Answer with just the number",
			"--config", filepath.Join(configDir, "settings.yaml"))
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
			require.NoError(t, err, "Command should complete successfully")

			output := stdout.String()
			t.Logf("stdout: %s", output)
			t.Logf("stderr: %s", stderr.String())

			// Output should be on stdout, not stderr
			assert.NotEmpty(t, output, "Response should be on stdout")

			// Should contain 4 somewhere in the response
			assert.Contains(t, output, "4", "Response should contain the answer")

		case <-time.After(30 * time.Second):
			cmd.Process.Kill()
			t.Fatal("Agent took too long to respond")
		}
	})

	t.Run("It handles multi-line prompts", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create a test config directory
		configDir := filepath.Join(tempDir, ".ryan")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// Multi-line prompt
		prompt := `List three colors:
1. Red
2. Blue
3. ?
Complete the list with one more color`

		cmd := exec.Command(binaryPath,
			"--headless",
			"--prompt", prompt,
			"--config", filepath.Join(configDir, "settings.yaml"))
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
			require.NoError(t, err, "Command should complete successfully")

			output := stdout.String()
			outputLower := strings.ToLower(output)
			t.Logf("Response to multi-line prompt: %s", output)

			// Should mention a color
			hasColor := strings.Contains(outputLower, "green") ||
				strings.Contains(outputLower, "yellow") ||
				strings.Contains(outputLower, "purple") ||
				strings.Contains(outputLower, "orange") ||
				strings.Contains(outputLower, "black") ||
				strings.Contains(outputLower, "white") ||
				strings.Contains(outputLower, "pink")

			assert.True(t, hasColor, "Response should include a color")

		case <-time.After(30 * time.Second):
			cmd.Process.Kill()
			t.Fatal("Agent took too long to respond")
		}
	})

	t.Run("It preserves conversation context with --continue", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create a test config directory
		configDir := filepath.Join(tempDir, ".ryan")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// First conversation: establish context
		cmd1 := exec.Command(binaryPath,
			"--headless",
			"--prompt", "My favorite number is 42. Remember this.",
			"--config", filepath.Join(configDir, "settings.yaml"))
		cmd1.Dir = tempDir

		var stdout1 bytes.Buffer
		cmd1.Stdout = &stdout1

		done1 := make(chan error, 1)
		go func() {
			done1 <- cmd1.Run()
		}()

		select {
		case err := <-done1:
			require.NoError(t, err, "First command should complete successfully")
			t.Logf("First response: %s", stdout1.String())
		case <-time.After(30 * time.Second):
			cmd1.Process.Kill()
			t.Fatal("First command took too long")
		}

		// Second conversation: test context retention with --continue
		cmd2 := exec.Command(binaryPath,
			"--headless",
			"--continue",
			"--prompt", "What was my favorite number?",
			"--config", filepath.Join(configDir, "settings.yaml"))
		cmd2.Dir = tempDir

		var stdout2 bytes.Buffer
		cmd2.Stdout = &stdout2

		done2 := make(chan error, 1)
		go func() {
			done2 <- cmd2.Run()
		}()

		select {
		case err := <-done2:
			require.NoError(t, err, "Second command should complete successfully")

			output := stdout2.String()
			t.Logf("Second response: %s", output)

			// Should remember the number 42
			assert.Contains(t, output, "42", "Agent should remember the favorite number with --continue")

		case <-time.After(30 * time.Second):
			cmd2.Process.Kill()
			t.Fatal("Second command took too long")
		}
	})

	t.Run("It forgets context without --continue", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create a test config directory
		configDir := filepath.Join(tempDir, ".ryan")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// First conversation: establish context
		cmd1 := exec.Command(binaryPath,
			"--headless",
			"--prompt", "My pet's name is Fluffy. Remember this.",
			"--config", filepath.Join(configDir, "settings.yaml"))
		cmd1.Dir = tempDir

		var stdout1 bytes.Buffer
		cmd1.Stdout = &stdout1

		done1 := make(chan error, 1)
		go func() {
			done1 <- cmd1.Run()
		}()

		select {
		case err := <-done1:
			require.NoError(t, err, "First command should complete successfully")
			t.Logf("First response: %s", stdout1.String())
		case <-time.After(30 * time.Second):
			cmd1.Process.Kill()
			t.Fatal("First command took too long")
		}

		// Second conversation: test context is lost without --continue
		cmd2 := exec.Command(binaryPath,
			"--headless",
			"--prompt", "What was my pet's name?",
			"--config", filepath.Join(configDir, "settings.yaml"))
		cmd2.Dir = tempDir

		var stdout2 bytes.Buffer
		cmd2.Stdout = &stdout2

		done2 := make(chan error, 1)
		go func() {
			done2 <- cmd2.Run()
		}()

		select {
		case err := <-done2:
			require.NoError(t, err, "Second command should complete successfully")

			output := stdout2.String()
			outputLower := strings.ToLower(output)
			t.Logf("Second response: %s", output)

			// Should NOT remember Fluffy, might say it doesn't know
			hasAdmissionOfNotKnowing := strings.Contains(outputLower, "don't know") ||
				strings.Contains(outputLower, "not sure") ||
				strings.Contains(outputLower, "didn't mention") ||
				strings.Contains(outputLower, "haven't told") ||
				strings.Contains(outputLower, "no information") ||
				!strings.Contains(outputLower, "fluffy")

			assert.True(t, hasAdmissionOfNotKnowing,
				"Agent should not remember the pet's name without --continue")

		case <-time.After(30 * time.Second):
			cmd2.Process.Kill()
			t.Fatal("Second command took too long")
		}
	})
}

func TestAgentErrorHandling(t *testing.T) {
	t.Run("It handles empty prompt gracefully", func(t *testing.T) {
		binaryPath := buildBinary(t)
		tempDir := t.TempDir()

		// Create a test config directory
		configDir := filepath.Join(tempDir, ".ryan")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// Run with empty prompt (should default to "hello")
		cmd := exec.Command(binaryPath,
			"--headless",
			"--prompt", "",
			"--config", filepath.Join(configDir, "settings.yaml"))
		cmd.Dir = tempDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		done := make(chan error, 1)
		go func() {
			done <- cmd.Run()
		}()

		select {
		case <-done:
			// Should either succeed with default or handle gracefully
			output := stdout.String()
			t.Logf("Response to empty prompt: %s", output)

			// Should have some output (either error message or default response)
			assert.True(t, len(output) > 0 || len(stderr.String()) > 0,
				"Should produce some output for empty prompt")

		case <-time.After(30 * time.Second):
			cmd.Process.Kill()
			t.Fatal("Command took too long")
		}
	})
}
