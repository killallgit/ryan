package tools

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Registry manages available tools and their execution
type Registry struct {
	tools        map[string]Tool
	statsTracker *ToolStatsTracker
	mu           sync.RWMutex
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools:        make(map[string]Tool),
		statsTracker: NewToolStatsTracker(),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	// Check if tool already exists
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}

	r.tools[name] = tool
	return nil
}

// Unregister removes a tool from the registry
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tools, name)
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tool names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// HasTools returns true if the registry has any tools registered
func (r *Registry) HasTools() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools) > 0
}

// GetTools returns all registered tools
func (r *Registry) GetTools() map[string]Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	tools := make(map[string]Tool, len(r.tools))
	for name, tool := range r.tools {
		tools[name] = tool
	}
	return tools
}

// Execute runs a tool with the given parameters
func (r *Registry) Execute(ctx context.Context, req ToolRequest) (ToolResult, error) {
	// Get the tool
	tool, exists := r.Get(req.Name)
	if !exists {
		return ToolResult{
				Success: false,
				Error:   fmt.Sprintf("tool %s not found", req.Name),
			}, ToolError{
				ToolName: req.Name,
				Message:  "tool not found",
			}
	}

	// Record the start of execution
	r.statsTracker.RecordStart(req.Name)
	startTime := time.Now()

	// Use the provided context or create a default one
	execCtx := req.Context
	if execCtx == nil {
		execCtx = ctx
	}

	// Execute the tool
	result, err := tool.Execute(execCtx, req.Parameters)
	
	// Record the end of execution
	duration := time.Since(startTime)
	success := err == nil && result.Success
	r.statsTracker.RecordEnd(req.Name, duration, success)

	return result, err
}

// ExecuteAsync runs a tool asynchronously and returns a channel for the result
func (r *Registry) ExecuteAsync(ctx context.Context, req ToolRequest) <-chan ToolResult {
	resultChan := make(chan ToolResult, 1)

	go func() {
		defer close(resultChan)

		result, err := r.Execute(ctx, req)
		if err != nil {
			// If Execute returned an error, create an error result
			result = ToolResult{
				Success: false,
				Error:   err.Error(),
			}
		}

		resultChan <- result
	}()

	return resultChan
}

// GetDefinitions returns tool definitions for a specific provider
func (r *Registry) GetDefinitions(provider string) ([]ToolDefinition, error) {
	tools := r.GetTools()
	definitions := make([]ToolDefinition, 0, len(tools))

	for _, tool := range tools {
		definition, err := ConvertToProvider(tool, provider)
		if err != nil {
			return nil, fmt.Errorf("failed to convert tool %s for provider %s: %w", tool.Name(), provider, err)
		}

		definitions = append(definitions, ToolDefinition{
			Provider:   provider,
			Definition: definition,
		})
	}

	return definitions, nil
}

// RegisterBuiltinTools registers the default tools (bash, file_read, web_fetch, grep, write_file)
func (r *Registry) RegisterBuiltinTools() error {
	// Register BashTool
	bashTool := NewBashTool()
	if err := r.Register(bashTool); err != nil {
		return fmt.Errorf("failed to register bash tool: %w", err)
	}

	// Register FileReadTool
	fileReadTool := NewFileReadTool()
	if err := r.Register(fileReadTool); err != nil {
		return fmt.Errorf("failed to register file read tool: %w", err)
	}

	// Register WebFetchTool
	webFetchTool := NewWebFetchTool()
	if err := r.Register(webFetchTool); err != nil {
		return fmt.Errorf("failed to register web fetch tool: %w", err)
	}

	// Register GrepTool
	grepTool := NewGrepTool()
	if err := r.Register(grepTool); err != nil {
		return fmt.Errorf("failed to register grep tool: %w", err)
	}

	// Register WriteTool
	writeTool := NewWriteTool()
	if err := r.Register(writeTool); err != nil {
		return fmt.Errorf("failed to register write tool: %w", err)
	}

	return nil
}

// GetToolStats returns statistics for a specific tool
func (r *Registry) GetToolStats(toolName string) *ToolStats {
	return r.statsTracker.GetStats(toolName)
}

// GetAllToolStats returns statistics for all tools
func (r *Registry) GetAllToolStats() map[string]*ToolStats {
	return r.statsTracker.GetAllStats()
}

// ResetToolStats resets statistics for a specific tool
func (r *Registry) ResetToolStats(toolName string) {
	r.statsTracker.Reset(toolName)
}

// ResetAllToolStats resets all tool statistics
func (r *Registry) ResetAllToolStats() {
	r.statsTracker.ResetAll()
}
