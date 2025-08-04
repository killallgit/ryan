package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// GrepTool implements advanced text search with ripgrep integration
type GrepTool struct {
	log         *logger.Logger
	maxResults  int
	maxFileSize int64
	ripgrepPath string
	workingDir  string
}

// GrepResult represents a single search result
type GrepResult struct {
	File         string   `json:"file"`
	LineNumber   int      `json:"line_number"`
	ColumnNumber int      `json:"column_number,omitempty"`
	Line         string   `json:"line"`
	Match        string   `json:"match"`
	Context      []string `json:"context,omitempty"`
}

// NewGrepTool creates a new Grep tool with default configuration
func NewGrepTool() *GrepTool {
	tool := &GrepTool{
		log:         logger.WithComponent("grep_tool"),
		maxResults:  1000,
		maxFileSize: 50 * 1024 * 1024, // 50MB limit
		workingDir:  ".",
	}

	// Try to find ripgrep binary
	tool.ripgrepPath = tool.findRipgrep()
	if tool.ripgrepPath == "" {
		tool.log.Warn("ripgrep not found, falling back to basic grep")
	}

	return tool
}

// Name returns the tool name
func (gt *GrepTool) Name() string {
	return "grep"
}

// Description returns the tool description
func (gt *GrepTool) Description() string {
	return "Advanced text search tool using ripgrep for fast, recursive file searching with pattern matching, context lines, and syntax highlighting. Supports regex patterns, file type filtering, and case-insensitive searching."
}

// JSONSchema returns the JSON schema for the tool parameters
func (gt *GrepTool) JSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The search pattern (supports regex)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory or file path to search in (defaults to current directory)",
				"default":     ".",
			},
			"case_sensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the search should be case sensitive",
				"default":     false,
			},
			"regex": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to treat pattern as a regular expression",
				"default":     true,
			},
			"whole_word": map[string]interface{}{
				"type":        "boolean",
				"description": "Match whole words only",
				"default":     false,
			},
			"file_types": map[string]interface{}{
				"type":        "array",
				"description": "File extensions to include (e.g., ['go', 'js', 'py'])",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"exclude_dirs": map[string]interface{}{
				"type":        "array",
				"description": "Directories to exclude from search (e.g., ['node_modules', '.git'])",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"context_before": map[string]interface{}{
				"type":        "integer",
				"description": "Number of lines to show before each match",
				"minimum":     0,
				"maximum":     10,
				"default":     0,
			},
			"context_after": map[string]interface{}{
				"type":        "integer",
				"description": "Number of lines to show after each match",
				"minimum":     0,
				"maximum":     10,
				"default":     0,
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return",
				"minimum":     1,
				"maximum":     2000,
				"default":     100,
			},
			"files_only": map[string]interface{}{
				"type":        "boolean",
				"description": "Return only file names that contain matches",
				"default":     false,
			},
			"line_numbers": map[string]interface{}{
				"type":        "boolean",
				"description": "Include line numbers in results",
				"default":     true,
			},
		},
		"required": []string{"pattern"},
	}
}

