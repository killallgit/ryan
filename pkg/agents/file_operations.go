package agents

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
)

// FileOperationsAgent handles all file-related operations with batching and caching
type FileOperationsAgent struct {
	batchProcessor *BatchProcessor
	fileCache      *FileCache
	toolRegistry   *tools.Registry
	log            *logger.Logger
}

// NewFileOperationsAgent creates a new file operations agent
func NewFileOperationsAgent(toolRegistry *tools.Registry) *FileOperationsAgent {
	return &FileOperationsAgent{
		batchProcessor: NewBatchProcessor(),
		fileCache:      NewFileCache(),
		toolRegistry:   toolRegistry,
		log:            logger.WithComponent("file_operations_agent"),
	}
}

// Name returns the agent name
func (f *FileOperationsAgent) Name() string {
	return "file_operations"
}

// Description returns the agent description
func (f *FileOperationsAgent) Description() string {
	return "Handles file reading, writing, creation, and batch operations with caching"
}

// CanHandle determines if this agent can handle the request
func (f *FileOperationsAgent) CanHandle(request string) (bool, float64) {
	lowerRequest := strings.ToLower(request)
	
	// High confidence keywords
	highConfidenceKeywords := []string{
		"list files", "read files", "read all files",
		"create file", "write file", "update file",
		"list and read", "gather files", "file content",
	}
	
	for _, keyword := range highConfidenceKeywords {
		if strings.Contains(lowerRequest, keyword) {
			return true, 0.9
		}
	}
	
	// Medium confidence keywords
	if strings.Contains(lowerRequest, "file") || strings.Contains(lowerRequest, "directory") {
		return true, 0.6
	}
	
	return false, 0.0
}

// Execute performs file operations
func (f *FileOperationsAgent) Execute(ctx context.Context, request AgentRequest) (AgentResult, error) {
	startTime := time.Now()
	f.log.Info("Executing file operation", "prompt", request.Prompt)

	// Determine operation type
	operation := f.determineOperation(request.Prompt)
	
	var result AgentResult
	var err error
	
	switch operation {
	case "list_and_read":
		result, err = f.handleListAndRead(ctx, request)
	case "read":
		result, err = f.handleRead(ctx, request)
	case "write":
		result, err = f.handleWrite(ctx, request)
	case "list":
		result, err = f.handleList(ctx, request)
	default:
		result = AgentResult{
			Success: false,
			Summary: "Unknown file operation",
			Details: fmt.Sprintf("Could not determine operation from: %s", request.Prompt),
		}
	}
	
	// Update metadata
	result.Metadata.AgentName = f.Name()
	result.Metadata.StartTime = startTime
	result.Metadata.EndTime = time.Now()
	result.Metadata.Duration = time.Since(startTime)
	
	// Update execution context if provided
	if execContext, ok := request.Context["execution_context"].(*ExecutionContext); ok {
		f.updateExecutionContext(execContext, result)
	}
	
	return result, err
}

// determineOperation determines what kind of file operation is requested
func (f *FileOperationsAgent) determineOperation(prompt string) string {
	lowerPrompt := strings.ToLower(prompt)
	
	if strings.Contains(lowerPrompt, "list and read") || 
	   strings.Contains(lowerPrompt, "read all files") ||
	   strings.Contains(lowerPrompt, "gather") {
		return "list_and_read"
	}
	
	if strings.Contains(lowerPrompt, "read") {
		return "read"
	}
	
	if strings.Contains(lowerPrompt, "write") || strings.Contains(lowerPrompt, "create") {
		return "write"
	}
	
	if strings.Contains(lowerPrompt, "list") {
		return "list"
	}
	
	return "unknown"
}

// handleListAndRead handles listing and reading all files in a directory
func (f *FileOperationsAgent) handleListAndRead(ctx context.Context, request AgentRequest) (AgentResult, error) {
	// Extract directory path
	dirPath := f.extractPath(request.Prompt)
	if dirPath == "" {
		dirPath = "."
	}
	
	f.log.Info("Listing and reading files", "directory", dirPath)
	
	// List files
	files, err := f.listFiles(dirPath)
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: fmt.Sprintf("Failed to list files in %s", dirPath),
			Details: err.Error(),
		}, err
	}
	
	// Read files in batch
	fileContents := make(map[string]string)
	filesProcessed := []string{}
	
	for _, file := range files {
		if f.shouldReadFile(file) {
			content, err := f.readFileWithCache(file)
			if err != nil {
				f.log.Warn("Failed to read file", "file", file, "error", err)
				continue
			}
			fileContents[file] = content
			filesProcessed = append(filesProcessed, file)
		}
	}
	
	// Build result
	details := f.buildFileListDetails(files, fileContents)
	
	return AgentResult{
		Success: true,
		Summary: fmt.Sprintf("Listed and read %d files in %s", len(filesProcessed), dirPath),
		Details: details,
		Artifacts: map[string]interface{}{
			"files":         files,
			"file_contents": fileContents,
			"directory":     dirPath,
		},
		Metadata: AgentMetadata{
			ToolsUsed:      []string{"file_list", "file_read"},
			FilesProcessed: filesProcessed,
		},
	}, nil
}

