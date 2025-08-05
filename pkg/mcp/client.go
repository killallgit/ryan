package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// Client implements the MCPClient interface
type Client struct {
	// Configuration
	config MCPClientConfig

	// Server management
	servers   map[string]*ServerConnection
	serversMu sync.RWMutex

	// Schema caching
	schemaCache *SchemaCache

	// Validation
	validator ToolValidator

	// Permission evaluation
	permissionEvaluator PermissionEvaluator

	// Discovery
	discovery ServerDiscovery

	// Logging
	logger *logger.Logger

	// HTTP client for server communication
	httpClient *http.Client

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ServerConnection represents a connection to an MCP server
type ServerConnection struct {
	config    ServerConfig
	client    *http.Client
	info      ServerInfo
	tools     map[string]ToolDefinition
	toolsMu   sync.RWMutex
	connected bool
	mu        sync.RWMutex
}

// NewClient creates a new MCP client with the given configuration
func NewClient(config MCPClientConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		config:      config,
		servers:     make(map[string]*ServerConnection),
		schemaCache: NewSchemaCache(config.SchemaCacheSize, config.SchemaCacheTTL),
		logger:      logger.WithComponent("mcp-client"),
		httpClient: &http.Client{
			Timeout: config.DefaultTimeout,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize validator if validation is enabled
	if config.EnableInputValidation || config.EnableOutputValidation {
		client.validator = NewJSONSchemaValidator()
	}

	return client
}

// ConnectToServer establishes a connection to an MCP server
func (c *Client) ConnectToServer(serverConfig ServerConfig) error {
	c.serversMu.Lock()
	defer c.serversMu.Unlock()

	// Check if server is already connected
	if conn, exists := c.servers[serverConfig.Name]; exists {
		if conn.connected {
			return nil // Already connected
		}
	}

	// Create server connection
	conn := &ServerConnection{
		config: serverConfig,
		client: &http.Client{
			Timeout: serverConfig.Timeout,
		},
		info: ServerInfo{
			Name:   serverConfig.Name,
			URL:    serverConfig.URL,
			Status: "connecting",
		},
		tools: make(map[string]ToolDefinition),
	}

	// Test connection
	if err := c.testServerConnection(conn); err != nil {
		return fmt.Errorf("failed to connect to server %s: %w", serverConfig.Name, err)
	}

	// Load available tools
	if err := c.loadServerTools(conn); err != nil {
		c.logger.Warn("Failed to load tools from server", "server", serverConfig.Name, "error", err)
	}

	// Mark as connected
	conn.mu.Lock()
	conn.connected = true
	conn.info.Status = "connected"
	conn.info.ConnectedAt = time.Now()
	conn.mu.Unlock()

	// Store connection
	c.servers[serverConfig.Name] = conn

	c.logger.Info("Connected to MCP server", "server", serverConfig.Name, "url", serverConfig.URL)
	return nil
}

// testServerConnection tests if we can communicate with the server
func (c *Client) testServerConnection(conn *ServerConnection) error {
	// Send a ping/health check request
	request := MCPRequest{
		Method: "ping",
		ID:     generateRequestID(),
	}

	response, err := c.sendRequest(conn, request)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("server error: %s", response.Error.Message)
	}

	return nil
}

// loadServerTools loads available tools from the server
func (c *Client) loadServerTools(conn *ServerConnection) error {
	// Request tools list
	request := MCPRequest{
		Method: "tools/list",
		ID:     generateRequestID(),
	}

	response, err := c.sendRequest(conn, request)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("server error: %s", response.Error.Message)
	}

	// Parse tools list
	var toolsResult struct {
		Tools []ToolDefinition `json:"tools"`
	}

	if err := mapToStruct(response.Result, &toolsResult); err != nil {
		return fmt.Errorf("failed to parse tools response: %w", err)
	}

	// Store tools
	conn.toolsMu.Lock()
	for _, tool := range toolsResult.Tools {
		tool.ServerName = conn.config.Name
		tool.ServerURL = conn.config.URL
		conn.tools[tool.Name] = tool
	}
	conn.info.AvailableTools = len(conn.tools)
	conn.toolsMu.Unlock()

	// Cache tool schemas if caching is enabled
	if c.config.EnableSchemaCache {
		if err := c.CacheToolSchemas(toolsResult.Tools); err != nil {
			c.logger.Warn("Failed to cache tool schemas", "server", conn.config.Name, "error", err)
		}
	}

	return nil
}

// DisconnectFromServer disconnects from an MCP server
func (c *Client) DisconnectFromServer(serverName string) error {
	c.serversMu.Lock()
	defer c.serversMu.Unlock()

	conn, exists := c.servers[serverName]
	if !exists {
		return fmt.Errorf("server not found: %s", serverName)
	}

	conn.mu.Lock()
	conn.connected = false
	conn.info.Status = "disconnected"
	conn.mu.Unlock()

	delete(c.servers, serverName)

	c.logger.Info("Disconnected from MCP server", "server", serverName)
	return nil
}

