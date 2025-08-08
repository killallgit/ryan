package integration

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOllamaRequired ensures Ollama is available before any tests run
func TestOllamaRequired(t *testing.T) {
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		t.Fatal("OLLAMA_HOST environment variable is required for integration tests")
	}

	// Verify Ollama is actually responding
	resp, err := http.Get(ollamaHost + "/api/tags")
	if err != nil {
		t.Fatalf("Failed to connect to Ollama at %s: %v", ollamaHost, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Ollama not healthy at %s: status %d", ollamaHost, resp.StatusCode)
	}

	t.Logf("Ollama is available at %s", ollamaHost)
}

func TestTokenCountingWithOllama(t *testing.T) {
	// This will fail if OLLAMA_HOST is not set
	ollamaHost := os.Getenv("OLLAMA_HOST")
	require.NotEmpty(t, ollamaHost, "OLLAMA_HOST must be set")

	binaryPath := buildBinary(t)
	tempDir := t.TempDir()

	// Create config directory
	configDir := filepath.Join(tempDir, ".ryan")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Run with simple prompt
	cmd := exec.Command(binaryPath,
		"--headless",
		"--prompt", "Say hello in exactly 3 words",
		"--config", filepath.Join(configDir, "settings.yaml"))
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "OLLAMA_HOST="+ollamaHost)

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
		if err != nil {
			t.Logf("stdout: %s", stdout.String())
			t.Logf("stderr: %s", stderr.String())
			t.Fatalf("Command failed: %v", err)
		}

		output := stdout.String() + stderr.String()
		t.Logf("Output: %s", output)

		// Verify token counts appear (check both cases)
		hasTokens := strings.Contains(output, "tokens") || strings.Contains(output, "Tokens")
		require.True(t, hasTokens, "Output must contain token information")

		// Parse and verify token counts
		sent, recv := parseTokenCounts(t, output)
		require.Greater(t, sent, 0, "Must have sent tokens")
		require.Greater(t, recv, 0, "Must have received tokens")

	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		t.Fatal("Command timed out - Ollama may not be responding")
	}
}

func TestStreamingTokenUpdates(t *testing.T) {
	ollamaHost := os.Getenv("OLLAMA_HOST")
	require.NotEmpty(t, ollamaHost, "OLLAMA_HOST must be set")

	binaryPath := buildBinary(t)
	tempDir := t.TempDir()

	configDir := filepath.Join(tempDir, ".ryan")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Use context for timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath,
		"--headless",
		"--prompt", "Count from 1 to 5 slowly",
		"--config", filepath.Join(configDir, "settings.yaml"))
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "OLLAMA_HOST="+ollamaHost)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	// Parse token counts
	sent, recv := parseTokenCounts(t, string(output))
	t.Logf("Tokens - Sent: %d, Received: %d", sent, recv)

	// Require minimum token counts
	require.GreaterOrEqual(t, sent, 5, "Prompt must be at least 5 tokens")
	require.GreaterOrEqual(t, recv, 5, "Response must be at least 5 tokens")
}

func TestMultiTurnConversationTokens(t *testing.T) {
	ollamaHost := os.Getenv("OLLAMA_HOST")
	require.NotEmpty(t, ollamaHost, "OLLAMA_HOST must be set")

	// Test that tokens accumulate across multiple turns
	binaryPath := buildBinary(t)
	tempDir := t.TempDir()

	configDir := filepath.Join(tempDir, ".ryan")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Create initial conversation
	historyFile := filepath.Join(configDir, "chat_history.json")
	history := `[{"role": "user", "content": "Hello"}, {"role": "assistant", "content": "Hi there! How can I help you today?"}]`
	err = os.WriteFile(historyFile, []byte(history), 0644)
	require.NoError(t, err)

	// Continue conversation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath,
		"--headless",
		"--continue",
		"--prompt", "What did I just say?",
		"--config", filepath.Join(configDir, "settings.yaml"))
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "OLLAMA_HOST="+ollamaHost)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	sent, recv := parseTokenCounts(t, string(output))

	// Should have more tokens due to history
	require.Greater(t, sent, 10, "With history, should send more tokens")
	require.Greater(t, recv, 0, "Should receive response tokens")
}

func TestTokenDisplayFormat(t *testing.T) {
	ollamaHost := os.Getenv("OLLAMA_HOST")
	require.NotEmpty(t, ollamaHost, "OLLAMA_HOST must be set")

	binaryPath := buildBinary(t)
	tempDir := t.TempDir()

	configDir := filepath.Join(tempDir, ".ryan")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Run a simple command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath,
		"--headless",
		"--prompt", "Hi",
		"--config", filepath.Join(configDir, "settings.yaml"))
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "OLLAMA_HOST="+ollamaHost)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Logf("stdout: %s", stdout.String())
		t.Logf("stderr: %s", stderr.String())
		t.Fatalf("Command failed: %v", err)
	}

	output := stdout.String() + stderr.String()

	// Check for token display in output (case insensitive)
	outputLower := strings.ToLower(output)
	assert.Contains(t, outputLower, "token", "Output should mention tokens")

	sent, recv := parseTokenCounts(t, output)
	t.Logf("Token display format test - Sent: %d, Received: %d", sent, recv)

	// Both should be non-zero
	assert.Greater(t, sent, 0, "Should display sent tokens")
	assert.Greater(t, recv, 0, "Should display received tokens")
}

func TestTokenCountAccuracy(t *testing.T) {
	ollamaHost := os.Getenv("OLLAMA_HOST")
	require.NotEmpty(t, ollamaHost, "OLLAMA_HOST must be set")

	binaryPath := buildBinary(t)
	tempDir := t.TempDir()

	configDir := filepath.Join(tempDir, ".ryan")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Test with a known prompt length
	testPrompt := "Write exactly the word 'test' and nothing else"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath,
		"--headless",
		"--prompt", testPrompt,
		"--config", filepath.Join(configDir, "settings.yaml"))
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "OLLAMA_HOST="+ollamaHost)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	sent, recv := parseTokenCounts(t, string(output))
	t.Logf("Accuracy test - Prompt: '%s'", testPrompt)
	t.Logf("Tokens - Sent: %d, Received: %d", sent, recv)

	// The prompt should be roughly 8-12 tokens depending on tokenizer
	assert.GreaterOrEqual(t, sent, 8, "Sent tokens seem too low for prompt")
	assert.LessOrEqual(t, sent, 15, "Sent tokens seem too high for prompt")

	// Response should be minimal (just "test" or similar)
	assert.GreaterOrEqual(t, recv, 1, "Should have at least 1 token in response")
	assert.LessOrEqual(t, recv, 10, "Response should be short")
}

// verifyOllamaConnection ensures Ollama is reachable
func verifyOllamaConnection(url string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url + "/api/tags")
	if err != nil {
		return fmt.Errorf("cannot connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}
	return nil
}
