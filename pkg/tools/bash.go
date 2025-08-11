package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BashTool implements bash command execution with permission checking
type BashTool struct {
	*SecuredTool
	timeout time.Duration
}

// NewBashTool creates a new bash tool
func NewBashTool() *BashTool {
	return NewBashToolWithBypass(false)
}

// NewBashToolWithBypass creates a new bash tool with optional permission bypass
func NewBashToolWithBypass(bypass bool) *BashTool {
	return &BashTool{
		SecuredTool: NewSecuredToolWithBypass(bypass),
		timeout:     30 * time.Second,
	}
}

// Name returns the tool name
func (t *BashTool) Name() string {
	return "bash"
}

// Description returns the tool description
func (t *BashTool) Description() string {
	return "Execute bash shell commands to interact with the file system and run system utilities. Use this to count files, check directory contents, search for patterns, or perform system operations. Input: bash command (e.g., 'ls -la', 'wc -l file.txt', 'find . -name \"*.go\" | wc -l')"
}

// Call executes the bash command
func (t *BashTool) Call(ctx context.Context, input string) (string, error) {
	// Trim whitespace from input
	command := strings.TrimSpace(input)
	if command == "" {
		return "", fmt.Errorf("bash command cannot be empty")
	}

	// Validate permissions - check the command
	if err := t.ValidateAccess("Bash", command); err != nil {
		return "", err
	}

	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Execute bash command using sh -c to support pipes, redirects, etc.
	// This allows complex commands like "ls | wc -l" to work properly
	cmd := exec.CommandContext(cmdCtx, "sh", "-c", command)

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	// Combine output
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	// Handle errors
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return output, fmt.Errorf("bash command timed out after %v", t.timeout)
		}
		// Include command's error output in the error message
		if output != "" {
			return output, fmt.Errorf("bash command failed: %w\nOutput: %s", err, output)
		}
		return "", fmt.Errorf("bash command failed: %w", err)
	}

	// Return output even if empty (some commands have no output when successful)
	if output == "" {
		output = fmt.Sprintf("Command '%s' completed successfully (no output)", command)
	}

	return strings.TrimSpace(output), nil
}
