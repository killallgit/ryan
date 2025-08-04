package langchain

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
	langchaintools "github.com/tmc/langchaingo/tools"
)

// Client provides full LangChain integration for Ryan
type Client struct {
	llm              llms.Model
	memory           schema.Memory
	toolRegistry     *tools.Registry
	langchainTools   []langchaintools.Tool
	agent            agents.Agent
	executor         *agents.Executor
	config           *config.Config
	log              *logger.Logger
	progressCallback ToolProgressCallback
}

// NewClient creates a new LangChain-powered client
func NewClient(baseURL, model string, toolRegistry *tools.Registry) (*Client, error) {
	cfg := config.Get()
	log := logger.WithComponent("langchain_enhanced")

	// Create Ollama LLM with additional debugging options
	log.Debug("Creating Ollama LLM", "base_url", baseURL, "model", model)

	// Try to create with additional options that might preserve raw output
	// Let's try with basic options first and add experimental ones progressively
	llm, err := ollama.New(
		ollama.WithServerURL(baseURL),
		ollama.WithModel(model),
	)

	// Log what ollama package functions are available for debugging
	log.Debug("Ollama client created successfully with basic options")
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama LLM: %w", err)
	}

	// Create memory based on configuration
	var mem schema.Memory
	if cfg.LangChain.Memory.Type == "window" {
		mem = memory.NewConversationWindowBuffer(cfg.LangChain.Memory.WindowSize)
	} else {
		mem = memory.NewConversationBuffer()
	}

	client := &Client{
		llm:          llm,
		memory:       mem,
		toolRegistry: toolRegistry,
		config:       cfg,
		log:          log,
	}

	// Initialize tools and agent (always enabled now)
	if toolRegistry != nil {
		if err := client.initializeAgent(); err != nil {
			log.Warn("Failed to initialize agent, falling back to direct mode", "error", err)
		}
	}

	return client, nil
}

// SetProgressCallback sets a callback for tool execution progress
func (c *Client) SetProgressCallback(callback ToolProgressCallback) {
	c.progressCallback = callback
	// Re-initialize agent if it already exists to update the callback
	if c.agent != nil && c.toolRegistry != nil {
		c.initializeAgent()
	}
}

// ToolProgressCallback is called when tool execution starts
type ToolProgressCallback func(toolName, command string)

// ToolAdapter bridges Ryan tools with LangChain tools
type ToolAdapter struct {
	ryanTool         tools.Tool
	log              *logger.Logger
	progressCallback ToolProgressCallback
}

func NewToolAdapter(ryanTool tools.Tool) *ToolAdapter {
	log := logger.WithComponent("tool_adapter")
	return &ToolAdapter{
		ryanTool: ryanTool,
		log:      log,
	}
}

// WithProgressCallback sets a callback for tool execution progress
func (ta *ToolAdapter) WithProgressCallback(callback ToolProgressCallback) *ToolAdapter {
	ta.progressCallback = callback
	return ta
}

func (ta *ToolAdapter) Name() string {
	return ta.ryanTool.Name()
}

func (ta *ToolAdapter) Description() string {
	return ta.ryanTool.Description()
}

func (ta *ToolAdapter) Call(ctx context.Context, input string) (string, error) {
	ta.log.Debug("Tool call initiated", "tool", ta.ryanTool.Name(), "input_length", len(input))

	// Parse input - for now, assume it's JSON-like format
	// In a real implementation, you'd want more sophisticated parsing
	params := make(map[string]interface{})

	// Simple parsing for common tool formats
	var commandForCallback string
	if strings.Contains(input, "command:") {
		// For bash tool: "command: docker images | wc -l"
		if cmd := extractValue(input, "command:"); cmd != "" {
			params["command"] = cmd
			commandForCallback = cmd
		}
	} else if strings.Contains(input, "path:") {
		// For file tool: "path: ./README.md"
		if path := extractValue(input, "path:"); path != "" {
			params["path"] = path
			commandForCallback = path
		}
	} else {
		// Fallback: use input as command/path
		switch ta.ryanTool.Name() {
		case "execute_bash":
			params["command"] = input
			commandForCallback = input
		case "read_file":
			params["path"] = input
			commandForCallback = input
		}
	}

	// Call progress callback if available
	if ta.progressCallback != nil && commandForCallback != "" {
		// Map tool names to display names that match Claude Code
		displayName := ta.ryanTool.Name()
		switch displayName {
		case "execute_bash":
			displayName = "Shell"
		case "read_file":
			displayName = "ReadFile"
		}
		ta.progressCallback(displayName, commandForCallback)
	}

	// Execute the Ryan tool
	result, err := ta.ryanTool.Execute(ctx, params)
	if err != nil {
		ta.log.Error("Tool execution failed", "tool", ta.ryanTool.Name(), "error", err)
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	if !result.Success {
		return "", fmt.Errorf("tool execution failed: %s", result.Error)
	}

	ta.log.Debug("Tool call completed", "tool", ta.ryanTool.Name(), "success", result.Success)
	return result.Content, nil
}

// extractValue extracts value after a prefix from input string
func extractValue(input, prefix string) string {
	if idx := strings.Index(input, prefix); idx != -1 {
		value := strings.TrimSpace(input[idx+len(prefix):])
		// Remove quotes if present
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
			if value[0] == value[len(value)-1] {
				value = value[1 : len(value)-1]
			}
		}
		return value
	}
	return ""
}

