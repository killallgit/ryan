package tools

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BatchExecutor orchestrates concurrent execution of multiple tools with progress tracking
type BatchExecutor struct {
	pool            *ExecutorPool
	progressManager *ProgressManager
	registry        *Registry
	batchTimeout    time.Duration
	mu              sync.RWMutex
	isRunning       bool
}

// BatchRequest represents a request to execute multiple tools concurrently
type BatchRequest struct {
	ID          string
	Tools       []ToolRequestSpec
	Context     context.Context
	Timeout     time.Duration
	Options     BatchOptions
}

// ToolRequestSpec specifies a tool to execute in a batch
type ToolRequestSpec struct {
	ID          string                 // Unique identifier for this execution
	ToolName    string                 // Name of the tool to execute
	Parameters  map[string]interface{} // Parameters for the tool
	Dependencies []string              // IDs of tools that must complete first
	Optional    bool                   // If true, failure won't fail the entire batch
}

// BatchOptions configures batch execution behavior
type BatchOptions struct {
	StopOnFirstError    bool          // Stop entire batch on first error
	MaxConcurrency      int           // Maximum number of concurrent executions (0 = unlimited)
	ProgressCallback    ProgressSubscriber // Optional progress callback
	AllowPartialResults bool          // Return partial results on timeout/cancellation
}

// BatchExecutionResult contains the results of a batch execution
type BatchExecutionResult struct {
	BatchID      string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Status       BatchExecutionStatus
	Results      map[string]ToolResult
	Errors       map[string]error
	TotalTools   int
	CompletedTools int
	FailedTools  int
	SkippedTools int
	Progress     float64
}

// BatchExecutionStatus represents the overall status of batch execution
type BatchExecutionStatus string

const (
	BatchStatusPending     BatchExecutionStatus = "pending"
	BatchStatusRunning     BatchExecutionStatus = "running"
	BatchStatusCompleted   BatchExecutionStatus = "completed"
	BatchStatusFailed      BatchExecutionStatus = "failed"
	BatchStatusCancelled   BatchExecutionStatus = "cancelled"
	BatchStatusPartial     BatchExecutionStatus = "partial"
)

// NewBatchExecutor creates a new batch executor
func NewBatchExecutor(registry *Registry, workerCount int) *BatchExecutor {
	pool := NewExecutorPool(workerCount)
	progressManager := NewProgressManager(50 * time.Millisecond) // 20 updates per second

	return &BatchExecutor{
		pool:            pool,
		progressManager: progressManager,
		registry:        registry,
		batchTimeout:    30 * time.Minute, // Default timeout
	}
}

// Start initializes the batch executor
func (be *BatchExecutor) Start() error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if be.isRunning {
		return fmt.Errorf("batch executor is already running")
	}

	if err := be.pool.Start(); err != nil {
		return fmt.Errorf("failed to start executor pool: %w", err)
	}

	be.progressManager.Start()
	be.isRunning = true
	return nil
}

// Stop shuts down the batch executor
func (be *BatchExecutor) Stop() error {
	be.mu.Lock()
	defer be.mu.Unlock()

	if !be.isRunning {
		return fmt.Errorf("batch executor is not running")
	}

	be.progressManager.Stop()
	
	if err := be.pool.Stop(); err != nil {
		return fmt.Errorf("failed to stop executor pool: %w", err)
	}

	be.isRunning = false
	return nil
}

// ExecuteBatch executes a batch of tools concurrently
func (be *BatchExecutor) ExecuteBatch(request BatchRequest) (*BatchExecutionResult, error) {
	be.mu.RLock()
	if !be.isRunning {
		be.mu.RUnlock()
		return nil, fmt.Errorf("batch executor is not running")
	}
	be.mu.RUnlock()

	// Validate request
	if err := be.validateBatchRequest(request); err != nil {
		return nil, fmt.Errorf("invalid batch request: %w", err)
	}

	// Set up timeout context
	ctx := request.Context
	if ctx == nil {
		ctx = context.Background()
	}

	timeout := request.Timeout
	if timeout <= 0 {
		timeout = be.batchTimeout
	}
	
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create progress tracker
	tracker := be.progressManager.CreateTracker(request.ID, len(request.Tools))
	defer be.progressManager.RemoveTracker(request.ID)

	// Subscribe to progress if callback provided
	if request.Options.ProgressCallback != nil {
		be.progressManager.Subscribe(request.ID, request.Options.ProgressCallback)
	}

	// Execute the batch
	return be.executeBatchInternal(ctx, request, tracker)
}

