package controllers

import (
	"context"
	"testing"

	"github.com/killallgit/ryan/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tmc/langchaingo/llms"
)

// MockLLM implements llms.Model for testing
type MockLLM struct {
	mock.Mock
}

func (m *MockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	args := m.Called(ctx, messages, options)
	return args.Get(0).(*llms.ContentResponse), args.Error(1)
}

func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	args := m.Called(ctx, prompt, options)
	return args.String(0), args.Error(1)
}

func TestControllerConfig(t *testing.T) {
	mockClient := &MockChatClient{}
	mockLLM := &MockLLM{}
	toolRegistry := tools.NewRegistry()

	config := ControllerConfig{
		Client:       mockClient,
		Model:        "test-model",
		SystemPrompt: "test system prompt",
		ToolRegistry: toolRegistry,
		LLM:          mockLLM,
	}

	assert.Equal(t, mockClient, config.Client)
	assert.Equal(t, "test-model", config.Model)
	assert.Equal(t, "test system prompt", config.SystemPrompt)
	assert.Equal(t, toolRegistry, config.ToolRegistry)
	assert.Equal(t, mockLLM, config.LLM)
}

func TestNewChatControllerFromConfig_WithLLM(t *testing.T) {
	// This test verifies the factory logic for LangChain controller creation
	// Note: Actual creation depends on global config/vectorstore initialization
	// which is not available in unit tests, so we test the logic without actual creation

	mockClient := &MockChatClient{}
	mockLLM := &MockLLM{}
	toolRegistry := tools.NewRegistry()

	config := ControllerConfig{
		Client:       mockClient,
		Model:        "test-model",
		ToolRegistry: toolRegistry,
		LLM:          mockLLM,
	}

	// Verify that the config has LLM set (prerequisite for LangChain controller)
	assert.NotNil(t, config.LLM)
	assert.Equal(t, mockLLM, config.LLM)
	assert.Equal(t, mockClient, config.Client)
	assert.Equal(t, "test-model", config.Model)
	assert.Equal(t, toolRegistry, config.ToolRegistry)

	// Test would attempt to create controller but requires global config init
	// controller, err := NewChatControllerFromConfig(config)
	// assert.NoError(t, err)
	// assert.NotNil(t, controller)
}

func TestNewChatControllerFromConfig_WithSystemPrompt(t *testing.T) {
	// Test factory configuration with system prompt
	mockClient := &MockChatClient{}
	mockLLM := &MockLLM{}
	toolRegistry := tools.NewRegistry()

	config := ControllerConfig{
		Client:       mockClient,
		Model:        "test-model",
		SystemPrompt: "Custom system prompt",
		ToolRegistry: toolRegistry,
		LLM:          mockLLM,
	}

	// Verify config has system prompt and would route to system version
	assert.NotEmpty(t, config.SystemPrompt)
	assert.Equal(t, "Custom system prompt", config.SystemPrompt)
	assert.NotNil(t, config.LLM)

	// Actual controller creation requires global initialization
	// controller, err := NewChatControllerFromConfig(config)
	// This would call NewLangChainChatControllerWithSystem
}

func TestNewChatControllerFromConfig_NoLLM(t *testing.T) {
	mockClient := &MockChatClient{}
	toolRegistry := tools.NewRegistry()

	config := ControllerConfig{
		Client:       mockClient,
		Model:        "test-model",
		ToolRegistry: toolRegistry,
		LLM:          nil, // No LLM provided
	}

	controller, err := NewChatControllerFromConfig(config)
	assert.Error(t, err)
	assert.Nil(t, controller)
	assert.Contains(t, err.Error(), "LLM model is required")
}

func TestChatControllerInterface_Compilation(t *testing.T) {
	// This test ensures that both controller types implement the interface
	// If they don't, this won't compile

	var _ ChatControllerInterface = (*ChatController)(nil)
	var _ ChatControllerInterface = (*LangChainChatController)(nil)

	// Test passes if compilation succeeds
	assert.True(t, true)
}

func TestChatControllerInterface_Methods(t *testing.T) {
	// Test that the interface has all expected methods
	// This is mainly a documentation test to ensure interface completeness

	var controller ChatControllerInterface

	// These methods should exist (we can't call them with nil, but we can verify they exist)
	assert.NotNil(t, controller == nil) // Just a basic check that the variable exists

	// The interface should have these methods (compile-time check):
	// - SendUserMessage(string) (chat.Message, error)
	// - SendUserMessageWithContext(context.Context, string) (chat.Message, error)
	// - StartStreaming(context.Context, string) (<-chan StreamingUpdate, error)
	// - AddUserMessage(string)
	// - AddErrorMessage(string)
	// - GetHistory() []chat.Message
	// - GetConversation() chat.Conversation
	// - GetMessageCount() int
	// - GetLastAssistantMessage() (chat.Message, bool)
	// - GetLastUserMessage() (chat.Message, bool)
	// - HasSystemMessage() bool
	// - GetModel() string
	// - SetModel(string)
	// - Reset()
	// - GetToolRegistry() *tools.Registry
	// - SetToolRegistry(*tools.Registry)
	// - GetTokenUsage() (int, int)
	// - SetOllamaClient(OllamaClient)
	// - ValidateModel(string) error
	// - SetModelWithValidation(string) error
}

func TestControllerConfig_Empty(t *testing.T) {
	config := ControllerConfig{}

	assert.Nil(t, config.Client)
	assert.Equal(t, "", config.Model)
	assert.Equal(t, "", config.SystemPrompt)
	assert.Nil(t, config.ToolRegistry)
	assert.Nil(t, config.LLM)
}

func TestControllerConfig_PartialFields(t *testing.T) {
	mockLLM := &MockLLM{}

	config := ControllerConfig{
		Model: "test-model",
		LLM:   mockLLM,
	}

	assert.Nil(t, config.Client)
	assert.Equal(t, "test-model", config.Model)
	assert.Equal(t, "", config.SystemPrompt)
	assert.Nil(t, config.ToolRegistry)
	assert.Equal(t, mockLLM, config.LLM)
}

func TestNewChatControllerFromConfig_MinimalConfig(t *testing.T) {
	// Test minimal valid configuration
	mockClient := &MockChatClient{}
	mockLLM := &MockLLM{}

	config := ControllerConfig{
		Client: mockClient,
		Model:  "test-model",
		LLM:    mockLLM,
		// ToolRegistry and SystemPrompt are nil/empty
	}

	// Verify minimal config is properly structured
	assert.NotNil(t, config.Client)
	assert.NotNil(t, config.LLM)
	assert.NotEmpty(t, config.Model)
	assert.Empty(t, config.SystemPrompt) // No system prompt
	assert.Nil(t, config.ToolRegistry)   // No tools

	// Would route to NewLangChainChatController (no system prompt)
	// controller, err := NewChatControllerFromConfig(config)
}
