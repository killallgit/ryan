package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/killallgit/ryan/pkg/vectorstore"
	"github.com/tmc/langchaingo/llms"
)

// LangChainChatController extends ChatController with LangChain memory integration
type LangChainChatController struct {
	*ChatController
	memory       *chat.LangChainVectorMemory
	hybridMemory *chat.HybridMemory
	llm          llms.Model
	useHybrid    bool
}

// createHybridMemory initializes hybrid memory with the global vector store
func createHybridMemory() (*chat.HybridMemory, *chat.LangChainVectorMemory, error) {
	log := logger.WithComponent("langchain_controller")

	// Get global vector store manager
	manager, err := vectorstore.GetGlobalManager()
	if err != nil {
		log.Warn("Failed to get vector store manager, falling back to vector memory only", "error", err)
		return nil, createFallbackVectorMemory(), nil
	}

	if manager == nil {
		log.Info("Vector store is disabled, using vector memory only")
		return nil, createFallbackVectorMemory(), nil
	}

	// Create hybrid memory with default configuration
	config := chat.DefaultHybridMemoryConfig()
	hybridMemory, err := chat.NewHybridMemory(manager, config)
	if err != nil {
		log.Error("Failed to create hybrid memory, falling back to vector memory", "error", err)
		return nil, createFallbackVectorMemory(), nil
	}

	log.Info("Successfully initialized hybrid memory for chat",
		"working_size", config.WorkingMemorySize,
		"vector_collection", config.VectorConfig.CollectionName)
	return hybridMemory, nil, nil
}

// createFallbackVectorMemory creates a fallback vector memory when hybrid fails
func createFallbackVectorMemory() *chat.LangChainVectorMemory {
	regularMemory := chat.NewLangChainMemory()
	return &chat.LangChainVectorMemory{
		LangChainMemory: regularMemory,
	}
}

// NewLangChainChatController creates a new controller with LangChain memory
func NewLangChainChatController(client chat.ChatClient, llm llms.Model, model string, toolRegistry *tools.Registry) (*LangChainChatController, error) {
	// Create base controller
	baseController := NewChatController(client, model, toolRegistry)

	// Try to create hybrid memory, fall back to vector memory
	hybridMemory, vectorMemory, err := createHybridMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to create memory system: %w", err)
	}

	// Create the controller
	lcc := &LangChainChatController{
		ChatController: baseController,
		memory:         vectorMemory,
		hybridMemory:   hybridMemory,
		llm:            llm,
		useHybrid:      hybridMemory != nil,
	}

	return lcc, nil
}

// NewLangChainChatControllerWithSystem creates a new controller with system prompt and LangChain memory
func NewLangChainChatControllerWithSystem(client chat.ChatClient, llm llms.Model, model, systemPrompt string, toolRegistry *tools.Registry) (*LangChainChatController, error) {
	// Create base controller with system prompt
	baseController := NewChatControllerWithSystem(client, model, systemPrompt, toolRegistry)

	// Try to create hybrid memory, fall back to vector memory
	hybridMemory, vectorMemory, err := createHybridMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to create memory system: %w", err)
	}

	ctx := context.Background()
	systemMsg := chat.NewSystemMessage(systemPrompt)

	// Add system message to appropriate memory
	if hybridMemory != nil {
		if err := hybridMemory.AddMessage(ctx, systemMsg); err != nil {
			return nil, fmt.Errorf("failed to add system message to hybrid memory: %w", err)
		}
	} else if vectorMemory != nil {
		if err := vectorMemory.AddMessage(ctx, systemMsg); err != nil {
			return nil, fmt.Errorf("failed to add system message to vector memory: %w", err)
		}
	}

	// Create the controller
	lcc := &LangChainChatController{
		ChatController: baseController,
		memory:         vectorMemory,
		hybridMemory:   hybridMemory,
		llm:            llm,
		useHybrid:      hybridMemory != nil,
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

	// Add user message to appropriate memory system
	userMsg := chat.NewUserMessage(content)
	if lcc.useHybrid {
		if err := lcc.hybridMemory.AddMessage(ctx, userMsg); err != nil {
			return chat.Message{}, fmt.Errorf("failed to add user message to hybrid memory: %w", err)
		}
		// Update base controller's conversation from hybrid memory
		lcc.conversation = lcc.hybridMemory.GetConversation()
	} else {
		if err := lcc.memory.AddMessage(ctx, userMsg); err != nil {
			return chat.Message{}, fmt.Errorf("failed to add user message to memory: %w", err)
		}
		// Update base controller's conversation from vector memory
		lcc.conversation = lcc.memory.GetConversation()
	}

	// For now, use the base controller implementation with enhanced memory
	// This provides tool support while leveraging LangChain memory
	log.Debug("Using base controller with LangChain memory enhancement")
	return lcc.ChatController.SendUserMessageWithContext(ctx, content)
}