// handleRead handles reading a specific file
func (f *FileOperationsAgent) handleRead(ctx context.Context, request AgentRequest) (AgentResult, error) {
	filePath := f.extractPath(request.Prompt)
	if filePath == "" {
		return AgentResult{
			Success: false,
			Summary: "No file path specified",
			Details: "Please specify a file path to read",
		}, nil
	}
	
	content, err := f.readFileWithCache(filePath)
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: fmt.Sprintf("Failed to read file: %s", filePath),
			Details: err.Error(),
		}, err
	}
	
	return AgentResult{
		Success: true,
		Summary: fmt.Sprintf("Successfully read file: %s", filePath),
		Details: content,
		Artifacts: map[string]interface{}{
			"file_path": filePath,
			"content":   content,
		},
		Metadata: AgentMetadata{
			ToolsUsed:      []string{"file_read"},
			FilesProcessed: []string{filePath},
		},
	}, nil
}

// handleWrite handles writing to a file
func (f *FileOperationsAgent) handleWrite(ctx context.Context, request AgentRequest) (AgentResult, error) {
	// Extract file path and content
	filePath := f.extractPath(request.Prompt)
	content := f.extractContent(request.Prompt, request.Context)
	
	if filePath == "" {
		return AgentResult{
			Success: false,
			Summary: "No file path specified",
			Details: "Please specify a file path to write to",
		}, nil
	}
	
	// Use write tool
	writeTool, exists := f.toolRegistry.Get("write_file")
	if !exists {
		return AgentResult{
			Success: false,
			Summary: "Write tool not available",
			Details: "The write_file tool is not registered",
		}, fmt.Errorf("write_file tool not available")
	}
	
	result, err := writeTool.Execute(ctx, map[string]interface{}{
		"file_path": filePath,
		"content":   content,
	})
	
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: fmt.Sprintf("Failed to write file: %s", filePath),
			Details: err.Error(),
		}, err
	}
	
	// Invalidate cache for this file
	f.fileCache.Invalidate(filePath)
	
	return AgentResult{
		Success: true,
		Summary: fmt.Sprintf("Successfully wrote to file: %s", filePath),
		Details: result.Content,
		Artifacts: map[string]interface{}{
			"file_path": filePath,
			"content":   content,
		},
		Metadata: AgentMetadata{
			ToolsUsed:      []string{"write_file"},
			FilesProcessed: []string{filePath},
		},
	}, nil
}

// handleList handles listing files in a directory
func (f *FileOperationsAgent) handleList(ctx context.Context, request AgentRequest) (AgentResult, error) {
	dirPath := f.extractPath(request.Prompt)
	if dirPath == "" {
		dirPath = "."
	}
	
	files, err := f.listFiles(dirPath)
	if err != nil {
		return AgentResult{
			Success: false,
			Summary: fmt.Sprintf("Failed to list files in %s", dirPath),
			Details: err.Error(),
		}, err
	}
	
	details := strings.Join(files, "\n")
	
	return AgentResult{
		Success: true,
		Summary: fmt.Sprintf("Listed %d files in %s", len(files), dirPath),
		Details: details,
		Artifacts: map[string]interface{}{
			"files":     files,
			"directory": dirPath,
		},
		Metadata: AgentMetadata{
			ToolsUsed: []string{"file_list"},
		},
	}, nil
}

// Helper methods

func (f *FileOperationsAgent) listFiles(dirPath string) ([]string, error) {
	var files []string
	
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}
		
		if !info.IsDir() {
			files = append(files, path)
		}
		
		return nil
	})
	
	return files, err
}

func (f *FileOperationsAgent) readFileWithCache(filePath string) (string, error) {
	// Check cache first
	if content, ok := f.fileCache.Get(filePath); ok {
		return content, nil
	}
	
	// Read file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	
	content := string(data)
	
	// Cache the content
	f.fileCache.Set(filePath, content)
	
	return content, nil
}