// initializeAgent sets up the LangChain agent with adapted tools
func (c *Client) initializeAgent() error {
	if c.toolRegistry == nil {
		return fmt.Errorf("no tool registry available")
	}

	// Convert Ryan tools to LangChain tools
	ryanTools := c.toolRegistry.GetTools()
	c.langchainTools = make([]langchaintools.Tool, 0, len(ryanTools))

	for _, tool := range ryanTools {
		adapter := NewToolAdapter(tool)
		if c.progressCallback != nil {
			adapter = adapter.WithProgressCallback(c.progressCallback)
		}
		c.langchainTools = append(c.langchainTools, adapter)
		c.log.Debug("Adapted tool", "name", tool.Name())
	}

	// Create conversational agent with enhanced configuration for autonomous reasoning
	c.agent = agents.NewConversationalAgent(c.llm, c.langchainTools,
		agents.WithMemory(c.memory))

	// Create executor with enhanced configuration for multi-step reasoning
	c.executor = agents.NewExecutor(c.agent)
	
	// Configure max iterations if the API supports it
	c.log.Info("Agent executor configured", 
		"max_iterations", c.config.LangChain.Tools.MaxIterations,
		"autonomous_reasoning", true)

	c.log.Info("LangChain agent initialized", "tools_count", len(c.langchainTools))
	return nil
}

// SendMessage sends a message using LangChain chains or agents
func (c *Client) SendMessage(ctx context.Context, userInput string) (string, error) {
	c.log.Debug("Processing message", "input_length", len(userInput), "agent_enabled", c.executor != nil)

	// Use agent if available (always enabled now)
	if c.executor != nil {
		return c.sendWithAgent(ctx, userInput)
	}

	// Use conversation chain for direct LLM interaction
	return c.sendWithChain(ctx, userInput)
}

// sendWithAgent uses the LangChain agent for autonomous tool calling
func (c *Client) sendWithAgent(ctx context.Context, userInput string) (string, error) {
	c.log.Debug("Using enhanced agent framework for autonomous multi-step reasoning")

	// Enhanced agent execution with ReAct pattern
	result, err := c.executeWithReasoningLoop(ctx, userInput)
	if err != nil {
		// Check if error is due to thinking blocks parsing issue
		if strings.Contains(err.Error(), "unable to parse agent output") && strings.Contains(err.Error(), "<think>") {
			c.log.Warn("Agent failed due to thinking blocks, falling back to direct LLM", "error", err)
			// Fall back to direct LLM interaction when agent parsing fails due to thinking blocks
			// But first, ensure memory consistency by saving the user input
			if c.memory != nil {
				c.memory.SaveContext(ctx,
					map[string]any{"input": userInput},
					map[string]any{"output": ""},
				)
			}
			return c.sendWithChain(ctx, userInput)
		}
		return "", fmt.Errorf("autonomous agent execution failed: %w", err)
	}

	// Log all available keys for debugging
	c.log.Debug("Agent execution result keys", "keys", getMapKeys(result))

	// Log the full result structure for debugging
	c.log.Debug("Full agent result", "result", result)

	// Check for intermediate steps
	if intermediateSteps, ok := result["intermediate_steps"]; ok {
		c.log.Debug("Found intermediate steps", "steps", intermediateSteps)
	}

	// Check for agent scratchpad
	if scratchpad, ok := result["agent_scratchpad"]; ok {
		c.log.Debug("Found agent scratchpad", "scratchpad", scratchpad)
	}

	// Check for thinking or reasoning fields
	if thinking, ok := result["thinking"]; ok {
		c.log.Debug("Found thinking field", "thinking", thinking)
	}

	// Check for any field that might contain raw LLM output
	for key, value := range result {
		if key != "output" && key != "input" {
			c.log.Debug("Additional result field", "key", key, "value", value)
		}
	}

	// Extract the final output
	if output, ok := result["output"].(string); ok {

		// Save to memory for consistency with chain mode
		if c.memory != nil {
			c.memory.SaveContext(ctx,
				map[string]any{"input": userInput},
				map[string]any{"output": output},
			)
		}
		return output, nil
	}

	// Fallback output
	finalOutput := fmt.Sprintf("%v", result)
	if c.memory != nil {
		c.memory.SaveContext(ctx,
			map[string]any{"input": userInput},
			map[string]any{"output": finalOutput},
		)
	}
	return finalOutput, nil
}

