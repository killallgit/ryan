package tools

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResultAggregator_NewResultAggregator(t *testing.T) {
	aggregator := NewResultAggregator(5, 10*time.Second)
	
	assert.NotNil(t, aggregator)
	assert.Equal(t, 5, aggregator.expectedCount)
	assert.Equal(t, 0, aggregator.completedCount)
	assert.Equal(t, 10*time.Second, aggregator.timeout)
	assert.NotNil(t, aggregator.results)
	assert.NotNil(t, aggregator.errors)
	assert.NotNil(t, aggregator.callbacks)
}

func TestResultAggregator_AddResult(t *testing.T) {
	aggregator := NewResultAggregator(3, 5*time.Second)
	
	// Add successful result
	result1 := ToolResult{Success: true, Content: "result1"}
	aggregator.AddResult("tool1", result1, nil)
	
	progress := aggregator.GetProgress()
	assert.Equal(t, 1, progress.CompletedCount)
	assert.Equal(t, 3, progress.TotalCount)
	assert.InDelta(t, 0.33, progress.Progress, 0.01)
	assert.False(t, progress.IsComplete)
	
	// Add error result
	result2 := ToolResult{Success: false, Error: "tool failed"}
	err2 := fmt.Errorf("execution error")
	aggregator.AddResult("tool2", result2, err2)
	
	progress = aggregator.GetProgress()
	assert.Equal(t, 2, progress.CompletedCount)
	assert.InDelta(t, 0.67, progress.Progress, 0.01)
	
	// Add final result
	result3 := ToolResult{Success: true, Content: "result3"}
	aggregator.AddResult("tool3", result3, nil)
	
	progress = aggregator.GetProgress()
	assert.Equal(t, 3, progress.CompletedCount)
	assert.Equal(t, 1.0, progress.Progress)
	assert.True(t, progress.IsComplete)
	assert.True(t, aggregator.IsComplete())
}

func TestResultAggregator_GetBatchResult(t *testing.T) {
	aggregator := NewResultAggregator(2, 5*time.Second)
	startTime := time.Now()
	
	// Add results
	result1 := ToolResult{Success: true, Content: "success"}
	result2 := ToolResult{Success: false, Error: "failure"}
	err2 := fmt.Errorf("tool error")
	
	aggregator.AddResult("tool1", result1, nil)
	aggregator.AddResult("tool2", result2, err2)
	
	batchResult := aggregator.GetBatchResult()
	
	assert.Equal(t, 2, batchResult.TotalCount)
	assert.Equal(t, 1, batchResult.SuccessCount)
	assert.Equal(t, 1, batchResult.ErrorCount)
	assert.True(t, batchResult.StartTime.After(startTime.Add(-1*time.Second)))
	assert.True(t, batchResult.EndTime.After(batchResult.StartTime))
	assert.Greater(t, batchResult.Duration, time.Duration(0))
	
	// Check results map
	assert.Len(t, batchResult.Results, 2)
	assert.Equal(t, result1, batchResult.Results["tool1"])
	assert.Equal(t, result2, batchResult.Results["tool2"])
	
	// Check errors map
	assert.Len(t, batchResult.Errors, 1)
	assert.Equal(t, err2, batchResult.Errors["tool2"])
}

func TestResultAggregator_Callbacks(t *testing.T) {
	aggregator := NewResultAggregator(3, 5*time.Second)
	
	var callbackResults []string
	var mu sync.Mutex
	
	// Register specific callback
	aggregator.OnResult("tool1", func(id string, result ToolResult, err error) {
		mu.Lock()
		defer mu.Unlock()
		callbackResults = append(callbackResults, fmt.Sprintf("specific:%s:%s", id, result.Content))
	})
	
	// Register wildcard callback
	aggregator.OnAnyResult(func(id string, result ToolResult, err error) {
		mu.Lock()
		defer mu.Unlock()
		callbackResults = append(callbackResults, fmt.Sprintf("wildcard:%s:%s", id, result.Content))
	})
	
	// Add results
	result1 := ToolResult{Success: true, Content: "content1"}
	result2 := ToolResult{Success: true, Content: "content2"}
	result3 := ToolResult{Success: true, Content: "content3"}
	
	aggregator.AddResult("tool1", result1, nil)
	aggregator.AddResult("tool2", result2, nil)
	aggregator.AddResult("tool3", result3, nil)
	
	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)
	
	mu.Lock()
	defer mu.Unlock()
	
	// Should have specific callback for tool1 and wildcard callbacks for all
	assert.Len(t, callbackResults, 4)
	assert.Contains(t, callbackResults, "specific:tool1:content1")
	assert.Contains(t, callbackResults, "wildcard:tool1:content1")
	assert.Contains(t, callbackResults, "wildcard:tool2:content2")
	assert.Contains(t, callbackResults, "wildcard:tool3:content3")
}

