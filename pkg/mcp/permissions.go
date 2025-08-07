package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
)

// PermissionManager implements Claude CLI's four-tier permission model:
// System → Tool → File → Context rules
type PermissionManager struct {
	// Rule storage
	systemRules  []PermissionRule
	toolRules    map[string][]PermissionRule
	fileRules    []FilePermissionRule
	contextRules map[string][]PermissionRule

	// Configuration
	defaultAction string // "allow", "deny", "ask"

	// Synchronization
	mu sync.RWMutex

	// Dependencies
	cfg    *config.Config
	logger *logger.Logger
}

// PermissionRule represents a permission rule
type PermissionRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// Rule matching
	ToolPattern    string          `json:"toolPattern,omitempty"`   // Regex pattern for tool names
	ActionPattern  string          `json:"actionPattern,omitempty"` // Regex pattern for actions
	ParameterRules []ParameterRule `json:"parameterRules,omitempty"`

	// Decision
	Action     string   `json:"action"` // "allow", "deny", "ask", "passthrough"
	Conditions []string `json:"conditions,omitempty"`

	// Scope and timing
	Scope     string     `json:"scope"` // "global", "project", "session"
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`

	// Metadata
	Priority int                    `json:"priority"` // Higher priority rules are evaluated first
	Tags     []string               `json:"tags,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ParameterRule represents a rule for tool parameters
type ParameterRule struct {
	Parameter string   `json:"parameter"`
	Pattern   string   `json:"pattern,omitempty"`  // Regex pattern for parameter value
	Required  *bool    `json:"required,omitempty"` // Whether parameter is required
	MinValue  *float64 `json:"minValue,omitempty"` // Minimum numeric value
	MaxValue  *float64 `json:"maxValue,omitempty"` // Maximum numeric value
}

// FilePermissionRule represents a file-specific permission rule
type FilePermissionRule struct {
	PermissionRule

	// File-specific fields
	PathPattern       string   `json:"pathPattern"` // Glob pattern for file paths
	Operation         string   `json:"operation"`   // "read", "write", "execute", "delete"
	AllowedPaths      []string `json:"allowedPaths,omitempty"`
	DeniedPaths       []string `json:"deniedPaths,omitempty"`
	MaxFileSize       int64    `json:"maxFileSize,omitempty"`
	AllowedExtensions []string `json:"allowedExtensions,omitempty"`
}

