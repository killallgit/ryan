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

// ExecutorMockTool implements Tool interface for testing
type ExecutorMockTool struct {
	name        string
	description string
	schema      map[string]interface{}
	executeFunc func(ctx context.Context, params map[string]interface{}) (ToolResult, error)
	delay       time.Duration
}

func (m *ExecutorMockTool) Name() string {
	return m.name
}

func (m *ExecutorMockTool) Description() string {
	return m.description
}

func (m *ExecutorMockTool) JSONSchema() map[string]interface{} {
	return m.schema
}

func (m *ExecutorMockTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	
	if m.executeFunc != nil {
		return m.executeFunc(ctx, params)
	}
	
	return ToolResult{
		Success: true,
		Content: fmt.Sprintf("Mock tool %s executed with params: %v", m.name, params),
	}, nil
}

func NewExecutorMockTool(name string) *ExecutorMockTool {
	return &ExecutorMockTool{
		name:        name,
		description: fmt.Sprintf("Mock tool %s for testing", name),
		schema:      NewJSONSchema(),
	}
}

func NewSlowExecutorMockTool(name string, delay time.Duration) *ExecutorMockTool {
	tool := NewExecutorMockTool(name)
	tool.delay = delay
	return tool
}

func NewErrorExecutorMockTool(name string, errMsg string) *ExecutorMockTool {
	tool := NewExecutorMockTool(name)
	tool.executeFunc = func(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
		return ToolResult{}, fmt.Errorf(errMsg)
	}
	return tool
}

func TestExecutorPool_NewExecutorPool(t *testing.T) {
	tests := []struct {
		name        string
		workerCount int
		expected    int
	}{
		{"positive worker count", 8, 8},
		{"zero worker count defaults to 4", 0, 4},
		{"negative worker count defaults to 4", -1, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewExecutorPool(tt.workerCount)
			assert.Equal(t, tt.expected, pool.workerCount)
			assert.False(t, pool.isRunning)
			assert.NotNil(t, pool.jobQueue)
			assert.NotNil(t, pool.resultChannels)
		})
	}
}

func TestExecutorPool_StartStop(t *testing.T) {
	pool := NewExecutorPool(2)

	// Test starting
	err := pool.Start()
	require.NoError(t, err)
	assert.True(t, pool.isRunning)
	assert.Len(t, pool.workers, 2)

	// Test starting already running pool
	err = pool.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test stopping
	err = pool.Stop()
	require.NoError(t, err)
	assert.False(t, pool.isRunning)

	// Test stopping already stopped pool
	err = pool.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestExecutorPool_SingleToolExecution(t *testing.T) {
	pool := NewExecutorPool(2)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	tool := NewExecutorMockTool("test_tool")
	request := ToolRequest{
		Name:       "test_tool",
		Parameters: map[string]interface{}{"param1": "value1"},
		Context:    context.Background(),
	}

	resultChan, err := pool.Submit("job1", tool, request)
	require.NoError(t, err)
	require.NotNil(t, resultChan)

	// Wait for result with timeout
	select {
	case result := <-resultChan:
		assert.True(t, result.Success)
		assert.Contains(t, result.Content, "test_tool executed")
		assert.Equal(t, "test_tool", result.Metadata.ToolName)
		assert.Greater(t, result.Metadata.ExecutionTime, time.Duration(0))
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for result")
	}
}

func TestExecutorPool_ConcurrentExecution(t *testing.T) {
	workerCount := 4
	jobCount := 10
	
	pool := NewExecutorPool(workerCount)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	// Create jobs that take some time to complete
	var wg sync.WaitGroup
	results := make([]ToolResult, jobCount)
	
	startTime := time.Now()
	
	for i := 0; i < jobCount; i++ {
		wg.Add(1)
		go func(jobID int) {
			defer wg.Done()
			
			tool := NewSlowExecutorMockTool(fmt.Sprintf("tool_%d", jobID), 100*time.Millisecond)
			request := ToolRequest{
				Name:       tool.Name(),
				Parameters: map[string]interface{}{"job_id": jobID},
				Context:    context.Background(),
			}

			resultChan, err := pool.Submit(fmt.Sprintf("job_%d", jobID), tool, request)
			require.NoError(t, err)

			select {
			case result := <-resultChan:
				results[jobID] = result
			case <-time.After(5 * time.Second):
				t.Errorf("Job %d timed out", jobID)
			}
		}(i)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	// Verify all jobs completed successfully
	for i, result := range results {
		assert.True(t, result.Success, "Job %d should succeed", i)
		assert.Contains(t, result.Content, fmt.Sprintf("tool_%d", i))
	}

	// With 4 workers and 10 jobs taking 100ms each, total time should be less than
	// sequential execution (1000ms) due to concurrency. Allow some buffer for timing variations.
	assert.Less(t, totalTime, 1200*time.Millisecond, "Concurrent execution should be faster than sequential")
}

func TestExecutorPool_ErrorHandling(t *testing.T) {
	pool := NewExecutorPool(2)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	tool := NewErrorExecutorMockTool("error_tool", "intentional error")
	request := ToolRequest{
		Name:       "error_tool",
		Parameters: map[string]interface{}{},
		Context:    context.Background(),
	}

	resultChan, err := pool.Submit("error_job", tool, request)
	require.NoError(t, err)

	select {
	case result := <-resultChan:
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "intentional error")
		assert.Equal(t, "error_tool", result.Metadata.ToolName)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for error result")
	}
}

func TestExecutorPool_ContextCancellation(t *testing.T) {
	pool := NewExecutorPool(2)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	
	tool := &ExecutorMockTool{
		name: "cancellable_tool",
		executeFunc: func(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
			// Simulate work that respects context cancellation
			select {
			case <-time.After(500 * time.Millisecond):
				return ToolResult{Success: true, Content: "Completed"}, nil
			case <-ctx.Done():
				return ToolResult{}, ctx.Err()
			}
		},
	}

	request := ToolRequest{
		Name:       "cancellable_tool",
		Parameters: map[string]interface{}{},
		Context:    ctx,
	}

	resultChan, err := pool.Submit("cancel_job", tool, request)
	require.NoError(t, err)

	// Cancel the context after a short delay
	time.AfterFunc(100*time.Millisecond, cancel)

	select {
	case result := <-resultChan:
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "context canceled")
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for cancellation result")
	}
}

