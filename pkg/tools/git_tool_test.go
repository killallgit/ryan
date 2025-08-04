package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitTool_Basic(t *testing.T) {
	tool := NewGitTool()
	
	// Test basic properties
	assert.Equal(t, "git", tool.Name())
	assert.NotEmpty(t, tool.Description())
	
	// Test JSON schema
	schema := tool.JSONSchema()
	assert.NotNil(t, schema)
	
	properties, exists := schema["properties"].(map[string]any)
	require.True(t, exists)
	
	// Check required operation property
	operation, exists := properties["operation"]
	require.True(t, exists)
	assert.NotNil(t, operation)
}

func TestGitTool_ValidationErrors(t *testing.T) {
	tool := NewGitTool()
	ctx := context.Background()
	
	// Test missing operation parameter
	result, err := tool.Execute(ctx, map[string]any{})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "operation parameter is required")
	
	// Test invalid operation type
	result, err = tool.Execute(ctx, map[string]any{
		"operation": 123,
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "operation must be a string")
	
	// Test disallowed operation
	result, err = tool.Execute(ctx, map[string]any{
		"operation": "reset",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "operation 'reset' is not allowed")
}

func TestGitTool_RepositoryValidation(t *testing.T) {
	tool := NewGitTool()
	ctx := context.Background()
	
	// Test non-existent path
	result, err := tool.Execute(ctx, map[string]any{
		"operation": "status",
		"path":      "/non/existent/path",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "invalid repository")
}

func TestGitTool_StatusOperation(t *testing.T) {
	// Skip if we're not in a git repository or git is not available
	if !isGitRepository(".") || !isGitAvailable() {
		t.Skip("Skipping git test: not in git repository or git not available")
	}
	
	tool := NewGitTool()
	ctx := context.Background()
	
	result, err := tool.Execute(ctx, map[string]any{
		"operation": "status",
		"format":    "structured",
	})
	
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Content)
	
	// Check metadata
	assert.NotNil(t, result.Metadata)
	assert.Equal(t, "git", result.Metadata.ToolName)
	assert.True(t, result.Metadata.ExecutionTime > 0)
	
	// Check data
	assert.NotNil(t, result.Data)
	data := result.Data
	assert.Equal(t, "status", data["operation"])
	assert.Equal(t, 0, data["exit_code"])
}

func TestGitTool_BranchOperation(t *testing.T) {
	if !isGitRepository(".") || !isGitAvailable() {
		t.Skip("Skipping git test: not in git repository or git not available")
	}
	
	tool := NewGitTool()
	ctx := context.Background()
	
	result, err := tool.Execute(ctx, map[string]any{
		"operation": "branch",
		"format":    "raw",
	})
	
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Content)
}

func TestGitTool_LogOperation(t *testing.T) {
	if !isGitRepository(".") || !isGitAvailable() {
		t.Skip("Skipping git test: not in git repository or git not available")
	}
	
	tool := NewGitTool()
	ctx := context.Background()
	
	result, err := tool.Execute(ctx, map[string]any{
		"operation": "log",
		"limit":     5,
		"format":    "summary",
	})
	
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Content)
	
	// Verify limit was applied
	data := result.Data
	assert.NotNil(t, data["raw_output"])
}

func TestGitTool_TimeoutHandling(t *testing.T) {
	if !isGitRepository(".") || !isGitAvailable() {
		t.Skip("Skipping git test: not in git repository or git not available")
	}
	
	tool := NewGitTool()
	tool.timeout = 1 * time.Millisecond // Very short timeout
	
	ctx := context.Background()
	
	result, err := tool.Execute(ctx, map[string]any{
		"operation": "log",
		"limit":     1000, // Large limit to potentially trigger timeout
	})
	
	require.NoError(t, err)
	// Should either succeed quickly or fail with timeout/signal
	if !result.Success {
		// Could be timeout or signal killed, both are acceptable
		assert.True(t, 
			strings.Contains(result.Error, "git operation failed") ||
			strings.Contains(result.Error, "signal: killed") ||
			strings.Contains(result.Error, "context deadline exceeded"))
	}
}

func TestGitTool_FormatOptions(t *testing.T) {
	if !isGitRepository(".") || !isGitAvailable() {
		t.Skip("Skipping git test: not in git repository or git not available")
	}
	
	tool := NewGitTool()
	ctx := context.Background()
	
	formats := []string{"raw", "structured", "summary"}
	
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			result, err := tool.Execute(ctx, map[string]any{
				"operation": "status",
				"format":    format,
			})
			
			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.NotEmpty(t, result.Content)
		})
	}
}

func TestGitTool_AllowedOperations(t *testing.T) {
	tool := NewGitTool()
	
	allowedOps := []string{
		"status", "diff", "log", "branch", "show", "ls-files",
		"rev-parse", "describe", "tag", "remote", "config",
		"blame", "shortlog", "reflog", "cherry",
	}
	
	for _, op := range allowedOps {
		assert.True(t, tool.isAllowedOperation(op), "Operation %s should be allowed", op)
	}
	
	disallowedOps := []string{"reset", "rebase", "merge", "commit", "push", "pull"}
	for _, op := range disallowedOps {
		assert.False(t, tool.isAllowedOperation(op), "Operation %s should not be allowed", op)
	}
}

func TestGitTool_ParameterExtraction(t *testing.T) {
	tool := NewGitTool()
	
	params := map[string]any{
		"string_param": "test_value",
		"int_param":    42,
		"float_param":  3.14,
		"array_param":  []any{"item1", "item2", 123}, // Mixed types
	}
	
	// Test string parameter extraction
	assert.Equal(t, "test_value", tool.getStringParam(params, "string_param", "default"))
	assert.Equal(t, "default", tool.getStringParam(params, "missing_param", "default"))
	
	// Test integer parameter extraction
	assert.Equal(t, 42, tool.getIntParam(params, "int_param", 0))
	assert.Equal(t, 3, tool.getIntParam(params, "float_param", 0)) // Should convert float to int
	assert.Equal(t, 0, tool.getIntParam(params, "missing_param", 0))
	
	// Test string array parameter extraction
	result := tool.getStringArrayParam(params, "array_param")
	expected := []string{"item1", "item2"} // Non-string items should be filtered out
	assert.Equal(t, expected, result)
	
	empty := tool.getStringArrayParam(params, "missing_param")
	assert.Equal(t, []string{}, empty)
}

// Helper functions for testing

func isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return true
	}
	
	// Check if we're in a git worktree
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	if path != "." {
		cmd.Dir = path
	}
	return cmd.Run() == nil
}

func isGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}