// GetHistory returns the chat history from memory
func (lcc *LangChainChatController) GetHistory() []chat.Message {
	if lcc.useHybrid {
		return chat.GetMessages(lcc.hybridMemory.GetConversation())
	}
	return chat.GetMessages(lcc.memory.GetConversation())
}

// GetConversation returns the current conversation from memory
func (lcc *LangChainChatController) GetConversation() chat.Conversation {
	if lcc.useHybrid {
		return lcc.hybridMemory.GetConversation()
	}
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

	// Clear appropriate memory system
	if lcc.useHybrid {
		if err := lcc.hybridMemory.Clear(ctx); err != nil {
			log := logger.WithComponent("langchain_chat_controller")
			log.Error("Failed to clear hybrid memory", "error", err)
		}
	} else {
		if err := lcc.memory.Clear(ctx); err != nil {
			log := logger.WithComponent("langchain_chat_controller")
			log.Error("Failed to clear memory", "error", err)
		}
	}

	// Re-add system prompt if it existed
	if systemPrompt != "" {
		if lcc.useHybrid {
			if err := lcc.hybridMemory.AddMessage(ctx, chat.NewSystemMessage(systemPrompt)); err != nil {
				log := logger.WithComponent("langchain_chat_controller")
				log.Error("Failed to re-add system message to hybrid memory", "error", err)
			}
		} else {
			if err := lcc.memory.AddMessage(ctx, chat.NewSystemMessage(systemPrompt)); err != nil {
				log := logger.WithComponent("langchain_chat_controller")
				log.Error("Failed to re-add system message", "error", err)
			}
		}
	}

	// Reset base controller
	lcc.ChatController.Reset()

	// Update conversation from appropriate memory
	if lcc.useHybrid {
		lcc.conversation = lcc.hybridMemory.GetConversation()
	} else {
		lcc.conversation = lcc.memory.GetConversation()
	}
}

// AddUserMessage adds a user message to memory (for optimistic updates)
func (lcc *LangChainChatController) AddUserMessage(content string) {
	if strings.TrimSpace(content) == "" {
		return
	}

	ctx := context.Background()
	userMsg := chat.NewUserMessage(content)

	// Add to appropriate memory system
	if lcc.useHybrid {
		if err := lcc.hybridMemory.AddMessage(ctx, userMsg); err != nil {
			log := logger.WithComponent("langchain_chat_controller")
			log.Error("Failed to add user message to hybrid memory", "error", err)
			return
		}
		// Update conversations from hybrid memory
		lcc.conversation = lcc.hybridMemory.GetConversation()
	} else {
		if err := lcc.memory.AddMessage(ctx, userMsg); err != nil {
			log := logger.WithComponent("langchain_chat_controller")
			log.Error("Failed to add user message to memory", "error", err)
			return
		}
		// Update conversations from vector memory
		lcc.conversation = lcc.memory.GetConversation()
	}

	lcc.ChatController.conversation = lcc.conversation
}

// AddErrorMessage adds an error message to memory
func (lcc *LangChainChatController) AddErrorMessage(errorMsg string) {
	ctx := context.Background()
	errMsg := chat.NewErrorMessage(errorMsg)

	// Add to appropriate memory system
	if lcc.useHybrid {
		if err := lcc.hybridMemory.AddMessage(ctx, errMsg); err != nil {
			log := logger.WithComponent("langchain_chat_controller")
			log.Error("Failed to add error message to hybrid memory", "error", err)
			return
		}
		// Update conversations from hybrid memory
		lcc.conversation = lcc.hybridMemory.GetConversation()
	} else {
		if err := lcc.memory.AddMessage(ctx, errMsg); err != nil {
			log := logger.WithComponent("langchain_chat_controller")
			log.Error("Failed to add error message to memory", "error", err)
			return
		}
		// Update conversations from vector memory
		lcc.conversation = lcc.memory.GetConversation()
	}

	lcc.ChatController.conversation = lcc.conversation
}

// GetMemory returns the underlying LangChain vector memory for advanced usage
func (lcc *LangChainChatController) GetMemory() *chat.LangChainVectorMemory {
	return lcc.memory
}

// GetHybridMemory returns the hybrid memory system if available
func (lcc *LangChainChatController) GetHybridMemory() *chat.HybridMemory {
	return lcc.hybridMemory
}

// IsUsingHybridMemory returns true if hybrid memory is active
func (lcc *LangChainChatController) IsUsingHybridMemory() bool {
	return lcc.useHybrid
}