// PermissionEvaluationContext provides context for permission evaluation
type PermissionEvaluationContext struct {
	// Request context
	ToolName    string                 `json:"toolName"`
	Parameters  map[string]interface{} `json:"parameters"`
	UserID      string                 `json:"userId,omitempty"`
	SessionID   string                 `json:"sessionId,omitempty"`
	ProjectRoot string                 `json:"projectRoot,omitempty"`

	// File context (for file operations)
	FilePath      string `json:"filePath,omitempty"`
	FileOperation string `json:"fileOperation,omitempty"`
	FileSize      int64  `json:"fileSize,omitempty"`

	// Working directory context
	WorkingDirectory string `json:"workingDirectory,omitempty"`

	// Additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewPermissionManager creates a new permission manager
func NewPermissionManager(defaultAction string, cfg *config.Config) *PermissionManager {
	if defaultAction == "" {
		defaultAction = "ask" // Default to asking user
	}

	return &PermissionManager{
		toolRules:     make(map[string][]PermissionRule),
		contextRules:  make(map[string][]PermissionRule),
		defaultAction: defaultAction,
		cfg:           cfg,
		logger:        logger.WithComponent("mcp-permissions"),
	}
}

// CanExecuteTool evaluates if a tool can be executed
func (pm *PermissionManager) CanExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (PermissionResult, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	evalCtx := PermissionEvaluationContext{
		ToolName:   toolName,
		Parameters: params,
	}

	// Extract context information if available
	// TODO: Reimplement using Viper configuration
	// if pm.cfg != nil {
	//     evalCtx.ProjectRoot = viper.GetString("project.root")
	// }

	// Add file context if this is a file operation
	pm.extractFileContext(&evalCtx)

	return pm.evaluatePermission(evalCtx)
}

// CanAccessResource evaluates if a resource can be accessed
func (pm *PermissionManager) CanAccessResource(ctx context.Context, resourceType, resourcePath string) (PermissionResult, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	evalCtx := PermissionEvaluationContext{
		ToolName:      fmt.Sprintf("%s_access", resourceType),
		FilePath:      resourcePath,
		FileOperation: "access",
	}

	return pm.evaluatePermission(evalCtx)
}

// evaluatePermission evaluates a permission request using the four-tier model
func (pm *PermissionManager) evaluatePermission(ctx PermissionEvaluationContext) (PermissionResult, error) {
	// Tier 1: System Rules (highest priority)
	if result, matched := pm.evaluateSystemRules(ctx); matched {
		pm.logger.Debug("Permission decision from system rules",
			"tool", ctx.ToolName, "action", result.Action, "reason", result.Reason)
		return result, nil
	}

	// Tier 2: Tool-specific Rules
	if result, matched := pm.evaluateToolRules(ctx); matched {
		pm.logger.Debug("Permission decision from tool rules",
			"tool", ctx.ToolName, "action", result.Action, "reason", result.Reason)
		return result, nil
	}

	// Tier 3: File Rules (if applicable)
	if ctx.FilePath != "" {
		if result, matched := pm.evaluateFileRules(ctx); matched {
			pm.logger.Debug("Permission decision from file rules",
				"tool", ctx.ToolName, "file", ctx.FilePath, "action", result.Action, "reason", result.Reason)
			return result, nil
		}
	}

	// Tier 4: Context Rules
	if result, matched := pm.evaluateContextRules(ctx); matched {
		pm.logger.Debug("Permission decision from context rules",
			"tool", ctx.ToolName, "action", result.Action, "reason", result.Reason)
		return result, nil
	}

	// Default action
	result := PermissionResult{
		Allowed: pm.defaultAction == "allow",
		Action:  pm.defaultAction,
		Reason:  fmt.Sprintf("Default action for tool %s", ctx.ToolName),
	}

	pm.logger.Debug("Permission decision from default action",
		"tool", ctx.ToolName, "action", result.Action)

	return result, nil
}

// evaluateSystemRules evaluates system-level permission rules
func (pm *PermissionManager) evaluateSystemRules(ctx PermissionEvaluationContext) (PermissionResult, bool) {
	for _, rule := range pm.systemRules {
		if pm.ruleMatches(rule, ctx) {
			return pm.createResultFromRule(rule, "system rule"), true
		}
	}
	return PermissionResult{}, false
}

// evaluateToolRules evaluates tool-specific permission rules
func (pm *PermissionManager) evaluateToolRules(ctx PermissionEvaluationContext) (PermissionResult, bool) {
	// Check rules specific to this tool
	if rules, exists := pm.toolRules[ctx.ToolName]; exists {
		for _, rule := range rules {
			if pm.ruleMatches(rule, ctx) {
				return pm.createResultFromRule(rule, "tool rule"), true
			}
		}
	}

	// Check wildcard tool rules
	for toolPattern, rules := range pm.toolRules {
		if matched, _ := regexp.MatchString(toolPattern, ctx.ToolName); matched {
			for _, rule := range rules {
				if pm.ruleMatches(rule, ctx) {
					return pm.createResultFromRule(rule, "tool pattern rule"), true
				}
			}
		}
	}

	return PermissionResult{}, false
}

// evaluateFileRules evaluates file-specific permission rules
func (pm *PermissionManager) evaluateFileRules(ctx PermissionEvaluationContext) (PermissionResult, bool) {
	for _, rule := range pm.fileRules {
		if pm.fileRuleMatches(rule, ctx) {
			return pm.createResultFromRule(rule.PermissionRule, "file rule"), true
		}
	}
	return PermissionResult{}, false
}

// evaluateContextRules evaluates context-specific permission rules
func (pm *PermissionManager) evaluateContextRules(ctx PermissionEvaluationContext) (PermissionResult, bool) {
	// Evaluate project context rules
	if ctx.ProjectRoot != "" {
		if rules, exists := pm.contextRules[ctx.ProjectRoot]; exists {
			for _, rule := range rules {
				if pm.ruleMatches(rule, ctx) {
					return pm.createResultFromRule(rule, "project context rule"), true
				}
			}
		}
	}

	// Evaluate session context rules
	if ctx.SessionID != "" {
		if rules, exists := pm.contextRules[ctx.SessionID]; exists {
			for _, rule := range rules {
				if pm.ruleMatches(rule, ctx) {
					return pm.createResultFromRule(rule, "session context rule"), true
				}
			}
		}
	}

	return PermissionResult{}, false
}

// ruleMatches checks if a permission rule matches the context
func (pm *PermissionManager) ruleMatches(rule PermissionRule, ctx PermissionEvaluationContext) bool {
	// Check if rule has expired
	if rule.ExpiresAt != nil && time.Now().After(*rule.ExpiresAt) {
		return false
	}

	// Check tool pattern
	if rule.ToolPattern != "" {
		if matched, _ := regexp.MatchString(rule.ToolPattern, ctx.ToolName); !matched {
			return false
		}
	}

	// Check parameter rules
	for _, paramRule := range rule.ParameterRules {
		if !pm.parameterRuleMatches(paramRule, ctx.Parameters) {
			return false
		}
	}

	return true
}

// fileRuleMatches checks if a file permission rule matches the context
func (pm *PermissionManager) fileRuleMatches(rule FilePermissionRule, ctx PermissionEvaluationContext) bool {
	// First check the base rule
	if !pm.ruleMatches(rule.PermissionRule, ctx) {
		return false
	}

	// Check file-specific conditions
	if rule.Operation != "" && rule.Operation != ctx.FileOperation {
		return false
	}

	// Check path pattern
	if rule.PathPattern != "" {
		if matched, _ := filepath.Match(rule.PathPattern, ctx.FilePath); !matched {
			return false
		}
	}

	// Check allowed paths
	if len(rule.AllowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range rule.AllowedPaths {
			if strings.HasPrefix(ctx.FilePath, allowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	// Check denied paths
	for _, deniedPath := range rule.DeniedPaths {
		if strings.HasPrefix(ctx.FilePath, deniedPath) {
			return false
		}
	}

	// Check file size
	if rule.MaxFileSize > 0 && ctx.FileSize > rule.MaxFileSize {
		return false
	}

	// Check file extension
	if len(rule.AllowedExtensions) > 0 {
		ext := strings.ToLower(filepath.Ext(ctx.FilePath))
		allowed := false
		for _, allowedExt := range rule.AllowedExtensions {
			if ext == strings.ToLower(allowedExt) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	return true
}

// parameterRuleMatches checks if a parameter rule matches
func (pm *PermissionManager) parameterRuleMatches(rule ParameterRule, params map[string]interface{}) bool {
	value, exists := params[rule.Parameter]

	// Check if required parameter is missing
	if rule.Required != nil && *rule.Required && !exists {
		return false
	}

	if !exists {
		return true // Parameter not present and not required
	}

	// Check pattern matching for string values
	if rule.Pattern != "" {
		if str, ok := value.(string); ok {
			if matched, _ := regexp.MatchString(rule.Pattern, str); !matched {
				return false
			}
		}
	}

	// Check numeric bounds
	if rule.MinValue != nil || rule.MaxValue != nil {
		var numValue float64
		var ok bool

		switch v := value.(type) {
		case float64:
			numValue, ok = v, true
		case int:
			numValue, ok = float64(v), true
		case int64:
			numValue, ok = float64(v), true
		}

		if ok {
			if rule.MinValue != nil && numValue < *rule.MinValue {
				return false
			}
			if rule.MaxValue != nil && numValue > *rule.MaxValue {
				return false
			}
		}
	}

	return true
}

// createResultFromRule creates a permission result from a rule
func (pm *PermissionManager) createResultFromRule(rule PermissionRule, ruleType string) PermissionResult {
	result := PermissionResult{
		Action:     rule.Action,
		Conditions: rule.Conditions,
		Reason:     fmt.Sprintf("Matched %s: %s", ruleType, rule.Name),
		Metadata: map[string]interface{}{
			"ruleId":   rule.ID,
			"ruleName": rule.Name,
			"ruleType": ruleType,
		},
	}

	result.Allowed = (rule.Action == "allow")

	return result
}

// extractFileContext extracts file operation context from parameters
func (pm *PermissionManager) extractFileContext(ctx *PermissionEvaluationContext) {
	// Common file parameter names
	fileParams := []string{"path", "file", "filename", "filepath", "file_path"}

	for _, param := range fileParams {
		if value, exists := ctx.Parameters[param]; exists {
			if filePath, ok := value.(string); ok {
				ctx.FilePath = filePath
				break
			}
		}
	}

	// Determine operation from tool name
	toolName := strings.ToLower(ctx.ToolName)
	if strings.Contains(toolName, "read") || strings.Contains(toolName, "get") {
		ctx.FileOperation = "read"
	} else if strings.Contains(toolName, "write") || strings.Contains(toolName, "create") || strings.Contains(toolName, "save") {
		ctx.FileOperation = "write"
	} else if strings.Contains(toolName, "delete") || strings.Contains(toolName, "remove") {
		ctx.FileOperation = "delete"
	} else if strings.Contains(toolName, "execute") || strings.Contains(toolName, "run") {
		ctx.FileOperation = "execute"
	}
}

// Permission management methods

// GrantToolPermission grants permission for a tool
func (pm *PermissionManager) GrantToolPermission(toolName string, scope string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	rule := PermissionRule{
		ID:          fmt.Sprintf("grant_%s_%d", toolName, time.Now().Unix()),
		Name:        fmt.Sprintf("Grant %s", toolName),
		Description: fmt.Sprintf("Auto-granted permission for %s", toolName),
		ToolPattern: fmt.Sprintf("^%s$", regexp.QuoteMeta(toolName)),
		Action:      "allow",
		Scope:       scope,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Priority:    100,
	}

	// Add to appropriate rule set based on scope
	switch scope {
	case "global":
		pm.systemRules = append(pm.systemRules, rule)
	case "project":
		// TODO: Reimplement project-scoped rules using Viper
		// For now, treat as tool-specific rule
		pm.toolRules[toolName] = append(pm.toolRules[toolName], rule)
	default:
		pm.toolRules[toolName] = append(pm.toolRules[toolName], rule)
	}

	pm.logger.Info("Granted tool permission", "tool", toolName, "scope", scope)
	return nil
}

// RevokeToolPermission revokes permission for a tool
func (pm *PermissionManager) RevokeToolPermission(toolName string, scope string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Add a deny rule with higher priority
	rule := PermissionRule{
		ID:          fmt.Sprintf("revoke_%s_%d", toolName, time.Now().Unix()),
		Name:        fmt.Sprintf("Revoke %s", toolName),
		Description: fmt.Sprintf("Revoked permission for %s", toolName),
		ToolPattern: fmt.Sprintf("^%s$", regexp.QuoteMeta(toolName)),
		Action:      "deny",
		Scope:       scope,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Priority:    200, // Higher priority than grant rules
	}

	// Add to appropriate rule set
	switch scope {
	case "global":
		pm.systemRules = append(pm.systemRules, rule)
	case "project":
		// TODO: Reimplement project-scoped rules using Viper
		// For now, treat as tool-specific rule
		pm.toolRules[toolName] = append(pm.toolRules[toolName], rule)
	default:
		pm.toolRules[toolName] = append(pm.toolRules[toolName], rule)
	}

	pm.logger.Info("Revoked tool permission", "tool", toolName, "scope", scope)
	return nil
}

// ListToolPermissions lists all permissions for a tool
func (pm *PermissionManager) ListToolPermissions(toolName string) ([]Permission, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var permissions []Permission

	// Check all rule sets for this tool
	allRules := make([]PermissionRule, 0)

	// System rules
	allRules = append(allRules, pm.systemRules...)

	// Tool rules
	if rules, exists := pm.toolRules[toolName]; exists {
		allRules = append(allRules, rules...)
	}

	// Context rules
	for _, rules := range pm.contextRules {
		allRules = append(allRules, rules...)
	}

	// Convert rules to permissions
	for _, rule := range allRules {
		if rule.ToolPattern != "" {
			if matched, _ := regexp.MatchString(rule.ToolPattern, toolName); matched {
				permission := Permission{
					ToolName:   toolName,
					Scope:      rule.Scope,
					Action:     rule.Action,
					Conditions: rule.Conditions,
					GrantedAt:  rule.CreatedAt,
					ExpiresAt:  rule.ExpiresAt,
					Metadata:   rule.Metadata,
				}
				permissions = append(permissions, permission)
			}
		}
	}

	return permissions, nil
}

// LoadDefaultRules loads default permission rules
func (pm *PermissionManager) LoadDefaultRules() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Default system rules for security
	pm.systemRules = []PermissionRule{
		{
			ID:          "system_deny_dangerous_bash",
			Name:        "Deny Dangerous Bash Commands",
			Description: "Prevent execution of potentially dangerous bash commands",
			ToolPattern: "bash|shell|execute",
			ParameterRules: []ParameterRule{
				{
					Parameter: "command",
					Pattern:   ".*(rm -rf|sudo|su|chmod 777|> /dev|dd if=).*",
				},
			},
			Action:    "deny",
			Scope:     "global",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Priority:  1000, // Very high priority
		},
		{
			ID:          "system_ask_file_operations",
			Name:        "Ask for File Operations",
			Description: "Ask user permission for file operations outside working directory",
			ToolPattern: "file_.*|read_.*|write_.*",
			Action:      "ask",
			Scope:       "global",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Priority:    100,
		},
	}

	pm.logger.Info("Loaded default permission rules", "count", len(pm.systemRules))
}
