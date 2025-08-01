package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// FileReadTool reads file contents with safety constraints
type FileReadTool struct {
	// AllowedPaths are directories where files can be read
	AllowedPaths []string
	
	// AllowedExtensions are file extensions that are allowed to be read
	AllowedExtensions []string
	
	// MaxFileSize is the maximum file size in bytes
	MaxFileSize int64
	
	// MaxLines is the maximum number of lines to read
	MaxLines int
}

// NewFileReadTool creates a new FileReadTool with default safety settings
func NewFileReadTool() *FileReadTool {
	home, _ := os.UserHomeDir()
	wd, _ := os.Getwd()
	
	// Common text and code file extensions
	allowedExtensions := []string{
		".txt", ".md", ".markdown",
		".go", ".py", ".js", ".ts", ".jsx", ".tsx",
		".c", ".cpp", ".h", ".hpp", ".cc", ".cxx",
		".java", ".kt", ".scala", ".clj", ".cljs",
		".rb", ".php", ".swift", ".rs", ".zig",
		".sh", ".bash", ".zsh", ".fish",
		".json", ".yaml", ".yml", ".toml", ".xml",
		".html", ".htm", ".css", ".scss", ".sass",
		".sql", ".csv", ".tsv",
		".ini", ".conf", ".config", ".cfg",
		".log", ".gitignore", ".dockerignore",
		".env", ".example",
	}
	
	return &FileReadTool{
		AllowedPaths: []string{
			home,
			wd,
		},
		AllowedExtensions: allowedExtensions,
		MaxFileSize:       10 * 1024 * 1024, // 10MB
		MaxLines:          10000,             // 10k lines
	}
}

// Name returns the tool name
func (ft *FileReadTool) Name() string {
	return "read_file"
}

// Description returns the tool description
func (ft *FileReadTool) Description() string {
	return "Read the contents of a text file. This tool can read source code, configuration files, documentation, and other text-based files."
}

// JSONSchema returns the JSON schema for the tool parameters
func (ft *FileReadTool) JSONSchema() map[string]interface{} {
	schema := NewJSONSchema()
	
	AddProperty(schema, "path", JSONSchemaProperty{
		Type:        "string",
		Description: "The path to the file to read",
	})
	
	AddProperty(schema, "start_line", JSONSchemaProperty{
		Type:        "number",
		Description: "Optional: line number to start reading from (1-based)",
		Default:     1,
	})
	
	AddProperty(schema, "end_line", JSONSchemaProperty{
		Type:        "number",
		Description: "Optional: line number to stop reading at (1-based)",
	})
	
	AddRequired(schema, "path")
	
	return schema
}

// Execute reads the file contents
func (ft *FileReadTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	startTime := time.Now()
	
	// Extract path parameter
	pathInterface, exists := params["path"]
	if !exists {
		return ft.createErrorResult(startTime, "path parameter is required"), nil
	}
	
	path, ok := pathInterface.(string)
	if !ok {
		return ft.createErrorResult(startTime, "path parameter must be a string"), nil
	}
	
	if strings.TrimSpace(path) == "" {
		return ft.createErrorResult(startTime, "path cannot be empty"), nil
	}
	
	// Validate file path
	if err := ft.validatePath(path); err != nil {
		return ft.createErrorResult(startTime, err.Error()), nil
	}
	
	// Get optional line range parameters
	startLine := 1
	endLine := -1
	
	if startLineInterface, exists := params["start_line"]; exists {
		if sl, ok := startLineInterface.(float64); ok {
			startLine = int(sl)
		}
	}
	
	if endLineInterface, exists := params["end_line"]; exists {
		if el, ok := endLineInterface.(float64); ok {
			endLine = int(el)
		}
	}
	
	// Validate line numbers
	if startLine < 1 {
		return ft.createErrorResult(startTime, "start_line must be >= 1"), nil
	}
	
	if endLine != -1 && endLine < startLine {
		return ft.createErrorResult(startTime, "end_line must be >= start_line"), nil
	}
	
	// Read file
	content, err := ft.readFile(path, startLine, endLine)
	endTime := time.Now()
	
	result := ToolResult{
		Success: err == nil,
		Content: content,
		Metadata: ToolMetadata{
			ExecutionTime: endTime.Sub(startTime),
			StartTime:     startTime,
			EndTime:       endTime,
			ToolName:      ft.Name(),
			Parameters:    params,
		},
	}
	
	if err != nil {
		result.Error = err.Error()
		result.Success = false
	}
	
	return result, nil
}