func TestResultAggregator_Wait(t *testing.T) {
	t.Run("successful completion", func(t *testing.T) {
		aggregator := NewResultAggregator(2, 5*time.Second)
		
		// Add results in goroutines to simulate concurrent execution
		go func() {
			time.Sleep(100 * time.Millisecond)
			aggregator.AddResult("tool1", ToolResult{Success: true, Content: "result1"}, nil)
		}()
		
		go func() {
			time.Sleep(200 * time.Millisecond)
			aggregator.AddResult("tool2", ToolResult{Success: true, Content: "result2"}, nil)
		}()
		
		ctx := context.Background()
		batchResult, err := aggregator.Wait(ctx)
		
		require.NoError(t, err)
		assert.Equal(t, 2, batchResult.TotalCount)
		assert.Equal(t, 2, batchResult.SuccessCount)
		assert.Equal(t, 0, batchResult.ErrorCount)
	})
	
	t.Run("timeout", func(t *testing.T) {
		aggregator := NewResultAggregator(2, 200*time.Millisecond)
		
		// Only add one result, so it will timeout waiting for the second
		go func() {
			time.Sleep(50 * time.Millisecond)
			aggregator.AddResult("tool1", ToolResult{Success: true, Content: "result1"}, nil)
		}()
		
		ctx := context.Background()
		batchResult, err := aggregator.Wait(ctx)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
		assert.Equal(t, 2, batchResult.TotalCount)
		assert.Equal(t, 1, batchResult.SuccessCount)
		assert.Equal(t, 0, batchResult.ErrorCount)
	})
	
	t.Run("context cancellation", func(t *testing.T) {
		aggregator := NewResultAggregator(2, 5*time.Second)
		
		ctx, cancel := context.WithCancel(context.Background())
		
		// Cancel context after short delay
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()
		
		batchResult, err := aggregator.Wait(ctx)
		
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Equal(t, 2, batchResult.TotalCount)
		assert.Equal(t, 0, batchResult.SuccessCount)
	})
}

func TestResultAggregator_ConcurrentAccess(t *testing.T) {
	aggregator := NewResultAggregator(100, 10*time.Second)
	
	var wg sync.WaitGroup
	
	// Simulate concurrent result additions
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			result := ToolResult{Success: true, Content: fmt.Sprintf("result%d", id)}
			aggregator.AddResult(fmt.Sprintf("tool%d", id), result, nil)
		}(i)
	}
	
	// Simulate concurrent progress checks
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				progress := aggregator.GetProgress()
				_ = progress // Just accessing to test concurrent reads
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}
	
	wg.Wait()
	
	assert.True(t, aggregator.IsComplete())
	batchResult := aggregator.GetBatchResult()
	assert.Equal(t, 100, batchResult.TotalCount)
	assert.Equal(t, 100, batchResult.SuccessCount)
	assert.Equal(t, 0, batchResult.ErrorCount)
}

func TestBatchExecutionContext(t *testing.T) {
	ctx := NewBatchExecutionContext("test-batch", 3, 2*time.Second)
	
	assert.Equal(t, "test-batch", ctx.ID)
	assert.Equal(t, 2*time.Second, ctx.Timeout)
	assert.NotNil(t, ctx.Aggregator)
	assert.NotNil(t, ctx.CancelFunc)
	
	// Test context
	contextFromBatch := ctx.Context()
	assert.NotNil(t, contextFromBatch)
	
	// Test cancellation
	ctx.Cancel()
	
	select {
	case <-contextFromBatch.Done():
		// Context was cancelled correctly
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context was not cancelled")
	}
}