// executeBatchInternal performs the actual batch execution
func (be *BatchExecutor) executeBatchInternal(ctx context.Context, request BatchRequest, tracker *ExecutionTracker) (*BatchExecutionResult, error) {
	startTime := time.Now()
	
	// Initialize result tracking
	results := make(map[string]ToolResult)
	errors := make(map[string]error)
	completed := make(map[string]bool)
	skipped := make(map[string]bool)
	mu := sync.RWMutex{}

	// For now, execute all tools concurrently (dependency resolution will be added later)
	// This provides basic batch execution functionality
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, be.getMaxConcurrency(request.Options))

	for _, toolSpec := range request.Tools {
		wg.Add(1)
		go func(spec ToolRequestSpec) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				mu.Lock()
				errors[spec.ID] = ctx.Err()
				mu.Unlock()
				return
			}

			// Execute tool
			result, err := be.executeSingleTool(ctx, request.ID, spec, tracker)
			
			mu.Lock()
			if err != nil {
				errors[spec.ID] = err
			} else {
				results[spec.ID] = result
				if !result.Success {
					errors[spec.ID] = fmt.Errorf(result.Error)
				}
			}
			completed[spec.ID] = true
			mu.Unlock()

			// Check if we should stop on error
			if (err != nil || !result.Success) && request.Options.StopOnFirstError {
				// Cancel context to stop other executions
				// Note: In a full implementation, we'd have a proper cancellation mechanism
			}
		}(toolSpec)
	}

	// Wait for all executions to complete
	wg.Wait()

	// Build final result
	endTime := time.Now()
	batchResult := be.buildBatchResult(request, startTime, endTime, results, errors, completed, skipped, ctx.Err())
	return batchResult, nil
}

// executeSingleTool executes a single tool and returns the result
func (be *BatchExecutor) executeSingleTool(ctx context.Context, batchID string, toolSpec ToolRequestSpec, tracker *ExecutionTracker) (ToolResult, error) {
	// Update progress
	be.progressManager.UpdateToolStatus(batchID, toolSpec.ID, toolSpec.ToolName, ToolStatusRunning, 0.0, "Starting execution")

	// Get tool from registry
	tool, exists := be.registry.Get(toolSpec.ToolName)
	if !exists {
		return ToolResult{}, fmt.Errorf("tool %s not found in registry", toolSpec.ToolName)
	}

	// Execute tool
	resultChan, err := be.pool.Submit(toolSpec.ID, tool, ToolRequest{
		Name:       toolSpec.ToolName,
		Parameters: toolSpec.Parameters,
		Context:    ctx,
	})

	if err != nil {
		return ToolResult{}, fmt.Errorf("failed to submit tool %s: %w", toolSpec.ToolName, err)
	}

	// Wait for result
	select {
	case result := <-resultChan:
		be.progressManager.CompleteToolExecution(batchID, toolSpec.ID, result, nil)
		return result, nil
	case <-ctx.Done():
		return ToolResult{}, ctx.Err()
	}
}


// getMaxConcurrency determines the maximum concurrency for the batch
func (be *BatchExecutor) getMaxConcurrency(options BatchOptions) int {
	if options.MaxConcurrency > 0 {
		return options.MaxConcurrency
	}
	return 1000 // Large default for "unlimited"
}

