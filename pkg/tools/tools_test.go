package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	t.Run("NewRegistry", func(t *testing.T) {
		registry := NewRegistry()
		assert.NotNil(t, registry)
		assert.Empty(t, registry.List())
	})

	t.Run("RegisterAndGet", func(t *testing.T) {
		registry := NewRegistry()
		mockTool := &MockTool{name: "test_tool"}

		err := registry.Register(mockTool)
		assert.NoError(t, err)

		tool, exists := registry.Get("test_tool")
		assert.True(t, exists)
		assert.Equal(t, mockTool, tool)

		// Test duplicate registration
		err = registry.Register(mockTool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("List", func(t *testing.T) {
		registry := NewRegistry()
		mockTool1 := &MockTool{name: "tool1"}
		mockTool2 := &MockTool{name: "tool2"}

		registry.Register(mockTool1)
		registry.Register(mockTool2)

		names := registry.List()
		assert.Len(t, names, 2)
		assert.Contains(t, names, "tool1")
		assert.Contains(t, names, "tool2")
	})

	t.Run("Unregister", func(t *testing.T) {
		registry := NewRegistry()
		mockTool := &MockTool{name: "test_tool"}

		registry.Register(mockTool)
		assert.Len(t, registry.List(), 1)

		registry.Unregister("test_tool")
		assert.Empty(t, registry.List())

		_, exists := registry.Get("test_tool")
		assert.False(t, exists)
	})

	t.Run("Execute", func(t *testing.T) {
		registry := NewRegistry()
		mockTool := &MockTool{
			name:   "test_tool",
			result: ToolResult{Success: true, Content: "test output"},
		}

		registry.Register(mockTool)

		req := ToolRequest{
			Name:       "test_tool",
			Parameters: map[string]any{"param": "value"},
			Context:    context.Background(),
		}

		result, err := registry.Execute(context.Background(), req)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "test output", result.Content)
	})

	t.Run("ExecuteNonExistentTool", func(t *testing.T) {
		registry := NewRegistry()
		req := ToolRequest{
			Name:       "nonexistent",
			Parameters: map[string]any{},
			Context:    context.Background(),
		}

		result, err := registry.Execute(context.Background(), req)
		assert.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "not found")
	})

	t.Run("RegisterBuiltinTools", func(t *testing.T) {
		registry := NewRegistry()
		err := registry.RegisterBuiltinTools()
		assert.NoError(t, err)

		names := registry.List()
		assert.Contains(t, names, "execute_bash")
		assert.Contains(t, names, "read_file")
	})
}

func TestBashTool(t *testing.T) {
	tool := NewBashTool()

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "execute_bash", tool.Name())
	})

	t.Run("Description", func(t *testing.T) {
		desc := tool.Description()
		assert.NotEmpty(t, desc)
		assert.Contains(t, strings.ToLower(desc), "bash")
	})

	t.Run("JSONSchema", func(t *testing.T) {
		schema := tool.JSONSchema()
		assert.Equal(t, "object", schema["type"])

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, properties, "command")

		required, ok := schema["required"].([]string)
		assert.True(t, ok)
		assert.Contains(t, required, "command")
	})

	t.Run("ExecuteSimpleCommand", func(t *testing.T) {
		params := map[string]any{
			"command": "echo 'hello world'",
		}

		result, err := tool.Execute(context.Background(), params)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Content, "hello world")
		assert.Equal(t, "execute_bash", result.Metadata.ToolName)
	})

	t.Run("ExecuteForbiddenCommand", func(t *testing.T) {
		params := map[string]any{
			"command": "sudo rm -rf /",
		}

		result, err := tool.Execute(context.Background(), params)
		assert.NoError(t, err) // Tool execution doesn't error, but result shows failure
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "forbidden")
	})

	t.Run("ExecuteEmptyCommand", func(t *testing.T) {
		params := map[string]any{
			"command": "",
		}

		result, err := tool.Execute(context.Background(), params)
		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "empty")
	})

	t.Run("ExecuteMissingCommand", func(t *testing.T) {
		params := map[string]any{}

		result, err := tool.Execute(context.Background(), params)
		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "required")
	})

	t.Run("ExecuteWithTimeout", func(t *testing.T) {
		// Set a very short timeout for testing
		tool.Timeout = 100 * time.Millisecond

		params := map[string]any{
			"command": "sleep 1", // Sleep longer than timeout
		}

		result, err := tool.Execute(context.Background(), params)
		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "timed out")
	})
}

