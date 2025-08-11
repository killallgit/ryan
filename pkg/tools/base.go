package tools

import (
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools/acl"
)

// SecuredTool provides permission checking for tool operations
type SecuredTool struct {
	permissionManager *acl.PermissionManager
}

// NewSecuredTool creates a new secured tool with permission checking
func NewSecuredTool() *SecuredTool {
	return NewSecuredToolWithBypass(false)
}

// NewSecuredToolWithBypass creates a new secured tool with optional permission bypass
func NewSecuredToolWithBypass(bypass bool) *SecuredTool {
	return &SecuredTool{
		permissionManager: acl.NewPermissionManagerWithBypass(bypass),
	}
}

// ValidateAccess checks if the tool operation is permitted
func (t *SecuredTool) ValidateAccess(toolName, input string) error {
	logger.Debug("Validating access for tool: %s, input: %s", toolName, input)
	err := t.permissionManager.Validate(toolName, input)
	if err != nil {
		logger.Warn("Access denied for tool %s: %v", toolName, err)
	} else {
		logger.Debug("Access granted for tool: %s", toolName)
	}
	return err
}
