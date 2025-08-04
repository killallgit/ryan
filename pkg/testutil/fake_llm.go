package testutil

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
)

// FakeLLM implements a fake language model for testing
type FakeLLM struct {
	mu           sync.Mutex
	responses    []string
	currentIndex int
	callCount    int
	lastPrompt   string
	lastOptions  []llms.CallOption
	errorOnCall  int // If > 0, return error on this call number
	errorMessage string
}

// NewFakeLLM creates a new fake LLM with predefined responses
func NewFakeLLM(responses ...string) *FakeLLM {
	return &FakeLLM{
		responses: responses,
	}
}

// Call implements the LLM interface
func (f *FakeLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.callCount++
	f.lastPrompt = prompt
	f.lastOptions = options

	// Check if we should return an error
	if f.errorOnCall > 0 && f.callCount == f.errorOnCall {
		if f.errorMessage != "" {
			return "", fmt.Errorf(f.errorMessage)
		}
		return "", fmt.Errorf("fake error on call %d", f.callCount)
	}

	// Return response based on current index
	if len(f.responses) == 0 {
		return "", fmt.Errorf("no responses configured")
	}

	response := f.responses[f.currentIndex]
	f.currentIndex = (f.currentIndex + 1) % len(f.responses)

	return response, nil
}

// GenerateContent implements the LLM interface for message-based generation
func (f *FakeLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	// Convert messages to a single prompt
	var parts []string
	for _, msg := range messages {
		for _, part := range msg.Parts {
			if text, ok := part.(llms.TextContent); ok {
				parts = append(parts, text.Text)
			}
		}
	}

	prompt := strings.Join(parts, "\n")
	response, err := f.Call(ctx, prompt, options...)
	if err != nil {
		return nil, err
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: response,
			},
		},
	}, nil
}

// Reset resets the response index and call count
func (f *FakeLLM) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.currentIndex = 0
	f.callCount = 0
	f.lastPrompt = ""
	f.lastOptions = nil
}

// AddResponse adds a new response to the LLM
func (f *FakeLLM) AddResponse(response string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.responses = append(f.responses, response)
}

// SetErrorOnCall configures the LLM to return an error on a specific call
func (f *FakeLLM) SetErrorOnCall(callNumber int, errorMessage string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errorOnCall = callNumber
	f.errorMessage = errorMessage
}

// GetCallCount returns the number of times Call was invoked
func (f *FakeLLM) GetCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount
}

// GetLastPrompt returns the last prompt passed to Call
func (f *FakeLLM) GetLastPrompt() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastPrompt
}

// GetLastOptions returns the last options passed to Call
func (f *FakeLLM) GetLastOptions() []llms.CallOption {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastOptions
}

// PredefinedResponses provides common test response patterns
var PredefinedResponses = struct {
	SimpleChat    []string
	ErrorResponse []string
	ToolCalling   []string
}{
	SimpleChat: []string{
		"Hello! How can I help you today?",
		"I understand your question. Here's my response.",
		"Thank you for the clarification.",
	},
	ErrorResponse: []string{
		"I'm sorry, I encountered an error processing your request.",
	},
	ToolCalling: []string{
		`{"tool_calls": [{"name": "test_tool", "arguments": {"param": "value"}}]}`,
	},
}
