package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/killallgit/ryan/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolsWithPermissions(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test settings file with permissions
	settingsDir := filepath.Join(tempDir, ".ryan")
	err := os.MkdirAll(settingsDir, 0755)
	require.NoError(t, err)

	settings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []string{
				"FileRead(*.go)",
				"FileRead(*.md)",
				"FileWrite(/tmp/*)",
				"Git(status:*)",
				"Git(diff:*)",
				"Ripgrep(*)",
			},
		},
	}

	settingsData, err := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, err)

	settingsPath := filepath.Join(settingsDir, "settings.json")
	err = os.WriteFile(settingsPath, settingsData, 0644)
	require.NoError(t, err)

	// Set HOME to temp directory so the ACL manager finds our test settings
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Test FileReadTool
	t.Run("FileReadTool", func(t *testing.T) {
		// Create a test file
		testFile := filepath.Join(tempDir, "test.go")
		testContent := "package main\n\nfunc main() {}\n"
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err)

		tool := tools.NewFileReadTool()
		ctx := context.Background()

		// Test allowed file
		result, err := tool.Call(ctx, testFile)
		assert.NoError(t, err)
		assert.Equal(t, testContent, result)

		// Test blocked file (not .go or .md)
		blockedFile := filepath.Join(tempDir, "secret.txt")
		err = os.WriteFile(blockedFile, []byte("secret"), 0644)
		require.NoError(t, err)

		_, err = tool.Call(ctx, blockedFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
	})

	// Test FileWriteTool
	t.Run("FileWriteTool", func(t *testing.T) {
		tool := tools.NewFileWriteTool()
		ctx := context.Background()

		// Test allowed path (/tmp/*)
		allowedPath := "/tmp/test_write.txt"
		input := allowedPath + ":::test content"
		result, err := tool.Call(ctx, input)
		assert.NoError(t, err)
		assert.Contains(t, result, "Successfully wrote")

		// Clean up
		os.Remove(allowedPath)

		// Test blocked path (outside /tmp)
		blockedPath := "/etc/passwd"
		input = blockedPath + ":::blocked content"
		_, err = tool.Call(ctx, input)
		assert.Error(t, err, "Should error on blocked path")
		if err != nil {
			assert.Contains(t, err.Error(), "permission denied")
		}
	})

	// Test GitTool
	t.Run("GitTool", func(t *testing.T) {
		tool := tools.NewGitTool()
		ctx := context.Background()

		// Test allowed command (status)
		_, err := tool.Call(ctx, "status")
		// May error if not in a git repo, but shouldn't be permission denied
		if err != nil {
			assert.NotContains(t, err.Error(), "permission denied")
		}

		// Test blocked command (commit)
		_, err = tool.Call(ctx, "commit -m 'test'")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
	})

	// Test RipgrepTool
	t.Run("RipgrepTool", func(t *testing.T) {
		tool := tools.NewRipgrepTool()
		ctx := context.Background()

		// Create a file to search
		searchFile := filepath.Join(tempDir, "search.go")
		err := os.WriteFile(searchFile, []byte("func TestFunction() {}"), 0644)
		require.NoError(t, err)

		// Change to temp directory for search
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		// Test search (always allowed with wildcard)
		result, err := tool.Call(ctx, "TestFunction")
		// May error if ripgrep not installed, but shouldn't be permission denied
		if err != nil {
			assert.NotContains(t, err.Error(), "permission denied")
		} else {
			// If ripgrep works, check result
			assert.NotEmpty(t, result)
		}
	})
}

func TestToolsWithSkipPermissions(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("All operations allowed with skip flag", func(t *testing.T) {
		// FileRead - any file should work with bypass enabled
		tool := tools.NewFileReadToolWithBypass(true)
		ctx := context.Background()

		testFile := filepath.Join(tempDir, "any.xyz")
		err := os.WriteFile(testFile, []byte("content"), 0644)
		require.NoError(t, err)

		result, err := tool.Call(ctx, testFile)
		assert.NoError(t, err)
		assert.Equal(t, "content", result)

		// FileWrite - any path should work with bypass enabled
		writeTool := tools.NewFileWriteToolWithBypass(true)
		writePath := filepath.Join(tempDir, "anywhere.txt")
		input := writePath + ":::content"

		result, err = writeTool.Call(ctx, input)
		assert.NoError(t, err)
		assert.Contains(t, result, "Successfully wrote")

		// Git - any command should work with bypass enabled (may fail for other reasons)
		gitTool := tools.NewGitToolWithBypass(true)
		_, err = gitTool.Call(ctx, "push --force")
		// Should not be permission denied
		if err != nil {
			assert.NotContains(t, err.Error(), "permission denied")
		}
	})
}
