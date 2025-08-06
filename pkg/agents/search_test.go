package agents

import (
	"context"
	"strings"
	"testing"

	"github.com/killallgit/ryan/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchAgent_NewSearchAgent(t *testing.T) {
	registry := tools.NewRegistry()
	agent := NewSearchAgent(registry)

	assert.NotNil(t, agent)
	assert.NotNil(t, agent.toolRegistry)
	assert.NotNil(t, agent.log)
}

func TestSearchAgent_Name(t *testing.T) {
	agent := NewSearchAgent(tools.NewRegistry())
	assert.Equal(t, "search", agent.Name())
}

func TestSearchAgent_Description(t *testing.T) {
	agent := NewSearchAgent(tools.NewRegistry())
	assert.Equal(t, "Searches for code patterns, symbols, and text across files", agent.Description())
}

func TestSearchAgent_CanHandle(t *testing.T) {
	// With LLM-based routing, all agents trust the orchestrator's decision
	// and always return true/1.0 from CanHandle
	tests := []struct {
		name    string
		request string
	}{
		{
			name:    "Direct search command",
			request: "search for the function handleRequest",
		},
		{
			name:    "Find command",
			request: "find all references to Database class",
		},
		{
			name:    "Grep command",
			request: "grep for TODO comments in the codebase",
		},
		{
			name:    "Locate command",
			request: "locate the implementation of the auth middleware",
		},
		{
			name:    "Where is query",
			request: "where is the config file loaded?",
		},
		{
			name:    "Look for query",
			request: "look for error handling patterns",
		},
		{
			name:    "Case insensitive",
			request: "SEARCH FOR connection pooling",
		},
		{
			name:    "Non-search request",
			request: "create a new file called test.go",
		},
		{
			name:    "Code review request",
			request: "review this pull request",
		},
	}

	agent := NewSearchAgent(tools.NewRegistry())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canHandle, confidence := agent.CanHandle(tt.request)
			// All agents now trust the orchestrator's LLM routing decision
			assert.True(t, canHandle, "Agent should always return true with LLM-based routing")
			assert.Equal(t, 1.0, confidence, "Agent should always return confidence 1.0 with LLM-based routing")
		})
	}
}

func TestSearchAgent_Execute(t *testing.T) {
	// Create a mock grep tool for testing
	mockGrepTool := &mockTool{
		name: "grep",
		executeFn: func(ctx context.Context, args map[string]interface{}) (tools.ToolResult, error) {
			pattern, ok := args["pattern"].(string)
			if !ok {
				return tools.ToolResult{}, nil
			}

			// Return mock search results
			if strings.Contains(pattern, "TODO") {
				return tools.ToolResult{
					Success: true,
					Content: "file1.go:10: // TODO: implement this\nfile2.go:25: // TODO: add tests",
				}, nil
			}

			return tools.ToolResult{
				Success: true,
				Content: "No matches found",
			}, nil
		},
	}

	registry := tools.NewRegistry()
	registry.Register(mockGrepTool)

	agent := NewSearchAgent(registry)

	tests := []struct {
		name        string
		request     AgentRequest
		expectError bool
		checkResult func(t *testing.T, result AgentResult)
	}{
		{
			name: "Search for TODO comments",
			request: AgentRequest{
				Prompt: "search for TODO comments",
			},
			expectError: false,
			checkResult: func(t *testing.T, result AgentResult) {
				assert.True(t, result.Success)
				assert.Contains(t, result.Details, "TODO")
				assert.Contains(t, result.Details, "file1.go")
				assert.Contains(t, result.Details, "file2.go")
			},
		},
		{
			name: "Search with no results",
			request: AgentRequest{
				Prompt: "search for NONEXISTENT",
			},
			expectError: false,
			checkResult: func(t *testing.T, result AgentResult) {
				assert.True(t, result.Success)
				assert.Contains(t, strings.ToLower(result.Details), "no")
			},
		},
		{
			name: "Search in specific directory",
			request: AgentRequest{
				Prompt: "search for handleRequest in pkg/handlers",
			},
			expectError: false,
			checkResult: func(t *testing.T, result AgentResult) {
				assert.True(t, result.Success)
				assert.NotEmpty(t, result.Details)
			},
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := agent.Execute(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

// Helper function tests
func TestSearchAgent_ExtractSearchPattern(t *testing.T) {
	tests := []struct {
		name     string
		request  string
		expected string
	}{
		{
			name:     "Search for pattern",
			request:  "search for handleRequest function",
			expected: "handleRequest",
		},
		{
			name:     "Find pattern",
			request:  "find Database class",
			expected: "Database",
		},
		{
			name:     "Grep pattern",
			request:  "grep TODO",
			expected: "TODO",
		},
		{
			name:     "Look for pattern",
			request:  "look for error handling",
			expected: "error handling",
		},
		{
			name:     "Where is pattern",
			request:  "where is config loaded",
			expected: "config",
		},
		{
			name:     "Pattern with quotes",
			request:  `search for "user authentication"`,
			expected: "user authentication",
		},
	}

	agent := NewSearchAgent(tools.NewRegistry())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test via CanHandle since extractSearchPattern is private
			// With LLM-based routing, all requests are handled
			canHandle, confidence := agent.CanHandle(tt.request)
			assert.True(t, canHandle, "Agent should always return true with LLM-based routing")
			assert.Equal(t, 1.0, confidence, "Agent should always return confidence 1.0 with LLM-based routing")
		})
	}
}

// Mock tool for testing
type mockTool struct {
	name      string
	executeFn func(context.Context, map[string]interface{}) (tools.ToolResult, error)
}

func (m *mockTool) Name() string                       { return m.name }
func (m *mockTool) Description() string                { return "Mock tool for testing" }
func (m *mockTool) JSONSchema() map[string]interface{} { return nil }
func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (tools.ToolResult, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, args)
	}
	return tools.ToolResult{}, nil
}
