package controllers

import (
	"context"
	"fmt"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/tmc/langchaingo/llms"
)

// ControllerConfig holds configuration for creating controllers
type ControllerConfig struct {
	Client       chat.ChatClient
	Model        string
	SystemPrompt string
	ToolRegistry *tools.Registry
	LLM          llms.Model // LLM model for LangChain controller
}

// NewChatControllerFromConfig creates a LangChain chat controller based on configuration
func NewChatControllerFromConfig(cfg ControllerConfig) (Controller, error) {
	if cfg.LLM == nil {
		return nil, fmt.Errorf("LLM model is required for LangChain controller")
	}

	if cfg.SystemPrompt != "" {
		return NewLangChainChatControllerWithSystem(cfg.Client, cfg.LLM, cfg.Model, cfg.SystemPrompt, cfg.ToolRegistry)
	}
	return NewLangChainChatController(cfg.Client, cfg.LLM, cfg.Model, cfg.ToolRegistry)
}

// Controller defines the common interface for all chat controllers
type Controller interface {
	SendUserMessage(content string) (chat.Message, error)
	SendUserMessageWithContext(ctx context.Context, content string) (chat.Message, error)
	StartStreaming(ctx context.Context, content string) (<-chan StreamingUpdate, error)
	AddUserMessage(content string)
	AddErrorMessage(errorMsg string)
	GetHistory() []chat.Message
	GetConversation() chat.Conversation
	GetMessageCount() int
	GetLastAssistantMessage() (chat.Message, bool)
	GetLastUserMessage() (chat.Message, bool)
	HasSystemMessage() bool
	GetModel() string
	SetModel(model string)
	Reset()
	GetToolRegistry() *tools.Registry
	SetToolRegistry(registry *tools.Registry)
	GetTokenUsage() (promptTokens, responseTokens int)
	SetOllamaClient(client any)
	ValidateModel(model string) error
	SetModelWithValidation(model string) error
}

// Ensure all controllers implement the interface
var _ Controller = (*ChatController)(nil)
var _ Controller = (*LangChainChatController)(nil)
var _ Controller = (*NativeController)(nil)
