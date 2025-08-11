package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GitTool implements git operations with permission checking
type GitTool struct {
	*SecuredTool
	timeout time.Duration
}

// NewGitTool creates a new git tool
func NewGitTool() *GitTool {
	return NewGitToolWithBypass(false)
}

// NewGitToolWithBypass creates a new git tool with optional permission bypass
func NewGitToolWithBypass(bypass bool) *GitTool {
	return &GitTool{
		SecuredTool: NewSecuredToolWithBypass(bypass),
		timeout:     30 * time.Second,
	}
}

// Name returns the tool name
func (t *GitTool) Name() string {
	return "git"
}

// Description returns the tool description
func (t *GitTool) Description() string {
	return "Execute git commands (read-only by default). Input: git command (e.g., 'status', 'diff HEAD', 'log -n 10')"
}

// Call executes the git command
func (t *GitTool) Call(ctx context.Context, input string) (string, error) {
	// Trim whitespace from input
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("git command cannot be empty")
	}

	// Validate permissions - check the full command
	if err := t.ValidateAccess("Git", input); err != nil {
		return "", err
	}

	// Parse command into arguments
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", fmt.Errorf("no git command provided")
	}

	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Execute git command
	cmd := exec.CommandContext(cmdCtx, "git", parts...)

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
			return output, fmt.Errorf("git command timed out after %v", t.timeout)
		}
		// Include git's error output in the error message
		if output != "" {
			return output, fmt.Errorf("git command failed: %w\nOutput: %s", err, output)
		}
		return "", fmt.Errorf("git command failed: %w", err)
	}

	// Return output even if empty (some commands have no output when successful)
	if output == "" {
		output = fmt.Sprintf("Git command '%s' completed successfully (no output)", input)
	}

	return output, nil
}