func TestFileReadTool(t *testing.T) {
	tool := NewFileReadTool()

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "read_file", tool.Name())
	})

	t.Run("Description", func(t *testing.T) {
		desc := tool.Description()
		assert.NotEmpty(t, desc)
		assert.Contains(t, strings.ToLower(desc), "read")
		assert.Contains(t, strings.ToLower(desc), "file")
	})

	t.Run("JSONSchema", func(t *testing.T) {
		schema := tool.JSONSchema()
		assert.Equal(t, "object", schema["type"])

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, properties, "path")

		required, ok := schema["required"].([]string)
		assert.True(t, ok)
		assert.Contains(t, required, "path")
	})

	t.Run("ReadExistingFile", func(t *testing.T) {
		// Create a temporary file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		content := "Hello, World!\nThis is a test file."

		err := os.WriteFile(tmpFile, []byte(content), 0644)
		require.NoError(t, err)

		// Update tool's allowed paths to include temp directory
		tool.AllowedPaths = append(tool.AllowedPaths, tmpDir)

		params := map[string]any{
			"path": tmpFile,
		}

		result, err := tool.Execute(context.Background(), params)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, content, result.Content)
		assert.Equal(t, "read_file", result.Metadata.ToolName)
	})

	t.Run("ReadNonExistentFile", func(t *testing.T) {
		params := map[string]any{
			"path": "/nonexistent/file.txt",
		}

		result, err := tool.Execute(context.Background(), params)
		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "does not exist")
	})

	t.Run("ReadFileWithLineRange", func(t *testing.T) {
		// Create a temporary file with multiple lines
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"

		err := os.WriteFile(tmpFile, []byte(content), 0644)
		require.NoError(t, err)

		// Update tool's allowed paths
		tool.AllowedPaths = append(tool.AllowedPaths, tmpDir)

		params := map[string]any{
			"path":       tmpFile,
			"start_line": float64(2),
			"end_line":   float64(4),
		}

		result, err := tool.Execute(context.Background(), params)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "Line 2\nLine 3\nLine 4", result.Content)
	})

	t.Run("ReadEmptyPath", func(t *testing.T) {
		params := map[string]any{
			"path": "",
		}

		result, err := tool.Execute(context.Background(), params)
		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "empty")
	})

	t.Run("ReadMissingPath", func(t *testing.T) {
		params := map[string]any{}

		result, err := tool.Execute(context.Background(), params)
		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "required")
	})
}

func TestProviderAdapters(t *testing.T) {
	mockTool := &MockTool{
		name:        "test_tool",
		description: "A test tool",
		schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"param": map[string]any{
					"type":        "string",
					"description": "A parameter",
				},
			},
			"required": []string{"param"},
		},
	}

	t.Run("ConvertToOpenAI", func(t *testing.T) {
		definition, err := ConvertToProvider(mockTool, "openai")
		assert.NoError(t, err)

		assert.Equal(t, "function", definition["type"])

		function, ok := definition["function"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "test_tool", function["name"])
		assert.Equal(t, "A test tool", function["description"])
		assert.Equal(t, mockTool.schema, function["parameters"])
	})

	t.Run("ConvertToAnthropic", func(t *testing.T) {
		definition, err := ConvertToProvider(mockTool, "anthropic")
		assert.NoError(t, err)

		assert.Equal(t, "test_tool", definition["name"])
		assert.Equal(t, "A test tool", definition["description"])
		assert.Equal(t, mockTool.schema, definition["input_schema"])
	})

	t.Run("ConvertToMCP", func(t *testing.T) {
		definition, err := ConvertToProvider(mockTool, "mcp")
		assert.NoError(t, err)

		assert.Equal(t, "test_tool", definition["name"])
		assert.Equal(t, "A test tool", definition["description"])
		assert.Equal(t, mockTool.schema, definition["inputSchema"])
		assert.Equal(t, "tool", definition["type"])
	})

	t.Run("ConvertToUnsupportedProvider", func(t *testing.T) {
		_, err := ConvertToProvider(mockTool, "unsupported")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported provider")
	})
}

func TestToolResult(t *testing.T) {
	t.Run("ConvertResultToOpenAI", func(t *testing.T) {
		result := ToolResult{
			Success: true,
			Content: "Success message",
		}

		converted, err := ConvertToolResult(result, "openai")
		assert.NoError(t, err)
		assert.Equal(t, "Success message", converted["content"])
		assert.Equal(t, "tool", converted["role"])
		assert.Nil(t, converted["error"])
	})

	t.Run("ConvertErrorResultToOpenAI", func(t *testing.T) {
		result := ToolResult{
			Success: false,
			Error:   "Error message",
		}

		converted, err := ConvertToolResult(result, "openai")
		assert.NoError(t, err)
		assert.Equal(t, "Error message", converted["content"])
		assert.Equal(t, "tool", converted["role"])
		assert.Equal(t, true, converted["error"])
	})

	t.Run("ConvertResultToAnthropic", func(t *testing.T) {
		result := ToolResult{
			Success: true,
			Content: "Success message",
		}

		converted, err := ConvertToolResult(result, "anthropic")
		assert.NoError(t, err)
		assert.Equal(t, "tool_result", converted["type"])
		assert.Equal(t, "Success message", converted["content"])
		assert.Nil(t, converted["is_error"])
	})

	t.Run("ConvertErrorResultToAnthropic", func(t *testing.T) {
		result := ToolResult{
			Success: false,
			Error:   "Error message",
		}

		converted, err := ConvertToolResult(result, "anthropic")
		assert.NoError(t, err)
		assert.Equal(t, "tool_result", converted["type"])
		assert.Equal(t, "Error message", converted["content"])
		assert.Equal(t, true, converted["is_error"])
	})
}

// MockTool is a test implementation of the Tool interface
type MockTool struct {
	name        string
	description string
	schema      map[string]any
	result      ToolResult
	executeErr  error
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	if m.description == "" {
		return "Mock tool for testing"
	}
	return m.description
}

func (m *MockTool) JSONSchema() map[string]any {
	if m.schema == nil {
		return NewJSONSchema()
	}
	return m.schema
}

func (m *MockTool) Execute(ctx context.Context, params map[string]any) (ToolResult, error) {
	if m.executeErr != nil {
		return ToolResult{}, m.executeErr
	}

	if m.result.Metadata.ToolName == "" {
		m.result.Metadata.ToolName = m.name
		m.result.Metadata.Parameters = params
		m.result.Metadata.StartTime = time.Now()
		m.result.Metadata.EndTime = time.Now()
	}

	return m.result, nil
}
