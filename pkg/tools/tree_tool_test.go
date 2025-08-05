package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeTool_Basic(t *testing.T) {
	tool := NewTreeTool()

	// Test basic properties
	assert.Equal(t, "tree", tool.Name())
	assert.NotEmpty(t, tool.Description())

	// Test JSON schema
	schema := tool.JSONSchema()
	assert.NotNil(t, schema)

	properties, exists := schema["properties"].(map[string]any)
	require.True(t, exists)

	// Check path property
	path, exists := properties["path"]
	require.True(t, exists)
	assert.NotNil(t, path)
}

func TestTreeTool_DefaultDirectory(t *testing.T) {
	tool := NewTreeTool()
	ctx := context.Background()

	// Test with default directory (current)
	result, err := tool.Execute(ctx, map[string]any{
		"max_depth": 2,
		"format":    "summary",
		"max_files": 100,
	})

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Content)

	// Check metadata
	assert.NotNil(t, result.Metadata)
	assert.Equal(t, "tree", result.Metadata.ToolName)
	assert.True(t, result.Metadata.ExecutionTime > 0)

	// Check data structure
	data := result.Data
	assert.NotNil(t, data["tree"])
	assert.NotNil(t, data["summary"])
	assert.Equal(t, "summary", data["format"])
}

func TestTreeTool_ValidationErrors(t *testing.T) {
	tool := NewTreeTool()
	ctx := context.Background()

	// Test non-existent path
	result, err := tool.Execute(ctx, map[string]any{
		"path": "/non/existent/path",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "path does not exist")

	// Test invalid max_depth
	result, err = tool.Execute(ctx, map[string]any{
		"max_depth": 100,
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "max_depth must be between 1 and 50")

	// Test invalid max_files
	result, err = tool.Execute(ctx, map[string]any{
		"max_files": 100000,
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "max_files must be between 1 and 50000")
}

func TestTreeTool_FormatOptions(t *testing.T) {
	tool := NewTreeTool()
	ctx := context.Background()

	formats := []string{"tree", "list", "json", "summary"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			result, err := tool.Execute(ctx, map[string]any{
				"max_depth": 2,
				"format":    format,
				"max_files": 50,
			})

			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.NotEmpty(t, result.Content)

			data := result.Data
			assert.Equal(t, format, data["format"])
		})
	}
}

func TestTreeTool_FileTypeFilter(t *testing.T) {
	tool := NewTreeTool()
	ctx := context.Background()

	// Filter for Go files only
	result, err := tool.Execute(ctx, map[string]any{
		"max_depth":  3,
		"file_types": []string{"go"},
		"format":     "list",
		"max_files":  100,
	})

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Content)

	// Should contain .go files information in the output
	if result.Success {
		data := result.Data
		summary := data["summary"]
		assert.NotNil(t, summary)
	}
}

func TestTreeTool_ExcludePatterns(t *testing.T) {
	tool := NewTreeTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"max_depth":        2,
		"exclude_patterns": []string{"test", ".git", "node_modules"},
		"format":           "tree",
		"max_files":        100,
	})

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Content)
}

func TestTreeTool_SizeFilters(t *testing.T) {
	tool := NewTreeTool()
	ctx := context.Background()

	// Filter files by size (larger than 1KB)
	result, err := tool.Execute(ctx, map[string]any{
		"max_depth": 3,
		"min_size":  1024,
		"format":    "list",
		"max_files": 100,
	})

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Content)
}

func TestTreeTool_SortOptions(t *testing.T) {
	tool := NewTreeTool()
	ctx := context.Background()

	sortOptions := []string{"name", "size", "date", "type"}

	for _, sortBy := range sortOptions {
		t.Run(sortBy, func(t *testing.T) {
			result, err := tool.Execute(ctx, map[string]any{
				"max_depth": 2,
				"sort_by":   sortBy,
				"format":    "tree",
				"max_files": 50,
			})

			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.NotEmpty(t, result.Content)
		})
	}
}