// Execute performs the search operation
func (gt *GrepTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	startTime := time.Now()

	// Extract and validate pattern
	pattern, exists := params["pattern"]
	if !exists {
		return gt.createErrorResult(startTime, "pattern parameter is required"), nil
	}

	patternStr, ok := pattern.(string)
	if !ok {
		return gt.createErrorResult(startTime, "pattern must be a string"), nil
	}

	if strings.TrimSpace(patternStr) == "" {
		return gt.createErrorResult(startTime, "pattern cannot be empty"), nil
	}

	// Extract optional parameters
	searchPath := gt.getStringParam(params, "path", ".")
	caseSensitive := gt.getBoolParam(params, "case_sensitive", false)
	useRegex := gt.getBoolParam(params, "regex", true)
	wholeWord := gt.getBoolParam(params, "whole_word", false)
	contextBefore := gt.getIntParam(params, "context_before", 0)
	contextAfter := gt.getIntParam(params, "context_after", 0)
	maxResults := gt.getIntParam(params, "max_results", 100)
	filesOnly := gt.getBoolParam(params, "files_only", false)
	lineNumbers := gt.getBoolParam(params, "line_numbers", true)

	// Get file types and exclude dirs
	fileTypes := gt.getStringArrayParam(params, "file_types")
	excludeDirs := gt.getStringArrayParam(params, "exclude_dirs")

	// Validate search path
	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(gt.workingDir, searchPath)
	}

	if _, err := os.Stat(searchPath); err != nil {
		return gt.createErrorResult(startTime, fmt.Sprintf("search path does not exist: %s", searchPath)), nil
	}

	gt.log.Debug("Starting search",
		"pattern", patternStr,
		"path", searchPath,
		"case_sensitive", caseSensitive,
		"regex", useRegex,
		"max_results", maxResults)

	var results []GrepResult
	var err error

	// Use ripgrep if available, otherwise fall back to basic grep
	if gt.ripgrepPath != "" {
		results, err = gt.executeRipgrep(ctx, patternStr, searchPath, GrepOptions{
			CaseSensitive: caseSensitive,
			UseRegex:      useRegex,
			WholeWord:     wholeWord,
			ContextBefore: contextBefore,
			ContextAfter:  contextAfter,
			MaxResults:    maxResults,
			FilesOnly:     filesOnly,
			LineNumbers:   lineNumbers,
			FileTypes:     fileTypes,
			ExcludeDirs:   excludeDirs,
		})
	} else {
		results, err = gt.executeBasicGrep(ctx, patternStr, searchPath, GrepOptions{
			CaseSensitive: caseSensitive,
			UseRegex:      useRegex,
			WholeWord:     wholeWord,
			ContextBefore: contextBefore,
			ContextAfter:  contextAfter,
			MaxResults:    maxResults,
			FilesOnly:     filesOnly,
			LineNumbers:   lineNumbers,
			FileTypes:     fileTypes,
			ExcludeDirs:   excludeDirs,
		})
	}

	if err != nil {
		return gt.createErrorResult(startTime, fmt.Sprintf("search failed: %v", err)), nil
	}

	// Format results
	content := gt.formatResults(results, patternStr, filesOnly)

	gt.log.Debug("Search completed",
		"results_count", len(results),
		"duration", time.Since(startTime))

	// Return successful result
	return ToolResult{
		Success: true,
		Content: content,
		Error:   "",
		Data: map[string]interface{}{
			"pattern":        patternStr,
			"search_path":    searchPath,
			"results_count":  len(results),
			"results":        results,
			"case_sensitive": caseSensitive,
			"regex":          useRegex,
			"files_only":     filesOnly,
		},
		Metadata: ToolMetadata{
			ExecutionTime: time.Since(startTime),
			StartTime:     startTime,
			EndTime:       time.Now(),
			ToolName:      gt.Name(),
			Parameters:    params,
		},
	}, nil
}

// GrepOptions holds search configuration
type GrepOptions struct {
	CaseSensitive bool
	UseRegex      bool
	WholeWord     bool
	ContextBefore int
	ContextAfter  int
	MaxResults    int
	FilesOnly     bool
	LineNumbers   bool
	FileTypes     []string
	ExcludeDirs   []string
}

// executeRipgrep performs search using ripgrep
func (gt *GrepTool) executeRipgrep(ctx context.Context, pattern, searchPath string, opts GrepOptions) ([]GrepResult, error) {
	args := []string{pattern, searchPath}

	// Add ripgrep-specific flags
	if !opts.CaseSensitive {
		args = append(args, "-i")
	}

	if !opts.UseRegex {
		args = append(args, "-F") // Fixed strings (literal)
	}

	if opts.WholeWord {
		args = append(args, "-w")
	}

	if opts.FilesOnly {
		args = append(args, "-l") // Files with matches only
	} else {
		args = append(args, "-n") // Line numbers
		if opts.ContextBefore > 0 {
			args = append(args, "-B", strconv.Itoa(opts.ContextBefore))
		}
		if opts.ContextAfter > 0 {
			args = append(args, "-A", strconv.Itoa(opts.ContextAfter))
		}
	}

	// File type filtering
	for _, fileType := range opts.FileTypes {
		args = append(args, "-t", fileType)
	}

	// Exclude directories
	for _, dir := range opts.ExcludeDirs {
		args = append(args, "--glob", "!"+dir)
	}

	// Limit results
	args = append(args, "-m", strconv.Itoa(opts.MaxResults))

	// Use line-oriented output instead of JSON for simpler parsing
	args = append(args, "--no-heading")

	gt.log.Debug("Executing ripgrep", "args", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, gt.ripgrepPath, args...)
	output, err := cmd.Output()

	if err != nil {
		// ripgrep returns exit code 1 when no matches found, which is not an error
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return []GrepResult{}, nil
		}
		return nil, fmt.Errorf("ripgrep command failed: %v", err)
	}

	return gt.parseRipgrepOutput(string(output), opts.FilesOnly)
}

