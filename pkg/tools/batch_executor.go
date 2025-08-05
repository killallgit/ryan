package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// BatchExecutor implements Claude Code's "multiple tools in single response" capability
type BatchExecutor struct {
	registry      *Registry
	log           *logger.Logger
	maxConcurrent int
	timeout       time.Duration
	progressSink  chan<- ProgressUpdate
}

// BatchRequest represents a request to execute multiple tools
type BatchRequest struct {
	Tools        []ToolRequest         `json:"tools"`
	Dependencies map[string][]string   `json:"dependencies,omitempty"` // tool_id -> [dependency_ids]
	Timeout      time.Duration         `json:"timeout,omitempty"`
	Context      context.Context       `json:"-"`
	Progress     chan<- ProgressUpdate `json:"-"`
}

// BatchResult represents the result of batch tool execution
type BatchResult struct {
	Results      map[string]ToolResult  `json:"results"`
	Errors       map[string]error       `json:"errors,omitempty"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Duration     time.Duration          `json:"duration"`
	ToolsCount   int                    `json:"tools_count"`
	SuccessCount int                    `json:"success_count"`
	ErrorCount   int                    `json:"error_count"`
	Metadata     BatchExecutionMetadata `json:"metadata"`
}

// BatchExecutionMetadata contains execution information for the batch
type BatchExecutionMetadata struct {
	ConcurrentExecutions int                `json:"concurrent_executions"`
	DependencyGraph      *DependencyGraph   `json:"dependency_graph,omitempty"`
	ExecutionOrder       []string           `json:"execution_order"`
	ResourceUsage        ResourceUsageStats `json:"resource_usage"`
}

// ProgressUpdate represents a progress update during batch execution
type ProgressUpdate struct {
	Type      ProgressType `json:"type"`
	ToolID    string       `json:"tool_id"`
	ToolName  string       `json:"tool_name"`
	Progress  float64      `json:"progress"` // 0.0 to 1.0
	Message   string       `json:"message"`
	Result    *ToolResult  `json:"result,omitempty"`
	Error     error        `json:"error,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

// ProgressType represents the type of progress update
type ProgressType string

const (
	ProgressStarted   ProgressType = "started"
	ProgressProgress  ProgressType = "progress"
	ProgressCompleted ProgressType = "completed"
	ProgressError     ProgressType = "error"
	ProgressCancelled ProgressType = "cancelled"
)

// ResourceUsageStats tracks resource usage during batch execution
type ResourceUsageStats struct {
	MaxConcurrent int   `json:"max_concurrent"`
	TotalMemoryMB int64 `json:"total_memory_mb"`
	PeakMemoryMB  int64 `json:"peak_memory_mb"`
	CPUTimeMs     int64 `json:"cpu_time_ms"`
	WallTimeMs    int64 `json:"wall_time_ms"`
}

// NewBatchExecutor creates a new batch executor
func NewBatchExecutor(registry *Registry) *BatchExecutor {
	return &BatchExecutor{
		registry:      registry,
		log:           logger.WithComponent("batch_executor"),
		maxConcurrent: 10, // Default limit
		timeout:       5 * time.Minute,
		progressSink:  nil,
	}
}

// WithMaxConcurrent sets the maximum number of concurrent tool executions
func (be *BatchExecutor) WithMaxConcurrent(max int) *BatchExecutor {
	be.maxConcurrent = max
	return be
}

// WithTimeout sets the default timeout for batch execution
func (be *BatchExecutor) WithTimeout(timeout time.Duration) *BatchExecutor {
	be.timeout = timeout
	return be
}

// WithProgressSink sets a channel to receive progress updates
func (be *BatchExecutor) WithProgressSink(sink chan<- ProgressUpdate) *BatchExecutor {
	be.progressSink = sink
	return be
}

// Execute performs batch tool execution with dependency resolution and concurrency
func (be *BatchExecutor) Execute(req BatchRequest) (*BatchResult, error) {
	startTime := time.Now()

	// Validate request
	if len(req.Tools) == 0 {
		return nil, fmt.Errorf("no tools specified in batch request")
	}

	// Set up context with timeout
	execCtx := req.Context
	if execCtx == nil {
		execCtx = context.Background()
	}

	timeout := req.Timeout
	if timeout == 0 {
		timeout = be.timeout
	}

	ctx, cancel := context.WithTimeout(execCtx, timeout)
	defer cancel()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	be.log.Info("Starting batch execution",
		"tools_count", len(req.Tools),
		"timeout", timeout,
		"max_concurrent", be.maxConcurrent)

	// Build dependency graph
	depGraph, err := be.buildDependencyGraph(req.Tools, req.Dependencies)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Get execution order
	executionOrder, err := depGraph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// Initialize result tracking
	results := make(map[string]ToolResult)
	errors := make(map[string]error)

	// Track resource usage
	resourceStats := ResourceUsageStats{
		MaxConcurrent: be.maxConcurrent,
	}

	// Execute tools in dependency order with concurrency control
	err = be.executeWithConcurrency(ctx, req, depGraph, executionOrder, results, errors, &resourceStats)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Count successes and failures
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	result := &BatchResult{
		Results:      results,
		Errors:       errors,
		StartTime:    startTime,
		EndTime:      endTime,
		Duration:     duration,
		ToolsCount:   len(req.Tools),
		SuccessCount: successCount,
		ErrorCount:   len(errors),
		Metadata: BatchExecutionMetadata{
			ConcurrentExecutions: be.maxConcurrent,
			DependencyGraph:      depGraph,
			ExecutionOrder:       executionOrder,
			ResourceUsage:        resourceStats,
		},
	}

	be.log.Info("Batch execution completed",
		"tools_count", len(req.Tools),
		"success_count", successCount,
		"error_count", len(errors),
		"duration", duration)

	return result, err
}

// executeWithConcurrency executes tools with concurrency control and dependency management
func (be *BatchExecutor) executeWithConcurrency(
	ctx context.Context,
	req BatchRequest,
	depGraph *DependencyGraph,
	executionOrder []string,
	results map[string]ToolResult,
	errors map[string]error,
	resourceStats *ResourceUsageStats,
) error {
	// Use a semaphore to control concurrency
	semaphore := make(chan struct{}, be.maxConcurrent)

	// Results channels
	resultChan := make(chan toolExecutionResult, len(req.Tools))

	// WaitGroup to track completion
	var wg sync.WaitGroup

	// Mutex to protect shared state
	var mu sync.Mutex

	// Track completion status
	completedTools := make(map[string]bool)

	// Track currently executing tools
	executingTools := make(map[string]bool)

	// Launch goroutine to process results
	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		be.processResults(resultChan, results, errors, completedTools, &mu)
	}()

	// Execute tools in dependency order
	for _, toolID := range executionOrder {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Wait for dependencies to complete
		if err := be.waitForDependencies(ctx, depGraph, toolID, completedTools, &mu); err != nil {
			return fmt.Errorf("dependency wait failed for tool %s: %w", toolID, err)
		}

		// Find the tool request
		var toolReq *ToolRequest
		for i := range req.Tools {
			if be.getToolID(&req.Tools[i], i) == toolID {
				// Create a deep copy to avoid pointer issues
				toolReqCopy := req.Tools[i]

				// Deep copy the parameters map
				paramsCopy := make(map[string]any)
				for k, v := range toolReqCopy.Parameters {
					paramsCopy[k] = v
				}
				toolReqCopy.Parameters = paramsCopy

				toolReq = &toolReqCopy
				break
			}
		}

		if toolReq == nil {
			return fmt.Errorf("tool request not found for ID: %s", toolID)
		}

		// Acquire semaphore (wait for available slot)
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}

		// Launch tool execution
		wg.Add(1)
		go func(id string, request ToolRequest) {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
			}()

			// Mark as executing
			mu.Lock()
			executingTools[id] = true
			mu.Unlock()

			// Send progress update
			be.sendProgressUpdate(req.Progress, ProgressUpdate{
				Type:      ProgressStarted,
				ToolID:    id,
				ToolName:  request.Name,
				Progress:  0.0,
				Message:   fmt.Sprintf("Starting execution of %s", request.Name),
				Timestamp: time.Now(),
			})

			// Execute the tool
			result, err := be.executeSingleTool(ctx, request)

			// Send result
			resultChan <- toolExecutionResult{
				ID:     id,
				Result: result,
				Error:  err,
			}

			// Mark as no longer executing
			mu.Lock()
			delete(executingTools, id)
			mu.Unlock()

			// Send completion update
			progressType := ProgressCompleted
			if err != nil {
				progressType = ProgressError
			}

			be.sendProgressUpdate(req.Progress, ProgressUpdate{
				Type:      progressType,
				ToolID:    id,
				ToolName:  request.Name,
				Progress:  1.0,
				Result:    &result,
				Error:     err,
				Timestamp: time.Now(),
			})

		}(toolID, *toolReq)
	}

	// Wait for all tools to complete
	wg.Wait()
	close(resultChan)

	// Wait for result processing to complete
	resultWg.Wait()

	return nil
}

