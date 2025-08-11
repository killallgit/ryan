package mocks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/orchestrator"
	"github.com/tmc/langchaingo/llms"
)

// CallRecord tracks LLM calls for testing
type CallRecord struct {
	Timestamp time.Time
	Input     string
	Output    string
	Agent     orchestrator.AgentType
	Messages  []llms.MessageContent
	Error     error
}

// MockLLM implements a configurable mock LLM for testing
type MockLLM struct {
	// Response configuration
	ResponseMap     map[string]string // Pattern-based responses
	ResponseQueue   []string          // Queue of responses to return
	DefaultResponse string            // Default response if no pattern matches

	// Behavior configuration
	SimulateLatency time.Duration
	ErrorRate       float64 // Probability of error (0.0 to 1.0)
	ErrorMessage    string

	// Intent analysis responses
	IntentResponses map[string]*orchestrator.TaskIntent

	// Tracking
	CallHistory []CallRecord
	callCount   int
	mu          sync.Mutex

	// Callbacks for testing
	OnCall     func(messages []llms.MessageContent)
	OnGenerate func(response string)
}

// NewMockLLM creates a new mock LLM with default configuration
func NewMockLLM() *MockLLM {
	return &MockLLM{
		ResponseMap:     make(map[string]string),
		ResponseQueue:   make([]string, 0),
		IntentResponses: make(map[string]*orchestrator.TaskIntent),
		CallHistory:     make([]CallRecord, 0),
		DefaultResponse: "Mock response",
	}
}

// WithResponseMap sets pattern-based responses
func (m *MockLLM) WithResponseMap(responses map[string]string) *MockLLM {
	m.ResponseMap = responses
	return m
}

// WithResponseQueue sets a queue of responses to return in order
func (m *MockLLM) WithResponseQueue(responses ...string) *MockLLM {
	m.ResponseQueue = responses
	return m
}

// WithLatency sets simulated latency
func (m *MockLLM) WithLatency(latency time.Duration) *MockLLM {
	m.SimulateLatency = latency
	return m
}

// WithErrorRate sets the probability of errors
func (m *MockLLM) WithErrorRate(rate float64) *MockLLM {
	m.ErrorRate = rate
	return m
}

// WithIntentResponse configures intent analysis responses
func (m *MockLLM) WithIntentResponse(query string, intent *orchestrator.TaskIntent) *MockLLM {
	m.IntentResponses[query] = intent
	return m
}

// Call implements the LLM interface for basic text generation
func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextContent{Text: prompt},
			},
		},
	}

	resp, err := m.GenerateContent(ctx, messages, options...)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Content, nil
	}

	return "", errors.New("no response generated")
}

// GenerateContent implements the LLM interface for message-based generation
func (m *MockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate latency if configured
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	// Extract the prompt from messages
	prompt := extractPrompt(messages)

	// Record the call
	record := CallRecord{
		Timestamp: time.Now(),
		Input:     prompt,
		Messages:  messages,
	}

	// Call callback if set
	if m.OnCall != nil {
		m.OnCall(messages)
	}

	// Simulate errors based on error rate
	if m.ErrorRate > 0 && rand.Float64() < m.ErrorRate {
		err := errors.New(m.ErrorMessage)
		if err.Error() == "" {
			err = errors.New("simulated error")
		}
		record.Error = err
		m.CallHistory = append(m.CallHistory, record)
		return nil, err
	}

	// Generate response
	response := m.generateResponse(prompt)
	record.Output = response
	m.CallHistory = append(m.CallHistory, record)
	m.callCount++

	// Call generate callback if set
	if m.OnGenerate != nil {
		m.OnGenerate(response)
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: response,
			},
		},
	}, nil
}

// generateResponse determines the appropriate response based on configuration
func (m *MockLLM) generateResponse(prompt string) string {
	// Check if this is an intent analysis request
	if strings.Contains(prompt, "Analyze the following user query") {
		return m.generateIntentResponse(prompt)
	}

	// Check response queue first
	if len(m.ResponseQueue) > 0 && m.callCount < len(m.ResponseQueue) {
		return m.ResponseQueue[m.callCount]
	}

	// Check pattern-based responses
	for pattern, response := range m.ResponseMap {
		if strings.Contains(prompt, pattern) {
			return response
		}
	}

	// Check for agent-specific patterns and format appropriately
	if strings.Contains(prompt, "You have access to the following tools") {
		// This is an agent call, format response appropriately
		return "I need to help the user.\n\nFinal Answer: " + m.DefaultResponse
	}

	// Return default response
	return m.DefaultResponse
}

