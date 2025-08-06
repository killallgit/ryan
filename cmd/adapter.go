package cmd

import (
	"context"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/tools"
)

// LangChainControllerAdapter adapts LangChainController to both ChatControllerInterface and TUI ControllerInterface
type LangChainControllerAdapter struct {
	*controllers.LangChainController
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