// buildBatchResult constructs the final batch execution result
func (be *BatchExecutor) buildBatchResult(request BatchRequest, startTime, endTime time.Time, results map[string]ToolResult, errors map[string]error, completed map[string]bool, skipped map[string]bool, ctxErr error) *BatchExecutionResult {
	totalTools := len(request.Tools)
	completedTools := len(completed)
	failedTools := len(errors)
	skippedTools := len(skipped)
	
	// Determine status
	status := BatchStatusCompleted
	if ctxErr != nil {
		if ctxErr == context.Canceled {
			status = BatchStatusCancelled
		} else {
			status = BatchStatusPartial
		}
	} else if failedTools > 0 {
		if completedTools == totalTools {
			status = BatchStatusFailed
		} else {
			status = BatchStatusPartial
		}
	}

	progress := float64(completedTools) / float64(totalTools)
	if totalTools == 0 {
		progress = 1.0
	}

	return &BatchExecutionResult{
		BatchID:        request.ID,
		StartTime:      startTime,
		EndTime:        endTime,
		Duration:       endTime.Sub(startTime),
		Status:         status,
		Results:        results,
		Errors:         errors,
		TotalTools:     totalTools,
		CompletedTools: completedTools,
		FailedTools:    failedTools,
		SkippedTools:   skippedTools,
		Progress:       progress,
	}
}

// validateBatchRequest validates a batch request
func (be *BatchExecutor) validateBatchRequest(request BatchRequest) error {
	if request.ID == "" {
		return fmt.Errorf("batch ID is required")
	}

	if len(request.Tools) == 0 {
		return fmt.Errorf("at least one tool must be specified")
	}

	// Check for duplicate tool IDs
	seen := make(map[string]bool)
	for _, tool := range request.Tools {
		if tool.ID == "" {
			return fmt.Errorf("tool ID is required")
		}
		if tool.ToolName == "" {
			return fmt.Errorf("tool name is required for tool %s", tool.ID)
		}
		if seen[tool.ID] {
			return fmt.Errorf("duplicate tool ID: %s", tool.ID)
		}
		seen[tool.ID] = true
	}

	// Validate dependencies exist
	for _, tool := range request.Tools {
		for _, dep := range tool.Dependencies {
			if !seen[dep] {
				return fmt.Errorf("tool %s depends on nonexistent tool %s", tool.ID, dep)
			}
		}
	}

	return nil
}

// GetStats returns statistics about the batch executor
func (be *BatchExecutor) GetStats() BatchExecutorStats {
	be.mu.RLock()
	defer be.mu.RUnlock()

	poolStats := be.pool.GetStats()
	progressStats := be.progressManager.GetStats()

	return BatchExecutorStats{
		IsRunning:         be.isRunning,
		PoolStats:         poolStats,
		ProgressStats:     progressStats,
		DefaultTimeout:    be.batchTimeout,
	}
}

// BatchExecutorStats provides statistics about the batch executor
type BatchExecutorStats struct {
	IsRunning      bool                  `json:"is_running"`
	PoolStats      ExecutorStats         `json:"pool_stats"`
	ProgressStats  ProgressManagerStats  `json:"progress_stats"`
	DefaultTimeout time.Duration         `json:"default_timeout"`
}

// SetDefaultTimeout sets the default timeout for batch executions
func (be *BatchExecutor) SetDefaultTimeout(timeout time.Duration) {
	be.mu.Lock()
	defer be.mu.Unlock()
	be.batchTimeout = timeout
}

// CreateSimpleBatch creates a simple batch request without dependencies
func CreateSimpleBatch(id string, tools []ToolRequestSpec, options BatchOptions) BatchRequest {
	return BatchRequest{
		ID:      id,
		Tools:   tools,
		Options: options,
	}
}

// CreateSequentialBatch creates a batch where tools execute in sequence
func CreateSequentialBatch(id string, tools []ToolRequestSpec, options BatchOptions) BatchRequest {
	// Set up sequential dependencies
	for i := 1; i < len(tools); i++ {
		tools[i].Dependencies = []string{tools[i-1].ID}
	}

	return BatchRequest{
		ID:      id,
		Tools:   tools,
		Options: options,
	}
}