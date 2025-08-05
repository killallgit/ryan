package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/killallgit/ryan/pkg/tools"
)

func TestFileOperationsAgent_Basic(t *testing.T) {
	registry := tools.NewRegistry()
	agent := NewFileOperationsAgent(registry)

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "file_operations", agent.Name())
	})

	t.Run("Description", func(t *testing.T) {
		desc := agent.Description()
		assert.Contains(t, desc, "file")
		assert.Contains(t, desc, "operations")
	})

	t.Run("CanHandle", func(t *testing.T) {
		tests := []struct {
			request  string
			expected bool
		}{
			{"read file test.go", true},
			{"write to file", true},
			{"create new file", true},
			{"list files in directory", true},
			{"analyze code structure", false},
			{"what is the weather", false},
		}

		for _, tt := range tests {
			t.Run(tt.request, func(t *testing.T) {
				canHandle, confidence := agent.CanHandle(tt.request)
				assert.Equal(t, tt.expected, canHandle)
				if tt.expected {
					assert.Greater(t, confidence, 0.0)
				}
			})
		}
	})
}

// Skipping Execute tests for now due to complex function signatures

func TestBatchProcessor(t *testing.T) {
	bp := NewBatchProcessor()
	assert.NotNil(t, bp)
}

func TestFileCache(t *testing.T) {
	cache := NewFileCache()

	t.Run("Set and Get", func(t *testing.T) {
		content := "test content"
		cache.Set("/test/file.go", content)

		retrieved, found := cache.Get("/test/file.go")
		assert.True(t, found)
		assert.Equal(t, content, retrieved)

		_, found = cache.Get("/nonexistent")
		assert.False(t, found)
	})

	t.Run("Invalidate", func(t *testing.T) {
		cache.Set("/test/file.go", "content")
		cache.Invalidate("/test/file.go")

		_, found := cache.Get("/test/file.go")
		assert.False(t, found)
	})
}

func TestDetermineOperation(t *testing.T) {
	agent := NewFileOperationsAgent(tools.NewRegistry())

	tests := []struct {
		request  string
		expected string
	}{
		{"read file test.go", "read"},
		{"write content to file.txt", "write"},
		{"create new file", "write"},
		{"list files in directory", "list"},
		{"show me the contents of file.go", "read"},
		{"list all files", "list"},
	}

	for _, tt := range tests {
		t.Run(tt.request, func(t *testing.T) {
			op := agent.determineOperation(tt.request)
			assert.Equal(t, tt.expected, op)
		})
	}
}

func TestExtractPath(t *testing.T) {
	agent := NewFileOperationsAgent(tools.NewRegistry())

	tests := []struct {
		request  string
		expected string
	}{
		{"read file /path/to/file.go", "/path/to/file.go"},
		{"read ./local/file.txt", "./local/file.txt"},
		{"show contents of main.go", "main.go"},
		{"list files in /usr/local", "/usr/local"},
		{"read package.json", "package.json"},
	}

	for _, tt := range tests {
		t.Run(tt.request, func(t *testing.T) {
			path := agent.extractPath(tt.request)
			assert.Equal(t, tt.expected, path)
		})
	}
}

// Skipping ExtractContent test due to complex function signature

func TestShouldReadFile(t *testing.T) {
	agent := NewFileOperationsAgent(tools.NewRegistry())

	tests := []struct {
		path     string
		expected bool
	}{
		{"test.go", true},
		{"main.py", true},
		{"config.json", true},
		{"data.txt", true},
		{"image.png", false},
		{"video.mp4", false},
		{"binary.exe", false},
		{"lib.so", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			should := agent.shouldReadFile(tt.path)
			assert.Equal(t, tt.expected, should)
		})
	}
}

// Skipping UpdateExecutionContext test due to complex function signature