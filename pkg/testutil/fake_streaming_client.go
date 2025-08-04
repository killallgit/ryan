package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/killallgit/ryan/pkg/chat"
)

// FakeStreamingChatClient implements the StreamingChatClient interface for testing
type FakeStreamingChatClient struct {
	*FakeChatClient
	chunkDelay   time.Duration // Delay between chunks
	chunkSize    int           // Characters per chunk
	failAfter    int           // Fail after N chunks (0 = no failure)
	errorMessage string        // Custom error message
}

// NewFakeStreamingChatClient creates a new fake streaming chat client
func NewFakeStreamingChatClient(model string, responses ...string) *FakeStreamingChatClient {
	return &FakeStreamingChatClient{
		FakeChatClient: NewFakeChatClient(model, responses...),
		chunkDelay:     10 * time.Millisecond,
		chunkSize:      5, // 5 characters per chunk by default
	}
}

// StreamMessage implements the StreamingChatClient interface
func (c *FakeStreamingChatClient) StreamMessage(ctx context.Context, req chat.ChatRequest) (<-chan chat.MessageChunk, error) {
	// Get the response from the underlying fake LLM
	prompt := ""
	for _, msg := range req.Messages {
		prompt += msg.Role + ": " + msg.Content + "\n"
	}

	response, err := c.fakeLLM.Call(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Check if this is a tool call response
	var toolCalls []chat.ToolCall
	if req.Tools != nil && len(req.Tools) > 0 && strings.Contains(response, "tool_calls") {
		var toolResponse struct {
			ToolCalls []struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments"`
			} `json:"tool_calls"`
		}
		if err := json.Unmarshal([]byte(response), &toolResponse); err == nil && len(toolResponse.ToolCalls) > 0 {
			for _, tc := range toolResponse.ToolCalls {
				toolCalls = append(toolCalls, chat.ToolCall{
					Function: chat.ToolFunction{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				})
			}
		}
	}

	// Create a channel for streaming chunks
	chunkChan := make(chan chat.MessageChunk, 100)
	streamID := uuid.New().String()

	// Start streaming in a goroutine
	go func() {
		defer close(chunkChan)

		chunkCount := 0

		// If we have tool calls, send them as a single chunk
		if len(toolCalls) > 0 {
			chunk := chat.MessageChunk{
				StreamID:  streamID,
				Content:   "",
				Done:      true,
				Timestamp: time.Now(),
				Message: chat.Message{
					Role:      "assistant",
					Content:   "",
					ToolCalls: toolCalls,
					Timestamp: time.Now(),
				},
			}

			select {
			case chunkChan <- chunk:
			case <-ctx.Done():
				return
			}
			return
		}

		// Otherwise, stream the text response in chunks
		for i := 0; i < len(response); i += c.chunkSize {
			// Check if we should fail
			chunkCount++
			if c.failAfter > 0 && chunkCount > c.failAfter {
				errMsg := c.errorMessage
				if errMsg == "" {
					errMsg = "simulated streaming error"
				}
				chunk := chat.MessageChunk{
					StreamID:  streamID,
					Error:     fmt.Errorf(errMsg),
					Timestamp: time.Now(),
				}
				select {
				case chunkChan <- chunk:
				case <-ctx.Done():
				}
				return
			}

			// Calculate chunk bounds
			end := i + c.chunkSize
			if end > len(response) {
				end = len(response)
			}

			chunkContent := response[i:end]
			isLast := end >= len(response)

			chunk := chat.MessageChunk{
				StreamID:  streamID,
				Content:   chunkContent,
				Done:      isLast,
				Timestamp: time.Now(),
				Message: chat.Message{
					Role:      "assistant",
					Content:   response[:end], // Accumulated content
					Timestamp: time.Now(),
				},
			}

			// Simulate processing delay
			if c.chunkDelay > 0 {
				select {
				case <-time.After(c.chunkDelay):
				case <-ctx.Done():
					return
				}
			}

			// Send chunk
			select {
			case chunkChan <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return chunkChan, nil
}

// SetChunkDelay sets the delay between chunks
func (c *FakeStreamingChatClient) SetChunkDelay(delay time.Duration) {
	c.chunkDelay = delay
}

// SetChunkSize sets the number of characters per chunk
func (c *FakeStreamingChatClient) SetChunkSize(size int) {
	c.chunkSize = size
}

// SetFailAfter configures the client to fail after N chunks
func (c *FakeStreamingChatClient) SetFailAfter(chunks int, errorMessage string) {
	c.failAfter = chunks
	c.errorMessage = errorMessage
}

// SupportsStreaming implements the StreamingCapable interface
func (c *FakeStreamingChatClient) SupportsStreaming() bool {
	return true
}
