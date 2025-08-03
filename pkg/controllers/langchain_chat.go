package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/tmc/langchaingo/llms"
)

// LangChainChatController extends ChatController with LangChain memory integration
type LangChainChatController struct {
	*ChatController
	memory *chat.LangChainMemory
	llm    llms.Model
}

// NewLangChainChatController creates a new controller with LangChain memory
func NewLangChainChatController(client chat.ChatClient, llm llms.Model, model string, toolRegistry *tools.Registry) (*LangChainChatController, error) {
	// Create base controller
	baseController := NewChatController(client, model, toolRegistry)

	// Create LangChain memory
	memory := chat.NewLangChainMemory()

	// Create the controller
	lcc := &LangChainChatController{
		ChatController: baseController,
		memory:         memory,
		llm:            llm,
	}

	return lcc, nil
}

// NewLangChainChatControllerWithSystem creates a new controller with system prompt and LangChain memory
func NewLangChainChatControllerWithSystem(client chat.ChatClient, llm llms.Model, model, systemPrompt string, toolRegistry *tools.Registry) (*LangChainChatController, error) {
	// Create base controller with system prompt
	baseController := NewChatControllerWithSystem(client, model, systemPrompt, toolRegistry)

	// Create LangChain memory and add system message
	memory := chat.NewLangChainMemory()
	ctx := context.Background()
	if err := memory.AddMessage(ctx, chat.NewSystemMessage(systemPrompt)); err != nil {
		return nil, fmt.Errorf("failed to add system message to memory: %w", err)
	}

	// Create the controller
	lcc := &LangChainChatController{
		ChatController: baseController,
		memory:         memory,
		llm:            llm,
	}

	return lcc, nil
}

// SendUserMessage sends a user message with enhanced LangChain memory
func (lcc *LangChainChatController) SendUserMessage(content string) (chat.Message, error) {
	if strings.TrimSpace(content) == "" {
		return chat.Message{}, fmt.Errorf("message content cannot be empty")
	}

	return lcc.SendUserMessageWithContext(context.Background(), content)
}

// SendUserMessageWithContext sends a user message with context using enhanced memory
func (lcc *LangChainChatController) SendUserMessageWithContext(ctx context.Context, content string) (chat.Message, error) {
	if strings.TrimSpace(content) == "" {
		return chat.Message{}, fmt.Errorf("message content cannot be empty")
	}

	log := logger.WithComponent("langchain_chat_controller")

	// Add user message to memory
	userMsg := chat.NewUserMessage(content)
	if err := lcc.memory.AddMessage(ctx, userMsg); err != nil {
		return chat.Message{}, fmt.Errorf("failed to add user message to memory: %w", err)
	}

	// Update base controller's conversation
	lcc.conversation = lcc.memory.GetConversation()

	// For now, use the base controller implementation with enhanced memory
	// This provides tool support while leveraging LangChain memory
	log.Debug("Using base controller with LangChain memory enhancement")
	return lcc.ChatController.SendUserMessageWithContext(ctx, content)
}

// GetHistory returns the chat history from memory
func (lcc *LangChainChatController) GetHistory() []chat.Message {
	return chat.GetMessages(lcc.memory.GetConversation())
}

// GetConversation returns the current conversation from memory
func (lcc *LangChainChatController) GetConversation() chat.Conversation {
	return lcc.memory.GetConversation()
}

// Reset clears the memory and resets the conversation
func (lcc *LangChainChatController) Reset() {
	ctx := context.Background()

	// Get system prompt if exists
	systemPrompt := ""
	if chat.HasSystemMessage(lcc.conversation) {
		messages := chat.GetMessagesByRole(lcc.conversation, chat.RoleSystem)
		if len(messages) > 0 {
			systemPrompt = messages[0].Content
		}
	}

	// Clear memory
	if err := lcc.memory.Clear(ctx); err != nil {
		log := logger.WithComponent("langchain_chat_controller")
		log.Error("Failed to clear memory", "error", err)
	}

	// Re-add system prompt if it existed
	if systemPrompt != "" {
		if err := lcc.memory.AddMessage(ctx, chat.NewSystemMessage(systemPrompt)); err != nil {
			log := logger.WithComponent("langchain_chat_controller")
			log.Error("Failed to re-add system message", "error", err)
		}
	}

	// Reset base controller
	lcc.ChatController.Reset()

	// Update conversation from memory
	lcc.conversation = lcc.memory.GetConversation()
}

// AddUserMessage adds a user message to memory (for optimistic updates)
func (lcc *LangChainChatController) AddUserMessage(content string) {
	if strings.TrimSpace(content) == "" {
		return
	}

	ctx := context.Background()
	userMsg := chat.NewUserMessage(content)

	// Add to memory
	if err := lcc.memory.AddMessage(ctx, userMsg); err != nil {
		log := logger.WithComponent("langchain_chat_controller")
		log.Error("Failed to add user message to memory", "error", err)
		return
	}

	// Update conversations
	lcc.conversation = lcc.memory.GetConversation()
	lcc.ChatController.conversation = lcc.conversation
}

// AddErrorMessage adds an error message to memory
func (lcc *LangChainChatController) AddErrorMessage(errorMsg string) {
	ctx := context.Background()
	errMsg := chat.NewErrorMessage(errorMsg)

	// Add to memory
	if err := lcc.memory.AddMessage(ctx, errMsg); err != nil {
		log := logger.WithComponent("langchain_chat_controller")
		log.Error("Failed to add error message to memory", "error", err)
		return
	}

	// Update conversations
	lcc.conversation = lcc.memory.GetConversation()
	lcc.ChatController.conversation = lcc.conversation
}

// GetMemory returns the underlying LangChain memory for advanced usage
func (lcc *LangChainChatController) GetMemory() *chat.LangChainMemory {
	return lcc.memory
}