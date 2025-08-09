package agent

import (
	"context"
	"fmt"

	"github.com/killallgit/ryan/pkg/memory"
	"github.com/killallgit/ryan/pkg/stream"
	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
	lcmemory "github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/tools"
)

// ExecutorAgent is a LangChain executor-based agent implementation
// It wraps LangChain's conversational agent with an executor for handling requests
type ExecutorAgent struct {
	llm      llms.Model
	executor *agents.Executor
	memory   *memory.Memory
	tools    []tools.Tool
}

// NewExecutorAgent creates a new executor-based agent with an injected LLM
func NewExecutorAgent(llm llms.Model) (*ExecutorAgent, error) {
	// Create memory with a session ID
	sessionID := "default"
	if viper.GetBool("continue") {
		sessionID = "continued"
	}

	mem, err := memory.New(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}

	// Initialize tools (empty for now, can add later)
	agentTools := []tools.Tool{}

	// Create the agent - using a conversational agent that can work without tools
	agent := agents.NewConversationalAgent(
		llm,
		agentTools,
	)

	// Create a LangChain memory wrapper around our SQLite history
	lcMem := lcmemory.NewConversationBuffer(
		lcmemory.WithChatHistory(mem.ChatMessageHistory()),
	)

	// Create executor with options
	maxIterations := viper.GetInt("langchain.tools.max_iterations")
	if maxIterations == 0 {
		maxIterations = 10
	}

	executor := agents.NewExecutor(
		agent,
		agents.WithMaxIterations(maxIterations),
		agents.WithMemory(lcMem),
	)

	return &ExecutorAgent{
		llm:      llm,
		executor: executor,
		memory:   mem,
		tools:    agentTools,
	}, nil
}

// Execute handles a request and returns a response
func (e *ExecutorAgent) Execute(ctx context.Context, prompt string) (string, error) {
	// The executor will handle memory management now
	// Just pass the input through
	input := map[string]any{
		"input": prompt,
	}

	// Execute through the agent (memory is handled by the executor)
	result, err := e.executor.Call(ctx, input)
	if err != nil {
		return "", fmt.Errorf("agent execution failed: %w", err)
	}

	// Extract the response
	response, ok := result["output"].(string)
	if !ok {
		// Try to get any string response from the result
		for _, v := range result {
			if str, ok := v.(string); ok {
				response = str
				break
			}
		}
		if response == "" {
			return "", fmt.Errorf("no valid response from agent")
		}
	}

	return response, nil
}

// ExecuteStream handles a request with streaming response
func (e *ExecutorAgent) ExecuteStream(ctx context.Context, prompt string, handler stream.Handler) error {
	// Create a LangChain streaming source using the agent's LLM
	source := stream.NewLangChainSource(e.llm)

	// Build conversation messages from memory
	messages := []stream.Message{}

	// Add conversation history if available
	if e.memory != nil {
		llmMessages, err := e.memory.ConvertToLLMMessages()
		if err == nil {
			for _, msg := range llmMessages {
				messages = append(messages, stream.Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
		}
	}

	// Add the current prompt
	messages = append(messages, stream.Message{
		Role:    "user",
		Content: prompt,
	})

	// Create a wrapper handler that also updates memory
	memoryHandler := &memoryUpdatingHandler{
		inner:  handler,
		memory: e.memory,
		prompt: prompt,
	}

	// Stream with full conversation history
	return source.StreamWithHistory(ctx, messages, memoryHandler)
}

// memoryUpdatingHandler wraps a stream handler to update memory
type memoryUpdatingHandler struct {
	inner   stream.Handler
	memory  *memory.Memory
	prompt  string
	content string
}

func (m *memoryUpdatingHandler) OnChunk(chunk string) error {
	m.content += chunk
	return m.inner.OnChunk(chunk)
}

func (m *memoryUpdatingHandler) OnComplete(finalContent string) error {
	if finalContent == "" {
		finalContent = m.content
	}

	// Update memory with the exchange
	if m.memory != nil {
		// Add user message
		_ = m.memory.AddUserMessage(m.prompt)

		// Add assistant response
		_ = m.memory.AddAssistantMessage(finalContent)
	}

	return m.inner.OnComplete(finalContent)
}

func (m *memoryUpdatingHandler) OnError(err error) {
	m.inner.OnError(err)
}

// GetLLM returns the underlying LLM for direct access if needed
func (e *ExecutorAgent) GetLLM() llms.Model {
	return e.llm
}

// GetMemory returns the memory instance
func (e *ExecutorAgent) GetMemory() *memory.Memory {
	return e.memory
}

// ClearMemory clears the conversation memory
func (e *ExecutorAgent) ClearMemory() error {
	if e.memory != nil {
		return e.memory.Clear()
	}
	return nil
}

// Close cleans up resources
func (e *ExecutorAgent) Close() error {
	if e.memory != nil {
		return e.memory.Close()
	}
	return nil
}