// CallTool calls an MCP tool
func (c *Client) CallTool(ctx context.Context, request ToolCallRequest) (*ToolCallResult, error) {
	startTime := time.Now()

	// Find the server that provides this tool
	serverConn, err := c.findToolServer(request.Name)
	if err != nil {
		return &ToolCallResult{
			IsError:       true,
			ErrorCode:     "TOOL_NOT_FOUND",
			ErrorDetail:   err.Error(),
			ToolName:      request.Name,
			ExecutionTime: time.Since(startTime),
		}, nil
	}

	// Check permissions
	if c.config.EnablePermissions && c.permissionEvaluator != nil {
		permResult, err := c.permissionEvaluator.CanExecuteTool(ctx, request.Name, request.Arguments)
		if err != nil {
			return &ToolCallResult{
				IsError:       true,
				ErrorCode:     "PERMISSION_ERROR",
				ErrorDetail:   fmt.Sprintf("Permission check failed: %v", err),
				ToolName:      request.Name,
				ServerName:    serverConn.config.Name,
				ExecutionTime: time.Since(startTime),
			}, nil
		}

		if !permResult.Allowed {
			return &ToolCallResult{
				IsError:       true,
				ErrorCode:     "PERMISSION_DENIED",
				ErrorDetail:   permResult.Reason,
				ToolName:      request.Name,
				ServerName:    serverConn.config.Name,
				ExecutionTime: time.Since(startTime),
			}, nil
		}
	}

	// Validate input if validation is enabled
	if c.config.EnableInputValidation && c.validator != nil {
		if err := c.validator.ValidateInput(request.Name, request.Arguments); err != nil {
			return &ToolCallResult{
				IsError:       true,
				ErrorCode:     "VALIDATION_ERROR",
				ErrorDetail:   fmt.Sprintf("Input validation failed: %v", err),
				ToolName:      request.Name,
				ServerName:    serverConn.config.Name,
				ExecutionTime: time.Since(startTime),
			}, nil
		}
	}

	// Create MCP request
	mcpRequest := MCPRequest{
		Method: "tools/call",
		Params: map[string]interface{}{
			"name":      request.Name,
			"arguments": request.Arguments,
		},
		ID: request.RequestID,
	}

	if mcpRequest.ID == "" {
		mcpRequest.ID = generateRequestID()
	}

	// Set timeout context
	if request.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, request.Timeout)
		defer cancel()
	}

	// Send request
	response, err := c.sendRequestWithContext(ctx, serverConn, mcpRequest)
	if err != nil {
		return &ToolCallResult{
			IsError:       true,
			ErrorCode:     "COMMUNICATION_ERROR",
			ErrorDetail:   fmt.Sprintf("Failed to communicate with server: %v", err),
			ToolName:      request.Name,
			ServerName:    serverConn.config.Name,
			ExecutionTime: time.Since(startTime),
		}, nil
	}

	// Handle server error
	if response.Error != nil {
		return &ToolCallResult{
			IsError:       true,
			ErrorCode:     fmt.Sprintf("SERVER_ERROR_%d", response.Error.Code),
			ErrorDetail:   response.Error.Message,
			ToolName:      request.Name,
			ServerName:    serverConn.config.Name,
			Metadata:      response.Error.Data,
			ExecutionTime: time.Since(startTime),
		}, nil
	}

	// Parse result
	result := &ToolCallResult{
		ToolName:      request.Name,
		ServerName:    serverConn.config.Name,
		ExecutionTime: time.Since(startTime),
	}

	if err := c.parseToolResult(response.Result, result); err != nil {
		result.IsError = true
		result.ErrorCode = "PARSING_ERROR"
		result.ErrorDetail = fmt.Sprintf("Failed to parse tool result: %v", err)
		return result, nil
	}

	// Validate output if validation is enabled
	if c.config.EnableOutputValidation && c.validator != nil && result.StructuredContent != nil {
		if err := c.validator.ValidateOutput(request.Name, result.StructuredContent); err != nil {
			result.IsError = true
			result.ErrorCode = "OUTPUT_VALIDATION_ERROR"
			result.ErrorDetail = fmt.Sprintf("Output validation failed: %v", err)
			return result, nil
		}
	}

	// Update server statistics
	c.updateServerStats(serverConn, result)

	return result, nil
}

