package agent

import (
	"context"
	"fmt"

	"github.com/killallgit/ryan/pkg/memory"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
	lcmemory "github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/tools"
)

// Orchestrator is the single agent that handles all requests
type Orchestrator struct {
	llm      llms.Model
	executor *agents.Executor
	memory   *memory.Memory
	tools    []tools.Tool
}

// NewOrchestrator creates a new orchestrator agent
func NewOrchestrator() (*Orchestrator, error) {
	// Create Ollama LLM
	ollamaClient := ollama.NewClient()

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
		ollamaClient.LLM,
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

	return &Orchestrator{
		llm:      ollamaClient.LLM,
		executor: executor,
		memory:   mem,
		tools:    agentTools,
	}, nil
}

// Execute handles a request and returns a response
func (o *Orchestrator) Execute(ctx context.Context, prompt string) (string, error) {
	// The executor will handle memory management now
	// Just pass the input through
	input := map[string]any{
		"input": prompt,
	}

	// Execute through the agent (memory is handled by the executor)
	result, err := o.executor.Call(ctx, input)
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
func (o *Orchestrator) ExecuteStream(ctx context.Context, prompt string, handler StreamHandler) error {
	// For now, we'll execute non-streaming and simulate streaming
	// This is because the agent executor doesn't support streaming directly
	response, err := o.Execute(ctx, prompt)
	if err != nil {
		handler.OnError(err)
		return err
	}

	// Simulate streaming by sending the response in chunks
	chunkSize := 10
	for i := 0; i < len(response); i += chunkSize {
		end := i + chunkSize
		if end > len(response) {
			end = len(response)
		}
		chunk := response[i:end]
		if err := handler.OnChunk(chunk); err != nil {
			return err
		}
	}

	return handler.OnComplete(response)
}

// StreamHandler handles streaming responses
type StreamHandler interface {
	OnChunk(chunk string) error
	OnComplete(finalContent string) error
	OnError(err error)
}

// GetLLM returns the underlying LLM for direct access if needed
func (o *Orchestrator) GetLLM() llms.Model {
	return o.llm
}

// GetMemory returns the memory instance
func (o *Orchestrator) GetMemory() *memory.Memory {
	return o.memory
}

// ClearMemory clears the conversation memory
func (o *Orchestrator) ClearMemory() error {
	if o.memory != nil {
		return o.memory.Clear()
	}
	return nil
}

// Close cleans up resources
func (o *Orchestrator) Close() error {
	if o.memory != nil {
		return o.memory.Close()
	}
	return nil
}
