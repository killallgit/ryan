package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/killallgit/ryan/pkg/tools"
)

type ChatController struct {
	client             chat.ChatClient
	conversation       chat.Conversation
	toolRegistry       *tools.Registry
	lastPromptTokens   int
	lastResponseTokens int
	ollamaClient       OllamaClient
}

type OllamaClient interface {
	Tags() (*ollama.TagsResponse, error)
}

func NewChatController(client chat.ChatClient, model string, toolRegistry *tools.Registry) *ChatController {
	return &ChatController{
		client:             client,
		conversation:       chat.NewConversation(model),
		toolRegistry:       toolRegistry,
		lastPromptTokens:   0,
		lastResponseTokens: 0,
		ollamaClient:       nil, // Will be set via SetOllamaClient
	}
}

func NewChatControllerWithSystem(client chat.ChatClient, model, systemPrompt string, toolRegistry *tools.Registry) *ChatController {
	return &ChatController{
		client:             client,
		conversation:       chat.NewConversationWithSystem(model, systemPrompt),
		toolRegistry:       toolRegistry,
		lastPromptTokens:   0,
		lastResponseTokens: 0,
		ollamaClient:       nil, // Will be set via SetOllamaClient
	}
}

func (cc *ChatController) SendUserMessage(content string) (chat.Message, error) {
	if strings.TrimSpace(content) == "" {
		return chat.Message{}, fmt.Errorf("message content cannot be empty")
	}

	return cc.SendUserMessageWithContext(context.Background(), content)
}

func (cc *ChatController) SendUserMessageWithContext(ctx context.Context, content string) (chat.Message, error) {
	if strings.TrimSpace(content) == "" {
		return chat.Message{}, fmt.Errorf("message content cannot be empty")
	}

	// Execute tool-enabled chat loop with user message
	return cc.executeToolEnabledChat(ctx, content)
}

func (cc *ChatController) executeToolEnabledChat(ctx context.Context, userMessage string) (chat.Message, error) {
	maxIterations := 10 // prevent infinite loops

	// Detect if the user message was already appended optimistically
	lastMsg, ok := chat.GetLastMessage(cc.conversation)
	userMessageAdded := ok && lastMsg.IsUser() && lastMsg.Content == strings.TrimSpace(userMessage)

	for i := 0; i < maxIterations; i++ {
		// Use current conversation messages
		messages := cc.conversation.Messages

		// Prepare chat request with tools if available
		var req chat.ChatRequest
		if cc.toolRegistry != nil {
			toolDefs, err := cc.toolRegistry.GetDefinitions("ollama")
			if err != nil {
				// Restore original conversation on error
				cc.conversation = originalConversation
				return chat.Message{}, fmt.Errorf("failed to get tool definitions: %w", err)
			}

			// Convert tool definitions to the format expected by chat request
			tools := make([]map[string]any, len(toolDefs))
			for i, def := range toolDefs {
				tools[i] = def.Definition
			}

			req = chat.ChatRequest{
				Model:    cc.conversation.Model,
				Messages: messages,
				Stream:   false,
				Tools:    tools,
			}
		} else {
			req = chat.ChatRequest{
				Model:    cc.conversation.Model,
				Messages: messages,
				Stream:   false,
			}
		}

		// Send chat request
		response, err := cc.client.SendMessageWithResponse(req)
		if err != nil {
			// Restore original conversation on error
			cc.conversation = originalConversation
			return chat.Message{}, fmt.Errorf("failed to send message: %w", err)
		}

		// Update token tracking
		cc.lastPromptTokens = response.PromptEvalCount
		cc.lastResponseTokens = response.EvalCount

		// Add assistant message to conversation
		cc.conversation = chat.AddMessage(cc.conversation, response.Message)

		// Check if assistant wants to call tools
		if !response.Message.HasToolCalls() {
			// No tool calls, return the final message
			return response.Message, nil
		}

		// Execute tool calls
		err = cc.executeToolCalls(ctx, response.Message.ToolCalls)
		if err != nil {
			return chat.Message{}, fmt.Errorf("failed to execute tools: %w", err)
		}

		// Continue the loop to get the final response after tool execution
	}

	return chat.Message{}, fmt.Errorf("maximum tool execution iterations reached")
}

