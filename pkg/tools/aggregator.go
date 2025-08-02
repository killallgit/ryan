package tools

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ResultAggregator collects and manages results from multiple concurrent tool executions
type ResultAggregator struct {
	results       map[string]ToolResult
	errors        map[string]error
	expectedCount int
	completedCount int
	callbacks     map[string][]ResultCallback
	mu            sync.RWMutex
	done          chan struct{}
	timeout       time.Duration
	startTime     time.Time
}

// ResultCallback is called when a result is received
type ResultCallback func(id string, result ToolResult, err error)

// BatchResult represents the final result of a batch execution
type BatchResult struct {
	Results      map[string]ToolResult `json:"results"`
	Errors       map[string]error      `json:"errors"`
	StartTime    time.Time             `json:"start_time"`
	EndTime      time.Time             `json:"end_time"`
	Duration     time.Duration         `json:"duration"`
	TotalCount   int                   `json:"total_count"`
	SuccessCount int                   `json:"success_count"`
	ErrorCount   int                   `json:"error_count"`
}

// NewResultAggregator creates a new result aggregator
func NewResultAggregator(expectedCount int, timeout time.Duration) *ResultAggregator {
	return &ResultAggregator{
		results:       make(map[string]ToolResult),
		errors:        make(map[string]error),
		expectedCount: expectedCount,
		completedCount: 0,
		callbacks:     make(map[string][]ResultCallback),
		done:          make(chan struct{}),
		timeout:       timeout,
		startTime:     time.Now(),
	}
}

// AddResult adds a result for a specific tool execution
func (ra *ResultAggregator) AddResult(id string, result ToolResult, err error) {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	// Store the result and error
	ra.results[id] = result
	if err != nil {
		ra.errors[id] = err
	}

	ra.completedCount++

	// Call registered callbacks
	if callbacks, exists := ra.callbacks[id]; exists {
		for _, callback := range callbacks {
			go callback(id, result, err)
		}
	}

	// Call wildcard callbacks (registered with "*")
	if wildcardCallbacks, exists := ra.callbacks["*"]; exists {
		for _, callback := range wildcardCallbacks {
			go callback(id, result, err)
		}
	}

	// Check if we're done
	if ra.completedCount >= ra.expectedCount {
		select {
		case <-ra.done:
			// Already closed
		default:
			close(ra.done)
		}
	}
}

// OnResult registers a callback for when a specific result is received
func (ra *ResultAggregator) OnResult(id string, callback ResultCallback) {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	if ra.callbacks[id] == nil {
		ra.callbacks[id] = make([]ResultCallback, 0)
	}
	ra.callbacks[id] = append(ra.callbacks[id], callback)
}

// OnAnyResult registers a callback for when any result is received
func (ra *ResultAggregator) OnAnyResult(callback ResultCallback) {
	ra.OnResult("*", callback)
}

// Wait waits for all results to be collected or timeout to occur
func (ra *ResultAggregator) Wait(ctx context.Context) (BatchResult, error) {
	var finalTimeout <-chan time.Time
	if ra.timeout > 0 {
		finalTimeout = time.After(ra.timeout)
	}

	select {
	case <-ra.done:
		// All results collected
		return ra.GetBatchResult(), nil
	case <-finalTimeout:
		// Timeout occurred
		return ra.GetBatchResult(), fmt.Errorf("batch execution timed out after %v", ra.timeout)
	case <-ctx.Done():
		// Context cancelled
		return ra.GetBatchResult(), ctx.Err()
	}
}

// GetBatchResult returns the current batch result
func (ra *ResultAggregator) GetBatchResult() BatchResult {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	endTime := time.Now()
	successCount := 0
	errorCount := 0

	// Count successes and errors
	for _, result := range ra.results {
		if result.Success {
			successCount++
		} else {
			errorCount++
		}
	}

	// Create copies of maps to prevent external modification
	results := make(map[string]ToolResult)
	errors := make(map[string]error)
	
	for k, v := range ra.results {
		results[k] = v
	}
	for k, v := range ra.errors {
		errors[k] = v
	}

	return BatchResult{
		Results:      results,
		Errors:       errors,
		StartTime:    ra.startTime,
		EndTime:      endTime,
		Duration:     endTime.Sub(ra.startTime),
		TotalCount:   ra.expectedCount,
		SuccessCount: successCount,
		ErrorCount:   errorCount,
	}
}