// validatePath checks if a file path is safe to read
func (ft *FileReadTool) validatePath(path string) error {
	// Clean and make absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	
	// Check if file exists
	fileInfo, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", absPath)
	}
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	
	// Check if it's a regular file
	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("path is not a regular file: %s", absPath)
	}
	
	// Check file size
	if fileInfo.Size() > ft.MaxFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d bytes)", fileInfo.Size(), ft.MaxFileSize)
	}
	
	// Check file extension
	ext := strings.ToLower(filepath.Ext(absPath))
	if len(ft.AllowedExtensions) > 0 {
		allowed := false
		for _, allowedExt := range ft.AllowedExtensions {
			if ext == allowedExt {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file extension not allowed: %s (allowed: %v)", ext, ft.AllowedExtensions)
		}
	}
	
	// Check if path is within allowed paths
	for _, allowed := range ft.AllowedPaths {
		allowedAbs, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		
		rel, err := filepath.Rel(allowedAbs, absPath)
		if err != nil {
			continue
		}
		
		// If the relative path doesn't start with "..", it's within the allowed path
		if !strings.HasPrefix(rel, "..") {
			return nil
		}
	}
	
	return fmt.Errorf("file not within allowed directories: %s", absPath)
}

// readFile reads file contents with optional line range
func (ft *FileReadTool) readFile(path string, startLine, endLine int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	// Read the entire file
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	
	// Check if content is valid UTF-8
	if !utf8.Valid(content) {
		return "", fmt.Errorf("file contains invalid UTF-8 content (binary file?)")
	}
	
	contentStr := string(content)
	
	// If no line range specified, return entire content
	if startLine == 1 && endLine == -1 {
		// Check line count limit
		lineCount := strings.Count(contentStr, "\n") + 1
		if lineCount > ft.MaxLines {
			return "", fmt.Errorf("file has too many lines: %d (max %d)", lineCount, ft.MaxLines)
		}
		return contentStr, nil
	}
	
	// Split content into lines for range selection
	lines := strings.Split(contentStr, "\n")
	totalLines := len(lines)
	
	// Validate line range
	if startLine > totalLines {
		return "", fmt.Errorf("start_line %d exceeds file length %d", startLine, totalLines)
	}
	
	if endLine == -1 {
		endLine = totalLines
	}
	
	if endLine > totalLines {
		endLine = totalLines
	}
	
	// Check that we're not returning too many lines
	requestedLines := endLine - startLine + 1
	if requestedLines > ft.MaxLines {
		return "", fmt.Errorf("requested range too large: %d lines (max %d)", requestedLines, ft.MaxLines)
	}
	
	// Extract the requested line range (convert to 0-based indexing)
	selectedLines := lines[startLine-1 : endLine]
	
	return strings.Join(selectedLines, "\n"), nil
}

// createErrorResult creates a ToolResult for an error case
func (ft *FileReadTool) createErrorResult(startTime time.Time, errorMsg string) ToolResult {
	endTime := time.Now()
	return ToolResult{
		Success: false,
		Error:   errorMsg,
		Metadata: ToolMetadata{
			ExecutionTime: endTime.Sub(startTime),
			StartTime:     startTime,
			EndTime:       endTime,
			ToolName:      ft.Name(),
		},
	}
}