func TestExecutorPool_Stats(t *testing.T) {
	pool := NewExecutorPool(3)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	// Get initial stats
	stats := pool.GetStats()
	assert.Equal(t, 3, stats.WorkerCount)
	assert.Equal(t, 0, stats.QueuedJobs)
	assert.Equal(t, 0, stats.ActiveJobs)
	assert.True(t, stats.IsRunning)
	assert.Equal(t, 3, stats.TotalWorkers)

	// Submit a slow job to see active job count
	tool := NewSlowExecutorMockTool("slow_tool", 200*time.Millisecond)
	request := ToolRequest{
		Name:       "slow_tool",
		Parameters: map[string]interface{}{},
		Context:    context.Background(),
	}

	resultChan, err := pool.Submit("slow_job", tool, request)
	require.NoError(t, err)

	// Check stats while job is running
	time.Sleep(50 * time.Millisecond) // Give job time to start
	stats = pool.GetStats()
	// Note: ActiveJobs count may be 0 or 1 depending on timing
	// We just verify the structure is correct
	assert.Equal(t, 3, stats.WorkerCount)
	assert.True(t, stats.IsRunning)

	// Wait for job to complete
	<-resultChan
}

func TestExecutorPool_QueueFullScenario(t *testing.T) {
	// Create a pool with small capacity to test queue behavior
	pool := NewExecutorPool(1)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	// Submit multiple jobs quickly to test queue handling
	tool := NewSlowExecutorMockTool("blocking_tool", 500*time.Millisecond)
	
	var results []<-chan ToolResult
	for i := 0; i < 5; i++ {
		request := ToolRequest{
			Name:       "blocking_tool",
			Parameters: map[string]interface{}{"job_id": i},
			Context:    context.Background(),
		}

		resultChan, err := pool.Submit(fmt.Sprintf("job_%d", i), tool, request)
		require.NoError(t, err, "Job %d should be submitted successfully", i)
		results = append(results, resultChan)
	}

	// All jobs should eventually complete despite limited workers
	for i, resultChan := range results {
		select {
		case result := <-resultChan:
			assert.True(t, result.Success, "Job %d should succeed", i)
		case <-time.After(10 * time.Second):
			t.Fatalf("Job %d timed out", i)
		}
	}
}

func TestExecutorPool_WorkerRecovery(t *testing.T) {
	pool := NewExecutorPool(2)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	// Create a tool that panics
	panicTool := &ExecutorMockTool{
		name: "panic_tool",
		executeFunc: func(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
			panic("intentional panic for testing")
		},
	}

	request := ToolRequest{
		Name:       "panic_tool",
		Parameters: map[string]interface{}{},
		Context:    context.Background(),
	}

	// Submit the panicking job
	resultChan, err := pool.Submit("panic_job", panicTool, request)
	require.NoError(t, err)

	// The worker should recover and the pool should continue functioning
	select {
	case <-resultChan:
		// The channel should be closed, indicating the job completed (even if with panic)
	case <-time.After(2 * time.Second):
		t.Fatal("Worker did not recover from panic")
	}

	// Verify pool is still functional by submitting a normal job
	normalTool := NewExecutorMockTool("normal_tool")
	normalRequest := ToolRequest{
		Name:       "normal_tool",
		Parameters: map[string]interface{}{},
		Context:    context.Background(),
	}

	normalResultChan, err := pool.Submit("normal_job", normalTool, normalRequest)
	require.NoError(t, err)

	select {
	case result := <-normalResultChan:
		assert.True(t, result.Success)
	case <-time.After(2 * time.Second):
		t.Fatal("Pool not functional after panic recovery")
	}
}