// GetProgress returns the current progress information
func (ra *ResultAggregator) GetProgress() ProgressInfo {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	progress := float64(ra.completedCount) / float64(ra.expectedCount)
	if ra.expectedCount == 0 {
		progress = 1.0
	}

	return ProgressInfo{
		CompletedCount: ra.completedCount,
		TotalCount:     ra.expectedCount,
		Progress:       progress,
		ElapsedTime:    time.Since(ra.startTime),
		IsComplete:     ra.completedCount >= ra.expectedCount,
	}
}

// IsComplete returns true if all expected results have been collected
func (ra *ResultAggregator) IsComplete() bool {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	return ra.completedCount >= ra.expectedCount
}

// GetPartialResults returns the currently available results
func (ra *ResultAggregator) GetPartialResults() map[string]ToolResult {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	// Return a copy to prevent external modification
	results := make(map[string]ToolResult)
	for k, v := range ra.results {
		results[k] = v
	}
	return results
}

// ProgressInfo provides information about batch execution progress
type ProgressInfo struct {
	CompletedCount int           `json:"completed_count"`
	TotalCount     int           `json:"total_count"`
	Progress       float64       `json:"progress"`        // 0.0 to 1.0
	ElapsedTime    time.Duration `json:"elapsed_time"`
	IsComplete     bool          `json:"is_complete"`
}

// BatchExecutionContext provides context for batch tool execution
type BatchExecutionContext struct {
	ID          string
	StartTime   time.Time
	Timeout     time.Duration
	Aggregator  *ResultAggregator
	CancelFunc  context.CancelFunc
	ctx         context.Context
}

// NewBatchExecutionContext creates a new batch execution context
func NewBatchExecutionContext(id string, expectedCount int, timeout time.Duration) *BatchExecutionContext {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	
	return &BatchExecutionContext{
		ID:         id,
		StartTime:  time.Now(),
		Timeout:    timeout,
		Aggregator: NewResultAggregator(expectedCount, timeout),
		CancelFunc: cancel,
		ctx:        ctx,
	}
}

// Context returns the context for this batch execution
func (bec *BatchExecutionContext) Context() context.Context {
	return bec.ctx
}

// Cancel cancels the batch execution
func (bec *BatchExecutionContext) Cancel() {
	bec.CancelFunc()
}

// Wait waits for the batch execution to complete
func (bec *BatchExecutionContext) Wait() (BatchResult, error) {
	return bec.Aggregator.Wait(bec.ctx)
}

// CollectorPool manages multiple result aggregators for different batch operations
type CollectorPool struct {
	collectors map[string]*ResultAggregator
	mu         sync.RWMutex
}

// NewCollectorPool creates a new collector pool
func NewCollectorPool() *CollectorPool {
	return &CollectorPool{
		collectors: make(map[string]*ResultAggregator),
	}
}

// CreateAggregator creates a new result aggregator with the given ID
func (cp *CollectorPool) CreateAggregator(id string, expectedCount int, timeout time.Duration) *ResultAggregator {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	aggregator := NewResultAggregator(expectedCount, timeout)
	cp.collectors[id] = aggregator
	return aggregator
}

// GetAggregator retrieves an aggregator by ID
func (cp *CollectorPool) GetAggregator(id string) (*ResultAggregator, bool) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	aggregator, exists := cp.collectors[id]
	return aggregator, exists
}

// RemoveAggregator removes an aggregator from the pool
func (cp *CollectorPool) RemoveAggregator(id string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	delete(cp.collectors, id)
}

// ListActiveAggregators returns a list of active aggregator IDs
func (cp *CollectorPool) ListActiveAggregators() []string {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	ids := make([]string, 0, len(cp.collectors))
	for id := range cp.collectors {
		ids = append(ids, id)
	}
	return ids
}

// GetStats returns statistics about the collector pool
func (cp *CollectorPool) GetStats() CollectorPoolStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	activeCount := len(cp.collectors)
	completedCount := 0

	for _, aggregator := range cp.collectors {
		if aggregator.IsComplete() {
			completedCount++
		}
	}

	return CollectorPoolStats{
		ActiveAggregators:    activeCount,
		CompletedAggregators: completedCount,
		PendingAggregators:   activeCount - completedCount,
	}
}

// CollectorPoolStats provides statistics about the collector pool
type CollectorPoolStats struct {
	ActiveAggregators    int `json:"active_aggregators"`
	CompletedAggregators int `json:"completed_aggregators"`
	PendingAggregators   int `json:"pending_aggregators"`
}