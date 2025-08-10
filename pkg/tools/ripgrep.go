package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RipgrepTool implements code searching with ripgrep
type RipgrepTool struct {
	*SecuredTool
	timeout     time.Duration
	maxResults  int
	maxFileSize string
}

// NewRipgrepTool creates a new ripgrep tool
func NewRipgrepTool() *RipgrepTool {
	return NewRipgrepToolWithBypass(false)
}

// NewRipgrepToolWithBypass creates a new ripgrep tool with optional permission bypass
func NewRipgrepToolWithBypass(bypass bool) *RipgrepTool {
	return &RipgrepTool{
		SecuredTool: NewSecuredToolWithBypass(bypass),
		timeout:     30 * time.Second,
		maxResults:  1000,
		maxFileSize: "50M",
	}
}

// Name returns the tool name
func (t *RipgrepTool) Name() string {
	return "search"
}

// Description returns the tool description
func (t *RipgrepTool) Description() string {
	return "Search files using ripgrep. Input: search pattern (supports regex)"
}

// Call executes the ripgrep search
func (t *RipgrepTool) Call(ctx context.Context, input string) (string, error) {
	// Trim whitespace from input
	pattern := strings.TrimSpace(input)
	if pattern == "" {
		return "", fmt.Errorf("search pattern cannot be empty")
	}

	// Validate permissions
	if err := t.ValidateAccess("Ripgrep", pattern); err != nil {
		return "", err
	}

	// Check if ripgrep is available
	if !t.isRipgrepAvailable() {
		return t.fallbackToGrep(ctx, pattern)
	}

	// Build ripgrep command with safety constraints
	args := []string{
		"--max-count", fmt.Sprintf("%d", t.maxResults),
		"--max-filesize", t.maxFileSize,
		"--line-number",
		"--with-filename",
		"--color", "never", // Disable color for clean output
		// Exclude common directories
		"--glob", "!.git/**",
		"--glob", "!node_modules/**",
		"--glob", "!vendor/**",
		"--glob", "!__pycache__/**",
		"--glob", "!*.pyc",
		"--glob", "!dist/**",
		"--glob", "!build/**",
		"--glob", "!target/**",
		"--glob", "!*.min.js",
		"--glob", "!*.min.css",
		pattern,
	}

	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Execute ripgrep
	cmd := exec.CommandContext(cmdCtx, "rg", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Get output
	output := stdout.String()
	if stderr.Len() > 0 && err != nil {
		// Only include stderr if there was an error
		output += "\n" + stderr.String()
	}

	// Handle errors
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return output, fmt.Errorf("search timed out after %v", t.timeout)
		}

		// Exit code 1 means no matches found (not an error)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "No matches found", nil
		}

		return output, fmt.Errorf("ripgrep failed: %w", err)
	}

	// Check if we hit the result limit
	lineCount := strings.Count(output, "\n")
	if lineCount >= t.maxResults {
		output += fmt.Sprintf("\n\n[Results truncated at %d matches]", t.maxResults)
	}

	if output == "" {
		return "No matches found", nil
	}

	return output, nil
}

// isRipgrepAvailable checks if ripgrep is installed
func (t *RipgrepTool) isRipgrepAvailable() bool {
	cmd := exec.Command("which", "rg")
	err := cmd.Run()
	return err == nil
}

// fallbackToGrep uses standard grep if ripgrep is not available
func (t *RipgrepTool) fallbackToGrep(ctx context.Context, pattern string) (string, error) {
	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Build grep command
	args := []string{
		"-r", // Recursive
		"-n", // Line numbers
		"-I", // Skip binary files
		"--exclude-dir=.git",
		"--exclude-dir=node_modules",
		"--exclude-dir=vendor",
		"--exclude-dir=__pycache__",
		"--exclude-dir=dist",
		"--exclude-dir=build",
		"--exclude-dir=target",
		"--exclude=*.pyc",
		"--exclude=*.min.js",
		"--exclude=*.min.css",
		pattern,
		".",
	}

	cmd := exec.CommandContext(cmdCtx, "grep", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return output, fmt.Errorf("search timed out after %v", t.timeout)
		}

		// Exit code 1 means no matches found (not an error)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "No matches found (using grep fallback)", nil
		}

		return output, fmt.Errorf("grep failed: %w", err)
	}

	// Limit results
	lines := strings.Split(output, "\n")
	if len(lines) > t.maxResults {
		lines = lines[:t.maxResults]
		output = strings.Join(lines, "\n")
		output += fmt.Sprintf("\n\n[Results truncated at %d matches (using grep fallback)]", t.maxResults)
	}

	if output == "" {
		return "No matches found (using grep fallback)", nil
	}

	return output + "\n\n[Note: Using grep fallback - install ripgrep for better performance]", nil
}
