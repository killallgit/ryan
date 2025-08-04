package chat

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// StreamingClient implements both ChatClient and StreamingChatClient using LangChain Go
type StreamingClient struct {
	*Client
}

// NewStreamingClient creates a new LangChain-based streaming chat client
func NewStreamingClient(baseURL, model string) (*StreamingClient, error) {
	return NewStreamingClientWithTimeout(baseURL, model, 60*time.Second)
}

// NewStreamingClientWithTimeout creates a new LangChain-based streaming chat client with custom timeout
func NewStreamingClientWithTimeout(baseURL, model string, timeout time.Duration) (*StreamingClient, error) {
	client, err := NewClientWithTimeout(baseURL, model, timeout)
	if err != nil {
		return nil, err
	}

	return &StreamingClient{
		Client: client,
	}, nil
}

// StreamMessage implements StreamingChatClient interface using LangChain Go's streaming
func (sc *StreamingClient) StreamMessage(ctx context.Context, req ChatRequest) (<-chan MessageChunk, error) {
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

		messages = append(messages, llms.TextParts(messageType, msg.Content))
	}

	// Create output channel
	outputChan := make(chan MessageChunk, 100)

	// Generate unique stream ID
	streamID := fmt.Sprintf("lc_%d", time.Now().UnixNano())

	// Start streaming in goroutine
	go func() {
		defer close(outputChan)

		var contentBuilder strings.Builder
		var mu sync.Mutex
		var chunkCount int

		// Create streaming function
		streamingFunc := func(ctx context.Context, chunk []byte) error {
			mu.Lock()
			defer mu.Unlock()

			chunkCount++
			chunkStr := string(chunk)
			contentBuilder.WriteString(chunkStr)

			// Create and send chunk
			messageChunk := MessageChunk{
				ID:        fmt.Sprintf("%s-%d", streamID, chunkCount),
				StreamID:  streamID,
				Model:     req.Model,
				Content:   chunkStr,
				Done:      false,
				Timestamp: time.Now(),
				CreatedAt: time.Now(),
				Error:     nil,
			}

			select {
			case outputChan <- messageChunk:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		}

		// Call LangChain Go with streaming
		response, err := sc.llm.GenerateContent(ctx, messages, llms.WithStreamingFunc(streamingFunc))
		if err != nil {
			// Send error chunk
			errorChunk := MessageChunk{
				StreamID:  streamID,
				Model:     req.Model,
				Content:   "",
				Done:      true,
				CreatedAt: time.Now(),
				Error:     err,
			}
			outputChan <- errorChunk
			return
		}

		// Send final chunk with complete information
		mu.Lock()
		finalContent := contentBuilder.String()

		// If no streaming chunks were received, use response content
		if finalContent == "" && len(response.Choices) > 0 {
			finalContent = response.Choices[0].Content
		}

		// TODO: Extract token usage if available from LangChain response
		// The llms.ContentResponse type doesn't expose usage information directly
		// This would require investigation into LangChain Go internals or using a different client

		finalChunk := MessageChunk{
			ID:        fmt.Sprintf("%s-final", streamID),
			StreamID:  streamID,
			Model:     req.Model,
			Content:   "", // No additional content in final chunk
			Done:      true,
			Timestamp: time.Now(),
			CreatedAt: time.Now(),
			Error:     nil,
			// Note: Token counts not available from LangChain Go llms.ContentResponse
			PromptEvalCount: 0,
			EvalCount:       0,
		}

		// Handle tool calls if present in response
		if len(response.Choices) > 0 && response.Choices[0].ToolCalls != nil {
			// Create message with tool calls and include it in the final chunk
			finalMessage := Message{
				Role:      RoleAssistant,
				Content:   finalContent,
				Timestamp: time.Now(),
			}

			toolCalls := make([]ToolCall, 0, len(response.Choices[0].ToolCalls))
			for _, tc := range response.Choices[0].ToolCalls {
				// Convert arguments string to map if needed
				var args map[string]any
				if tc.FunctionCall.Arguments != "" {
					args = map[string]any{"raw": tc.FunctionCall.Arguments}
				}

				toolCalls = append(toolCalls, ToolCall{
					Function: ToolFunction{
						Name:      tc.FunctionCall.Name,
						Arguments: args,
					},
				})
			}
			finalMessage.ToolCalls = toolCalls
			finalChunk.Message = finalMessage
		}

		mu.Unlock()

		outputChan <- finalChunk
	}()

	return outputChan, nil
}

// CreateStreamingChatRequest creates a streaming-enabled chat request
func CreateStreamingChatRequest(conversation Conversation, userMessage string) ChatRequest {
	conv := AddMessage(conversation, NewUserMessage(userMessage))

	return ChatRequest{
		Model:    conv.Model,
		Messages: conv.Messages,
		Stream:   true,
	}
}

// CreateStreamingChatRequestWithTools creates a streaming-enabled chat request with tools
func CreateStreamingChatRequestWithTools(conversation Conversation, userMessage string, tools []map[string]any) ChatRequest {
	conv := AddMessage(conversation, NewUserMessage(userMessage))

	return ChatRequest{
		Model:    conv.Model,
		Messages: conv.Messages,
		Stream:   true,
		Tools:    tools,
	}
}

// generateStreamID creates a unique identifier for this stream
var streamCounter int64

func generateStreamID() string {
	id := atomic.AddInt64(&streamCounter, 1)
	return fmt.Sprintf("stream-%d-%d", time.Now().UnixNano(), id)
}

// Verify interface compliance
var _ StreamingChatClient = (*StreamingClient)(nil)
