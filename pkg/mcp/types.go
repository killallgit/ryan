package mcp

import (
	"context"
	"encoding/json"
	"time"
)

// MCP Protocol Types and Interfaces
// Based on Claude CLI's Model Context Protocol implementation

// MCPClient defines the interface for MCP client operations
type MCPClient interface {
	// Core tool calling functionality
	CallTool(ctx context.Context, request ToolCallRequest) (*ToolCallResult, error)
	
	// Server management
	ConnectToServer(serverConfig ServerConfig) error
	DisconnectFromServer(serverName string) error
	ListServers() []ServerInfo
	
	// Tool management
	ListTools(serverName string) ([]ToolDefinition, error)
	GetToolSchema(serverName, toolName string) (*json.RawMessage, error)
	CacheToolSchemas(tools []ToolDefinition) error
	
	// Validation
	ValidateToolOutput(toolName string, output interface{}) error
	
	// Lifecycle
	Close() error
}

// ToolCallRequest represents a request to call an MCP tool
type ToolCallRequest struct {
	// Tool identification
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	
	// Request metadata
	RequestID string `json:"requestId,omitempty"`
	Timeout   time.Duration `json:"-"`
	
	// Context and permissions
	Context     map[string]interface{} `json:"context,omitempty"`
	Permissions []string              `json:"permissions,omitempty"`
}

// ToolCallResult represents the result of an MCP tool call
type ToolCallResult struct {
	// Core result data
	Content           string                 `json:"content,omitempty"`
	StructuredContent map[string]interface{} `json:"structuredContent,omitempty"`
	ContentArray      []ContentItem          `json:"contentArray,omitempty"`
	
	// Result metadata
	IsError     bool   `json:"isError"`
	ErrorCode   string `json:"errorCode,omitempty"`
	ErrorDetail string `json:"errorDetail,omitempty"`
	
	// Execution metadata
	ExecutionTime time.Duration              `json:"executionTime"`
	Metadata      map[string]interface{}     `json:"metadata,omitempty"`
	
	// Tool information
	ToolName   string `json:"toolName"`
	ServerName string `json:"serverName"`
}

// ContentItem represents a single content item in a content array
type ContentItem struct {
	Type     string                 `json:"type"`
	Content  string                 `json:"content,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolDefinition represents an MCP tool definition
type ToolDefinition struct {
	// Basic tool information
	Name        string `json:"name"`
	Description string `json:"description"`
	
	// Schema definitions
	InputSchema  *json.RawMessage `json:"inputSchema,omitempty"`
	OutputSchema *json.RawMessage `json:"outputSchema,omitempty"`
	
	// Tool metadata
	Category    string                 `json:"category,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	
	// Server information
	ServerName string `json:"serverName"`
	ServerURL  string `json:"serverUrl,omitempty"`
}