func TestTreeTool_HiddenFiles(t *testing.T) {
	tool := NewTreeTool()
	ctx := context.Background()

	// Test without hidden files
	result1, err := tool.Execute(ctx, map[string]any{
		"max_depth":      2,
		"include_hidden": false,
		"format":         "summary",
		"max_files":      100,
	})
	require.NoError(t, err)
	assert.True(t, result1.Success)

	// Test with hidden files
	result2, err := tool.Execute(ctx, map[string]any{
		"max_depth":      2,
		"include_hidden": true,
		"format":         "summary",
		"max_files":      100,
	})
	require.NoError(t, err)
	assert.True(t, result2.Success)

	// The version with hidden files should typically find more files
	// (though this depends on the directory structure)
}

func TestTreeTool_WithTestData(t *testing.T) {
	// Create a temporary directory with test structure
	tmpDir, err := os.MkdirTemp("", "tree_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test structure
	testStructure := map[string]string{
		"file1.txt":            "content1",
		"file2.go":             "package main",
		"dir1/file3.py":        "print('hello')",
		"dir1/file4.js":        "console.log('test')",
		"dir2/subdir/file5.md": "# Test",
		".hidden":              "hidden content",
	}

	for path, content := range testStructure {
		fullPath := filepath.Join(tmpDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	tool := NewTreeTool()
	ctx := context.Background()

	// Test tree analysis on our test data
	result, err := tool.Execute(ctx, map[string]any{
		"path":       tmpDir,
		"max_depth":  5,
		"format":     "tree",
		"show_stats": true,
		"max_files":  100,
	})

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Content)

	// Check that we found the expected structure
	data := result.Data
	summary := data["summary"]
	assert.NotNil(t, summary)

	// Should contain statistics about our test files
	content := result.Content
	assert.Contains(t, content, tmpDir)
}

func TestTreeTool_ContextCancellation(t *testing.T) {
	tool := NewTreeTool()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := tool.Execute(ctx, map[string]any{
		"max_depth": 10,
		"max_files": 1000,
	})

	// Should handle cancellation gracefully
	require.NoError(t, err)
	// Result might succeed or fail depending on timing, but shouldn't panic
}

func TestTreeTool_ParameterExtraction(t *testing.T) {
	tool := NewTreeTool()

	params := map[string]any{
		"string_param": "test_value",
		"bool_param":   true,
		"int_param":    42,
		"float_param":  3.14,
		"array_param":  []any{"item1", "item2", 123},
	}

	// Test parameter extraction methods
	assert.Equal(t, "test_value", tool.getStringParam(params, "string_param", "default"))
	assert.Equal(t, "default", tool.getStringParam(params, "missing_param", "default"))

	assert.Equal(t, true, tool.getBoolParam(params, "bool_param", false))
	assert.Equal(t, false, tool.getBoolParam(params, "missing_param", false))

	assert.Equal(t, 42, tool.getIntParam(params, "int_param", 0))
	assert.Equal(t, 3, tool.getIntParam(params, "float_param", 0))
	assert.Equal(t, 0, tool.getIntParam(params, "missing_param", 0))

	result := tool.getStringArrayParam(params, "array_param")
	expected := []string{"item1", "item2"}
	assert.Equal(t, expected, result)
}

func TestTreeTool_SizeFormatting(t *testing.T) {
	tool := NewTreeTool()

	testCases := []struct {
		bytes    int64
		expected string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tc := range testCases {
		result := tool.formatSize(tc.bytes)
		assert.Equal(t, tc.expected, result, "Size formatting for %d bytes", tc.bytes)
	}
}

func TestTreeTool_ExcludePatternMatching(t *testing.T) {
	tool := NewTreeTool()

	patterns := []string{"test", "*.log", "node_modules"}

	testCases := []struct {
		name     string
		fullPath string
		expected bool
	}{
		{"test_file.go", "/path/to/test_file.go", true}, // matches "test"
		{"app.log", "/path/to/app.log", true},           // matches "*.log"
		{"node_modules", "/path/to/node_modules", true}, // matches "node_modules"
		{"normal.go", "/path/to/normal.go", false},      // doesn't match any pattern
	}

	for _, tc := range testCases {
		result := tool.matchesExcludePattern(tc.name, tc.fullPath, patterns)
		assert.Equal(t, tc.expected, result, "Pattern matching for %s", tc.name)
	}
}