func (f *FileOperationsAgent) shouldReadFile(filePath string) bool {
	// Skip binary files, large files, etc.
	ext := strings.ToLower(filepath.Ext(filePath))
	
	// Common text file extensions
	textExts := []string{
		".go", ".js", ".ts", ".py", ".java", ".c", ".cpp", ".h",
		".txt", ".md", ".yml", ".yaml", ".json", ".xml", ".html",
		".css", ".scss", ".less", ".sh", ".bash", ".zsh",
	}
	
	for _, textExt := range textExts {
		if ext == textExt {
			return true
		}
	}
	
	// Check if it's a no-extension text file (like Makefile, Dockerfile)
	base := filepath.Base(filePath)
	textFiles := []string{
		"Makefile", "Dockerfile", "README", "LICENSE",
		"Taskfile", ".gitignore", ".env",
	}
	
	for _, textFile := range textFiles {
		if base == textFile {
			return true
		}
	}
	
	return false
}

func (f *FileOperationsAgent) extractPath(prompt string) string {
	// Look for quoted paths
	if start := strings.Index(prompt, "\""); start != -1 {
		if end := strings.Index(prompt[start+1:], "\""); end != -1 {
			return prompt[start+1 : start+1+end]
		}
	}
	
	// Look for paths after common keywords
	keywords := []string{" in ", " from ", " at ", " of "}
	for _, keyword := range keywords {
		if idx := strings.Index(prompt, keyword); idx != -1 {
			pathPart := prompt[idx+len(keyword):]
			// Take the first word/path
			if spaceIdx := strings.IndexAny(pathPart, " ,;"); spaceIdx != -1 {
				pathPart = pathPart[:spaceIdx]
			}
			return strings.TrimSpace(pathPart)
		}
	}
	
	// Look for path-like patterns
	words := strings.Fields(prompt)
	for _, word := range words {
		if strings.Contains(word, "/") || strings.HasSuffix(word, ".go") {
			return strings.Trim(word, "\"',.")
		}
	}
	
	return ""
}

func (f *FileOperationsAgent) extractContent(prompt string, context map[string]interface{}) string {
	// Check if content is in context
	if content, ok := context["content"].(string); ok {
		return content
	}
	
	// Try to extract from prompt
	patterns := []string{
		"with content:",
		"content:",
		"containing:",
	}
	
	lowerPrompt := strings.ToLower(prompt)
	for _, pattern := range patterns {
		if idx := strings.Index(lowerPrompt, pattern); idx != -1 {
			return strings.TrimSpace(prompt[idx+len(pattern):])
		}
	}
	
	return ""
}

func (f *FileOperationsAgent) buildFileListDetails(files []string, contents map[string]string) string {
	var details []string
	
	details = append(details, fmt.Sprintf("Found %d files:", len(files)))
	details = append(details, "")
	
	for _, file := range files {
		details = append(details, fmt.Sprintf("File: %s", file))
		if content, ok := contents[file]; ok {
			lines := strings.Split(content, "\n")
			preview := fmt.Sprintf("  Lines: %d", len(lines))
			if len(lines) > 0 {
				preview += fmt.Sprintf(" (First line: %s)", truncateString(lines[0], 60))
			}
			details = append(details, preview)
		}
		details = append(details, "")
	}
	
	return strings.Join(details, "\n")
}

func (f *FileOperationsAgent) updateExecutionContext(ctx *ExecutionContext, result AgentResult) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	
	// Update file context
	for _, file := range result.Metadata.FilesProcessed {
		found := false
		for i, fc := range ctx.FileContext {
			if fc.Path == file {
				ctx.FileContext[i].LastModified = time.Now()
				found = true
				break
			}
		}
		if !found {
			ctx.FileContext = append(ctx.FileContext, FileInfo{
				Path:         file,
				LastModified: time.Now(),
			})
		}
	}
	
	// Store file contents in shared data
	if contents, ok := result.Artifacts["file_contents"].(map[string]string); ok {
		if ctx.SharedData == nil {
			ctx.SharedData = make(map[string]interface{})
		}
		ctx.SharedData["file_contents"] = contents
	}
}

// BatchProcessor handles batch file operations
type BatchProcessor struct {
	batchSize int
}

func NewBatchProcessor() *BatchProcessor {
	return &BatchProcessor{
		batchSize: 10,
	}
}

// FileCache provides simple file content caching
type FileCache struct {
	cache map[string]string
	mu    sync.RWMutex
}

func NewFileCache() *FileCache {
	return &FileCache{
		cache: make(map[string]string),
	}
}

func (fc *FileCache) Get(path string) (string, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	content, ok := fc.cache[path]
	return content, ok
}

func (fc *FileCache) Set(path string, content string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.cache[path] = content
}

func (fc *FileCache) Invalidate(path string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	delete(fc.cache, path)
}