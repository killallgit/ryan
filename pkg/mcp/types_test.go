package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToolCallRequest_Validation(t *testing.T) {
	request := ToolCallRequest{
		Name: "test-tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
			"param2": 42,
		},
		RequestID: "req-123",
		Timeout:   30 * time.Second,
		Context: map[string]interface{}{
			"user": "test-user",
		},
		Permissions: []string{"read", "write"},
	}

	assert.Equal(t, "test-tool", request.Name)
	assert.NotNil(t, request.Arguments)
	assert.Equal(t, "req-123", request.RequestID)
	assert.Equal(t, 30*time.Second, request.Timeout)
	assert.NotNil(t, request.Context)
	assert.Len(t, request.Permissions, 2)
}

func TestToolCallResult_Validation(t *testing.T) {
	result := ToolCallResult{
		Content: "Operation completed successfully",
		StructuredContent: map[string]interface{}{
			"status": "success",
			"count":  42,
		},
		ContentArray: []ContentItem{
			{
				Type:    "text",
				Content: "First item",
			},
			{
				Type: "data",
				Data: map[string]interface{}{
					"key": "value",
				},
			},
		},
		IsError:       false,
		ExecutionTime: 150 * time.Millisecond,
		ToolName:      "test-tool",
		ServerName:    "test-server",
		Metadata: map[string]interface{}{
			"version": "1.0",
		},
	}

	assert.Equal(t, "Operation completed successfully", result.Content)
	assert.False(t, result.IsError)
	assert.Equal(t, 150*time.Millisecond, result.ExecutionTime)
	assert.Equal(t, "test-tool", result.ToolName)
	assert.Equal(t, "test-server", result.ServerName)
	assert.Len(t, result.ContentArray, 2)
}

func TestContentItem_Validation(t *testing.T) {
	textItem := ContentItem{
		Type:    "text",
		Content: "Hello, World!",
		Metadata: map[string]interface{}{
			"language": "en",
		},
	}

	dataItem := ContentItem{
		Type: "json",
		Data: map[string]interface{}{
			"users": []string{"alice", "bob"},
			"count": 2,
		},
	}

	assert.Equal(t, "text", textItem.Type)
	assert.Equal(t, "Hello, World!", textItem.Content)
	assert.Equal(t, "json", dataItem.Type)
	assert.NotNil(t, dataItem.Data)
}

func TestToolDefinition_Validation(t *testing.T) {
	inputSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"filename": {"type": "string"},
			"content": {"type": "string"}
		},
		"required": ["filename"]
	}`)

	outputSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"success": {"type": "boolean"},
			"message": {"type": "string"}
		}
	}`)

	toolDef := ToolDefinition{
		Name:        "write_file",
		Description: "Writes content to a file",
		InputSchema: &inputSchema,
		OutputSchema: &outputSchema,
		Category:    "file_operations",
		Tags:        []string{"file", "write", "io"},
		Version:     "1.2.0",
		ServerName:  "file-server",
		ServerURL:   "http://localhost:8080",
		Metadata: map[string]interface{}{
			"author": "test",
		},
	}

	assert.Equal(t, "write_file", toolDef.Name)
	assert.Equal(t, "Writes content to a file", toolDef.Description)
	assert.NotNil(t, toolDef.InputSchema)
	assert.NotNil(t, toolDef.OutputSchema)
	assert.Equal(t, "file_operations", toolDef.Category)
	assert.Len(t, toolDef.Tags, 3)
	assert.Equal(t, "1.2.0", toolDef.Version)
}

func TestServerConfig_Validation(t *testing.T) {
	config := ServerConfig{
		Name:           "test-server",
		URL:            "http://localhost:8080",
		Timeout:        30 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     1 * time.Second,
		KeepAlive:      true,
		MaxConnections: 10,
		AuthType:       "bearer",
		Credentials: map[string]string{
			"token": "secret-token",
		},
		SupportedFeatures: []string{"tool-calling", "streaming"},
		Enabled:           true,
		Metadata: map[string]interface{}{
			"version": "1.0",
		},
	}

	assert.Equal(t, "test-server", config.Name)
	assert.Equal(t, "http://localhost:8080", config.URL)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.RetryAttempts)
	assert.True(t, config.KeepAlive)
	assert.Equal(t, 10, config.MaxConnections)
	assert.True(t, config.Enabled)
}

func TestServerInfo_Validation(t *testing.T) {
	now := time.Now()
	info := ServerInfo{
		Name:            "test-server",
		URL:             "http://localhost:8080",
		Status:          "connected",
		ConnectedAt:     now,
		LastPing:        now.Add(-5 * time.Second),
		ResponseTime:    100 * time.Millisecond,
		Version:         "1.0.0",
		SupportedFeatures: []string{"tool-calling"},
		AvailableTools:  5,
		TotalCalls:      100,
		SuccessfulCalls: 95,
		FailedCalls:     5,
		AverageLatency:  150 * time.Millisecond,
	}

	assert.Equal(t, "test-server", info.Name)
	assert.Equal(t, "connected", info.Status)
	assert.Equal(t, 5, info.AvailableTools)
	assert.Equal(t, int64(100), info.TotalCalls)
	assert.Equal(t, int64(95), info.SuccessfulCalls)
	assert.Equal(t, int64(5), info.FailedCalls)
}

func TestMCPRequest_Validation(t *testing.T) {
	request := MCPRequest{
		Method: "tools/call",
		Params: map[string]interface{}{
			"name": "test-tool",
			"arguments": map[string]interface{}{
				"param": "value",
			},
		},
		ID: "req-123",
	}

	assert.Equal(t, "tools/call", request.Method)
	assert.NotNil(t, request.Params)
	assert.Equal(t, "req-123", request.ID)
}

func TestMCPResponse_Validation(t *testing.T) {
	response := MCPResponse{
		Result: map[string]interface{}{
			"success": true,
			"data":    "result data",
		},
		ID: "req-123",
		Metadata: map[string]interface{}{
			"server": "test-server",
		},
	}

	assert.NotNil(t, response.Result)
	assert.Nil(t, response.Error)
	assert.Equal(t, "req-123", response.ID)

	// Test error response
	errorResponse := MCPResponse{
		Error: &MCPError{
			Code:    400,
			Message: "Bad Request",
			Data: map[string]interface{}{
				"details": "Invalid parameter",
			},
		},
		ID: "req-124",
	}

	assert.Nil(t, errorResponse.Result)
	assert.NotNil(t, errorResponse.Error)
	assert.Equal(t, 400, errorResponse.Error.Code)
	assert.Equal(t, "Bad Request", errorResponse.Error.Message)
}

func TestMCPError_Validation(t *testing.T) {
	err := MCPError{
		Code:    404,
		Message: "Tool not found",
		Data: map[string]interface{}{
			"toolName": "unknown-tool",
			"server":   "test-server",
		},
	}

	assert.Equal(t, 404, err.Code)
	assert.Equal(t, "Tool not found", err.Message)
	assert.NotNil(t, err.Data)
	assert.Equal(t, "unknown-tool", err.Data["toolName"])
}