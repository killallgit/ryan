package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tmc/langchaingo/documentloaders"
)

// FileReadTool implements file reading with permission checking
type FileReadTool struct {
	*SecuredTool
}

// NewFileReadTool creates a new file read tool
func NewFileReadTool() *FileReadTool {
	return NewFileReadToolWithBypass(false)
}

// NewFileReadToolWithBypass creates a new file read tool with optional permission bypass
func NewFileReadToolWithBypass(bypass bool) *FileReadTool {
	return &FileReadTool{
		SecuredTool: NewSecuredToolWithBypass(bypass),
	}
}

// Name returns the tool name
func (t *FileReadTool) Name() string {
	return "file_read"
}

// Description returns the tool description
func (t *FileReadTool) Description() string {
	return "Read file contents. Input: file path"
}

// Call executes the file read operation
func (t *FileReadTool) Call(ctx context.Context, input string) (string, error) {
	// Trim whitespace from input
	path := strings.TrimSpace(input)
	if path == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	// Validate permissions
	if err := t.ValidateAccess("FileRead", path); err != nil {
		return "", err
	}

	// Check if file exists
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist: %s", path)
		}
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if it's a regular file
	if !stat.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file: %s", path)
	}

	// Check file size (limit to 10MB)
	const maxSize = 10 * 1024 * 1024
	if stat.Size() > maxSize {
		return "", fmt.Errorf("file too large: %d bytes (max %d bytes)", stat.Size(), maxSize)
	}

	// Open file
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Use langchain document loader
	loader := documentloaders.NewText(file)
	docs, err := loader.Load(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load document: %w", err)
	}

	// Return content from first document
	if len(docs) > 0 {
		return docs[0].PageContent, nil
	}

	return "", nil
}