// findToolServer finds the server that provides the specified tool
func (c *Client) findToolServer(toolName string) (*ServerConnection, error) {
	c.serversMu.RLock()
	defer c.serversMu.RUnlock()

	for _, conn := range c.servers {
		if !conn.connected {
			continue
		}

		conn.toolsMu.RLock()
		_, exists := conn.tools[toolName]
		conn.toolsMu.RUnlock()

		if exists {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("tool not found: %s", toolName)
}

// sendRequest sends an MCP request to a server
func (c *Client) sendRequest(conn *ServerConnection, request MCPRequest) (*MCPResponse, error) {
	return c.sendRequestWithContext(c.ctx, conn, request)
}

// sendRequestWithContext sends an MCP request with context
func (c *Client) sendRequestWithContext(ctx context.Context, conn *ServerConnection, request MCPRequest) (*MCPResponse, error) {
	// Serialize request
	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", conn.config.URL, bytes.NewReader(requestData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Add authentication if configured
	if err := c.addAuthentication(httpReq, conn.config); err != nil {
		return nil, fmt.Errorf("failed to add authentication: %w", err)
	}

	// Send request
	httpResp, err := conn.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Parse response
	var mcpResponse MCPResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&mcpResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &mcpResponse, nil
}

// addAuthentication adds authentication to the HTTP request
func (c *Client) addAuthentication(req *http.Request, config ServerConfig) error {
	switch config.AuthType {
	case "bearer":
		if token, ok := config.Credentials["token"]; ok {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	case "basic":
		if username, ok := config.Credentials["username"]; ok {
			if password, ok := config.Credentials["password"]; ok {
				req.SetBasicAuth(username, password)
			}
		}
	case "apikey":
		if key, ok := config.Credentials["key"]; ok {
			if header, ok := config.Credentials["header"]; ok {
				req.Header.Set(header, key)
			} else {
				req.Header.Set("X-API-Key", key)
			}
		}
	}
	return nil
}

// parseToolResult parses the tool execution result
func (c *Client) parseToolResult(result interface{}, toolResult *ToolCallResult) error {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		// Simple string result
		if str, ok := result.(string); ok {
			toolResult.Content = str
			return nil
		}
		return fmt.Errorf("unexpected result type: %T", result)
	}

	// Extract content fields
	if content, ok := resultMap["content"].(string); ok {
		toolResult.Content = content
	}

	if structuredContent, ok := resultMap["structuredContent"].(map[string]interface{}); ok {
		toolResult.StructuredContent = structuredContent
	}

	if contentArray, ok := resultMap["contentArray"].([]interface{}); ok {
		for _, item := range contentArray {
			if itemMap, ok := item.(map[string]interface{}); ok {
				contentItem := ContentItem{}
				if err := mapToStruct(itemMap, &contentItem); err == nil {
					toolResult.ContentArray = append(toolResult.ContentArray, contentItem)
				}
			}
		}
	}

	// Extract error information
	if isError, ok := resultMap["isError"].(bool); ok {
		toolResult.IsError = isError
	}

	if errorCode, ok := resultMap["errorCode"].(string); ok {
		toolResult.ErrorCode = errorCode
	}

	if errorDetail, ok := resultMap["errorDetail"].(string); ok {
		toolResult.ErrorDetail = errorDetail
	}

	// Extract metadata
	if metadata, ok := resultMap["metadata"].(map[string]interface{}); ok {
		toolResult.Metadata = metadata
	}

	return nil
}

// updateServerStats updates server execution statistics
func (c *Client) updateServerStats(conn *ServerConnection, result *ToolCallResult) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.info.TotalCalls++
	if result.IsError {
		conn.info.FailedCalls++
		conn.info.LastError = result.ErrorDetail
		conn.info.LastErrorTime = time.Now()
	} else {
		conn.info.SuccessfulCalls++
	}

	// Update average latency
	if conn.info.TotalCalls > 0 {
		totalLatency := conn.info.AverageLatency * time.Duration(conn.info.TotalCalls-1)
		conn.info.AverageLatency = (totalLatency + result.ExecutionTime) / time.Duration(conn.info.TotalCalls)
	} else {
		conn.info.AverageLatency = result.ExecutionTime
	}

	conn.info.LastPing = time.Now()
	conn.info.ResponseTime = result.ExecutionTime
}

// ListServers returns information about all connected servers
func (c *Client) ListServers() []ServerInfo {
	c.serversMu.RLock()
	defer c.serversMu.RUnlock()

	servers := make([]ServerInfo, 0, len(c.servers))
	for _, conn := range c.servers {
		conn.mu.RLock()
		servers = append(servers, conn.info)
		conn.mu.RUnlock()
	}

	return servers
}

// ListTools returns all available tools from a specific server
func (c *Client) ListTools(serverName string) ([]ToolDefinition, error) {
	c.serversMu.RLock()
	conn, exists := c.servers[serverName]
	c.serversMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("server not found: %s", serverName)
	}

	conn.toolsMu.RLock()
	defer conn.toolsMu.RUnlock()

	tools := make([]ToolDefinition, 0, len(conn.tools))
	for _, tool := range conn.tools {
		tools = append(tools, tool)
	}

	return tools, nil
}

// Close closes the MCP client and all server connections
func (c *Client) Close() error {
	c.cancel()  // Cancel context to stop background operations
	c.wg.Wait() // Wait for all goroutines to finish

	// Disconnect from all servers
	c.serversMu.Lock()
	for serverName := range c.servers {
		delete(c.servers, serverName)
	}
	c.serversMu.Unlock()

	c.logger.Info("MCP client closed")
	return nil
}

// Utility functions

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// mapToStruct converts a map to a struct using JSON marshaling
func mapToStruct(m interface{}, s interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, s)
}
