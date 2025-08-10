package acl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPermissionPatterns(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		toolName string
		input    string
		allowed  bool
	}{
		// File extension patterns
		{"Go file allowed", "FileRead(*.go)", "FileRead", "main.go", true},
		{"Go file in subdir allowed", "FileRead(*.go)", "FileRead", "pkg/tools/main.go", true},
		{"MD file blocked when only Go allowed", "FileRead(*.go)", "FileRead", "README.md", false},
		{"Wrong tool name", "FileRead(*.go)", "FileWrite", "main.go", false},

		// Git command patterns
		{"Git status allowed", "Git(status:*)", "Git", "status", true},
		{"Git status with args allowed", "Git(status:*)", "Git", "status --short", true},
		{"Git diff allowed", "Git(diff:*)", "Git", "diff HEAD", true},
		{"Git commit blocked", "Git(diff:*)", "Git", "commit -m test", false},

		// Directory patterns
		{"File in /tmp allowed", "FileWrite(/tmp/*)", "FileWrite", "/tmp/test.txt", true},
		{"File in /tmp subdir allowed", "FileWrite(/tmp/*)", "FileWrite", "/tmp/subdir/test.txt", true},
		{"File outside /tmp blocked", "FileWrite(/tmp/*)", "FileWrite", "/etc/passwd", false},
		{"File in relative path blocked", "FileWrite(/tmp/*)", "FileWrite", "tmp/test.txt", false},

		// Domain patterns
		{"GitHub domain allowed", "WebFetch(github.com/*)", "WebFetch", "github.com/user/repo", true},
		{"GitHub path allowed", "WebFetch(github.com/*)", "WebFetch", "github.com/tmc/langchaingo", true},
		{"Other domain blocked", "WebFetch(github.com/*)", "WebFetch", "evil.com/malware", false},

		// Wildcard patterns
		{"Ripgrep any pattern", "Ripgrep(*)", "Ripgrep", "func main", true},
		{"Ripgrep empty pattern", "Ripgrep(*)", "Ripgrep", "", true},
		{"Ripgrep complex pattern", "Ripgrep(*)", "Ripgrep", "^[a-zA-Z]+.*test$", true},

		// Exact match patterns
		{"Exact command match", "Git(status)", "Git", "status", true},
		{"Exact command no match", "Git(status)", "Git", "status --short", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &PermissionManager{
				allowedPatterns: []string{tt.pattern},
				bypassEnabled:   false,
			}
			result := pm.IsAllowed(tt.toolName, tt.input)
			assert.Equal(t, tt.allowed, result,
				"Pattern %s with input %s(%s) should be %v",
				tt.pattern, tt.toolName, tt.input, tt.allowed)
		})
	}
}

func TestPermissionManagerBypass(t *testing.T) {
	pm := &PermissionManager{
		allowedPatterns: []string{}, // No patterns allowed
		bypassEnabled:   true,       // But bypass is enabled
	}

	// Everything should be allowed when bypass is enabled
	assert.True(t, pm.IsAllowed("FileRead", "/etc/passwd"))
	assert.True(t, pm.IsAllowed("Git", "push --force"))
	assert.True(t, pm.IsAllowed("FileWrite", "/system/critical.conf"))
	assert.NoError(t, pm.Validate("AnyTool", "any input"))
}

func TestPermissionManagerValidate(t *testing.T) {
	pm := &PermissionManager{
		allowedPatterns: []string{
			"FileRead(*.go)",
			"Git(status:*)",
		},
		bypassEnabled: false,
	}

	// Allowed operations
	assert.NoError(t, pm.Validate("FileRead", "main.go"))
	assert.NoError(t, pm.Validate("Git", "status"))

	// Blocked operations
	err := pm.Validate("FileRead", "secret.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	err = pm.Validate("Git", "push origin main")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed by ACL")
}

func TestMultiplePatterns(t *testing.T) {
	pm := &PermissionManager{
		allowedPatterns: []string{
			"FileRead(*.go)",
			"FileRead(*.md)",
			"FileRead(*.txt)",
			"Git(status:*)",
			"Git(diff:*)",
			"Git(log:*)",
		},
		bypassEnabled: false,
	}

	// Test various allowed patterns
	assert.True(t, pm.IsAllowed("FileRead", "main.go"))
	assert.True(t, pm.IsAllowed("FileRead", "README.md"))
	assert.True(t, pm.IsAllowed("FileRead", "notes.txt"))
	assert.True(t, pm.IsAllowed("Git", "status --porcelain"))
	assert.True(t, pm.IsAllowed("Git", "diff HEAD~1"))
	assert.True(t, pm.IsAllowed("Git", "log --oneline"))

	// Test blocked patterns
	assert.False(t, pm.IsAllowed("FileRead", "config.yaml"))
	assert.False(t, pm.IsAllowed("FileWrite", "main.go"))
	assert.False(t, pm.IsAllowed("Git", "commit -m 'test'"))
}

func TestDefaultPermissions(t *testing.T) {
	defaults := getDefaultPermissions()

	// Check that defaults include safe operations
	assert.Contains(t, defaults, "FileRead(*.go)")
	assert.Contains(t, defaults, "FileRead(*.md)")
	assert.Contains(t, defaults, "Git(status:*)")
	assert.Contains(t, defaults, "Git(diff:*)")
	assert.Contains(t, defaults, "Ripgrep(*)")

	// Ensure defaults are reasonable (at least 5 patterns)
	assert.GreaterOrEqual(t, len(defaults), 5)
}
