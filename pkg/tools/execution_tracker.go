package tools

import (
	"context"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// ToolExecutionTracker wraps tool execution to provide feedback and tracking
type ToolExecutionTracker struct {
	registry *Registry
	log      *logger.Logger
	// Callback functions for different execution stages
	onStart    func(toolName string, args map[string]any)
	onProgress func(toolName string, progress string)
	onComplete func(toolName string, result ToolResult)
	onError    func(toolName string, err error)
}

// NewToolExecutionTracker creates a new tool execution tracker
func NewToolExecutionTracker(registry *Registry) *ToolExecutionTracker {
	log := logger.WithComponent("tool_tracker")
	return &ToolExecutionTracker{
		registry: registry,
		log:      log,
	}
}

// SetCallbacks sets the callback functions for different execution stages
func (tet *ToolExecutionTracker) SetCallbacks(
	onStart func(toolName string, args map[string]any),
	onProgress func(toolName string, progress string),
	onComplete func(toolName string, result ToolResult),
	onError func(toolName string, err error),
) {
	tet.onStart = onStart
	tet.onProgress = onProgress
	tet.onComplete = onComplete
	tet.onError = onError
}

// ExecuteWithTracking executes a tool with full tracking and feedback
func (tet *ToolExecutionTracker) ExecuteWithTracking(ctx context.Context, req ToolRequest) (ToolResult, error) {
	toolName := req.Name
	args := req.Parameters

	tet.log.Debug("Starting tracked tool execution", "tool", toolName, "args", args)

	// Notify start
	if tet.onStart != nil {
		tet.onStart(toolName, args)
	}

	// Record start time
	startTime := time.Now()

	// Execute the tool using the underlying registry
	result, err := tet.registry.Execute(ctx, req)

	// Record execution time
	executionTime := time.Since(startTime)
	tet.log.Debug("Tool execution completed",
		"tool", toolName,
		"success", result.Success,
		"duration", executionTime)

	// Update result metadata with execution time
	if result.Metadata.ExecutionTime == 0 {
		result.Metadata.ExecutionTime = executionTime
		result.Metadata.StartTime = startTime
		result.Metadata.EndTime = startTime.Add(executionTime)
	}

	// Handle callbacks based on result
	if err != nil {
		tet.log.Error("Tool execution failed", "tool", toolName, "error", err)
		if tet.onError != nil {
			tet.onError(toolName, err)
		}
		return result, err
	}

	if !result.Success {
		tet.log.Warn("Tool execution unsuccessful", "tool", toolName, "error", result.Error)
		if tet.onError != nil {
			tet.onError(toolName, ToolError{
				ToolName: toolName,
				Message:  "Tool execution failed",
				Cause:    err,
			})
		}
	} else {
		tet.log.Debug("Tool execution successful", "tool", toolName, "result_length", len(result.Content))
		if tet.onComplete != nil {
			tet.onComplete(toolName, result)
		}
	}

	return result, nil
}

// ExecuteAsync executes a tool asynchronously with tracking
func (tet *ToolExecutionTracker) ExecuteAsync(ctx context.Context, req ToolRequest) <-chan ToolResult {
	resultChan := make(chan ToolResult, 1)

	go func() {
		defer close(resultChan)

		result, err := tet.ExecuteWithTracking(ctx, req)
		if err != nil {
			// Convert error to failed result
			result = ToolResult{
				Success: false,
				Error:   err.Error(),
				Metadata: ToolMetadata{
					ToolName:   req.Name,
					Parameters: req.Parameters,
				},
			}
		}

		resultChan <- result
	}()

	return resultChan
}

// GetRegistry returns the underlying tool registry
func (tet *ToolExecutionTracker) GetRegistry() *Registry {
	return tet.registry
}

// HasTools returns true if the registry has any tools
func (tet *ToolExecutionTracker) HasTools() bool {
	return tet.registry.HasTools()
}

// List returns all available tool names
func (tet *ToolExecutionTracker) List() []string {
	return tet.registry.List()
}

// Get retrieves a tool by name
func (tet *ToolExecutionTracker) Get(name string) (Tool, bool) {
	return tet.registry.Get(name)
}
