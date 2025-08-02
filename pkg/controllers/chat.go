package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

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

	// Store original conversation in case we need to rollback on error
	originalConversation := cc.conversation

	// Check if the user message was already added (e.g., by optimistic UI update)
	needsUserMessage := true
	if len(cc.conversation.Messages) > 0 {
		lastMsg := cc.conversation.Messages[len(cc.conversation.Messages)-1]
		if lastMsg.Role == chat.RoleUser && lastMsg.Content == userMessage {
			needsUserMessage = false
		}
	}

	// Add user message to conversation if not already present
	if needsUserMessage {
		cc.conversation = chat.AddMessage(cc.conversation, chat.NewUserMessage(userMessage))
	}

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

// Streaming functionality

// StreamingUpdate represents updates from the streaming process
type StreamingUpdate struct {
	Type     StreamingUpdateType
	StreamID string
	Content  string
	Chunk    chat.MessageChunk
	Message  chat.Message
	Error    error
	Metadata StreamingMetadata
}

// StreamingUpdateType indicates the type of streaming update
type StreamingUpdateType int

const (
	StreamStarted StreamingUpdateType = iota
	ChunkReceived
	MessageComplete
	StreamError
	ToolExecutionStarted
	ToolExecutionComplete
)

// StreamingMetadata provides additional context about the stream
type StreamingMetadata struct {
	ChunkCount    int
	ContentLength int
	Duration      time.Duration
	Model         string
}

// StartStreaming initiates streaming for a user message
func (cc *ChatController) StartStreaming(ctx context.Context, content string) (<-chan StreamingUpdate, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("message content cannot be empty")
	}

	// Check if client supports streaming
	streamingClient, ok := cc.client.(chat.StreamingChatClient)
	if !ok {
		// Fallback to non-streaming
		return cc.fallbackToNonStreaming(ctx, content)
	}

	return cc.executeStreamingChat(ctx, streamingClient, content)
}