// generateIntentResponse creates a JSON response for intent analysis
func (m *MockLLM) generateIntentResponse(prompt string) string {
	// Extract the user query from the prompt
	lines := strings.Split(prompt, "\n")
	var userQuery string
	for i, line := range lines {
		if strings.HasPrefix(line, "User Query:") {
			userQuery = strings.TrimSpace(strings.TrimPrefix(line, "User Query:"))
			break
		} else if i == len(lines)-1 {
			// Last line might be the query
			userQuery = strings.TrimSpace(line)
		}
	}

	// Check if we have a configured response for this query
	if intent, ok := m.IntentResponses[userQuery]; ok {
		data, _ := json.Marshal(map[string]interface{}{
			"type":                  intent.Type,
			"confidence":            intent.Confidence,
			"required_capabilities": intent.RequiredCapabilities,
			"reasoning":             "Mock intent analysis",
		})
		return string(data)
	}

	// Generate default intent based on keywords
	intentType := "reasoning"
	confidence := 0.8
	capabilities := []string{"general"}

	// More specific intent detection for test scenarios
	if strings.Contains(strings.ToLower(userQuery), "list files") ||
		strings.Contains(strings.ToLower(userQuery), "bash") ||
		strings.Contains(strings.ToLower(userQuery), "run") ||
		strings.Contains(strings.ToLower(userQuery), "save it to") ||
		strings.Contains(strings.ToLower(userQuery), "write back") ||
		strings.Contains(strings.ToLower(userQuery), "read package.json") ||
		strings.Contains(strings.ToLower(userQuery), "modify version") {
		intentType = "tool_use"
		capabilities = []string{"file", "bash"}
		confidence = 0.95
	} else if strings.Contains(strings.ToLower(userQuery), "write a function") ||
		strings.Contains(strings.ToLower(userQuery), "write a http server") ||
		strings.Contains(strings.ToLower(userQuery), "code") ||
		strings.Contains(strings.ToLower(userQuery), "function") ||
		strings.Contains(strings.ToLower(userQuery), "implement") {
		intentType = "code_generation"
		capabilities = []string{"coding"}
		confidence = 0.9
	} else if strings.Contains(strings.ToLower(userQuery), "find all go files") ||
		strings.Contains(strings.ToLower(userQuery), "analyze the codebase") ||
		strings.Contains(strings.ToLower(userQuery), "search") ||
		strings.Contains(strings.ToLower(userQuery), "find") {
		intentType = "search"
		capabilities = []string{"search"}
		confidence = 0.85
	} else if strings.Contains(strings.ToLower(userQuery), "create a new go module") ||
		strings.Contains(strings.ToLower(userQuery), "plan") {
		intentType = "planning"
		capabilities = []string{"planning"}
		confidence = 0.8
	}

	data, _ := json.Marshal(map[string]interface{}{
		"type":                  intentType,
		"confidence":            confidence,
		"required_capabilities": capabilities,
		"reasoning":             "Mock intent analysis based on keywords",
	})

	return string(data)
}

// GetCallHistory returns the call history
func (m *MockLLM) GetCallHistory() []CallRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]CallRecord{}, m.CallHistory...)
}

// GetCallCount returns the number of calls made
func (m *MockLLM) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// Reset clears the call history and resets counters
func (m *MockLLM) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallHistory = make([]CallRecord, 0)
	m.callCount = 0
}

// extractPrompt extracts the prompt text from messages
func extractPrompt(messages []llms.MessageContent) string {
	var parts []string
	for _, msg := range messages {
		for _, part := range msg.Parts {
			switch p := part.(type) {
			case llms.TextContent:
				parts = append(parts, p.Text)
			// Handle other content types as needed
			default:
				// Handle other content types by converting to string
				parts = append(parts, fmt.Sprintf("%v", p))
			}
		}
	}
	return strings.Join(parts, "\n")
}

// CreateModel is not implemented for the mock
func (m *MockLLM) CreateEmbedding(ctx context.Context, inputTexts []string) ([][]float32, error) {
	return nil, errors.New("embeddings not supported in mock")
}
