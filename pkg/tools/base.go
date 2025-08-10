package tools

import (
	"github.com/killallgit/ryan/pkg/tools/acl"
)

// SecuredTool provides permission checking for tool operations
type SecuredTool struct {
	permissionManager *acl.PermissionManager
}

// NewSecuredTool creates a new secured tool with permission checking
func NewSecuredTool() *SecuredTool {
	return &SecuredTool{
		permissionManager: acl.NewPermissionManager(),
	}
}

// ValidateAccess checks if the tool operation is permitted
func (t *SecuredTool) ValidateAccess(toolName, input string) error {
	return t.permissionManager.Validate(toolName, input)
}
