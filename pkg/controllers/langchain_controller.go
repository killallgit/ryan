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

// LangChainController wraps the LangChain client to work with the existing controller interface
type LangChainController struct {
	client       *langchain.Client
	model        string
	toolRegistry *tools.Registry
	conversation chat.Conversation
	log          *logger.Logger
}

// NewLangChainController creates a new controller using the LangChain client
func NewLangChainController(baseURL, model string, toolRegistry *tools.Registry) (*LangChainController, error) {
	client, err := langchain.NewClient(baseURL, model, toolRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create LangChain client: %w", err)
	}

	log := logger.WithComponent("langchain_controller")

	return &LangChainController{
		client:       client,
		model:        model,
		toolRegistry: toolRegistry,
		conversation: chat.NewConversation(model),
		log:          log,
	}, nil
}

// NewLangChainControllerWithSystem creates a new controller with system prompt
func NewLangChainControllerWithSystem(baseURL, model, systemPrompt string, toolRegistry *tools.Registry) (*LangChainController, error) {
	controller, err := NewLangChainController(baseURL, model, toolRegistry)
	if err != nil {
		return nil, err
	}

	// Add system message to conversation
	if systemPrompt != "" {
		controller.conversation = chat.AddMessage(controller.conversation, chat.NewSystemMessage(systemPrompt))
	}

	return controller, nil
}

// SendUserMessage sends a user message using the LangChain agent
func (lc *LangChainController) SendUserMessage(content string) (chat.Message, error) {
	return lc.SendUserMessageWithContext(context.Background(), content)
}

// SendUserMessageWithContext sends a user message with context using the LangChain agent
func (lc *LangChainController) SendUserMessageWithContext(ctx context.Context, content string) (chat.Message, error) {
	if strings.TrimSpace(content) == "" {
		return chat.Message{}, fmt.Errorf("message content cannot be empty")
	}

	lc.log.Debug("Sending user message with LangChain agent", 
		"content_length", len(content), 
		"has_tools", lc.toolRegistry != nil,
		"agent_enabled", lc.client != nil)

	// Add user message to conversation
	userMsg := chat.NewUserMessage(content)
	lc.conversation = chat.AddMessage(lc.conversation, userMsg)

	// Use the enhanced client to send the message
	response, err := lc.client.SendMessage(ctx, content)
	if err != nil {
		errorMsg := fmt.Sprintf("LangChain agent failed: %v", err)
		lc.log.Error("Enhanced LangChain client failed", "error", err)
		
		// Add error message to conversation
		errMsg := chat.NewErrorMessage(errorMsg)
		lc.conversation = chat.AddMessage(lc.conversation, errMsg)
		
		return errMsg, fmt.Errorf("failed to send message: %w", err)
	}

	// Create assistant message from response
	assistantMsg := chat.NewAssistantMessage(response)
	lc.conversation = chat.AddMessage(lc.conversation, assistantMsg)

	lc.log.Debug("Enhanced LangChain agent response received", 
		"response_length", len(response))

	return assistantMsg, nil
}

// SendUserMessageWithStreamingContext sends a message with streaming support
func (lc *LangChainController) SendUserMessageWithStreamingContext(ctx context.Context, content string, outputChan chan<- chat.MessageChunk) (chat.Message, error) {
	if strings.TrimSpace(content) == "" {
		return chat.Message{}, fmt.Errorf("message content cannot be empty")
	}

	lc.log.Debug("Starting streaming message with LangChain agent")

	// Add user message to conversation
	userMsg := chat.NewUserMessage(content)
	lc.conversation = chat.AddMessage(lc.conversation, userMsg)

	// Create a channel to collect streamed content
	streamChan := make(chan string, 100)
	var fullResponse strings.Builder

	// Start streaming in a goroutine
	go func() {
		defer close(streamChan)
		if err := lc.client.StreamMessage(ctx, content, streamChan); err != nil {
			lc.log.Error("Streaming failed", "error", err)
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
	lc.conversation = chat.AddMessage(lc.conversation, assistantMsg)

	lc.log.Debug("Streaming response completed", "response_length", len(response))

	return assistantMsg, nil
}

// GetHistory returns the chat history
func (lc *LangChainController) GetHistory() []chat.Message {
	return chat.GetMessages(lc.conversation)
}

// GetConversation returns the current conversation
func (lc *LangChainController) GetConversation() chat.Conversation {
	return lc.conversation
}

// Reset clears the conversation and memory
func (lc *LangChainController) Reset() {
	lc.log.Debug("Resetting LangChain controller")

	// Get system prompt if exists
	systemPrompt := ""
	if chat.HasSystemMessage(lc.conversation) {
		messages := chat.GetMessagesByRole(lc.conversation, chat.RoleSystem)
		if len(messages) > 0 {
			systemPrompt = messages[0].Content
		}
	}

	// Clear conversation
	lc.conversation = chat.NewConversation(lc.model)

	// Clear LangChain memory
	ctx := context.Background()
	if err := lc.client.ClearMemory(ctx); err != nil {
		lc.log.Error("Failed to clear LangChain memory", "error", err)
	}

	// Re-add system prompt if it existed
	if systemPrompt != "" {
		lc.conversation = chat.AddMessage(lc.conversation, chat.NewSystemMessage(systemPrompt))
	}
}

// AddUserMessage adds a user message to the conversation (for optimistic updates)
func (lc *LangChainController) AddUserMessage(content string) {
	if strings.TrimSpace(content) == "" {
		return
	}

	userMsg := chat.NewUserMessage(content)
	lc.conversation = chat.AddMessage(lc.conversation, userMsg)
}

// AddErrorMessage adds an error message to the conversation
func (lc *LangChainController) AddErrorMessage(errorMsg string) {
	errMsg := chat.NewErrorMessage(errorMsg)
	lc.conversation = chat.AddMessage(lc.conversation, errMsg)
}

// GetModel returns the model name
func (lc *LangChainController) GetModel() string {
	return lc.model
}

// GetToolRegistry returns the tool registry
func (lc *LangChainController) GetToolRegistry() *tools.Registry {
	return lc.toolRegistry
}

// GetClient returns the underlying LangChain client
func (lc *LangChainController) GetClient() *langchain.Client {
	return lc.client
}