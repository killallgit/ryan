package acl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PermissionManager handles tool access control based on configured patterns
type PermissionManager struct {
	allowedPatterns []string
	bypassEnabled   bool
}

// NewPermissionManager creates a new permission manager from settings
func NewPermissionManager() *PermissionManager {
	return NewPermissionManagerWithBypass(false)
}

// NewPermissionManagerWithBypass creates a new permission manager with optional bypass
func NewPermissionManagerWithBypass(bypass bool) *PermissionManager {
	return &PermissionManager{
		allowedPatterns: loadPermissions(),
		bypassEnabled:   bypass,
	}
}

func loadPermissions() []string {
	// Load from ~/.ryan/settings.json
	home, err := os.UserHomeDir()
	if err != nil {
		return getDefaultPermissions()
	}

	settingsPath := filepath.Join(home, ".ryan", "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		// Default permissions if no settings file
		return getDefaultPermissions()
	}

	var settings struct {
		Permissions struct {
			Allow []string `json:"allow"`
		} `json:"permissions"`
	}

	if err := json.Unmarshal(data, &settings); err != nil {
		return getDefaultPermissions()
	}

	if len(settings.Permissions.Allow) == 0 {
		return getDefaultPermissions()
	}

	return settings.Permissions.Allow
}

func getDefaultPermissions() []string {
	return []string{
		"FileRead(*.go)",
		"FileRead(*.md)",
		"FileRead(*.txt)",
		"FileRead(*.json)",
		"FileRead(*.yaml)",
		"FileRead(*.yml)",
		"Git(status:*)",
		"Git(diff:*)",
		"Git(log:*)",
		"Git(branch:*)",
		"Git(show:*)",
		"Ripgrep(*)",
	}
}

// Validate checks if a tool operation is permitted
func (pm *PermissionManager) Validate(toolName string, input string) error {
	// Bypass all checks if flag is set
	if pm.bypassEnabled {
		return nil
	}

	// Check against allowed patterns
	if pm.IsAllowed(toolName, input) {
		return nil
	}

	return fmt.Errorf("permission denied: %s(%s) not allowed by ACL", toolName, input)
}

// IsAllowed checks if the tool operation matches any allowed pattern
func (pm *PermissionManager) IsAllowed(toolName string, input string) bool {
	// Bypass all checks if flag is set
	if pm.bypassEnabled {
		return true
	}

	// Check each allowed pattern
	for _, pattern := range pm.allowedPatterns {
		if matchesPattern(toolName, input, pattern) {
			return true
		}
	}

	return false
}

func matchesPattern(toolName, input, pattern string) bool {
	// Parse pattern: Tool(matcher)
	if !strings.HasPrefix(pattern, toolName+"(") {
		return false
	}

	// Extract matcher from pattern
	start := strings.Index(pattern, "(")
	end := strings.LastIndex(pattern, ")")
	if start == -1 || end == -1 {
		return false
	}

	matcher := pattern[start+1 : end]

	// Handle different matcher types
	if strings.Contains(matcher, ":*") {
		// Prefix match (e.g., "status:*" matches "status --short")
		prefix := strings.TrimSuffix(matcher, ":*")
		// Add the colon back for matching
		return input == prefix || strings.HasPrefix(input, prefix+" ") || strings.HasPrefix(input, prefix+":")
	} else if strings.HasPrefix(matcher, "*.") {
		// Extension match (e.g., "*.go" matches any .go file)
		ext := matcher[1:] // Remove *
		return strings.HasSuffix(input, ext)
	} else if strings.HasSuffix(matcher, "/*") {
		// Directory match (e.g., "/tmp/*" matches any file in /tmp)
		dir := strings.TrimSuffix(matcher, "/*")
		return strings.HasPrefix(input, dir+"/")
	} else if strings.Contains(matcher, "*") && strings.Index(matcher, "*") == len(matcher)-1 {
		// Suffix wildcard (e.g., "github.com/*" matches "github.com/user/repo")
		prefix := strings.TrimSuffix(matcher, "*")
		return strings.HasPrefix(input, prefix)
	} else if matcher == "*" {
		// Wildcard - matches anything
		return true
	} else {
		// Exact match
		return input == matcher
	}
}
