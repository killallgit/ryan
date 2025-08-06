package cmd

import (
	"context"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/tools"
)

// LangChainControllerAdapter adapts LangChainController to both Controller and TUI ControllerInterface
type LangChainControllerAdapter struct {
	*controllers.LangChainController
}

// NativeControllerAdapter adapts NativeController to both Controller and TUI ControllerInterface
type NativeControllerAdapter struct {
	*controllers.NativeController
}

// Implement any missing methods needed by the interface
func (lca *LangChainControllerAdapter) StartStreaming(ctx context.Context, content string) (<-chan controllers.StreamingUpdate, error) {
	return lca.LangChainController.StartStreaming(ctx, content)
}

// SetOllamaClient accepts any type to satisfy tui.ControllerInterface
func (lca *LangChainControllerAdapter) SetOllamaClient(client any) {
	// LangChainController also expects any type, so we can pass it directly
	lca.LangChainController.SetOllamaClient(client)
}

func (lca *LangChainControllerAdapter) ValidateModel(model string) error {
	return lca.LangChainController.ValidateModel(model)
}

func (lca *LangChainControllerAdapter) GetTokenUsage() (promptTokens, responseTokens int) {
	return lca.LangChainController.GetTokenUsage()
}

func (lca *LangChainControllerAdapter) CleanThinkingBlocks() {
	lca.LangChainController.CleanThinkingBlocks()
}

// GetLastAssistantMessage returns the last assistant message from the conversation
func (lca *LangChainControllerAdapter) GetLastAssistantMessage() (chat.Message, bool) {
	history := lca.GetHistory()
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			return history[i], true
		}
	}
	return chat.Message{}, false
}

// GetLastUserMessage returns the last user message from the conversation
func (lca *LangChainControllerAdapter) GetLastUserMessage() (chat.Message, bool) {
	history := lca.GetHistory()
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "human" {
			return history[i], true
		}
	}
	return chat.Message{}, false
}

// GetMessageCount returns the number of messages in the conversation
func (lca *LangChainControllerAdapter) GetMessageCount() int {
	return len(lca.GetHistory())
}

// HasSystemMessage returns true if the conversation has a system message
func (lca *LangChainControllerAdapter) HasSystemMessage() bool {
	history := lca.GetHistory()
	for _, msg := range history {
		if msg.Role == "system" {
			return true
		}
	}
	return false
}

// SetModelWithValidation sets the model after validating it
func (lca *LangChainControllerAdapter) SetModelWithValidation(model string) error {
	if err := lca.ValidateModel(model); err != nil {
		return err
	}
	lca.SetModel(model)
	return nil
}

// SetToolRegistry sets the tool registry
func (lca *LangChainControllerAdapter) SetToolRegistry(registry *tools.Registry) {
	// LangChainController doesn't have SetToolRegistry method
	// The tool registry is set during construction via NewLangChainController
	// This is a no-op for compatibility with the interface
}

// AddAssistantMessage adds an assistant message (for compatibility)
func (lca *LangChainControllerAdapter) AddAssistantMessage(content string) {
	// LangChainController handles this internally during streaming
	// This is a no-op for compatibility with the interface
}

// Native Controller Adapter Methods
// Implement the TUI interface methods for NativeControllerAdapter

// StartStreaming implements streaming for native controller
func (nca *NativeControllerAdapter) StartStreaming(ctx context.Context, content string) (<-chan controllers.StreamingUpdate, error) {
	return nca.NativeController.StartStreaming(ctx, content)
}

// SetOllamaClient accepts any type to satisfy tui.ControllerInterface
func (nca *NativeControllerAdapter) SetOllamaClient(client any) {
	nca.NativeController.SetOllamaClient(client)
}

func (nca *NativeControllerAdapter) ValidateModel(model string) error {
	return nca.NativeController.ValidateModel(model)
}

func (nca *NativeControllerAdapter) GetTokenUsage() (promptTokens, responseTokens int) {
	return nca.NativeController.GetTokenUsage()
}

func (nca *NativeControllerAdapter) CleanThinkingBlocks() {
	// Native controller doesn't need thinking block cleaning like LangChain
	// This is a no-op for compatibility with the TUI interface
}

// GetLastAssistantMessage returns the last assistant message from the conversation
func (nca *NativeControllerAdapter) GetLastAssistantMessage() (chat.Message, bool) {
	return nca.NativeController.GetLastAssistantMessage()
}

// GetLastUserMessage returns the last user message from the conversation
func (nca *NativeControllerAdapter) GetLastUserMessage() (chat.Message, bool) {
	return nca.NativeController.GetLastUserMessage()
}

// GetMessageCount returns the number of messages in the conversation
func (nca *NativeControllerAdapter) GetMessageCount() int {
	return nca.NativeController.GetMessageCount()
}

// HasSystemMessage returns true if the conversation has a system message
func (nca *NativeControllerAdapter) HasSystemMessage() bool {
	return nca.NativeController.HasSystemMessage()
}

// SetModelWithValidation sets the model after validating it
func (nca *NativeControllerAdapter) SetModelWithValidation(model string) error {
	return nca.NativeController.SetModelWithValidation(model)
}

// SetToolRegistry sets the tool registry
func (nca *NativeControllerAdapter) SetToolRegistry(registry *tools.Registry) {
	nca.NativeController.SetToolRegistry(registry)
}

// AddAssistantMessage adds an assistant message
func (nca *NativeControllerAdapter) AddAssistantMessage(content string) {
	nca.NativeController.AddAssistantMessage(content)
}