// ServerConfig represents MCP server configuration
type ServerConfig struct {
	// Server identification
	Name string `json:"name"`
	URL  string `json:"url"`
	
	// Connection settings
	Timeout         time.Duration `json:"timeout"`
	RetryAttempts   int           `json:"retryAttempts"`
	RetryDelay      time.Duration `json:"retryDelay"`
	KeepAlive       bool          `json:"keepAlive"`
	MaxConnections  int           `json:"maxConnections"`
	
	// Authentication
	AuthType    string            `json:"authType,omitempty"`
	Credentials map[string]string `json:"credentials,omitempty"`
	
	// Capabilities
	SupportedFeatures []string `json:"supportedFeatures,omitempty"`
	
	// Configuration metadata
	Enabled  bool                   `json:"enabled"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ServerInfo represents information about a connected MCP server
type ServerInfo struct {
	// Basic information
	Name   string `json:"name"`
	URL    string `json:"url"`
	Status string `json:"status"` // "connected", "disconnected", "error"
	
	// Connection information
	ConnectedAt   time.Time     `json:"connectedAt,omitempty"`
	LastPing      time.Time     `json:"lastPing,omitempty"`
	ResponseTime  time.Duration `json:"responseTime,omitempty"`
	
	// Server capabilities
	Version           string   `json:"version,omitempty"`
	SupportedFeatures []string `json:"supportedFeatures,omitempty"`
	AvailableTools    int      `json:"availableTools"`
	
	// Statistics
	TotalCalls    int64         `json:"totalCalls"`
	SuccessfulCalls int64       `json:"successfulCalls"`
	FailedCalls   int64         `json:"failedCalls"`
	AverageLatency time.Duration `json:"averageLatency"`
	
	// Error information
	LastError     string    `json:"lastError,omitempty"`
	LastErrorTime time.Time `json:"lastErrorTime,omitempty"`
}

// MCPRequest represents a generic MCP protocol request
type MCPRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
	ID     string                 `json:"id,omitempty"`
}

// MCPResponse represents a generic MCP protocol response
type MCPResponse struct {
	Result interface{}            `json:"result,omitempty"`
	Error  *MCPError              `json:"error,omitempty"`
	ID     string                 `json:"id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MCPError represents an MCP protocol error
type MCPError struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// ToolValidator defines the interface for tool validation
type ToolValidator interface {
	ValidateInput(toolName string, input map[string]interface{}) error
	ValidateOutput(toolName string, output interface{}) error
	GetInputSchema(toolName string) (*json.RawMessage, error)
	GetOutputSchema(toolName string) (*json.RawMessage, error)
}

// ServerDiscovery defines the interface for MCP server discovery
type ServerDiscovery interface {
	DiscoverServers(ctx context.Context) ([]ServerConfig, error)
	RegisterServer(config ServerConfig) error
	UnregisterServer(serverName string) error
	GetServerConfig(serverName string) (*ServerConfig, error)
	ListRegisteredServers() []ServerConfig
}

// PermissionEvaluator defines the interface for tool permission evaluation
type PermissionEvaluator interface {
	// Permission checking
	CanExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (PermissionResult, error)
	CanAccessResource(ctx context.Context, resourceType, resourcePath string) (PermissionResult, error)
	
	// Permission management
	GrantToolPermission(toolName string, scope string) error
	RevokeToolPermission(toolName string, scope string) error
	ListToolPermissions(toolName string) ([]Permission, error)
}

// PermissionResult represents the result of a permission check
type PermissionResult struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
	Action  string `json:"action"` // "allow", "deny", "ask"
	
	// Additional context
	Conditions []string               `json:"conditions,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Permission represents a tool permission
type Permission struct {
	ToolName     string                 `json:"toolName"`
	Scope        string                 `json:"scope"`        // "global", "project", "session"
	Action       string                 `json:"action"`       // "allow", "deny", "ask"
	Conditions   []string               `json:"conditions,omitempty"`
	GrantedAt    time.Time              `json:"grantedAt"`
	ExpiresAt    *time.Time             `json:"expiresAt,omitempty"`
	GrantedBy    string                 `json:"grantedBy,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ToolExecutionContext provides context for tool execution
type ToolExecutionContext struct {
	// Request context
	RequestID   string                 `json:"requestId"`
	UserID      string                 `json:"userId,omitempty"`
	SessionID   string                 `json:"sessionId,omitempty"`
	ProjectRoot string                 `json:"projectRoot,omitempty"`
	
	// Execution environment
	WorkingDirectory string            `json:"workingDirectory,omitempty"`    
	EnvironmentVars  map[string]string `json:"environmentVars,omitempty"`
	
	// Permission context
	Permissions []Permission           `json:"permissions,omitempty"`
	
	// Timing and limits
	Timeout     time.Duration          `json:"timeout,omitempty"`
	MaxMemory   int64                  `json:"maxMemory,omitempty"`
	MaxCPUTime  time.Duration          `json:"maxCpuTime,omitempty"`
	
	// Additional context
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ToolExecutionResult represents the complete result of tool execution
type ToolExecutionResult struct {
	// Core result
	Result *ToolCallResult `json:"result"`
	
	// Execution metadata
	Context     *ToolExecutionContext `json:"context"`
	StartTime   time.Time             `json:"startTime"`
	EndTime     time.Time             `json:"endTime"`
	Duration    time.Duration         `json:"duration"`
	
	// Resource usage
	MemoryUsed  int64 `json:"memoryUsed,omitempty"`
	CPUTime     time.Duration `json:"cpuTime,omitempty"`
	
	// Hook results
	HookResults []HookResult `json:"hookResults,omitempty"`
}

// HookResult represents the result of a lifecycle hook
type HookResult struct {
	HookName  string                 `json:"hookName"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MCPClientConfig represents configuration for the MCP client
type MCPClientConfig struct {
	// Connection settings  
	DefaultTimeout    time.Duration `json:"defaultTimeout"`
	MaxRetryAttempts  int           `json:"maxRetryAttempts"`
	RetryDelay        time.Duration `json:"retryDelay"`
	KeepAliveInterval time.Duration `json:"keepAliveInterval"`
	
	// Caching settings
	EnableSchemaCache bool          `json:"enableSchemaCache"`
	SchemaCacheSize   int           `json:"schemaCacheSize"`
	SchemaCacheTTL    time.Duration `json:"schemaCacheTTL"`
	
	// Security settings
	EnablePermissions    bool `json:"enablePermissions"`
	DefaultPermissionAction string `json:"defaultPermissionAction"` // "allow", "deny", "ask"
	
	// Validation settings
	EnableInputValidation  bool `json:"enableInputValidation"`
	EnableOutputValidation bool `json:"enableOutputValidation"`
	StrictValidation       bool `json:"strictValidation"`
	
	// Execution settings
	MaxConcurrentCalls int           `json:"maxConcurrentCalls"`
	DefaultMemoryLimit int64         `json:"defaultMemoryLimit"`
	DefaultCPULimit    time.Duration `json:"defaultCpuLimit"`
	
	// Logging and debugging
	EnableRequestLogging  bool `json:"enableRequestLogging"`
	EnableResponseLogging bool `json:"enableResponseLogging"`
	LogLevel              string `json:"logLevel"`
}