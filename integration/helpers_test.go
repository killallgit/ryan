package integration

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// parseTokenCounts extracts sent and received token counts from output
func parseTokenCounts(t *testing.T, output string) (sent, recv int) {
	// Try multiple patterns to find token counts

	// Pattern 1: Look for "X tokens" format (total display)
	tokenPattern := regexp.MustCompile(`(\d+)\s+tokens`)
	matches := tokenPattern.FindAllStringSubmatch(output, -1)

	if len(matches) >= 1 {
		if val, err := strconv.Atoi(matches[0][1]); err == nil {
			// If we only find one count, it might be the total
			// We'll need to look for more specific patterns
			if len(matches) == 1 {
				// This is likely a total, try to find sent/recv separately
			} else {
				sent = val
			}
		}
	}
	if len(matches) >= 2 {
		if val, err := strconv.Atoi(matches[1][1]); err == nil {
			recv = val
		}
	}

	// Pattern 2: Look for our specific format "[Tokens - Sent: X, Received: Y, Total: Z]"
	bracketPattern := regexp.MustCompile(`\[Tokens - Sent:\s*(\d+),\s*Received:\s*(\d+),\s*Total:\s*\d+\]`)
	if match := bracketPattern.FindStringSubmatch(output); match != nil {
		if s, err := strconv.Atoi(match[1]); err == nil {
			sent = s
		}
		if r, err := strconv.Atoi(match[2]); err == nil {
			recv = r
		}
		t.Logf("Found bracket pattern: sent=%d, recv=%d", sent, recv)
		return sent, recv
	}

	// Pattern 3: Look for "sent: X, recv: Y" or similar patterns
	sentRecvPattern := regexp.MustCompile(`(?i)(?:sent|sending)[\s:]*(\d+).*?(?:recv|received|receiving)[\s:]*(\d+)`)
	if match := sentRecvPattern.FindStringSubmatch(output); match != nil {
		if s, err := strconv.Atoi(match[1]); err == nil {
			sent = s
		}
		if r, err := strconv.Atoi(match[2]); err == nil {
			recv = r
		}
		t.Logf("Found sent/recv pattern: sent=%d, recv=%d", sent, recv)
		return sent, recv
	}

	// Pattern 3: Look for separate "Sent:" and "Received:" lines
	sentPattern := regexp.MustCompile(`(?i)(?:tokens?\s+)?sent[\s:]+(\d+)`)
	recvPattern := regexp.MustCompile(`(?i)(?:tokens?\s+)?(?:received?|recv)[\s:]+(\d+)`)

	if match := sentPattern.FindStringSubmatch(output); match != nil {
		if s, err := strconv.Atoi(match[1]); err == nil {
			sent = s
		}
	}

	if match := recvPattern.FindStringSubmatch(output); match != nil {
		if r, err := strconv.Atoi(match[1]); err == nil {
			recv = r
		}
	}

	// Pattern 4: Look in stderr/debug output for token updates
	// The UpdateTokensMsg might appear in debug output
	updatePattern := regexp.MustCompile(`UpdateTokensMsg\{Sent:(\d+)\s+Recv:(\d+)\}`)
	if match := updatePattern.FindStringSubmatch(output); match != nil {
		if s, err := strconv.Atoi(match[1]); err == nil {
			sent = s
		}
		if r, err := strconv.Atoi(match[2]); err == nil {
			recv = r
		}
		t.Logf("Found token update message: sent=%d, recv=%d", sent, recv)
		return sent, recv
	}

	// Pattern 5: Status bar format (might appear as "12 tokens" for total)
	// In this case, we might see the accumulation
	// Look for lines that might indicate token counts during streaming
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Skip empty lines
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for lines containing token information
		if strings.Contains(strings.ToLower(line), "token") {
			t.Logf("Token line found: %s", line)

			// Try to extract numbers from this line
			numberPattern := regexp.MustCompile(`\d+`)
			numbers := numberPattern.FindAllString(line, -1)
			if len(numbers) > 0 {
				// If we haven't found sent/recv yet, try to use these
				if sent == 0 && len(numbers) >= 1 {
					if val, err := strconv.Atoi(numbers[0]); err == nil && val > 0 {
						sent = val
					}
				}
				if recv == 0 && len(numbers) >= 2 {
					if val, err := strconv.Atoi(numbers[1]); err == nil && val > 0 {
						recv = val
					}
				}
			}
		}
	}

	// If we still haven't found values, look for any reasonable numbers
	// This is a fallback for when the format might be different
	if sent == 0 || recv == 0 {
		// Look for prompt/response indicators with numbers
		promptPattern := regexp.MustCompile(`(?i)prompt.*?(\d+)`)
		responsePattern := regexp.MustCompile(`(?i)response.*?(\d+)`)

		if match := promptPattern.FindStringSubmatch(output); match != nil && sent == 0 {
			if s, err := strconv.Atoi(match[1]); err == nil {
				sent = s
			}
		}

		if match := responsePattern.FindStringSubmatch(output); match != nil && recv == 0 {
			if r, err := strconv.Atoi(match[1]); err == nil {
				recv = r
			}
		}
	}

	t.Logf("Parsed tokens from output - Sent: %d, Received: %d", sent, recv)

	// Log a sample of the output for debugging
	if sent == 0 && recv == 0 {
		// Show first and last 500 chars of output for debugging
		sample := output
		if len(output) > 1000 {
			sample = output[:500] + "\n...[truncated]...\n" + output[len(output)-500:]
		}
		t.Logf("Could not parse tokens from output. Sample:\n%s", sample)
	}

	return sent, recv
}


