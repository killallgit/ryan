package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// BashTool executes shell commands with safety constraints
type BashTool struct {
	// AllowedPaths are directories where commands can be executed
	AllowedPaths []string

	// ForbiddenCommands are commands that are not allowed to be executed
	ForbiddenCommands []string

	// Timeout is the maximum time a command can run
	Timeout time.Duration

	// WorkingDirectory is the directory where commands are executed
	WorkingDirectory string
}

// NewBashTool creates a new BashTool with default safety settings
func NewBashTool() *BashTool {
	home, _ := os.UserHomeDir()
	wd, _ := os.Getwd()
	bashTimeout := viper.GetDuration("tools.bash.timeout")
	return &BashTool{
		AllowedPaths: []string{
			home,
			"/tmp",
			wd,
		},
		ForbiddenCommands: []string{
			"sudo",
			"su",
			"rm -rf /",
			"dd",
			"mkfs",
			"fdisk",
			"shutdown",
			"reboot",
			"halt",
			"poweroff",
		},
		Timeout:          bashTimeout,
		WorkingDirectory: wd,
	}
}

// Name returns the tool name
func (bt *BashTool) Name() string {
	return "execute_bash"
}

// Description returns the tool description
func (bt *BashTool) Description() string {
	return "Execute a bash command and return its output. Use this tool to run shell commands, check file systems, run scripts, or interact with the system."
}

// JSONSchema returns the JSON schema for the tool parameters
func (bt *BashTool) JSONSchema() map[string]interface{} {
	schema := NewJSONSchema()

	AddProperty(schema, "command", JSONSchemaProperty{
		Type:        "string",
		Description: "The bash command to execute",
	})

	AddProperty(schema, "working_directory", JSONSchemaProperty{
		Type:        "string",
		Description: "Optional working directory for the command (must be within allowed paths)",
	})

	AddRequired(schema, "command")

	return schema
}

// Execute runs the bash command
func (bt *BashTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	startTime := time.Now()

	// Extract command parameter
	commandInterface, exists := params["command"]
	if !exists {
		return bt.createErrorResult(startTime, "command parameter is required"), nil
	}

	command, ok := commandInterface.(string)
	if !ok {
		return bt.createErrorResult(startTime, "command parameter must be a string"), nil
	}

	if strings.TrimSpace(command) == "" {
		return bt.createErrorResult(startTime, "command cannot be empty"), nil
	}

	// Validate command safety
	if err := bt.validateCommand(command); err != nil {
		return bt.createErrorResult(startTime, err.Error()), nil
	}

	// Get working directory
	workingDir := bt.WorkingDirectory
	if wdInterface, exists := params["working_directory"]; exists {
		if wd, ok := wdInterface.(string); ok && wd != "" {
			if err := bt.validatePath(wd); err != nil {
				return bt.createErrorResult(startTime, fmt.Sprintf("invalid working directory: %v", err)), nil
			}
			workingDir = wd
		}
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, bt.Timeout)
	defer cancel()

	// Execute command
	cmd := exec.CommandContext(execCtx, "bash", "-c", command)
	cmd.Dir = workingDir

	output, err := cmd.CombinedOutput()
	endTime := time.Now()

	result := ToolResult{
		Success: err == nil,
		Content: string(output),
		Metadata: ToolMetadata{
			ExecutionTime: endTime.Sub(startTime),
			StartTime:     startTime,
			EndTime:       endTime,
			ToolName:      bt.Name(),
			Parameters:    params,
		},
	}

	if err != nil {
		result.Error = fmt.Sprintf("command failed: %v", err)

		// Add context for specific error types
		if execCtx.Err() == context.DeadlineExceeded {
			result.Error = fmt.Sprintf("command timed out after %v", bt.Timeout)
		}

		// Include the output in error cases as it might contain useful error messages
		if len(output) > 0 {
			result.Content = string(output)
		}
	}

	return result, nil
}

// validateCommand checks if a command is safe to execute
func (bt *BashTool) validateCommand(command string) error {
	// Check for forbidden commands
	lowerCommand := strings.ToLower(strings.TrimSpace(command))

	for _, forbidden := range bt.ForbiddenCommands {
		if strings.Contains(lowerCommand, strings.ToLower(forbidden)) {
			return fmt.Errorf("forbidden command detected: %s", forbidden)
		}
	}

	// Additional safety checks
	dangerousPatterns := []string{
		"rm -rf",
		"rm -fr",
		"> /dev/",
		"mkfs",
		"format",
		"del /f",
		"del /q",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerCommand, strings.ToLower(pattern)) {
			return fmt.Errorf("potentially dangerous command pattern detected: %s", pattern)
		}
	}

	return nil
}

// validatePath checks if a path is allowed
func (bt *BashTool) validatePath(path string) error {
	// Clean and make absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	// Check if path is within allowed paths
	for _, allowed := range bt.AllowedPaths {
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

	return fmt.Errorf("path not within allowed directories: %s", absPath)
}

// createErrorResult creates a ToolResult for an error case
func (bt *BashTool) createErrorResult(startTime time.Time, errorMsg string) ToolResult {
	endTime := time.Now()
	return ToolResult{
		Success: false,
		Error:   errorMsg,
		Metadata: ToolMetadata{
			ExecutionTime: endTime.Sub(startTime),
			StartTime:     startTime,
			EndTime:       endTime,
			ToolName:      bt.Name(),
		},
	}
}
