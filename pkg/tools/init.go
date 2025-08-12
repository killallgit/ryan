package tools

import (
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools/registry"
	"github.com/tmc/langchaingo/tools"
)

// init registers all tools with the global registry during package initialization
func init() {
	// Register bash tool
	registry.Global().Register("bash", func(skipPermissions bool) tools.Tool {
		return NewBashToolWithBypass(skipPermissions)
	})

	// Register file read tool
	registry.Global().Register("file_read", func(skipPermissions bool) tools.Tool {
		return NewFileReadToolWithBypass(skipPermissions)
	})

	// Register file write tool
	registry.Global().Register("file_write", func(skipPermissions bool) tools.Tool {
		return NewFileWriteToolWithBypass(skipPermissions)
	})

	// Register git tool
	registry.Global().Register("git", func(skipPermissions bool) tools.Tool {
		return NewGitToolWithBypass(skipPermissions)
	})

	// Register ripgrep tool
	registry.Global().Register("ripgrep", func(skipPermissions bool) tools.Tool {
		return NewRipgrepToolWithBypass(skipPermissions)
	})

	// Register webfetch tool
	registry.Global().Register("webfetch", func(skipPermissions bool) tools.Tool {
		return NewWebFetchToolWithBypass(skipPermissions)
	})

	logger.Debug("Registered tools with global registry")
}