// executeStreamingChat handles the streaming chat process with tool support
func (cc *ChatController) executeStreamingChat(ctx context.Context, streamingClient chat.StreamingChatClient, userMessage string) (<-chan StreamingUpdate, error) {
	maxIterations := 10

	// Store original conversation in case we need to rollback on error
	originalConversation := cc.conversation

	// Check if the user message was already added (optimistic UI update)
	needsUserMessage := true
	if len(cc.conversation.Messages) > 0 {
		lastMsg := cc.conversation.Messages[len(cc.conversation.Messages)-1]
		if lastMsg.Role == chat.RoleUser && lastMsg.Content == userMessage {
			needsUserMessage = false
		}
	}

	// Add user message if not already present
	if needsUserMessage {
		cc.conversation = chat.AddMessage(cc.conversation, chat.NewUserMessage(userMessage))
	}

	// Create update channel
	updates := make(chan StreamingUpdate, 100)

	// Start streaming goroutine
	go func() {
		defer close(updates)

		accumulator := chat.NewMessageAccumulator()

		for i := 0; i < maxIterations; i++ {
			// Prepare chat request with tools if available
			var req chat.ChatRequest
			if cc.toolRegistry != nil {
				toolDefs, err := cc.toolRegistry.GetDefinitions("ollama")
				if err != nil {
					updates <- StreamingUpdate{
						Type:  StreamError,
						Error: fmt.Errorf("failed to get tool definitions: %w", err),
					}
					cc.conversation = originalConversation
					return
				}

				// Convert tool definitions
				tools := make([]map[string]any, len(toolDefs))
				for j, def := range toolDefs {
					tools[j] = def.Definition
				}

				req = chat.CreateStreamingChatRequestWithTools(cc.conversation, userMessage, tools)
			} else {
				req = chat.CreateStreamingChatRequest(cc.conversation, userMessage)
			}

			// Start streaming
			chunks, err := streamingClient.StreamMessage(ctx, req)
			if err != nil {
				updates <- StreamingUpdate{
					Type:  StreamError,
					Error: fmt.Errorf("failed to start streaming: %w", err),
				}
				cc.conversation = originalConversation
				return
			}

			streamID := ""
			startTime := time.Now()
			var assistantMessage chat.Message

			// Send stream started event
			updates <- StreamingUpdate{
				Type:     StreamStarted,
				StreamID: streamID,
				Metadata: StreamingMetadata{
					Model: cc.conversation.Model,
				},
			}

			// Process chunks
			for chunk := range chunks {
				if chunk.Error != nil {
					updates <- StreamingUpdate{
						Type:     StreamError,
						StreamID: chunk.StreamID,
						Error:    chunk.Error,
					}
					continue
				}

				if streamID == "" {
					streamID = chunk.StreamID
				}

				// Add chunk to accumulator
				accumulator.AddChunk(chunk)

				// Send chunk update
				stats, _ := accumulator.GetStreamStats(chunk.StreamID)
				updates <- StreamingUpdate{
					Type:     ChunkReceived,
					StreamID: chunk.StreamID,
					Content:  chunk.Content,
					Chunk:    chunk,
					Metadata: StreamingMetadata{
						ChunkCount:    stats.ChunkCount,
						ContentLength: len(accumulator.GetCurrentContent(chunk.StreamID)),
						Duration:      time.Since(startTime),
						Model:         chunk.Model,
					},
				}

				// Check if stream is complete
				if chunk.Done {
					// Get final message
					finalMessage, exists := accumulator.GetCompleteMessage(chunk.StreamID)
					if exists {
						assistantMessage = finalMessage

						// Update token tracking from last chunk
						cc.lastPromptTokens = chunk.PromptEvalCount
						cc.lastResponseTokens = chunk.EvalCount

						// Add assistant message to conversation
						cc.conversation = chat.AddMessage(cc.conversation, assistantMessage)

						// Send completion event
						finalStats, _ := accumulator.GetStreamStats(chunk.StreamID)
						updates <- StreamingUpdate{
							Type:     MessageComplete,
							StreamID: chunk.StreamID,
							Message:  assistantMessage,
							Metadata: StreamingMetadata{
								ChunkCount:    finalStats.ChunkCount,
								ContentLength: len(assistantMessage.Content),
								Duration:      time.Since(startTime),
								Model:         chunk.Model,
							},
						}
					}
					break
				}
			}

			// Check if assistant wants to call tools
			if !assistantMessage.HasToolCalls() {
				// No tool calls, streaming complete
				return
			}

			// Execute tool calls
			updates <- StreamingUpdate{
				Type:     ToolExecutionStarted,
				StreamID: streamID,
			}

			err = cc.executeToolCalls(ctx, assistantMessage.ToolCalls)
			if err != nil {
				updates <- StreamingUpdate{
					Type:  StreamError,
					Error: fmt.Errorf("failed to execute tools: %w", err),
				}
				return
			}

			updates <- StreamingUpdate{
				Type:     ToolExecutionComplete,
				StreamID: streamID,
			}

			// Continue the loop to get the final response after tool execution
		}

		updates <- StreamingUpdate{
			Type:  StreamError,
			Error: fmt.Errorf("maximum tool execution iterations reached"),
		}
	}()

	return updates, nil
}

// fallbackToNonStreaming provides non-streaming fallback when streaming is not available
func (cc *ChatController) fallbackToNonStreaming(ctx context.Context, content string) (<-chan StreamingUpdate, error) {
	updates := make(chan StreamingUpdate, 1)

	go func() {
		defer close(updates)

		response, err := cc.SendUserMessageWithContext(ctx, content)
		if err != nil {
			updates <- StreamingUpdate{
				Type:  StreamError,
				Error: err,
			}
			return
		}

		updates <- StreamingUpdate{
			Type:    MessageComplete,
			Message: response,
		}
	}()

	return updates, nil
}
