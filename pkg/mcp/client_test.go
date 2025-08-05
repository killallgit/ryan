package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	config := MCPClientConfig{
		DefaultTimeout:         30 * time.Second,
		MaxRetryAttempts:       3,
		EnableInputValidation:  true,
		EnableOutputValidation: true,
		SchemaCacheSize:        100,
		SchemaCacheTTL:         5 * time.Minute,
	}

	client := NewClient(config)

	assert.NotNil(t, client)
	assert.Equal(t, config.DefaultTimeout, client.config.DefaultTimeout)
	assert.Equal(t, config.MaxRetryAttempts, client.config.MaxRetryAttempts)
	assert.NotNil(t, client.servers)
	assert.NotNil(t, client.schemaCache)
	assert.NotNil(t, client.httpClient)
}

func TestClient_ConnectToServer(t *testing.T) {
	client := NewClient(MCPClientConfig{
		DefaultTimeout: 10 * time.Second,
	})
	defer client.Close()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": {"tools": []}}`))
	}))
	defer server.Close()

	serverConfig := ServerConfig{
		Name:          "test-server",
		URL:           server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 2,
		Enabled:       true,
	}

	err := client.ConnectToServer(serverConfig)
	assert.NoError(t, err)

	// Verify server was added
	servers := client.ListServers()
	assert.Len(t, servers, 1)
	assert.Equal(t, "test-server", servers[0].Name)
}

func TestClient_ConnectToServer_InvalidURL(t *testing.T) {
	client := NewClient(MCPClientConfig{
		DefaultTimeout: 1 * time.Second,
	})
	defer client.Close()

	serverConfig := ServerConfig{
		Name:          "invalid-server",
		URL:           "http://invalid-url-that-does-not-exist.local",
		Timeout:       1 * time.Second,
		RetryAttempts: 1,
		Enabled:       true,
	}

	err := client.ConnectToServer(serverConfig)
	assert.Error(t, err)
}

func TestClient_DisconnectFromServer(t *testing.T) {
	client := NewClient(MCPClientConfig{})
	defer client.Close()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": {"tools": []}}`))
	}))
	defer server.Close()

	serverConfig := ServerConfig{
		Name:    "test-server",
		URL:     server.URL,
		Enabled: true,
	}

	// Connect first
	err := client.ConnectToServer(serverConfig)
	require.NoError(t, err)

	// Then disconnect
	err = client.DisconnectFromServer("test-server")
	assert.NoError(t, err)

	// Verify server was removed
	servers := client.ListServers()
	assert.Len(t, servers, 0)
}

func TestClient_DisconnectFromServer_NotFound(t *testing.T) {
	client := NewClient(MCPClientConfig{})
	defer client.Close()

	err := client.DisconnectFromServer("non-existent-server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server not found")
}

func TestClient_CallTool_ServerNotFound(t *testing.T) {
	client := NewClient(MCPClientConfig{})
	defer client.Close()

	request := ToolCallRequest{
		Name:      "test-tool",
		Arguments: map[string]interface{}{"param": "value"},
		RequestID: "req-123",
	}

	ctx := context.Background()
	result, err := client.CallTool(ctx, request)
	assert.NoError(t, err) // The method returns ToolCallResult with error info, not an error
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Equal(t, "TOOL_NOT_FOUND", result.ErrorCode)
}

func TestClient_ListServers_Empty(t *testing.T) {
	client := NewClient(MCPClientConfig{})
	defer client.Close()

	servers := client.ListServers()
	assert.Empty(t, servers)
}

func TestClient_ListTools_ServerNotFound(t *testing.T) {
	client := NewClient(MCPClientConfig{})
	defer client.Close()

	tools, err := client.ListTools("non-existent-server")
	assert.Error(t, err)
	assert.Nil(t, tools)
	assert.Contains(t, err.Error(), "server not found")
}

