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

// ReactMode defines the ReAct agent's operating mode
type ReactMode string

const (
	ReactExecuteMode ReactMode = "execute"
	ReactPlanMode    ReactMode = "plan"
)

// ReactAgent implements a ReAct pattern agent with visible reasoning
type ReactAgent struct {
	llm        llms.Model
	tools      []tools.Tool
	controller *react.Controller
	memory     *memory.Memory
	stream     stream.Handler
	mode       ReactMode
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
		mode:       ReactExecuteMode,
	}

	// Setup the controller with initial mode
	if err := agent.setupController(); err != nil {
		return nil, fmt.Errorf("failed to setup controller: %w", err)
	}

	return agent, nil
}

// setupController configures the controller with the appropriate mode
func (a *ReactAgent) setupController() error {
	// Convert our ReactMode to react.Mode and set on controller
	if a.mode == ReactExecuteMode {
		a.controller.SetMode(react.ExecuteMode)
	} else {
		a.controller.SetMode(react.PlanMode)
	}

	return nil
}

// SetMode changes the operating mode
func (a *ReactAgent) SetMode(mode ReactMode) error {
	if a.mode != mode {
		a.mode = mode
		return a.setupController()
	}
	return nil
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

// GetMode returns the current operating mode
func (a *ReactAgent) GetMode() ReactMode {
	return a.mode
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