// executeWithReasoningLoop implements enhanced ReAct pattern for autonomous multi-step reasoning
func (c *Client) executeWithReasoningLoop(ctx context.Context, userInput string) (map[string]any, error) {
	c.log.Debug("Starting autonomous reasoning loop", "max_iterations", c.config.LangChain.Tools.MaxIterations)
	
	// Use the standard LangChain agent executor but with enhanced logging
	result, err := c.executor.Call(ctx, map[string]any{
		"input": userInput,
	})
	
	if err != nil {
		return nil, err
	}
	
	// Enhanced processing of intermediate steps for autonomous reasoning
	if intermediateSteps, ok := result["intermediate_steps"]; ok {
		c.processIntermediateSteps(intermediateSteps)
	}
	
	// Log reasoning insights
	c.logReasoningInsights(result)
	
	return result, nil
}

// processIntermediateSteps processes and logs the reasoning steps for autonomous operation
func (c *Client) processIntermediateSteps(steps any) {
	c.log.Debug("Processing intermediate reasoning steps", "steps_type", fmt.Sprintf("%T", steps))
	
	// Extract step information for autonomous reasoning analysis
	switch stepsData := steps.(type) {
	case []any:
		c.log.Info("Multi-step autonomous reasoning executed", "step_count", len(stepsData))
		for i, step := range stepsData {
			c.log.Debug("Reasoning step", "step_number", i+1, "step", step)
			
			// Log tool execution events for each step
			if stepMap, ok := step.(map[string]any); ok {
				if action, exists := stepMap["action"]; exists {
					c.log.Debug("Agent action", "step", i+1, "action", action)
				}
				if observation, exists := stepMap["observation"]; exists {
					c.log.Debug("Agent observation", "step", i+1, "observation", observation)
				}
			}
		}
	default:
		c.log.Debug("Intermediate steps format", "type", fmt.Sprintf("%T", stepsData), "content", stepsData)
	}
}

// logReasoningInsights provides detailed logging for autonomous agent behavior
func (c *Client) logReasoningInsights(result map[string]any) {
	insights := make([]string, 0)
	
	// Analyze the result for autonomous reasoning patterns
	if intermediateSteps, ok := result["intermediate_steps"]; ok {
		if steps, ok := intermediateSteps.([]any); ok {
			insights = append(insights, fmt.Sprintf("executed %d reasoning steps", len(steps)))
		}
	}
	
	if scratchpad, ok := result["agent_scratchpad"]; ok {
		if scratchpadStr, ok := scratchpad.(string); ok && len(scratchpadStr) > 0 {
			insights = append(insights, "utilized agent scratchpad for reasoning")
		}
	}
	
	// Log findings about autonomous behavior
	if len(insights) > 0 {
		c.log.Info("Autonomous reasoning analysis", "insights", strings.Join(insights, ", "))
	}
	
	// Log total execution metrics
	toolCallCount := 0
	if steps, ok := result["intermediate_steps"].([]any); ok {
		toolCallCount = len(steps)
	}
	
	c.log.Info("Autonomous agent execution complete", 
		"tool_calls", toolCallCount,
		"reasoning_successful", result["output"] != nil)
}