// toolExecutionResult represents the result of a single tool execution
type toolExecutionResult struct {
	ID     string
	Result ToolResult
	Error  error
}

// processResults processes tool execution results
func (be *BatchExecutor) processResults(
	resultChan <-chan toolExecutionResult,
	results map[string]ToolResult,
	errors map[string]error,
	completedTools map[string]bool,
	mu *sync.Mutex,
) {
	for result := range resultChan {
		mu.Lock()

		if result.Error != nil {
			// Tool execution failed (couldn't run the tool at all)
			errors[result.ID] = result.Error
		} else {
			// Tool executed (but may have failed internally)
			results[result.ID] = result.Result

			// If the tool result indicates failure, also add to errors
			if !result.Result.Success && result.Result.Error != "" {
				errors[result.ID] = fmt.Errorf("tool execution failed: %s", result.Result.Error)
			}
		}

		completedTools[result.ID] = true
		mu.Unlock()
	}
}

// waitForDependencies waits for all dependencies of a tool to complete
func (be *BatchExecutor) waitForDependencies(
	ctx context.Context,
	depGraph *DependencyGraph,
	toolID string,
	completedTools map[string]bool,
	mu *sync.Mutex,
) error {
	node := depGraph.GetNode(toolID)
	if node == nil {
		return fmt.Errorf("tool not found in dependency graph: %s", toolID)
	}

	// If no dependencies, return immediately
	if len(node.Dependencies) == 0 {
		return nil
	}

	// Poll for dependency completion
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			mu.Lock()
			allCompleted := true
			for _, depID := range node.Dependencies {
				if !completedTools[depID] {
					allCompleted = false
					break
				}
			}
			mu.Unlock()

			if allCompleted {
				return nil
			}
		}
	}
}

