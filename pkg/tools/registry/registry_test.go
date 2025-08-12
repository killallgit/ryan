package registry

import (
	"context"
	"fmt"
	"testing"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/tools"
)

// mockTool is a simple mock tool for testing
type mockTool struct {
	name string
	desc string
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.desc
}

func (m *mockTool) Call(ctx context.Context, input string) (string, error) {
	return "mock response", nil
}

func TestNewRegistry(t *testing.T) {
	r := New()
	assert.NotNil(t, r)
	assert.Empty(t, r.GetAll())
}

func TestRegister(t *testing.T) {
	r := New()

	// Test successful registration
	err := r.Register("test_tool", func(skipPermissions bool) tools.Tool {
		return &mockTool{name: "test_tool", desc: "Test tool"}
	})
	assert.NoError(t, err)
	assert.True(t, r.IsRegistered("test_tool"))

	// Test duplicate registration
	err = r.Register("test_tool", func(skipPermissions bool) tools.Tool {
		return &mockTool{name: "test_tool", desc: "Test tool"}
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test nil factory
	err = r.Register("nil_tool", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "factory cannot be nil")
}

func TestGet(t *testing.T) {
	r := New()

	// Register a tool
	r.Register("test_tool", func(skipPermissions bool) tools.Tool {
		return &mockTool{name: "test_tool", desc: "Test tool"}
	})

	// Test successful retrieval
	tool, err := r.Get("test_tool", false)
	assert.NoError(t, err)
	assert.NotNil(t, tool)
	assert.Equal(t, "test_tool", tool.Name())

	// Test non-existent tool
	tool, err = r.Get("non_existent", false)
	assert.Error(t, err)
	assert.Nil(t, tool)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetAll(t *testing.T) {
	r := New()

	// Register multiple tools
	r.Register("tool1", func(skipPermissions bool) tools.Tool {
		return &mockTool{name: "tool1", desc: "Tool 1"}
	})
	r.Register("tool2", func(skipPermissions bool) tools.Tool {
		return &mockTool{name: "tool2", desc: "Tool 2"}
	})

	names := r.GetAll()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "tool1")
	assert.Contains(t, names, "tool2")
}

func TestIsRegistered(t *testing.T) {
	r := New()

	// Register a tool
	r.Register("test_tool", func(skipPermissions bool) tools.Tool {
		return &mockTool{name: "test_tool", desc: "Test tool"}
	})

	assert.True(t, r.IsRegistered("test_tool"))
	assert.False(t, r.IsRegistered("non_existent"))
}

func TestClear(t *testing.T) {
	r := New()

	// Register tools
	r.Register("tool1", func(skipPermissions bool) tools.Tool {
		return &mockTool{name: "tool1", desc: "Tool 1"}
	})
	r.Register("tool2", func(skipPermissions bool) tools.Tool {
		return &mockTool{name: "tool2", desc: "Tool 2"}
	})

	assert.Len(t, r.GetAll(), 2)

	// Clear registry
	r.Clear()
	assert.Empty(t, r.GetAll())
	assert.False(t, r.IsRegistered("tool1"))
	assert.False(t, r.IsRegistered("tool2"))
}

func TestGetEnabled(t *testing.T) {
	r := New()

	// Register tools
	r.Register("file_read", func(skipPermissions bool) tools.Tool {
		return &mockTool{name: "file_read", desc: "File Read"}
	})
	r.Register("file_write", func(skipPermissions bool) tools.Tool {
		return &mockTool{name: "file_write", desc: "File Write"}
	})
	r.Register("bash", func(skipPermissions bool) tools.Tool {
		return &mockTool{name: "bash", desc: "Bash"}
	})

	// Test with tools disabled
	settings := &config.Settings{}
	settings.Tools.Enabled = false
	enabledTools := r.GetEnabled(settings, false)
	assert.Empty(t, enabledTools)

	// Test with tools enabled but individual tools disabled
	settings.Tools.Enabled = true
	settings.Tools.File.Read.Enabled = false
	settings.Tools.File.Write.Enabled = false
	settings.Tools.Bash.Enabled = false
	enabledTools = r.GetEnabled(settings, false)
	assert.Empty(t, enabledTools)

	// Test with specific tools enabled
	settings.Tools.File.Read.Enabled = true
	settings.Tools.Bash.Enabled = true
	enabledTools = r.GetEnabled(settings, false)
	assert.Len(t, enabledTools, 2)

	// Verify the correct tools are returned
	toolNames := make([]string, len(enabledTools))
	for i, tool := range enabledTools {
		toolNames[i] = tool.Name()
	}
	assert.Contains(t, toolNames, "file_read")
	assert.Contains(t, toolNames, "bash")
	assert.NotContains(t, toolNames, "file_write")
}

func TestGlobalRegistry(t *testing.T) {
	// Test that global registry is initialized
	g := Global()
	assert.NotNil(t, g)

	// Clear global registry for clean test
	g.Clear()

	// Test global registry operations
	err := g.Register("global_test", func(skipPermissions bool) tools.Tool {
		return &mockTool{name: "global_test", desc: "Global test tool"}
	})
	require.NoError(t, err)

	tool, err := g.Get("global_test", false)
	assert.NoError(t, err)
	assert.NotNil(t, tool)
	assert.Equal(t, "global_test", tool.Name())

	// Clean up
	g.Clear()
}

func TestConcurrentAccess(t *testing.T) {
	r := New()
	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			name := fmt.Sprintf("tool_%d", i)
			r.Register(name, func(skipPermissions bool) tools.Tool {
				return &mockTool{name: name, desc: "Concurrent tool"}
			})
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			r.GetAll()
			r.IsRegistered(fmt.Sprintf("tool_%d", i))
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify some tools were registered
	assert.True(t, len(r.GetAll()) > 0)
}
