package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileWriteTool implements file writing with permission checking
type FileWriteTool struct {
	*SecuredTool
}

// NewFileWriteTool creates a new file write tool
func NewFileWriteTool() *FileWriteTool {
	return NewFileWriteToolWithBypass(false)
}

// NewFileWriteToolWithBypass creates a new file write tool with optional permission bypass
func NewFileWriteToolWithBypass(bypass bool) *FileWriteTool {
	return &FileWriteTool{
		SecuredTool: NewSecuredToolWithBypass(bypass),
	}
}

// Name returns the tool name
func (t *FileWriteTool) Name() string {
	return "file_write"
}

// Description returns the tool description
func (t *FileWriteTool) Description() string {
	return "Write content to a file. Input format: 'path:::content'"
}

// Call executes the file write operation
func (t *FileWriteTool) Call(ctx context.Context, input string) (string, error) {
	// Parse input format: "path:::content"
	parts := strings.SplitN(input, ":::", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid format, use: path:::content")
	}

	path := strings.TrimSpace(parts[0])
	content := parts[1] // Don't trim content - preserve formatting

	if path == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	// Validate permissions
	if err := t.ValidateAccess("FileWrite", path); err != nil {
		return "", err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create backup if file exists
	if stat, err := os.Stat(path); err == nil && stat.Mode().IsRegular() {
		backupPath := fmt.Sprintf("%s.backup.%d", path, time.Now().Unix())
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read existing file for backup: %w", err)
		}
		if err := os.WriteFile(backupPath, data, stat.Mode()); err != nil {
			// Log warning but don't fail
			fmt.Fprintf(os.Stderr, "Warning: failed to create backup: %v\n", err)
		}
	}

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}
