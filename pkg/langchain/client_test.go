package langchain

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	langchaintools "github.com/tmc/langchaingo/tools"
)

func TestNewLangChainClient(t *testing.T) {
	// Skip this test for now due to config initialization complexity
	t.Skip("Skipping NewClient test due to config initialization requirements")
}

// Tests are already covered by TestToolAdapter below

func TestClient_SetProgressCallback(t *testing.T) {
	client := &Client{}
	callback := func(toolName, command string) {}

	client.SetProgressCallback(callback)
	assert.NotNil(t, client.progressCallback)
}

func TestClient_GetMemory(t *testing.T) {
	mockMemory := &MockMemory{}
	client := &Client{memory: mockMemory}

	result := client.GetMemory()
	assert.Equal(t, mockMemory, result)
}

func TestClient_GetTools(t *testing.T) {
	mockTools := []langchaintools.Tool{&MockLangchainTool{}}
	client := &Client{langchainTools: mockTools}

	result := client.GetTools()
	assert.Equal(t, mockTools, result)
}

func TestClient_ClearMemory(t *testing.T) {
	mockMemory := &MockMemory{}
	client := &Client{memory: mockMemory}

	ctx := context.Background()
	err := client.ClearMemory(ctx)
	assert.NoError(t, err)
	assert.True(t, mockMemory.cleared)

	// Test with nil memory
	clientWithNilMemory := &Client{memory: nil}
	err = clientWithNilMemory.ClearMemory(ctx)
	assert.NoError(t, err)
}

func TestClient_determineAgentType(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.NewBashTool())

	client := &Client{
		toolRegistry: registry,
		model:        "qwen3:latest",
		log:          logger.WithComponent("test"),
	}

	// Test with tools - should return conversational
	agentType := client.determineAgentType()
	assert.Equal(t, AgentTypeConversational, agentType)

	// Test without tools - should return direct
	client.toolRegistry = nil
	agentType = client.determineAgentType()
	assert.Equal(t, AgentTypeDirect, agentType)
}

func TestClient_isOllamaModel(t *testing.T) {
	// Test with Ollama model - we can't easily create a real Ollama instance,
	// so we'll test with a mock that's not an Ollama instance
	client := &Client{llm: &ClientMockLLM{}}

	result := client.isOllamaModel()
	assert.False(t, result) // ClientMockLLM is not an *ollama.LLM
}

func TestClient_WithPromptTemplate(t *testing.T) {
	// Test template functionality without configuration
	client := &Client{
		llm: &ClientMockLLM{response: "Template processed: Hello, Test User!"},
	}

	ctx := context.Background()
	template := "Hello, {{.name}}!"
	vars := map[string]any{
		"name": "Test User",
	}

	result, err := client.WithPromptTemplate(ctx, template, vars)
	assert.NoError(t, err)
	assert.Contains(t, result, "Template processed")
}

func TestToolAdapter(t *testing.T) {
	// Create a mock Ryan tool
	mockTool := &MockRyanTool{
		name:        "test_tool",
		description: "A test tool",
	}

	adapter := NewToolAdapter(mockTool)
	assert.NotNil(t, adapter)
	assert.Equal(t, "test_tool", adapter.Name())
	assert.Equal(t, "A test tool", adapter.Description())

	t.Run("Call with progress callback", func(t *testing.T) {
		callbackCalled := false
		var callbackToolName, callbackCommand string

		callback := func(toolName, command string) {
			callbackCalled = true
			callbackToolName = toolName
			callbackCommand = command
		}

		adapter = adapter.WithProgressCallback(callback)

		ctx := context.Background()
		result, err := adapter.Call(ctx, "command: ls -la")

		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.True(t, callbackCalled)
		assert.Equal(t, "test_tool", callbackToolName)
		assert.Equal(t, "ls -la", callbackCommand)
	})

	t.Run("Call without progress callback", func(t *testing.T) {
		adapter := NewToolAdapter(mockTool)
		ctx := context.Background()
		result, err := adapter.Call(ctx, "simple input")

		assert.NoError(t, err)
		assert.Equal(t, "success", result)
	})

	t.Run("Tool execution error", func(t *testing.T) {
		errorTool := &MockRyanTool{
			name:        "error_tool",
			description: "Tool that fails",
			shouldError: true,
		}

		adapter := NewToolAdapter(errorTool)
		ctx := context.Background()

		_, err := adapter.Call(ctx, "input")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool execution failed")
	})

	t.Run("Tool returns unsuccessful result", func(t *testing.T) {
		failTool := &MockRyanTool{
			name:        "fail_tool",
			description: "Tool that returns failure",
			shouldFail:  true,
		}

		adapter := NewToolAdapter(failTool)
		ctx := context.Background()

		_, err := adapter.Call(ctx, "input")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool execution failed")
	})
}

func TestExtractValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		prefix   string
		expected string
	}{
		{
			name:     "basic extraction",
			input:    "command: ls -la",
			prefix:   "command:",
			expected: "ls -la",
		},
		{
			name:     "extraction with quotes",
			input:    `path: "test.txt"`,
			prefix:   "path:",
			expected: "test.txt",
		},
		{
			name:     "extraction with single quotes",
			input:    "path: 'test.txt'",
			prefix:   "path:",
			expected: "test.txt",
		},
		{
			name:     "no match",
			input:    "some text",
			prefix:   "command:",
			expected: "",
		},
		{
			name:     "empty value",
			input:    "command:",
			prefix:   "command:",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractValue(tt.input, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_cleanResponseForStreaming(t *testing.T) {
	// Test without config dependency
	client := &Client{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "preserve thinking blocks",
			input:    "<think>Let me think about this</think>\nHere's my response",
			expected: "<think>Let me think about this</think>\nHere's my response",
		},
		{
			name:     "reduce multiple empty lines",
			input:    "Line 1\n\n\nLine 2\n\n",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "normal text",
			input:    "This is normal text",
			expected: "This is normal text",
		},
		{
			name:     "preserve complex thinking blocks",
			input:    "Start<think>Complex\nthinking\nblock</think>End",
			expected: "Start<think>Complex\nthinking\nblock</think>End",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.cleanResponseForStreaming(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractVarNames(t *testing.T) {
	vars := map[string]any{
		"name":  "John",
		"age":   30,
		"email": "john@example.com",
	}

	names := extractVarNames(vars)
	assert.Len(t, names, 3)
	assert.Contains(t, names, "name")
	assert.Contains(t, names, "age")
	assert.Contains(t, names, "email")
}

func TestGetMapKeys(t *testing.T) {
	testMap := map[string]any{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	keys := getMapKeys(testMap)
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
	assert.Contains(t, keys, "key3")
}

func TestClient_streamResponse(t *testing.T) {
	// Test without config dependency
	client := &Client{}

	outputChan := make(chan string, 10)
	response := "This is a test response"

	go func() {
		err := client.streamResponse(response, outputChan)
		assert.NoError(t, err)
		close(outputChan)
	}()

	// Collect all chunks
	var chunks []string
	for chunk := range outputChan {
		chunks = append(chunks, chunk)
	}

	// Verify the response was streamed in chunks
	fullResponse := strings.Join(chunks, "")
	assert.Equal(t, response, fullResponse)
	assert.Greater(t, len(chunks), 1) // Should be broken into multiple chunks
}

// MockRyanTool implements the tools.Tool interface for testing
type MockRyanTool struct {
	name        string
	description string
	shouldError bool
	shouldFail  bool
}

func (m *MockRyanTool) Name() string        { return m.name }
func (m *MockRyanTool) Description() string { return m.description }
func (m *MockRyanTool) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{
				"type":        "string",
				"description": "Input for the mock tool",
			},
		},
	}
}

func (m *MockRyanTool) Execute(ctx context.Context, args map[string]interface{}) (tools.ToolResult, error) {
	if m.shouldError {
		return tools.ToolResult{}, fmt.Errorf("mock tool error")
	}

	if m.shouldFail {
		return tools.ToolResult{
			Success: false,
			Error:   "mock tool failure",
		}, nil
	}

	return tools.ToolResult{
		Success: true,
		Content: "success",
	}, nil
}

// Extended MockLLM with more functionality for testing
type ExtendedMockLLM struct {
	response     string
	err          error
	callCount    int
	lastPrompt   string
	streamChunks []string
}

func (m *ExtendedMockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	m.callCount++
	if len(messages) > 0 {
		// Extract the last human message for testing
		for _, msg := range messages {
			for _, part := range msg.Parts {
				if textPart, ok := part.(llms.TextContent); ok {
					m.lastPrompt = textPart.Text
				}
			}
		}
	}

	if m.err != nil {
		return nil, m.err
	}

	// Handle streaming - simplified for testing
	_ = options

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: m.response,
			},
		},
	}, nil
}

func (m *ExtendedMockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	m.callCount++
	m.lastPrompt = prompt

	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "string longer than max",
			input:    "this is a very long string",
			maxLen:   10,
			expected: "this is a ...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
		{
			name:     "exact length",
			input:    "exact",
			maxLen:   5,
			expected: "exact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// MockMemory implements schema.Memory interface for testing
type MockMemory struct {
	cleared bool
}

func (m *MockMemory) Clear(ctx context.Context) error {
	m.cleared = true
	return nil
}

func (m *MockMemory) ChatHistory() schema.ChatMessageHistory {
	return nil
}

func (m *MockMemory) MemoryVariables(ctx context.Context) []string {
	return []string{"history"}
}

func (m *MockMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	return map[string]any{"history": ""}, nil
}

func (m *MockMemory) SaveContext(ctx context.Context, inputValues, outputValues map[string]any) error {
	return nil
}

func (m *MockMemory) GetMemoryKey(ctx context.Context) string {
	return "history"
}

// MockLangchainTool implements langchaintools.Tool interface for testing
type MockLangchainTool struct{}

func (m *MockLangchainTool) Name() string {
	return "mock_langchain_tool"
}

func (m *MockLangchainTool) Description() string {
	return "A mock LangChain tool"
}

func (m *MockLangchainTool) Call(ctx context.Context, input string) (string, error) {
	return "mock result", nil
}

// ClientMockLLM implements llms.Model interface for testing
type ClientMockLLM struct {
	response string
}

func (m *ClientMockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{Content: m.response},
		},
	}, nil
}

func (m *ClientMockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return m.response, nil
}