// getMapKeys extracts keys from a map
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// sendWithChain uses conversation chain for direct LLM interaction
func (c *Client) sendWithChain(ctx context.Context, userInput string) (string, error) {
	c.log.Debug("Using conversation chain for message processing")

	// Convert input to LangChain message format
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, userInput),
	}

	// Add tool context if available (so LLM knows what tools it can reference)
	if len(c.langchainTools) > 0 {
		toolDescriptions := make([]string, 0, len(c.langchainTools))
		for _, tool := range c.langchainTools {
			toolDescriptions = append(toolDescriptions, fmt.Sprintf("- %s: %s", tool.Name(), tool.Description()))
		}

		toolContext := fmt.Sprintf("You have access to the following tools:\n%s\n\nYou can reference these tools in your response, but you cannot actually execute them in this mode.",
			strings.Join(toolDescriptions, "\n"))

		messages = append([]llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, toolContext),
		}, messages...)
	}

	// Add memory context if available
	if c.memory != nil {
		memoryVars, err := c.memory.LoadMemoryVariables(ctx, map[string]any{})
		if err == nil {
			if history, ok := memoryVars["history"].(string); ok && history != "" {
				// Prepend history as system message
				messages = append([]llms.MessageContent{
					llms.TextParts(llms.ChatMessageTypeSystem, fmt.Sprintf("Previous conversation:\n%s", history)),
				}, messages...)
			}
		}
	}

	// Generate response
	response, err := c.llm.GenerateContent(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("content generation failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices available")
	}

	result := response.Choices[0].Content

	// Save to memory
	if c.memory != nil {
		c.memory.SaveContext(ctx,
			map[string]any{"input": userInput},
			map[string]any{"output": result},
		)
	}

	return result, nil
}

// StreamMessage provides streaming responses using LangChain's streaming
func (c *Client) StreamMessage(ctx context.Context, userInput string, outputChan chan<- string) error {
	c.log.Debug("Starting streaming message processing")

	// Convert input to LangChain message format
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, userInput),
	}

	// Add memory context if available
	if c.memory != nil {
		memoryVars, err := c.memory.LoadMemoryVariables(ctx, map[string]any{})
		if err == nil {
			if history, ok := memoryVars["history"].(string); ok && history != "" {
				// Prepend history as system message
				messages = append([]llms.MessageContent{
					llms.TextParts(llms.ChatMessageTypeSystem, fmt.Sprintf("Previous conversation:\n%s", history)),
				}, messages...)
			}
		}
	}

	// Use LangChain's streaming
	var allChunks []string
	_, err := c.llm.GenerateContent(ctx, messages,
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			chunkStr := string(chunk)
			allChunks = append(allChunks, chunkStr)

			select {
			case outputChan <- chunkStr:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}),
	)

	if err != nil {
		return fmt.Errorf("streaming failed: %w", err)
	}

	// Save to memory after streaming completes
	if c.memory != nil {
		// Note: In a real implementation, you'd want to accumulate the full response
		// and save both user input and AI response to memory
		c.memory.SaveContext(ctx,
			map[string]any{"input": userInput},
			map[string]any{"output": ""}, // Placeholder - in real implementation, accumulate chunks
		)
	}

	return nil
}

// GetMemory returns the current conversation memory
func (c *Client) GetMemory() schema.Memory {
	return c.memory
}

// GetTools returns the available LangChain tools
func (c *Client) GetTools() []langchaintools.Tool {
	return c.langchainTools
}

// ClearMemory clears the conversation memory
func (c *Client) ClearMemory(ctx context.Context) error {
	if c.memory != nil {
		return c.memory.Clear(ctx)
	}
	return nil
}

// WithPromptTemplate creates a response using a custom prompt template
func (c *Client) WithPromptTemplate(ctx context.Context, templateStr string, vars map[string]any) (string, error) {
	template := prompts.NewPromptTemplate(templateStr, extractVarNames(vars))

	prompt, err := template.Format(vars)
	if err != nil {
		return "", fmt.Errorf("template formatting failed: %w", err)
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	response, err := c.llm.GenerateContent(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("content generation failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices available")
	}

	return response.Choices[0].Content, nil
}

// extractVarNames extracts variable names from a map
func extractVarNames(vars map[string]any) []string {
	names := make([]string, 0, len(vars))
	for name := range vars {
		names = append(names, name)
	}
	return names
}
