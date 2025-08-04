package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// WriteTool implements safe file writing with backup functionality
type WriteTool struct {
	log               *logger.Logger
	maxFileSize       int64
	createBackups     bool
	backupDir         string
	allowedExtensions []string
	restrictedPaths   []string
}

// NewWriteTool creates a new Write tool with default configuration
func NewWriteTool() *WriteTool {
	return &WriteTool{
		log:           logger.WithComponent("write_tool"),
		maxFileSize:   100 * 1024 * 1024, // 100MB limit
		createBackups: true,
		backupDir:     ".backups",
		allowedExtensions: []string{
			".txt", ".md", ".go", ".js", ".ts", ".py", ".java", ".c", ".cpp", ".h",
			".css", ".html", ".xml", ".json", ".yaml", ".yml", ".toml", ".ini",
			".sh", ".bat", ".ps1", ".sql", ".r", ".rb", ".php", ".kt", ".swift",
			".rs", ".scala", ".clj", ".hs", ".elm", ".dart", ".vue", ".jsx", ".tsx",
			".conf", ".config", ".env", ".gitignore", ".gitattributes", ".dockerfile",
		},
		restrictedPaths: []string{
			"/etc", "/usr", "/var", "/sys", "/proc", "/dev", "/boot", "/root",
			"node_modules", ".git", ".svn", ".hg", "vendor", "target", "build",
			"dist", "out", ".idea", ".vscode", "__pycache__", ".pytest_cache",
		},
	}
}

// Name returns the tool name
func (wt *WriteTool) Name() string {
	return "write_file"
}

// Description returns the tool description
func (wt *WriteTool) Description() string {
	return "Safely write content to files with automatic backup creation. Supports creating new files and editing existing ones. Includes safety checks for file extensions and paths to prevent accidental system file modification."
}

// JSONSchema returns the JSON schema for the tool parameters
func (wt *WriteTool) JSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to write (absolute or relative)",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
			"create_backup": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to create a backup of existing file before writing",
				"default":     true,
			},
			"append": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to append to existing file instead of overwriting",
				"default":     false,
			},
			"create_dirs": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to create parent directories if they don't exist",
				"default":     true,
			},
			"force": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to bypass some safety checks (use with caution)",
				"default":     false,
			},
			"encoding": map[string]interface{}{
				"type":        "string",
				"description": "File encoding (utf-8, ascii, etc.)",
				"default":     "utf-8",
			},
			"line_ending": map[string]interface{}{
				"type":        "string",
				"description": "Line ending style (lf, crlf, cr)",
				"enum":        []string{"lf", "crlf", "cr"},
				"default":     "lf",
			},
		},
		"required": []string{"file_path", "content"},
	}
}

// Execute performs the file write operation
func (wt *WriteTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	startTime := time.Now()

	// Extract and validate parameters
	filePath, exists := params["file_path"]
	if !exists {
		return wt.createErrorResult(startTime, "file_path parameter is required"), nil
	}

	filePathStr, ok := filePath.(string)
	if !ok {
		return wt.createErrorResult(startTime, "file_path must be a string"), nil
	}

	content, exists := params["content"]
	if !exists {
		return wt.createErrorResult(startTime, "content parameter is required"), nil
	}

	contentStr, ok := content.(string)
	if !ok {
		return wt.createErrorResult(startTime, "content must be a string"), nil
	}

	// Extract optional parameters
	createBackup := wt.getBoolParam(params, "create_backup", true)
	appendMode := wt.getBoolParam(params, "append", false)
	createDirs := wt.getBoolParam(params, "create_dirs", true)
	force := wt.getBoolParam(params, "force", false)
	encoding := wt.getStringParam(params, "encoding", "utf-8")
	lineEnding := wt.getStringParam(params, "line_ending", "lf")

	// Clean and validate file path
	cleanPath := filepath.Clean(filePathStr)
	if !filepath.IsAbs(cleanPath) {
		workingDir, err := os.Getwd()
		if err != nil {
			return wt.createErrorResult(startTime, fmt.Sprintf("failed to get working directory: %v", err)), nil
		}
		cleanPath = filepath.Join(workingDir, cleanPath)
	}

	// Safety checks
	if !force {
		if err := wt.validatePath(cleanPath); err != nil {
			return wt.createErrorResult(startTime, err.Error()), nil
		}

		if err := wt.validateFileExtension(cleanPath); err != nil {
			return wt.createErrorResult(startTime, err.Error()), nil
		}

		if len(contentStr) > int(wt.maxFileSize) {
			return wt.createErrorResult(startTime, fmt.Sprintf("content size exceeds maximum allowed size of %d bytes", wt.maxFileSize)), nil
		}
	}

	// Check if file exists
	fileExists := false
	var originalInfo os.FileInfo
	if info, err := os.Stat(cleanPath); err == nil {
		fileExists = true
		originalInfo = info
	}

	// Create parent directories if needed
	if createDirs {
		parentDir := filepath.Dir(cleanPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return wt.createErrorResult(startTime, fmt.Sprintf("failed to create parent directories: %v", err)), nil
		}
	}

	// Create backup if requested and file exists
	var backupPath string
	if createBackup && fileExists && !appendMode {
		var err error
		backupPath, err = wt.createBackup(cleanPath)
		if err != nil {
			return wt.createErrorResult(startTime, fmt.Sprintf("failed to create backup: %v", err)), nil
		}
		wt.log.Debug("Created backup", "original", cleanPath, "backup", backupPath)
	}

	// Normalize line endings
	normalizedContent := wt.normalizeLineEndings(contentStr, lineEnding)

	// Write file
	var writeMode int
	if appendMode {
		writeMode = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	} else {
		writeMode = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}

	file, err := os.OpenFile(cleanPath, writeMode, 0644)
	if err != nil {
		return wt.createErrorResult(startTime, fmt.Sprintf("failed to open file for writing: %v", err)), nil
	}
	defer file.Close()

	bytesWritten, err := file.WriteString(normalizedContent)
	if err != nil {
		return wt.createErrorResult(startTime, fmt.Sprintf("failed to write content: %v", err)), nil
	}

	// Get final file info
	finalInfo, err := file.Stat()
	if err != nil {
		wt.log.Warn("Failed to get final file info", "error", err)
	}

	wt.log.Debug("File write completed",
		"path", cleanPath,
		"bytes_written", bytesWritten,
		"append_mode", appendMode,
		"backup_created", backupPath != "",
		"duration", time.Since(startTime))

	// Prepare result data
	resultData := map[string]interface{}{
		"file_path":      cleanPath,
		"bytes_written":  bytesWritten,
		"append_mode":    appendMode,
		"backup_created": backupPath != "",
		"file_existed":   fileExists,
		"encoding":       encoding,
		"line_ending":    lineEnding,
	}

	if backupPath != "" {
		resultData["backup_path"] = backupPath
	}

	if finalInfo != nil {
		resultData["file_size"] = finalInfo.Size()
		resultData["modified_time"] = finalInfo.ModTime()
	}

	if originalInfo != nil {
		resultData["original_size"] = originalInfo.Size()
		resultData["original_modified"] = originalInfo.ModTime()
	}

	// Format success message
	action := "Created"
	if fileExists {
		if appendMode {
			action = "Appended to"
		} else {
			action = "Updated"
		}
	}

	resultContent := fmt.Sprintf("%s file: %s\nBytes written: %d", action, cleanPath, bytesWritten)
	if backupPath != "" {
		resultContent += fmt.Sprintf("\nBackup created: %s", backupPath)
	}

	return ToolResult{
		Success: true,
		Content: resultContent,
		Error:   "",
		Data:    resultData,
		Metadata: ToolMetadata{
			ExecutionTime: time.Since(startTime),
			StartTime:     startTime,
			EndTime:       time.Now(),
			ToolName:      wt.Name(),
			Parameters:    params,
		},
	}, nil
}

