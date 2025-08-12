package agent

import (
	"context"
	"fmt"

	"github.com/killallgit/ryan/pkg/agent/react"
	"github.com/killallgit/ryan/pkg/memory"
	"github.com/killallgit/ryan/pkg/stream"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// ReactAgent implements a ReAct pattern agent with visible reasoning
type ReactAgent struct {
	llm        llms.Model
	tools      []tools.Tool
	controller *react.Controller
	memory     *memory.Memory
	stream     stream.Handler
}

// NewReactAgent creates a new ReAct pattern agent
func NewReactAgent(llm llms.Model, toolList []tools.Tool, memory *memory.Memory, stream stream.Handler) (*ReactAgent, error) {
	// Create controller with options
	controllerOpts := []react.Option{
		react.WithMaxIterations(5),
	}
	if stream != nil {
		controllerOpts = append(controllerOpts, react.WithStreamHandler(stream))
	}

	controller := react.NewController(llm, toolList, controllerOpts...)

	agent := &ReactAgent{
		llm:        llm,
		tools:      toolList,
		controller: controller,
		memory:     memory,
		stream:     stream,
	}

	return agent, nil
}

// SetCustomPrompt sets a custom system prompt
func (a *ReactAgent) SetCustomPrompt(customPrompt string) {
	if a.controller != nil && a.controller.GetPromptBuilder() != nil {
		a.controller.GetPromptBuilder().SetCustomPrompt(customPrompt)
	}
}

// Execute runs the agent with the given input
func (a *ReactAgent) Execute(ctx context.Context, input string) (string, error) {
	// Add user message to memory if available
	if a.memory != nil && a.memory.IsEnabled() {
		if err := a.memory.AddUserMessage(input); err != nil {
			return "", fmt.Errorf("failed to add user message to memory: %w", err)
		}
	}

	// Execute through the controller
	result, err := a.controller.Execute(ctx, input)
	if err != nil {
		return "", fmt.Errorf("agent execution failed: %w", err)
	}

	// Add assistant response to memory if available
	if a.memory != nil && a.memory.IsEnabled() {
		if err := a.memory.AddAssistantMessage(result); err != nil {
			return "", fmt.Errorf("failed to add assistant message to memory: %w", err)
		}
	}

	return result, nil
}

// ExecuteStream runs the agent with streaming output
func (a *ReactAgent) ExecuteStream(ctx context.Context, input string, handler stream.Handler) error {
	// Create a new controller with the streaming handler
	controllerOpts := []react.Option{
		react.WithMaxIterations(5),
		react.WithStreamHandler(handler),
	}

	controller := react.NewController(a.llm, a.tools, controllerOpts...)

	// Execute with the streaming controller
	result, err := controller.Execute(ctx, input)
	if err != nil {
		handler.OnError(err)
		return err
	}

	// Send completion
	return handler.OnComplete(result)
}

// ClearMemory clears the conversation memory
func (a *ReactAgent) ClearMemory() error {
	if a.memory != nil {
		return a.memory.Clear()
	}
	return nil
}

// Close cleans up the agent resources
func (a *ReactAgent) Close() error {
	if a.memory != nil {
		return a.memory.Close()
	}
	return nil
}

// GetTokenStats returns token usage statistics
func (a *ReactAgent) GetTokenStats() (int, int) {
	// TODO: Implement token tracking
	return 0, 0
}

// Ensure ReactAgent implements Agent interface
var _ Agent = (*ReactAgent)(nil)
