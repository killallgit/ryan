package react

import (
	"context"
	"fmt"

	"github.com/killallgit/ryan/pkg/stream"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// Controller manages the ReAct reasoning loop
type Controller struct {
	llm           llms.Model
	tools         []tools.Tool
	maxIters      int
	parser        *ResponseParser
	promptBuilder *PromptBuilder
	toolExecutor  *ToolExecutor
	stateManager  *StateManager
	decision      *DecisionMaker
	stream        stream.Handler
}

// NewController creates a new ReAct loop controller
func NewController(llm llms.Model, toolList []tools.Tool, options ...Option) *Controller {
	promptBuilder := NewPromptBuilder()
	promptBuilder.SetTools(toolList)

	c := &Controller{
		llm:           llm,
		tools:         toolList,
		maxIters:      5,
		parser:        NewResponseParser(),
		promptBuilder: promptBuilder,
		toolExecutor:  NewToolExecutor(toolList),
		stateManager:  NewStateManager(),
		decision:      NewDecisionMaker(),
	}

	// Apply options
	for _, opt := range options {
		opt(c)
	}

	return c
}

// Option configures the controller
type Option func(*Controller)

// WithMaxIterations sets the maximum number of reasoning iterations
func WithMaxIterations(n int) Option {
	return func(c *Controller) {
		c.maxIters = n
	}
}

// WithStreamHandler sets the streaming handler
func WithStreamHandler(h stream.Handler) Option {
	return func(c *Controller) {
		c.stream = h
	}
}

// SetPromptBuilder sets a custom prompt builder
func (c *Controller) SetPromptBuilder(pb *PromptBuilder) {
	c.promptBuilder = pb
}

// GetPromptBuilder returns the prompt builder for customization
func (c *Controller) GetPromptBuilder() *PromptBuilder {
	return c.promptBuilder
}

// Execute runs the ReAct loop for the given input
func (c *Controller) Execute(ctx context.Context, input string) (string, error) {
	// Initialize state
	c.stateManager.Reset()
	c.stateManager.SetInput(input)

	// Main ReAct loop
	for i := 0; i < c.maxIters; i++ {
		// Build prompt with current state
		prompt := c.promptBuilder.Build(c.stateManager.GetState())

		// Call LLM
		response, err := c.callLLM(ctx, prompt)
		if err != nil {
			return "", fmt.Errorf("LLM call failed: %w", err)
		}

		// Parse response to extract Thought/Action/Action Input
		parsed, err := c.parser.Parse(response)
		if err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}

		// Update state with parsed response
		c.stateManager.AddIteration(parsed)

		// Stream the thought if handler is available
		if c.stream != nil && parsed.Thought != "" {
			c.stream.OnChunk([]byte(fmt.Sprintf("🤔 **Thinking:** %s\n", parsed.Thought)))
		}

		// Check if we have a final answer
		if parsed.FinalAnswer != "" {
			if c.stream != nil {
				c.stream.OnChunk([]byte(fmt.Sprintf("✅ **Answer:** %s\n", parsed.FinalAnswer)))
			}
			return parsed.FinalAnswer, nil
		}

		// Execute action if provided
		if parsed.Action != "" {
			if c.stream != nil {
				c.stream.OnChunk([]byte(fmt.Sprintf("⚡ **Action:** %s\n", parsed.Action)))
				c.stream.OnChunk([]byte(fmt.Sprintf("📝 **Input:** %s\n", parsed.ActionInput)))
			}

			observation, err := c.toolExecutor.Execute(ctx, parsed.Action, parsed.ActionInput)
			if err != nil {
				observation = fmt.Sprintf("Error: %v", err)
			}

			if c.stream != nil {
				c.stream.OnChunk([]byte(fmt.Sprintf("👁️ **Observation:** %s\n", observation)))
			}

			c.stateManager.AddObservation(observation)
		}

		// Check if we should continue
		if shouldStop := c.decision.ShouldStop(c.stateManager.GetState()); shouldStop {
			break
		}
	}

	// If we've exhausted iterations without a final answer
	return c.stateManager.GetBestAnswer(), nil
}

// callLLM calls the language model with the given prompt
func (c *Controller) callLLM(ctx context.Context, prompt string) (string, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	completion, err := c.llm.GenerateContent(ctx, messages)
	if err != nil {
		return "", err
	}

	if completion == nil || len(completion.Choices) == 0 {
		return "", fmt.Errorf("no completion returned")
	}

	return completion.Choices[0].Content, nil
}
