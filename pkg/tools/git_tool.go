package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// GitTool implements git repository operations with safety constraints
type GitTool struct {
	log        *logger.Logger
	gitBinary  string
	workingDir string
	timeout    time.Duration
	allowedOps []string
}

// GitOperation represents a git operation result
type GitOperation struct {
	Command    string            `json:"command"`
	Output     string            `json:"output"`
	Error      string            `json:"error,omitempty"`
	ExitCode   int               `json:"exit_code"`
	Repository string            `json:"repository"`
	Branch     string            `json:"branch,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// GitStatus represents repository status information
type GitStatus struct {
	Branch       string   `json:"branch"`
	Staged       []string `json:"staged"`
	Modified     []string `json:"modified"`
	Untracked    []string `json:"untracked"`
	Deleted      []string `json:"deleted"`
	Renamed      []string `json:"renamed"`
	Ahead        int      `json:"ahead"`
	Behind       int      `json:"behind"`
	Clean        bool     `json:"clean"`
	HasConflicts bool     `json:"has_conflicts"`
}

// NewGitTool creates a new Git tool with default configuration
func NewGitTool() *GitTool {
	wd, _ := os.Getwd()

	tool := &GitTool{
		log:        logger.WithComponent("git_tool"),
		workingDir: wd,
		timeout:    30 * time.Second,
		allowedOps: []string{
			"status", "diff", "log", "branch", "show", "ls-files",
			"rev-parse", "describe", "tag", "remote", "config",
			"blame", "shortlog", "reflog", "cherry",
		},
	}

	// Find git binary
	tool.gitBinary = tool.findGitBinary()
	if tool.gitBinary == "" {
		tool.log.Warn("git binary not found")
	}

	return tool
}

// Name returns the tool name
func (gt *GitTool) Name() string {
	return "git"
}

// Description returns the tool description
func (gt *GitTool) Description() string {
	return "Git repository operations including status checking, diff viewing, log inspection, branch management, and repository analysis. Provides comprehensive repository intelligence for code review and development workflows."
}

// JSONSchema returns the JSON schema for the tool parameters
func (gt *GitTool) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"description": "Git operation to perform",
				"enum": []string{
					"status", "diff", "log", "branch", "show", "ls-files",
					"rev-parse", "describe", "tag", "remote", "config",
					"blame", "shortlog", "reflog", "cherry",
				},
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Repository path (defaults to current directory)",
				"default":     ".",
			},
			"args": map[string]any{
				"type":        "array",
				"description": "Additional arguments for the git command",
				"items": map[string]any{
					"type": "string",
				},
			},
			"format": map[string]any{
				"type":        "string",
				"description": "Output format (raw, structured, summary)",
				"enum":        []string{"raw", "structured", "summary"},
				"default":     "structured",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Limit number of results (for log, branch operations)",
				"minimum":     1,
				"maximum":     1000,
				"default":     50,
			},
			"branch": map[string]any{
				"type":        "string",
				"description": "Specific branch for operations (optional)",
			},
			"file": map[string]any{
				"type":        "string",
				"description": "Specific file for operations like diff, blame, log",
			},
			"commit": map[string]any{
				"type":        "string",
				"description": "Specific commit hash or reference",
			},
		},
		"required": []string{"operation"},
	}
}

// Execute performs the git operation
func (gt *GitTool) Execute(ctx context.Context, params map[string]any) (ToolResult, error) {
	startTime := time.Now()

	if gt.gitBinary == "" {
		return gt.createErrorResult(startTime, "git binary not available"), nil
	}

	// Extract and validate operation
	operation, exists := params["operation"]
	if !exists {
		return gt.createErrorResult(startTime, "operation parameter is required"), nil
	}

	opStr, ok := operation.(string)
	if !ok {
		return gt.createErrorResult(startTime, "operation must be a string"), nil
	}

	// Validate operation is allowed
	if !gt.isAllowedOperation(opStr) {
		return gt.createErrorResult(startTime, fmt.Sprintf("operation '%s' is not allowed", opStr)), nil
	}

	// Extract optional parameters
	repoPath := gt.getStringParam(params, "path", ".")
	format := gt.getStringParam(params, "format", "structured")
	limit := gt.getIntParam(params, "limit", 50)
	branch := gt.getStringParam(params, "branch", "")
	file := gt.getStringParam(params, "file", "")
	commit := gt.getStringParam(params, "commit", "")
	args := gt.getStringArrayParam(params, "args")

	// Validate and resolve repository path
	if !filepath.IsAbs(repoPath) {
		repoPath = filepath.Join(gt.workingDir, repoPath)
	}

	if err := gt.validateRepository(repoPath); err != nil {
		return gt.createErrorResult(startTime, fmt.Sprintf("invalid repository: %v", err)), nil
	}

	gt.log.Debug("Executing git operation",
		"operation", opStr,
		"path", repoPath,
		"format", format)

	// Execute the git operation
	result, err := gt.executeGitOperation(ctx, opStr, repoPath, GitOperationParams{
		Args:   args,
		Format: format,
		Limit:  limit,
		Branch: branch,
		File:   file,
		Commit: commit,
	})

	if err != nil {
		return gt.createErrorResult(startTime, fmt.Sprintf("git operation failed: %v", err)), nil
	}

	// Format the response based on requested format
	content := gt.formatResult(result, format)

	return ToolResult{
		Success: result.ExitCode == 0,
		Content: content,
		Error:   result.Error,
		Data: map[string]any{
			"operation":  opStr,
			"repository": repoPath,
			"exit_code":  result.ExitCode,
			"branch":     result.Branch,
			"raw_output": result.Output,
			"metadata":   result.Metadata,
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

// GitOperationParams holds parameters for git operations
type GitOperationParams struct {
	Args   []string
	Format string
	Limit  int
	Branch string
	File   string
	Commit string
}

// executeGitOperation executes a specific git operation
func (gt *GitTool) executeGitOperation(ctx context.Context, operation, repoPath string, params GitOperationParams) (*GitOperation, error) {
	// Build git command arguments
	args := gt.buildGitArgs(operation, params)

	// Create command with timeout
	execCtx, cancel := context.WithTimeout(ctx, gt.timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, gt.gitBinary, args...)
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	exitCode := 0
	errorMsg := ""

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
		errorMsg = err.Error()
	}

	// Get current branch
	branch := gt.getCurrentBranch(repoPath)

	// Build metadata
	metadata := map[string]string{
		"repository": repoPath,
		"git_dir":    filepath.Join(repoPath, ".git"),
	}

	return &GitOperation{
		Command:    fmt.Sprintf("git %s", strings.Join(args, " ")),
		Output:     string(output),
		Error:      errorMsg,
		ExitCode:   exitCode,
		Repository: repoPath,
		Branch:     branch,
		Metadata:   metadata,
	}, nil
}

// buildGitArgs constructs git command arguments based on operation and parameters
func (gt *GitTool) buildGitArgs(operation string, params GitOperationParams) []string {
	args := []string{operation}

	switch operation {
	case "status":
		args = append(args, "--porcelain=v1", "--branch")

	case "diff":
		if params.Commit != "" {
			args = append(args, params.Commit)
		} else {
			args = append(args, "--cached")
		}
		if params.File != "" {
			args = append(args, "--", params.File)
		}

	case "log":
		args = append(args, "--oneline", fmt.Sprintf("-n%d", params.Limit))
		if params.Branch != "" {
			args = append(args, params.Branch)
		}
		if params.File != "" {
			args = append(args, "--", params.File)
		}

	case "branch":
		args = append(args, "-v", "-a")

	case "show":
		if params.Commit != "" {
			args = append(args, params.Commit)
		} else {
			args = append(args, "HEAD")
		}
		args = append(args, "--stat")

	case "ls-files":
		args = append(args, "--cached", "--others", "--exclude-standard")

	case "remote":
		args = append(args, "-v")

	case "blame":
		if params.File != "" {
			args = append(args, params.File)
		}

	case "reflog":
		args = append(args, "--oneline", fmt.Sprintf("-n%d", params.Limit))
	}

	// Add any additional arguments
	args = append(args, params.Args...)

	return args
}

// getCurrentBranch gets the current branch name
func (gt *GitTool) getCurrentBranch(repoPath string) string {
	cmd := exec.Command(gt.gitBinary, "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

// formatResult formats the git operation result based on requested format
func (gt *GitTool) formatResult(result *GitOperation, format string) string {
	switch format {
	case "raw":
		return result.Output

	case "summary":
		return gt.createSummary(result)

	case "structured":
		fallthrough
	default:
		return gt.createStructuredOutput(result)
	}
}

// createSummary creates a brief summary of the git operation
func (gt *GitTool) createSummary(result *GitOperation) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Git %s completed", strings.Fields(result.Command)[1]))
	if result.Branch != "" {
		summary.WriteString(fmt.Sprintf(" on branch '%s'", result.Branch))
	}

	if result.ExitCode != 0 {
		summary.WriteString(fmt.Sprintf(" with exit code %d", result.ExitCode))
		if result.Error != "" {
			summary.WriteString(fmt.Sprintf(": %s", result.Error))
		}
	}

	return summary.String()
}

// createStructuredOutput creates formatted output for the git operation
func (gt *GitTool) createStructuredOutput(result *GitOperation) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("=== Git %s ===\n", strings.Fields(result.Command)[1]))
	output.WriteString(fmt.Sprintf("Repository: %s\n", result.Repository))
	if result.Branch != "" {
		output.WriteString(fmt.Sprintf("Branch: %s\n", result.Branch))
	}
	output.WriteString(fmt.Sprintf("Exit Code: %d\n", result.ExitCode))
	output.WriteString("\n")

	if result.Output != "" {
		output.WriteString("Output:\n")
		output.WriteString(result.Output)
		if !strings.HasSuffix(result.Output, "\n") {
			output.WriteString("\n")
		}
	}

	if result.Error != "" {
		output.WriteString("\nError:\n")
		output.WriteString(result.Error)
		output.WriteString("\n")
	}

	return output.String()
}

// Helper methods

func (gt *GitTool) findGitBinary() string {
	candidates := []string{"git", "/usr/bin/git", "/usr/local/bin/git", "/opt/homebrew/bin/git"}

	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}

	return ""
}

func (gt *GitTool) isAllowedOperation(operation string) bool {
	for _, allowed := range gt.allowedOps {
		if operation == allowed {
			return true
		}
	}
	return false
}

func (gt *GitTool) validateRepository(path string) error {
	// Check if path exists
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("path does not exist: %s", path)
	}

	// Check if it's a git repository
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		// Try to find git dir in parent directories
		cmd := exec.Command(gt.gitBinary, "rev-parse", "--git-dir")
		cmd.Dir = path
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("not a git repository: %s", path)
		}
	}

	return nil
}

func (gt *GitTool) createErrorResult(startTime time.Time, errorMsg string) ToolResult {
	gt.log.Error("Git tool error", "error", errorMsg)
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

func (gt *GitTool) getStringParam(params map[string]any, key, defaultValue string) string {
	if value, exists := params[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

func (gt *GitTool) getIntParam(params map[string]any, key string, defaultValue int) int {
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

func (gt *GitTool) getStringArrayParam(params map[string]any, key string) []string {
	if value, exists := params[key]; exists {
		if arrayValue, ok := value.([]any); ok {
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
