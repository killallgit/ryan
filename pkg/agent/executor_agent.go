package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/killallgit/ryan/pkg/memory"
	"github.com/killallgit/ryan/pkg/stream"
	"github.com/killallgit/ryan/pkg/tokens"
	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
	lcmemory "github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/tools"
)

// ExecutorAgent is a LangChain executor-based agent implementation
// It wraps LangChain's conversational agent with an executor for handling requests
type ExecutorAgent struct {
	llm          llms.Model
	executor     *agents.Executor
	memory       *memory.Memory
	tools        []tools.Tool
	tokenCounter *tokens.TokenCounter
	tokensSent   int
	tokensRecv   int
	tokensMu     sync.RWMutex
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

	// Initialize token counter
	modelName := viper.GetString("ollama.default_model")
	tokenCounter, err := tokens.NewTokenCounter(modelName)
	if err != nil {
		// Don't fail if token counter can't be initialized, just log warning
		fmt.Printf("Warning: Could not initialize token counter: %v\n", err)
		tokenCounter = nil
	}

	return &ExecutorAgent{
		llm:          llm,
		executor:     executor,
		memory:       mem,
		tools:        agentTools,
		tokenCounter: tokenCounter,
		tokensSent:   0,
		tokensRecv:   0,
	}, nil
}

// Execute handles a request and returns a response
func (e *ExecutorAgent) Execute(ctx context.Context, prompt string) (string, error) {
	// Count input tokens
	if e.tokenCounter != nil {
		inputTokens := e.tokenCounter.CountTokens(prompt)
		e.tokensMu.Lock()
		e.tokensSent += inputTokens
		e.tokensMu.Unlock()
	}

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

	// Count output tokens
	if e.tokenCounter != nil {
		outputTokens := e.tokenCounter.CountTokens(response)
		e.tokensMu.Lock()
		e.tokensRecv += outputTokens
		e.tokensMu.Unlock()
	}

	return response, nil
}

// ExecuteStream handles a request with streaming response
func (e *ExecutorAgent) ExecuteStream(ctx context.Context, prompt string, handler stream.Handler) error {
	// Count input tokens
	if e.tokenCounter != nil {
		inputTokens := e.tokenCounter.CountTokens(prompt)
		e.tokensMu.Lock()
		e.tokensSent += inputTokens
		e.tokensMu.Unlock()
	}

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

	// Create a wrapper handler that tracks tokens and updates memory
	tokenAndMemoryHandler := &tokenAndMemoryHandler{
		inner:      handler,
		memory:     e.memory,
		prompt:     prompt,
		agent:      e,
		buffer:     "",
		lastTokens: 0,
	}

	// Stream with full conversation history
	return source.StreamWithHistory(ctx, messages, tokenAndMemoryHandler)
}

// tokenAndMemoryHandler wraps a stream handler to track tokens and update memory
type tokenAndMemoryHandler struct {
	inner      stream.Handler
	memory     *memory.Memory
	prompt     string
	agent      *ExecutorAgent
	buffer     string
	lastTokens int
}

func (h *tokenAndMemoryHandler) OnChunk(chunk string) error {
	// Accumulate chunks for memory and token tracking
	h.buffer += chunk

	// Count tokens in accumulated buffer
	if h.agent.tokenCounter != nil {
		currentTokens := h.agent.tokenCounter.CountTokens(h.buffer)
		// Only update if token count changed
		if currentTokens > h.lastTokens {
			tokenDiff := currentTokens - h.lastTokens
			h.agent.tokensMu.Lock()
			h.agent.tokensRecv += tokenDiff
			h.agent.tokensMu.Unlock()
			h.lastTokens = currentTokens
		}
	}

	// Forward to original handler
	return h.inner.OnChunk(chunk)
}

func (h *tokenAndMemoryHandler) OnComplete(finalContent string) error {
	if finalContent == "" {
		finalContent = h.buffer
	}

	// Final token count (in case there's any discrepancy)
	if h.agent.tokenCounter != nil && finalContent != "" {
		finalTokens := h.agent.tokenCounter.CountTokens(finalContent)
		if finalTokens > h.lastTokens {
			tokenDiff := finalTokens - h.lastTokens
			h.agent.tokensMu.Lock()
			h.agent.tokensRecv += tokenDiff
			h.agent.tokensMu.Unlock()
		}
	}

	// Update memory with the exchange
	if h.memory != nil {
		// Add user message
		_ = h.memory.AddUserMessage(h.prompt)

		// Add assistant response
		_ = h.memory.AddAssistantMessage(finalContent)
	}

	return h.inner.OnComplete(finalContent)
}

func (h *tokenAndMemoryHandler) OnError(err error) {
	h.inner.OnError(err)
}

// GetLLM returns the underlying LLM for direct access if needed
func (e *ExecutorAgent) GetLLM() llms.Model {
	return e.llm
}

// GetMemory returns the memory instance
func (e *ExecutorAgent) GetMemory() *memory.Memory {
	return e.memory
}

// ClearMemory clears the conversation memory and resets token counts
func (e *ExecutorAgent) ClearMemory() error {
	// Reset token counts
	e.tokensMu.Lock()
	e.tokensSent = 0
	e.tokensRecv = 0
	e.tokensMu.Unlock()

	// Clear memory
	if e.memory != nil {
		return e.memory.Clear()
	}
	return nil
}

// GetTokenStats returns the cumulative token usage statistics
func (e *ExecutorAgent) GetTokenStats() (int, int) {
	e.tokensMu.RLock()
	defer e.tokensMu.RUnlock()
	return e.tokensSent, e.tokensRecv
}

// Close cleans up resources
func (e *ExecutorAgent) Close() error {
	if e.memory != nil {
		return e.memory.Close()
	}
	return nil
}
