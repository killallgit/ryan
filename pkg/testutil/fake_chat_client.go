package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/tmc/langchaingo/llms"
)

// FakeChatClient implements the ChatClient interface for testing
type FakeChatClient struct {
	fakeLLM      *FakeLLM
	model        string
	responseTime time.Duration // Simulated response time
}

// NewFakeChatClient creates a new fake chat client
func NewFakeChatClient(model string, responses ...string) *FakeChatClient {
	return &FakeChatClient{
		fakeLLM:      NewFakeLLM(responses...),
		model:        model,
		responseTime: 10 * time.Millisecond, // Default simulated response time
	}
}

// SendMessage implements the ChatClient interface
func (c *FakeChatClient) SendMessage(req chat.ChatRequest) (chat.Message, error) {
	resp, err := c.SendMessageWithResponse(req)
	if err != nil {
		return chat.Message{}, err
	}
	return resp.Message, nil
}

// SendMessageWithResponse implements the ChatClient interface
func (c *FakeChatClient) SendMessageWithResponse(req chat.ChatRequest) (chat.ChatResponse, error) {
	// Simulate response time
	if c.responseTime > 0 {
		time.Sleep(c.responseTime)
	}

	// Convert messages to prompt
	prompt := ""
	for _, msg := range req.Messages {
		prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	// Get response from fake LLM
	ctx := context.Background()
	response, err := c.fakeLLM.Call(ctx, prompt)
	if err != nil {
		return chat.ChatResponse{}, err
	}

	// Check if response looks like a tool call
	var toolCalls []chat.ToolCall
	if req.Tools != nil && len(req.Tools) > 0 {
		// Try to parse as JSON to see if it contains tool calls
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

	now := time.Now()
	return chat.ChatResponse{
		Model:     c.model,
		CreatedAt: now,
		Message: chat.Message{
			Role:      "assistant",
			Content:   response,
			ToolCalls: toolCalls,
		},
		Done:               true,
		DoneReason:         "complete",
		PromptEvalCount:    len(prompt),
		EvalCount:          len(response),
		PromptEvalDuration: c.responseTime / 2,
		EvalDuration:       c.responseTime / 2,
	}, nil
}

// GetFakeLLM returns the underlying fake LLM for test assertions
func (c *FakeChatClient) GetFakeLLM() *FakeLLM {
	return c.fakeLLM
}

// SetResponseTime sets the simulated response time
func (c *FakeChatClient) SetResponseTime(duration time.Duration) {
	c.responseTime = duration
}

// CreateFakeOllamaLLM creates a fake LLM that can be used in place of Ollama
func CreateFakeOllamaLLM(responses ...string) llms.Model {
	return &fakeLLMAdapter{
		fakeLLM: NewFakeLLM(responses...),
	}
}

// fakeLLMAdapter adapts FakeLLM to implement llms.Model interface
type fakeLLMAdapter struct {
	fakeLLM *FakeLLM
}

func (a *fakeLLMAdapter) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return a.fakeLLM.Call(ctx, prompt, options...)
}

func (a *fakeLLMAdapter) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	return a.fakeLLM.GenerateContent(ctx, messages, options...)
}