// Helper methods

func (wt *WriteTool) validatePath(path string) error {
	// Normalize path for consistent checking
	cleanPath := filepath.Clean(path)

	// Check for restricted paths (must be exact matches or start with restricted path)
	for _, restricted := range wt.restrictedPaths {
		// Check if path starts with restricted directory
		if strings.HasPrefix(cleanPath, restricted+"/") || cleanPath == restricted {
			return fmt.Errorf("path contains restricted directory: %s", restricted)
		}

		// For system paths, be more strict - they must start with the path
		if strings.HasPrefix(restricted, "/") && strings.HasPrefix(cleanPath, restricted) {
			return fmt.Errorf("path contains restricted directory: %s", restricted)
		}

		// For relative restricted paths, check if they appear as complete path components
		pathComponents := strings.Split(cleanPath, string(filepath.Separator))
		for _, component := range pathComponents {
			if component == restricted {
				return fmt.Errorf("path contains restricted directory: %s", restricted)
			}
		}
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains potential directory traversal: %s", path)
	}

	return nil
}

func (wt *WriteTool) validateFileExtension(path string) error {
	ext := strings.ToLower(filepath.Ext(path))

	// Allow files without extensions (like Makefile, Dockerfile, etc.)
	if ext == "" {
		return nil
	}

	for _, allowed := range wt.allowedExtensions {
		if ext == allowed {
			return nil
		}
	}

	return fmt.Errorf("file extension %s is not in the allowed list", ext)
}

func (wt *WriteTool) createBackup(originalPath string) (string, error) {
	// Ensure backup directory exists
	backupDir := filepath.Join(filepath.Dir(originalPath), wt.backupDir)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Generate backup filename with timestamp
	filename := filepath.Base(originalPath)
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("%s.%s.backup", filename, timestamp)
	backupPath := filepath.Join(backupDir, backupFilename)

	// Copy original file to backup location
	originalFile, err := os.Open(originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to open original file: %v", err)
	}
	defer originalFile.Close()

	backupFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %v", err)
	}
	defer backupFile.Close()

	_, err = io.Copy(backupFile, originalFile)
	if err != nil {
		return "", fmt.Errorf("failed to copy file content: %v", err)
	}

	return backupPath, nil
}

func (wt *WriteTool) normalizeLineEndings(content, lineEnding string) string {
	// First normalize all line endings to \n
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// Then apply the desired line ending
	switch lineEnding {
	case "crlf":
		return strings.ReplaceAll(content, "\n", "\r\n")
	case "cr":
		return strings.ReplaceAll(content, "\n", "\r")
	case "lf":
		fallthrough
	default:
		return content
	}
}

func (wt *WriteTool) createErrorResult(startTime time.Time, errorMsg string) ToolResult {
	wt.log.Error("Write tool error", "error", errorMsg)
	return ToolResult{
		Success: false,
		Content: "",
		Error:   errorMsg,
		Metadata: ToolMetadata{
			ExecutionTime: time.Since(startTime),
			StartTime:     startTime,
			EndTime:       time.Now(),
			ToolName:      wt.Name(),
		},
	}
}

func (wt *WriteTool) getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if value, exists := params[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

func (wt *WriteTool) getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if value, exists := params[key]; exists {
		if boolValue, ok := value.(bool); ok {
			return boolValue
		}
	}
	return defaultValue
}