// setupTestConfig copies the shared test configuration to a temporary directory
// and returns the config file path. It ensures OLLAMA_HOST is properly set.
func setupTestConfig(t *testing.T, tempDir string) string {
	configDir := filepath.Join(tempDir, ".ryan")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Copy the shared test configuration
	sourceConfig := filepath.Join(".", "settings.test.yaml")
	targetConfig := filepath.Join(configDir, "settings.yaml")

	sourceFile, err := os.Open(sourceConfig)
	require.NoError(t, err)
	defer sourceFile.Close()

	targetFile, err := os.Create(targetConfig)
	require.NoError(t, err)
	defer targetFile.Close()

	_, err = io.Copy(targetFile, sourceFile)
	require.NoError(t, err)

	return targetConfig
}

// setupEnvWithOverrides creates environment with config overrides
// PANICS if OLLAMA_HOST is not set - integration tests require this
func setupEnvWithOverrides(overrides map[string]string) []string {
	env := setupEnvForOllama() // This will panic if OLLAMA_HOST not set

	// Add config overrides as environment variables
	// Viper will pick these up and override the config file values
	for key, value := range overrides {
		// Convert config keys to environment variable format
		// e.g., "memory_window_size" -> "LANGCHAIN_MEMORY_WINDOW_SIZE"
		envKey := ""
		switch key {
		case "memory_window_size":
			envKey = "LANGCHAIN_MEMORY_WINDOW_SIZE"
		case "memory_type":
			envKey = "LANGCHAIN_MEMORY_TYPE"
		default:
			// Generic conversion: convert dots/underscores to uppercase env format
			envKey = strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		}
		env = append(env, envKey+"="+value)
	}

	return env
}

// setupEnvForOllama ensures OLLAMA_HOST environment variable is available for commands
// PANICS if OLLAMA_HOST is not set - integration tests require this
func setupEnvForOllama() []string {
	env := os.Environ()

	// Check if OLLAMA_HOST is already set
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		panic("OLLAMA_HOST environment variable must be set for integration tests")
	}

	// Ensure OLLAMA_HOST is in the environment for the command
	// This will override the config file's ollama.url setting
	found := false
	for _, envVar := range env {
		if strings.HasPrefix(envVar, "OLLAMA_HOST=") {
			found = true
			break
		}
	}

	if !found {
		env = append(env, "OLLAMA_HOST="+ollamaHost)
	}

	return env
}
