package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchExecutor_Basic(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)
	
	executor := NewBatchExecutor(registry)
	assert.NotNil(t, executor)
}

func TestBatchExecutor_SimpleBatch(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)
	
	executor := NewBatchExecutor(registry).WithMaxConcurrent(5)
	
	// Create a simple batch request with multiple tools
	req := BatchRequest{
		Tools: []ToolRequest{
			{
				Name: "execute_bash",
				Parameters: map[string]any{
					"command": "echo 'Hello World'",
				},
			},
			{
				Name: "execute_bash", 
				Parameters: map[string]any{
					"command": "date",
				},
			},
		},
		Timeout: 30 * time.Second,
		Context: context.Background(),
	}
	
	result, err := executor.Execute(req)
	require.NoError(t, err)
	require.NotNil(t, result)
	
	// Check basic result structure
	assert.Equal(t, 2, result.ToolsCount)
	assert.Equal(t, 2, len(result.Results))
	assert.Equal(t, 0, len(result.Errors))
	assert.Equal(t, 2, result.SuccessCount)
	assert.Equal(t, 0, result.ErrorCount)
	assert.True(t, result.Duration > 0)
	
	// Check that tools executed successfully
	for _, toolResult := range result.Results {
		assert.True(t, toolResult.Success)
		assert.NotEmpty(t, toolResult.Content)
	}
}

func TestBatchExecutor_WithDependencies(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)
	
	executor := NewBatchExecutor(registry).WithMaxConcurrent(1) // Force sequential execution
	
	// Use a temporary file in the current directory (which should be allowed)
	tmpFile := filepath.Join(".", "batch_test_temp.txt")
	
	// Manually inspect what getToolID returns
	tools := []ToolRequest{
		{
			Name: "execute_bash",
			Parameters: map[string]any{
				"id":      "create_file",
				"command": fmt.Sprintf("echo 'test content' > %s", tmpFile),
			},
		},
		{
			Name: "read_file",
			Parameters: map[string]any{
				"id":   "read_file", 
				"path": tmpFile,
			},
		},
	}
	
	// Create batch with dependencies - second tool depends on first
	req := BatchRequest{
		Tools: tools,
		Dependencies: map[string][]string{
			"read_file": {"create_file"}, // read_file depends on create_file
		},
		Timeout: 30 * time.Second,
		Context: context.Background(),
	}
	
	result, err := executor.Execute(req)
	require.NoError(t, err)
	require.NotNil(t, result)
	
	// Verify execution order was respected
	assert.Equal(t, 2, result.ToolsCount)
	assert.Equal(t, 2, result.SuccessCount)
	
	// Verify dependency graph was created
	assert.NotNil(t, result.Metadata.DependencyGraph)
	assert.Equal(t, []string{"create_file", "read_file"}, result.Metadata.ExecutionOrder)
	
	// Clean up
	registry.Execute(context.Background(), ToolRequest{
		Name: "execute_bash",
		Parameters: map[string]any{
			"command": fmt.Sprintf("rm -f %s", tmpFile),
		},
	})
}

func TestBatchExecutor_ErrorHandling(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)
	
	executor := NewBatchExecutor(registry)
	
	// Create batch with one failing tool
	req := BatchRequest{
		Tools: []ToolRequest{
			{
				Name: "execute_bash",
				Parameters: map[string]any{
					"command": "echo 'passed'",
				},
			},
			{
				Name: "execute_bash",
				Parameters: map[string]any{
					"command": "exit 1", // This will fail
				},
			},
		},
		Timeout: 30 * time.Second,
		Context: context.Background(),
	}
	
	result, err := executor.Execute(req)
	require.NoError(t, err) // Batch execution itself shouldn't error
	require.NotNil(t, result)
	
	// Check that we have mixed results  
	assert.Equal(t, 2, result.ToolsCount)
	assert.Equal(t, 1, result.SuccessCount) // One success
	assert.Equal(t, 1, result.ErrorCount)   // One failure
}