func (cc *ChatController) executeToolCalls(ctx context.Context, toolCalls []chat.ToolCall) error {
	if cc.toolRegistry == nil {
		return fmt.Errorf("tool registry not available")
	}

	for _, toolCall := range toolCalls {
		// Execute the tool
		toolReq := tools.ToolRequest{
			Name:       toolCall.Function.Name,
			Parameters: toolCall.Function.Arguments,
			Context:    ctx,
		}

		result, err := cc.toolRegistry.Execute(ctx, toolReq)
		if err != nil {
			// Add error result to conversation
			errorMsg := fmt.Sprintf("Tool execution failed: %s", err.Error())
			toolResult := chat.NewToolResultMessage(toolCall.Function.Name, errorMsg)
			cc.conversation = chat.AddMessage(cc.conversation, toolResult)
			continue
		}

		// Add successful result to conversation
		content := result.Content
		if !result.Success && result.Error != "" {
			content = result.Error
		}

		toolResult := chat.NewToolResultMessage(toolCall.Function.Name, content)
		cc.conversation = chat.AddMessage(cc.conversation, toolResult)
	}

	return nil
}

// AddUserMessage adds a user message to the conversation immediately (optimistic UI update)
func (cc *ChatController) AddUserMessage(content string) {
	if strings.TrimSpace(content) == "" {
		return
	}
	cc.conversation = chat.AddMessage(cc.conversation, chat.NewUserMessage(content))
}

func (cc *ChatController) GetHistory() []chat.Message {
	return chat.GetMessages(cc.conversation)
}

func (cc *ChatController) GetConversation() chat.Conversation {
	return cc.conversation
}

func (cc *ChatController) GetMessageCount() int {
	return chat.GetMessageCount(cc.conversation)
}

func (cc *ChatController) GetLastAssistantMessage() (chat.Message, bool) {
	return chat.GetLastAssistantMessage(cc.conversation)
}

func (cc *ChatController) GetLastUserMessage() (chat.Message, bool) {
	return chat.GetLastUserMessage(cc.conversation)
}

func (cc *ChatController) HasSystemMessage() bool {
	return chat.HasSystemMessage(cc.conversation)
}

func (cc *ChatController) GetModel() string {
	return cc.conversation.Model
}

func (cc *ChatController) SetModel(model string) {
	cc.conversation = chat.WithModel(cc.conversation, model)
}

func (cc *ChatController) Reset() {
	systemPrompt := ""
	if chat.HasSystemMessage(cc.conversation) {
		messages := chat.GetMessagesByRole(cc.conversation, chat.RoleSystem)
		if len(messages) > 0 {
			systemPrompt = messages[0].Content
		}
	}

	cc.conversation = chat.NewConversationWithSystem(cc.conversation.Model, systemPrompt)
	cc.lastPromptTokens = 0
	cc.lastResponseTokens = 0
}

func (cc *ChatController) GetToolRegistry() *tools.Registry {
	return cc.toolRegistry
}

func (cc *ChatController) SetToolRegistry(registry *tools.Registry) {
	cc.toolRegistry = registry
}

func (cc *ChatController) GetTokenUsage() (promptTokens, responseTokens int) {
	return cc.lastPromptTokens, cc.lastResponseTokens
}

func (cc *ChatController) AddErrorMessage(errorMsg string) {
	cc.conversation = chat.AddMessage(cc.conversation, chat.NewErrorMessage(errorMsg))
}

func (cc *ChatController) SetOllamaClient(client OllamaClient) {
	cc.ollamaClient = client
}

func (cc *ChatController) ValidateModel(model string) error {
	if cc.ollamaClient == nil {
		return fmt.Errorf("ollama client not configured")
	}

	response, err := cc.ollamaClient.Tags()
	if err != nil {
		return fmt.Errorf("failed to check available models: %w", err)
	}

	for _, availableModel := range response.Models {
		if availableModel.Name == model {
			return nil
		}
	}

	return fmt.Errorf("model %s not found locally", model)
}

func (cc *ChatController) SetModelWithValidation(model string) error {
	if err := cc.ValidateModel(model); err != nil {
		return err
	}

	cc.SetModel(model)
	return nil
}