func TestClient_Close(t *testing.T) {
	client := NewClient(MCPClientConfig{})

	err := client.Close()
	assert.NoError(t, err)
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	// IDs might be the same due to timing, so just check they're not empty and have expected prefix
	assert.Contains(t, id1, "req_")
	assert.Contains(t, id2, "req_")
}

func TestMapToStruct(t *testing.T) {
	// Test successful mapping
	source := map[string]interface{}{
		"name":    "test",
		"value":   42,
		"enabled": true,
	}

	type TestStruct struct {
		Name    string `json:"name"`
		Value   int    `json:"value"`
		Enabled bool   `json:"enabled"`
	}

	var target TestStruct
	err := mapToStruct(source, &target)
	assert.NoError(t, err)
	assert.Equal(t, "test", target.Name)
	assert.Equal(t, 42, target.Value)
	assert.True(t, target.Enabled)

	// Test with invalid target (not a pointer)
	var invalidTarget TestStruct
	err = mapToStruct(source, invalidTarget)
	assert.Error(t, err)
}

func TestClient_AuthenticationMethods(t *testing.T) {
	client := NewClient(MCPClientConfig{})
	defer client.Close()

	req, _ := http.NewRequest("GET", "http://example.com", nil)

	t.Run("Bearer token authentication", func(t *testing.T) {
		config := ServerConfig{
			AuthType: "bearer",
			Credentials: map[string]string{
				"token": "test-token",
			},
		}

		err := client.addAuthentication(req, config)
		assert.NoError(t, err)
		assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
	})

	t.Run("Basic authentication", func(t *testing.T) {
		config := ServerConfig{
			AuthType: "basic",
			Credentials: map[string]string{
				"username": "user",
				"password": "pass",
			},
		}

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		err := client.addAuthentication(req, config)
		assert.NoError(t, err)
		assert.NotEmpty(t, req.Header.Get("Authorization"))
		assert.Contains(t, req.Header.Get("Authorization"), "Basic")
	})

	t.Run("No authentication", func(t *testing.T) {
		config := ServerConfig{
			AuthType: "",
		}

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		err := client.addAuthentication(req, config)
		assert.NoError(t, err)
		assert.Empty(t, req.Header.Get("Authorization"))
	})
}

func TestClient_ParseToolResult(t *testing.T) {
	client := NewClient(MCPClientConfig{})
	defer client.Close()

	t.Run("Parse string result", func(t *testing.T) {
		result := "test result"
		toolResult := &ToolCallResult{}

		err := client.parseToolResult(result, toolResult)
		assert.NoError(t, err)
		assert.Equal(t, "test result", toolResult.Content)
	})

	t.Run("Parse map result", func(t *testing.T) {
		result := map[string]interface{}{
			"content": "structured content",
			"structuredContent": map[string]interface{}{"key": "value"},
		}
		toolResult := &ToolCallResult{}

		err := client.parseToolResult(result, toolResult)
		assert.NoError(t, err)
		assert.Equal(t, "structured content", toolResult.Content)
		assert.NotNil(t, toolResult.StructuredContent)
	})

	t.Run("Parse result with content array", func(t *testing.T) {
		result := map[string]interface{}{
			"content": "main content",
			"contentArray": []interface{}{
				map[string]interface{}{
					"type":    "text",
					"content": "item1",
				},
				map[string]interface{}{
					"type": "data",
					"data": map[string]interface{}{"key": "value"},
				},
			},
		}
		toolResult := &ToolCallResult{}

		err := client.parseToolResult(result, toolResult)
		assert.NoError(t, err)
		assert.Equal(t, "main content", toolResult.Content)
		assert.Len(t, toolResult.ContentArray, 2)
	})

	t.Run("Parse invalid result type", func(t *testing.T) {
		result := []interface{}{"array", "not", "supported"}
		toolResult := &ToolCallResult{}

		err := client.parseToolResult(result, toolResult)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected result type")
	})
}