func TestBatchExecutor_ContextCancellation(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)
	
	executor := NewBatchExecutor(registry)
	
	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	
	req := BatchRequest{
		Tools: []ToolRequest{
			{
				Name: "execute_bash",
				Parameters: map[string]any{
					"command": "sleep 10", // Long running command
				},
			},
		},
		Context: ctx,
	}
	
	// Cancel context immediately
	cancel()
	
	result, err := executor.Execute(req)
	
	// Should get context cancelled error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.Nil(t, result)
}

func TestBatchExecutor_ProgressUpdates(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)
	
	executor := NewBatchExecutor(registry)
	
	// Create progress channel
	progressChan := make(chan ProgressUpdate, 10)
	
	req := BatchRequest{
		Tools: []ToolRequest{
			{
				Name: "execute_bash",
				Parameters: map[string]any{
					"command": "echo 'test'",
				},
			},
		},
		Progress: progressChan,
		Timeout:  30 * time.Second,
		Context:  context.Background(),
	}
	
	result, err := executor.Execute(req)
	require.NoError(t, err)
	require.NotNil(t, result)
	
	close(progressChan)
	
	// Collect progress updates
	var updates []ProgressUpdate
	for update := range progressChan {
		updates = append(updates, update)
	}
	
	// Should have at least started and completed updates
	assert.GreaterOrEqual(t, len(updates), 2)
	
	// First update should be started
	assert.Equal(t, ProgressStarted, updates[0].Type)
	
	// Last update should be completed
	lastUpdate := updates[len(updates)-1]
	assert.Equal(t, ProgressCompleted, lastUpdate.Type)
	assert.Equal(t, 1.0, lastUpdate.Progress)
}

func TestBatchExecutor_ConcurrencyLimit(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)
	
	// Create executor with very limited concurrency
	executor := NewBatchExecutor(registry).WithMaxConcurrent(1)
	
	// Create multiple tools that should run sequentially due to limit
	var tools []ToolRequest
	for i := 0; i < 3; i++ {
		tools = append(tools, ToolRequest{
			Name: "execute_bash",
			Parameters: map[string]any{
				"command": "echo 'test'",
			},
		})
	}
	
	req := BatchRequest{
		Tools:   tools,
		Timeout: 30 * time.Second,
		Context: context.Background(),
	}
	
	start := time.Now()
	result, err := executor.Execute(req)
	duration := time.Since(start)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	
	// All tools should succeed
	assert.Equal(t, 3, result.SuccessCount)
	assert.Equal(t, 0, result.ErrorCount)
	
	// Should take some minimum time due to sequential execution
	// This is a rough check - timing tests can be flaky
	assert.True(t, duration > 10*time.Millisecond)
}

func TestBatchExecutor_EmptyBatch(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)
	
	executor := NewBatchExecutor(registry)
	
	req := BatchRequest{
		Tools: []ToolRequest{}, // Empty tools list
	}
	
	result, err := executor.Execute(req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no tools specified")
}

func TestBatchExecutor_InvalidDependencies(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)
	
	executor := NewBatchExecutor(registry)
	
	req := BatchRequest{
		Tools: []ToolRequest{
			{
				Name: "execute_bash",
				Parameters: map[string]any{
					"id":      "tool1",
					"command": "echo 'test'",
				},
			},
		},
		Dependencies: map[string][]string{
			"tool1": {"nonexistent"}, // Dependency on non-existent tool
		},
	}
	
	result, err := executor.Execute(req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "dependency graph")
}

func TestBatchExecutor_CircularDependencies(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterBuiltinTools()
	require.NoError(t, err)
	
	executor := NewBatchExecutor(registry)
	
	req := BatchRequest{
		Tools: []ToolRequest{
			{
				Name: "execute_bash",
				Parameters: map[string]any{
					"id":      "tool1",
					"command": "echo 'test1'",
				},
			},
			{
				Name: "execute_bash",
				Parameters: map[string]any{
					"id":      "tool2",
					"command": "echo 'test2'",
				},
			},
		},
		Dependencies: map[string][]string{
			"tool1": {"tool2"}, // tool1 depends on tool2
			"tool2": {"tool1"}, // tool2 depends on tool1 -> circular!
		},
	}
	
	result, err := executor.Execute(req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cycle")
}