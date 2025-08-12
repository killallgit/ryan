package structured

import (
	"context"
	"fmt"

	"github.com/killallgit/ryan/pkg/tools"
	langchaintools "github.com/tmc/langchaingo/tools"
)

// MigrateFileRead creates a structured version of the FileRead tool
func MigrateFileRead(skipPermissions bool) langchaintools.Tool {
	originalTool := tools.NewFileReadToolWithBypass(skipPermissions)

	return NewBuilder("file_read", "Read contents of a file").
		WithParameter("path", "string", "Path to the file to read", true).
		WithExecutor(func(ctx context.Context, params map[string]interface{}) (string, error) {
			path, ok := params["path"].(string)
			if !ok {
				return "", fmt.Errorf("path must be a string")
			}
			return originalTool.Call(ctx, path)
		}).
		Build()
}

// MigrateFileWrite creates a structured version of the FileWrite tool
func MigrateFileWrite(skipPermissions bool) langchaintools.Tool {
	originalTool := tools.NewFileWriteToolWithBypass(skipPermissions)

	return NewBuilder("file_write", "Write content to a file").
		WithParameter("path", "string", "Path to the file to write", true).
		WithParameter("content", "string", "Content to write to the file", true).
		WithExecutor(func(ctx context.Context, params map[string]interface{}) (string, error) {
			path, ok := params["path"].(string)
			if !ok {
				return "", fmt.Errorf("path must be a string")
			}
			content, ok := params["content"].(string)
			if !ok {
				return "", fmt.Errorf("content must be a string")
			}
			// Original tool expects "path:::content" format
			input := fmt.Sprintf("%s:::%s", path, content)
			return originalTool.Call(ctx, input)
		}).
		Build()
}

// MigrateBash creates a structured version of the Bash tool
func MigrateBash(skipPermissions bool) langchaintools.Tool {
	originalTool := tools.NewBashToolWithBypass(skipPermissions)

	return NewBuilder("bash", "Execute bash shell commands").
		WithParameter("command", "string", "The bash command to execute", true).
		WithExecutor(func(ctx context.Context, params map[string]interface{}) (string, error) {
			command, ok := params["command"].(string)
			if !ok {
				return "", fmt.Errorf("command must be a string")
			}
			return originalTool.Call(ctx, command)
		}).
		Build()
}

// MigrateRipgrep creates a structured version of the Ripgrep tool
func MigrateRipgrep(skipPermissions bool) langchaintools.Tool {
	originalTool := tools.NewRipgrepToolWithBypass(skipPermissions)

	return NewBuilder("ripgrep", "Search for patterns in files using ripgrep").
		WithParameter("pattern", "string", "Pattern to search for", true).
		WithParameter("path", "string", "Path to search in", false).
		WithExecutor(func(ctx context.Context, params map[string]interface{}) (string, error) {
			pattern, ok := params["pattern"].(string)
			if !ok {
				return "", fmt.Errorf("pattern must be a string")
			}
			path, _ := params["path"].(string)
			if path == "" {
				path = "."
			}
			// Original tool expects "pattern:::path" format
			input := fmt.Sprintf("%s:::%s", pattern, path)
			return originalTool.Call(ctx, input)
		}).
		Build()
}

// MigrateWebFetch creates a structured version of the WebFetch tool
func MigrateWebFetch(skipPermissions bool) langchaintools.Tool {
	originalTool := tools.NewWebFetchToolWithBypass(skipPermissions)

	return NewBuilder("web_fetch", "Fetch content from a URL").
		WithParameter("url", "string", "URL to fetch", true).
		WithExecutor(func(ctx context.Context, params map[string]interface{}) (string, error) {
			url, ok := params["url"].(string)
			if !ok {
				return "", fmt.Errorf("url must be a string")
			}
			return originalTool.Call(ctx, url)
		}).
		Build()
}

// MigrateGit creates a structured version of the Git tool
func MigrateGit(skipPermissions bool) langchaintools.Tool {
	originalTool := tools.NewGitToolWithBypass(skipPermissions)

	return NewBuilder("git", "Execute git commands").
		WithParameter("command", "string", "Git command to execute", true).
		WithExecutor(func(ctx context.Context, params map[string]interface{}) (string, error) {
			command, ok := params["command"].(string)
			if !ok {
				return "", fmt.Errorf("command must be a string")
			}
			return originalTool.Call(ctx, command)
		}).
		Build()
}