// executeSingleTool executes a single tool
func (be *BatchExecutor) executeSingleTool(ctx context.Context, req ToolRequest) (ToolResult, error) {
	// Set context if not provided
	if req.Context == nil {
		req.Context = ctx
	}

	return be.registry.Execute(ctx, req)
}

// buildDependencyGraph builds a dependency graph from tool requests and dependencies
func (be *BatchExecutor) buildDependencyGraph(tools []ToolRequest, dependencies map[string][]string) (*DependencyGraph, error) {
	graph := NewDependencyGraph()

	// Add all tools as nodes
	for i, tool := range tools {
		toolID := be.getToolID(&tool, i)
		err := graph.AddNode(toolID, tool.Name, tool.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to add tool %s to graph: %w", toolID, err)
		}
	}

	// Add dependencies
	for toolID, deps := range dependencies {
		for _, depID := range deps {
			err := graph.AddDependency(toolID, depID)
			if err != nil {
				return nil, fmt.Errorf("failed to add dependency %s -> %s: %w", depID, toolID, err)
			}
		}
	}

	return graph, nil
}

// GetToolID generates a unique ID for a tool request (exported for testing)
func (be *BatchExecutor) GetToolID(req *ToolRequest, index int) string {
	return be.getToolID(req, index)
}

// getToolID generates a unique ID for a tool request
func (be *BatchExecutor) getToolID(req *ToolRequest, index int) string {
	// Use tool name + index as ID if no explicit ID is provided
	if id, exists := req.Parameters["id"]; exists {
		if idStr, ok := id.(string); ok {
			return idStr
		}
	}
	return fmt.Sprintf("%s_%d", req.Name, index)
}

// sendProgressUpdate sends a progress update if a sink is configured
func (be *BatchExecutor) sendProgressUpdate(sink chan<- ProgressUpdate, update ProgressUpdate) {
	if sink != nil {
		select {
		case sink <- update:
		default:
			// Don't block if the channel is full
		}
	}

	// Also send to batch executor's sink if configured
	if be.progressSink != nil {
		select {
		case be.progressSink <- update:
		default:
		}
	}
}