func TestCollectorPool(t *testing.T) {
	pool := NewCollectorPool()
	
	// Create aggregators
	agg1 := pool.CreateAggregator("batch1", 2, 5*time.Second)
	agg2 := pool.CreateAggregator("batch2", 3, 5*time.Second)
	
	assert.NotNil(t, agg1)
	assert.NotNil(t, agg2)
	
	// Test retrieval
	retrieved1, exists1 := pool.GetAggregator("batch1")
	assert.True(t, exists1)
	assert.Equal(t, agg1, retrieved1)
	
	_, exists3 := pool.GetAggregator("nonexistent")
	assert.False(t, exists3)
	
	// Test listing
	active := pool.ListActiveAggregators()
	assert.Len(t, active, 2)
	assert.Contains(t, active, "batch1")
	assert.Contains(t, active, "batch2")
	
	// Complete one aggregator
	agg1.AddResult("tool1", ToolResult{Success: true}, nil)
	agg1.AddResult("tool2", ToolResult{Success: true}, nil)
	
	// Test stats
	stats := pool.GetStats()
	assert.Equal(t, 2, stats.ActiveAggregators)
	assert.Equal(t, 1, stats.CompletedAggregators)
	assert.Equal(t, 1, stats.PendingAggregators)
	
	// Remove aggregator
	pool.RemoveAggregator("batch1")
	active = pool.ListActiveAggregators()
	assert.Len(t, active, 1)
	assert.Contains(t, active, "batch2")
}

func TestResultAggregator_GetPartialResults(t *testing.T) {
	aggregator := NewResultAggregator(3, 5*time.Second)
	
	// Add partial results
	result1 := ToolResult{Success: true, Content: "result1"}
	result2 := ToolResult{Success: false, Error: "error2"}
	
	aggregator.AddResult("tool1", result1, nil)
	aggregator.AddResult("tool2", result2, fmt.Errorf("some error"))
	
	// Get partial results
	partial := aggregator.GetPartialResults()
	assert.Len(t, partial, 2)
	assert.Equal(t, result1, partial["tool1"])
	assert.Equal(t, result2, partial["tool2"])
	
	// Ensure it's a copy (modifying returned map shouldn't affect internal state)
	partial["tool3"] = ToolResult{Success: true, Content: "modified"}
	
	partialAgain := aggregator.GetPartialResults()
	assert.Len(t, partialAgain, 2) // Should still be 2, not 3
	_, exists := partialAgain["tool3"]
	assert.False(t, exists)
}

func TestResultAggregator_EdgeCases(t *testing.T) {
	t.Run("zero expected count", func(t *testing.T) {
		aggregator := NewResultAggregator(0, 1*time.Second)
		
		progress := aggregator.GetProgress()
		assert.Equal(t, 0, progress.CompletedCount)
		assert.Equal(t, 0, progress.TotalCount)
		assert.Equal(t, 1.0, progress.Progress) // Should be 1.0 when expected count is 0
		assert.True(t, progress.IsComplete)
		assert.True(t, aggregator.IsComplete())
	})
	
	t.Run("more results than expected", func(t *testing.T) {
		aggregator := NewResultAggregator(2, 1*time.Second)
		
		// Add more results than expected
		aggregator.AddResult("tool1", ToolResult{Success: true}, nil)
		aggregator.AddResult("tool2", ToolResult{Success: true}, nil)
		aggregator.AddResult("tool3", ToolResult{Success: true}, nil)
		
		progress := aggregator.GetProgress()
		assert.Equal(t, 3, progress.CompletedCount)
		assert.Equal(t, 2, progress.TotalCount)
		assert.Greater(t, progress.Progress, 1.0) // Progress can exceed 1.0
		assert.True(t, progress.IsComplete)
		
		batchResult := aggregator.GetBatchResult()
		assert.Equal(t, 2, batchResult.TotalCount) // Expected count remains unchanged
		assert.Len(t, batchResult.Results, 3)      // But all results are included
	})
	
	t.Run("no timeout", func(t *testing.T) {
		aggregator := NewResultAggregator(1, 0) // No timeout
		
		go func() {
			time.Sleep(100 * time.Millisecond)
			aggregator.AddResult("tool1", ToolResult{Success: true}, nil)
		}()
		
		ctx := context.Background()
		batchResult, err := aggregator.Wait(ctx)
		
		require.NoError(t, err)
		assert.True(t, aggregator.IsComplete())
		assert.Equal(t, 1, batchResult.SuccessCount)
	})
}