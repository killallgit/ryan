package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	Stream   bool             `json:"stream"`
	Tools    []map[string]any `json:"tools,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Model              string        `json:"model"`
	CreatedAt          time.Time     `json:"created_at"`
	Message            Message       `json:"message"`
	Done               bool          `json:"done"`
	DoneReason         string        `json:"done_reason,omitempty"`
	PromptEvalCount    int           `json:"prompt_eval_count"`
	EvalCount          int           `json:"eval_count"`
	PromptEvalDuration time.Duration `json:"prompt_eval_duration"`
	EvalDuration       time.Duration `json:"eval_duration"`
}

// ChatClient interface defines methods for chat completion
type ChatClient interface {
	SendMessage(req ChatRequest) (Message, error)
	SendMessageWithResponse(req ChatRequest) (ChatResponse, error)
}

// MessageChunk represents a single chunk from a streaming response
type MessageChunk struct {
	ID        string    `json:"-"` // Unique chunk identifier
	Content   string    `json:"-"` // Incremental text content
	Done      bool      `json:"-"` // Stream completion indicator
	Timestamp time.Time `json:"-"` // When chunk was received
	StreamID  string    `json:"-"` // Which stream this belongs to
	Error     error     `json:"-"` // Error if chunk processing failed

	// Raw Ollama response fields
	Model              string        `json:"model"`
	CreatedAt          time.Time     `json:"created_at"`
	Message            Message       `json:"message"`
	DoneFlag           bool          `json:"done"`
	DoneReason         string        `json:"done_reason,omitempty"`
	PromptEvalCount    int           `json:"prompt_eval_count"`
	EvalCount          int           `json:"eval_count"`
	PromptEvalDuration time.Duration `json:"prompt_eval_duration"`
	EvalDuration       time.Duration `json:"eval_duration"`
}

// StreamingChatClient interface extends ChatClient with streaming support
type StreamingChatClient interface {
	ChatClient
	StreamMessage(ctx context.Context, req ChatRequest) (<-chan MessageChunk, error)
}

// Client implements ChatClient using LangChain Go's Ollama provider
type Client struct {
	llm     llms.Model
	baseURL string
	model   string
}

// NewClient creates a new LangChain-based chat client
func NewClient(baseURL, model string) (*Client, error) {
	return NewClientWithTimeout(baseURL, model, 60*time.Second)
}

// NewClientWithTimeout creates a new LangChain-based chat client with custom timeout
func NewClientWithTimeout(baseURL, model string, timeout time.Duration) (*Client, error) {
	var opts []ollama.Option

	if baseURL != "" {
		opts = append(opts, ollama.WithServerURL(baseURL))
	}

	if model != "" {
		opts = append(opts, ollama.WithModel(model))
	}

	llm, err := ollama.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create LangChain Ollama client: %w", err)
	}

	return &Client{
		llm:     llm,
		baseURL: baseURL,
		model:   model,
	}, nil
}

// SendMessage implements ChatClient interface using LangChain Go
func (c *Client) SendMessage(req ChatRequest) (Message, error) {
	resp, err := c.SendMessageWithResponse(req)
	if err != nil {
		return Message{}, err
	}
	return resp.Message, nil
}

// SendMessageWithResponse implements ChatClient interface using LangChain Go
func (c *Client) SendMessageWithResponse(req ChatRequest) (ChatResponse, error) {
	ctx := context.Background()

	// Convert chat messages to LangChain format
	messages := make([]llms.MessageContent, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messageType := llms.ChatMessageTypeHuman
		switch msg.Role {
		case RoleSystem:
			messageType = llms.ChatMessageTypeSystem
		case RoleAssistant:
			messageType = llms.ChatMessageTypeAI
		case RoleUser:
			messageType = llms.ChatMessageTypeHuman
		case RoleTool:
			messageType = llms.ChatMessageTypeTool
		}

		// Handle tool calls in assistant messages
		if msg.Role == RoleAssistant && len(msg.ToolCalls) > 0 {
			// For messages with tool calls, we need to include both content and tool calls
			// LangChain Go handles this through the content response structure
			messages = append(messages, llms.TextParts(messageType, msg.Content))
		} else {
			messages = append(messages, llms.TextParts(messageType, msg.Content))
		}
	}

	// Prepare call options
	var opts []llms.CallOption
	if req.Stream {
		// Streaming will be handled separately in streaming client
		// For now, we'll force non-streaming for compatibility
		opts = append(opts, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			// This will be implemented in streaming support
			return nil
		}))
	}

	// Add tools if provided
	if len(req.Tools) > 0 {
		// Convert tools to LangChain format
		// Note: Tool conversion will need to be implemented based on LangChain Go's tool interface
		// For now, we'll proceed without tools to establish basic functionality
	}

	// Call LangChain Go LLM
	response, err := c.llm.GenerateContent(ctx, messages, opts...)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("LangChain content generation failed: %w", err)
	}

	// Convert response back to our format
	if len(response.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("no response choices returned from LangChain")
	}

	choice := response.Choices[0]

	// Create response message
	responseMessage := Message{
		Role:      RoleAssistant,
		Content:   choice.Content,
		Timestamp: time.Now(),
	}

	// Handle tool calls if present
	if len(choice.ToolCalls) > 0 {
		toolCalls := make([]ToolCall, 0, len(choice.ToolCalls))
		for _, tc := range choice.ToolCalls {
			// Convert arguments string to map if needed
			var args map[string]any
			if tc.FunctionCall.Arguments != "" {
				// For now, create a simple wrapper - this may need adjustment based on actual format
				args = map[string]any{"raw": tc.FunctionCall.Arguments}
			}

			toolCalls = append(toolCalls, ToolCall{
				Function: ToolFunction{
					Name:      tc.FunctionCall.Name,
					Arguments: args,
				},
			})
		}
		responseMessage.ToolCalls = toolCalls
	}

	chatResponse := ChatResponse{
		Model:     req.Model,
		CreatedAt: time.Now(),
		Message:   responseMessage,
		Done:      true,
		// TODO: Extract token counts from LangChain response if available
		// The llms.ContentResponse type doesn't expose usage information directly
		PromptEvalCount: 0,
		EvalCount:       0,
	}

	return chatResponse, nil
}

// CreateChatRequest creates a chat request from a conversation and user message
func CreateChatRequest(conversation Conversation, userMessage string) ChatRequest {
	conv := AddMessage(conversation, NewUserMessage(userMessage))

	return ChatRequest{
		Model:    conv.Model,
		Messages: GetMessages(conv),
		Stream:   false,
	}
}

// CreateChatRequestWithTools creates a chat request with tools from a conversation and user message
func CreateChatRequestWithTools(conversation Conversation, userMessage string, tools []map[string]any) ChatRequest {
	conv := AddMessage(conversation, NewUserMessage(userMessage))

	return ChatRequest{
		Model:    conv.Model,
		Messages: GetMessages(conv),
		Stream:   false,
		Tools:    tools,
	}
}
