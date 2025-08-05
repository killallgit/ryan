package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMCP_BasicTypes(t *testing.T) {
	t.Run("ToolCallRequest", func(t *testing.T) {
		request := ToolCallRequest{
			Name: "test-tool",
			Arguments: map[string]interface{}{
				"param": "value",
			},
			RequestID: "req-123",
			Timeout:   30 * time.Second,
		}

		assert.Equal(t, "test-tool", request.Name)
		assert.NotNil(t, request.Arguments)
		assert.Equal(t, "req-123", request.RequestID)
	})

	t.Run("ToolCallResult", func(t *testing.T) {
		result := ToolCallResult{
			Content:       "success",
			IsError:       false,
			ExecutionTime: 100 * time.Millisecond,
			ToolName:      "test-tool",
			ServerName:    "test-server",
		}

		assert.Equal(t, "success", result.Content)
		assert.False(t, result.IsError)
		assert.Equal(t, "test-tool", result.ToolName)
	})

	t.Run("ToolDefinition", func(t *testing.T) {
		schema := json.RawMessage(`{"type": "object"}`)
		toolDef := ToolDefinition{
			Name:         "test-tool",
			Description:  "A test tool",
			InputSchema:  &schema,
			OutputSchema: &schema,
			ServerName:   "test-server",
		}

		assert.Equal(t, "test-tool", toolDef.Name)
		assert.Equal(t, "A test tool", toolDef.Description)
		assert.NotNil(t, toolDef.InputSchema)
	})

	t.Run("ServerConfig", func(t *testing.T) {
		config := ServerConfig{
			Name:          "test-server",
			URL:           "http://localhost:8080",
			Timeout:       30 * time.Second,
			RetryAttempts: 3,
			Enabled:       true,
		}

		assert.Equal(t, "test-server", config.Name)
		assert.Equal(t, "http://localhost:8080", config.URL)
		assert.True(t, config.Enabled)
	})

	t.Run("ServerInfo", func(t *testing.T) {
		info := ServerInfo{
			Name:            "test-server",
			URL:             "http://localhost:8080",
			Status:          "connected",
			AvailableTools:  5,
			TotalCalls:      100,
			SuccessfulCalls: 95,
			FailedCalls:     5,
		}

		assert.Equal(t, "test-server", info.Name)
		assert.Equal(t, "connected", info.Status)
		assert.Equal(t, 5, info.AvailableTools)
	})
}

func TestMCP_ProtocolTypes(t *testing.T) {
	t.Run("MCPRequest", func(t *testing.T) {
		request := MCPRequest{
			Method: "tools/call",
			Params: map[string]interface{}{
				"name": "test-tool",
			},
			ID: "req-123",
		}

		assert.Equal(t, "tools/call", request.Method)
		assert.NotNil(t, request.Params)
	})

	t.Run("MCPResponse", func(t *testing.T) {
		response := MCPResponse{
			Result: map[string]interface{}{
				"success": true,
			},
			ID: "req-123",
		}

		assert.NotNil(t, response.Result)
		assert.Nil(t, response.Error)
	})

	t.Run("MCPError", func(t *testing.T) {
		mcpError := MCPError{
			Code:    400,
			Message: "Bad Request",
			Data: map[string]interface{}{
				"details": "Invalid parameters",
			},
		}

		assert.Equal(t, 400, mcpError.Code)
		assert.Equal(t, "Bad Request", mcpError.Message)
		assert.NotNil(t, mcpError.Data)
	})
}

func TestMCP_ContentTypes(t *testing.T) {
	t.Run("ContentItem", func(t *testing.T) {
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
				"key": "value",
			},
		}

		assert.Equal(t, "text", textItem.Type)
		assert.Equal(t, "Hello, World!", textItem.Content)
		assert.Equal(t, "json", dataItem.Type)
		assert.NotNil(t, dataItem.Data)
	})
}

func TestMCP_ClientConfig(t *testing.T) {
	config := MCPClientConfig{
		DefaultTimeout:         30 * time.Second,
		MaxRetryAttempts:       3,
		EnableInputValidation:  true,
		EnableOutputValidation: true,
		SchemaCacheSize:        100,
		SchemaCacheTTL:         5 * time.Minute,
	}

	assert.Equal(t, 30*time.Second, config.DefaultTimeout)
	assert.Equal(t, 3, config.MaxRetryAttempts)
	assert.True(t, config.EnableInputValidation)
	assert.True(t, config.EnableOutputValidation)
}