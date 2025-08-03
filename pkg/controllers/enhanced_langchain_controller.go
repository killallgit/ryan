package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/langchain"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
)

// EnhancedLangChainController wraps the enhanced LangChain client to work with the existing controller interface
type EnhancedLangChainController struct {
	client       *langchain.EnhancedClient
	model        string
	toolRegistry *tools.Registry
	conversation chat.Conversation
	log          *logger.Logger
}

// NewEnhancedLangChainController creates a new controller using the enhanced LangChain client
func NewEnhancedLangChainController(baseURL, model string, toolRegistry *tools.Registry) (*EnhancedLangChainController, error) {
	client, err := langchain.NewEnhancedClient(baseURL, model, toolRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create enhanced LangChain client: %w", err)
	}

	log := logger.WithComponent("enhanced_langchain_controller")

	return &EnhancedLangChainController{
		client:       client,
		model:        model,
		toolRegistry: toolRegistry,
		conversation: chat.NewConversation(model),
		log:          log,
	}, nil
}

// NewEnhancedLangChainControllerWithSystem creates a new controller with system prompt
func NewEnhancedLangChainControllerWithSystem(baseURL, model, systemPrompt string, toolRegistry *tools.Registry) (*EnhancedLangChainController, error) {
	controller, err := NewEnhancedLangChainController(baseURL, model, toolRegistry)
	if err != nil {
		return nil, err
	}

	// Add system message to conversation
	if systemPrompt != "" {
		controller.conversation = chat.AddMessage(controller.conversation, chat.NewSystemMessage(systemPrompt))
	}

	return controller, nil
}

// SendUserMessage sends a user message using the enhanced LangChain agent
func (elc *EnhancedLangChainController) SendUserMessage(content string) (chat.Message, error) {
	return elc.SendUserMessageWithContext(context.Background(), content)
}

// SendUserMessageWithContext sends a user message with context using the enhanced LangChain agent
func (elc *EnhancedLangChainController) SendUserMessageWithContext(ctx context.Context, content string) (chat.Message, error) {
	if strings.TrimSpace(content) == "" {
		return chat.Message{}, fmt.Errorf("message content cannot be empty")
	}

	elc.log.Debug("Sending user message with enhanced LangChain agent", 
		"content_length", len(content), 
		"has_tools", elc.toolRegistry != nil,
		"agent_enabled", elc.client != nil)

	// Add user message to conversation
	userMsg := chat.NewUserMessage(content)
	elc.conversation = chat.AddMessage(elc.conversation, userMsg)

	// Use the enhanced client to send the message
	response, err := elc.client.SendMessage(ctx, content)
	if err != nil {
		errorMsg := fmt.Sprintf("LangChain agent failed: %v", err)
		elc.log.Error("Enhanced LangChain client failed", "error", err)
		
		// Add error message to conversation
		errMsg := chat.NewErrorMessage(errorMsg)
		elc.conversation = chat.AddMessage(elc.conversation, errMsg)
		
		return errMsg, fmt.Errorf("failed to send message: %w", err)
	}

	// Create assistant message from response
	assistantMsg := chat.NewAssistantMessage(response)
	elc.conversation = chat.AddMessage(elc.conversation, assistantMsg)

	elc.log.Debug("Enhanced LangChain agent response received", 
		"response_length", len(response))

	return assistantMsg, nil
}

// SendUserMessageWithStreamingContext sends a message with streaming support
func (elc *EnhancedLangChainController) SendUserMessageWithStreamingContext(ctx context.Context, content string, outputChan chan<- chat.MessageChunk) (chat.Message, error) {
	if strings.TrimSpace(content) == "" {
		return chat.Message{}, fmt.Errorf("message content cannot be empty")
	}

	elc.log.Debug("Starting streaming message with enhanced LangChain agent")

	// Add user message to conversation
	userMsg := chat.NewUserMessage(content)
	elc.conversation = chat.AddMessage(elc.conversation, userMsg)

	// Create a channel to collect streamed content
	streamChan := make(chan string, 100)
	var fullResponse strings.Builder

	// Start streaming in a goroutine
	go func() {
		defer close(streamChan)
		if err := elc.client.StreamMessage(ctx, content, streamChan); err != nil {
			elc.log.Error("Streaming failed", "error", err)
			// Send error through output channel
			select {
			case outputChan <- chat.MessageChunk{Content: fmt.Sprintf("Error: %v", err), Done: true}:
			case <-ctx.Done():
			}
		}
	}()

	// Forward chunks to output channel and accumulate content
	for chunk := range streamChan {
		fullResponse.WriteString(chunk)
		select {
		case outputChan <- chat.MessageChunk{Content: chunk, Done: false}:
		case <-ctx.Done():
			return chat.Message{}, ctx.Err()
		}
	}

	// Send completion signal
	select {
	case outputChan <- chat.MessageChunk{Content: "", Done: true}:
	case <-ctx.Done():
		return chat.Message{}, ctx.Err()
	}

	// Create assistant message from full response
	response := fullResponse.String()
	assistantMsg := chat.NewAssistantMessage(response)
	elc.conversation = chat.AddMessage(elc.conversation, assistantMsg)

	elc.log.Debug("Streaming response completed", "response_length", len(response))

	return assistantMsg, nil
}

// GetHistory returns the chat history
func (elc *EnhancedLangChainController) GetHistory() []chat.Message {
	return chat.GetMessages(elc.conversation)
}

// GetConversation returns the current conversation
func (elc *EnhancedLangChainController) GetConversation() chat.Conversation {
	return elc.conversation
}

// Reset clears the conversation and memory
func (elc *EnhancedLangChainController) Reset() {
	elc.log.Debug("Resetting enhanced LangChain controller")

	// Get system prompt if exists
	systemPrompt := ""
	if chat.HasSystemMessage(elc.conversation) {
		messages := chat.GetMessagesByRole(elc.conversation, chat.RoleSystem)
		if len(messages) > 0 {
			systemPrompt = messages[0].Content
		}
	}

	// Clear conversation
	elc.conversation = chat.NewConversation(elc.model)

	// Clear LangChain memory
	ctx := context.Background()
	if err := elc.client.ClearMemory(ctx); err != nil {
		elc.log.Error("Failed to clear LangChain memory", "error", err)
	}

	// Re-add system prompt if it existed
	if systemPrompt != "" {
		elc.conversation = chat.AddMessage(elc.conversation, chat.NewSystemMessage(systemPrompt))
	}
}

// AddUserMessage adds a user message to the conversation (for optimistic updates)
func (elc *EnhancedLangChainController) AddUserMessage(content string) {
	if strings.TrimSpace(content) == "" {
		return
	}

	userMsg := chat.NewUserMessage(content)
	elc.conversation = chat.AddMessage(elc.conversation, userMsg)
}

// AddErrorMessage adds an error message to the conversation
func (elc *EnhancedLangChainController) AddErrorMessage(errorMsg string) {
	errMsg := chat.NewErrorMessage(errorMsg)
	elc.conversation = chat.AddMessage(elc.conversation, errMsg)
}

// GetModel returns the model name
func (elc *EnhancedLangChainController) GetModel() string {
	return elc.model
}

// GetToolRegistry returns the tool registry
func (elc *EnhancedLangChainController) GetToolRegistry() *tools.Registry {
	return elc.toolRegistry
}

// GetEnhancedClient returns the underlying enhanced LangChain client
func (elc *EnhancedLangChainController) GetEnhancedClient() *langchain.EnhancedClient {
	return elc.client
}