// executeBasicGrep performs search using basic grep command
func (gt *GrepTool) executeBasicGrep(ctx context.Context, pattern, searchPath string, opts GrepOptions) ([]GrepResult, error) {
	args := []string{"-r"}

	if !opts.CaseSensitive {
		args = append(args, "-i")
	}

	if !opts.UseRegex {
		args = append(args, "-F")
	}

	if opts.WholeWord {
		args = append(args, "-w")
	}

	if opts.FilesOnly {
		args = append(args, "-l")
	} else if opts.LineNumbers {
		args = append(args, "-n")
	}

	if opts.ContextBefore > 0 {
		args = append(args, "-B", strconv.Itoa(opts.ContextBefore))
	}
	if opts.ContextAfter > 0 {
		args = append(args, "-A", strconv.Itoa(opts.ContextAfter))
	}

	// Add exclusions
	for _, dir := range opts.ExcludeDirs {
		args = append(args, "--exclude-dir="+dir)
	}

	args = append(args, pattern, searchPath)

	cmd := exec.CommandContext(ctx, "grep", args...)
	output, err := cmd.Output()

	if err != nil {
		// grep returns exit code 1 when no matches found
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return []GrepResult{}, nil
		}
		return nil, fmt.Errorf("grep command failed: %v", err)
	}

	return gt.parseGrepOutput(string(output), opts.FilesOnly)
}

// Helper methods

func (gt *GrepTool) findRipgrep() string {
	// Check common locations for ripgrep
	candidates := []string{"rg", "/usr/bin/rg", "/usr/local/bin/rg", "/opt/homebrew/bin/rg"}

	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}

	return ""
}

func (gt *GrepTool) parseRipgrepOutput(output string, filesOnly bool) ([]GrepResult, error) {
	var results []GrepResult
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if filesOnly {
		// Simple file list parsing
		for _, line := range lines {
			if line != "" {
				results = append(results, GrepResult{
					File: line,
				})
			}
		}
		return results, nil
	}

	// Parse ripgrep line-oriented output
	for _, line := range lines {
		if line == "" {
			continue
		}

		result := gt.parseGrepLine(line)
		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

func (gt *GrepTool) parseGrepOutput(output string, filesOnly bool) ([]GrepResult, error) {
	var results []GrepResult
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		if filesOnly {
			results = append(results, GrepResult{
				File: line,
			})
		} else {
			result := gt.parseGrepLine(line)
			if result != nil {
				results = append(results, *result)
			}
		}
	}

	return results, nil
}

func (gt *GrepTool) parseGrepLine(line string) *GrepResult {
	// Parse format: filename:line_number:content
	parts := strings.SplitN(line, ":", 3)
	if len(parts) < 2 {
		return nil
	}

	result := &GrepResult{
		File: parts[0],
		Line: line,
	}

	if len(parts) >= 3 {
		if lineNum, err := strconv.Atoi(parts[1]); err == nil {
			result.LineNumber = lineNum
			result.Line = parts[2]
		} else {
			result.Line = parts[1]
		}
	}

	return result
}

func (gt *GrepTool) formatResults(results []GrepResult, pattern string, filesOnly bool) string {
	if len(results) == 0 {
		return fmt.Sprintf("No matches found for pattern: %s", pattern)
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d matches for pattern: %s\n\n", len(results), pattern))

	if filesOnly {
		output.WriteString("Files containing matches:\n")
		for _, result := range results {
			output.WriteString(fmt.Sprintf("  %s\n", result.File))
		}
	} else {
		for i, result := range results {
			if i > 0 {
				output.WriteString("\n")
			}

			if result.LineNumber > 0 {
				output.WriteString(fmt.Sprintf("%s:%d: %s\n", result.File, result.LineNumber, result.Line))
			} else {
				output.WriteString(fmt.Sprintf("%s: %s\n", result.File, result.Line))
			}

			// Add context if available
			for _, contextLine := range result.Context {
				output.WriteString(fmt.Sprintf("    %s\n", contextLine))
			}
		}
	}

	return output.String()
}

func (gt *GrepTool) createErrorResult(startTime time.Time, errorMsg string) ToolResult {
	gt.log.Error("Grep tool error", "error", errorMsg)
	return ToolResult{
		Success: false,
		Content: "",
		Error:   errorMsg,
		Metadata: ToolMetadata{
			ExecutionTime: time.Since(startTime),
			StartTime:     startTime,
			EndTime:       time.Now(),
			ToolName:      gt.Name(),
		},
	}
}

func (gt *GrepTool) getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if value, exists := params[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

func (gt *GrepTool) getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if value, exists := params[key]; exists {
		if boolValue, ok := value.(bool); ok {
			return boolValue
		}
	}
	return defaultValue
}

func (gt *GrepTool) getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, exists := params[key]; exists {
		if intValue, ok := value.(int); ok {
			return intValue
		}
		if floatValue, ok := value.(float64); ok {
			return int(floatValue)
		}
	}
	return defaultValue
}

func (gt *GrepTool) getStringArrayParam(params map[string]interface{}, key string) []string {
	if value, exists := params[key]; exists {
		if arrayValue, ok := value.([]interface{}); ok {
			var result []string
			for _, item := range arrayValue {
				if strValue, ok := item.(string); ok {
					result = append(result, strValue)
				}
			}
			return result
		}
	}
	return []string{}
}
