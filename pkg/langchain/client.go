package langchain

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/models"
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
	
	// New fields for enhanced agent support
	agentSelector    *AgentSelector
	outputProcessor  *OutputProcessor
	ollamaToolCaller *OllamaToolCaller
	agentType        AgentType
	model            string
}

// NewClient creates a new LangChain-powered client
func NewClient(baseURL, model string, toolRegistry *tools.Registry) (*Client, error) {
	cfg := config.Get()
	log := logger.WithComponent("langchain_enhanced")

	// Create Ollama LLM with additional debugging options
	log.Debug("Creating Ollama LLM", "base_url", baseURL, "model", model)

	// Try to create with basic options - we'll handle thinking block prevention via better error handling
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
		model:        model,
	}
	
	// Initialize agent selector
	if toolRegistry != nil {
		client.agentSelector = NewAgentSelector(toolRegistry, model)
	}

	// Initialize tools and agent (always enabled now)
	if toolRegistry != nil {
		if err := client.initializeAgent(); err != nil {
			log.Error("Failed to initialize LangChain agent - tools will not work properly", "error", err)
			return nil, fmt.Errorf("failed to initialize LangChain agent: %w", err)
		}
		log.Info("LangChain agent initialized successfully", "tools_count", len(client.langchainTools))
	} else {
		log.Warn("No tool registry provided - agent will run without tools")
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

	c.log.Debug("Starting agent initialization", "registry_tools", len(c.toolRegistry.GetTools()))

	// Convert Ryan tools to LangChain tools
	ryanTools := c.toolRegistry.GetTools()
	if len(ryanTools) == 0 {
		return fmt.Errorf("no tools available in registry")
	}

	c.langchainTools = make([]langchaintools.Tool, 0, len(ryanTools))

	for _, tool := range ryanTools {
		c.log.Debug("Adapting tool for LangChain", "tool_name", tool.Name(), "tool_description", tool.Description())
		adapter := NewToolAdapter(tool)
		if c.progressCallback != nil {
			adapter = adapter.WithProgressCallback(c.progressCallback)
		}
		c.langchainTools = append(c.langchainTools, adapter)
		c.log.Debug("Successfully adapted tool", "name", tool.Name())
	}

	// Determine the best agent type for this model
	c.agentType = c.determineAgentType()
	c.log.Debug("Selected agent type", 
		"type", c.agentType,
		"model", c.model,
		"tools_count", len(c.langchainTools))

	// Create appropriate agent based on type
	switch c.agentType {
	case AgentTypeOllamaFunctions:
		// Initialize Ollama tool caller
		c.ollamaToolCaller = NewOllamaToolCaller(c.llm, c.toolRegistry)
		c.log.Info("Using native Ollama function calling")
		// Don't create traditional agent/executor for native calling
		return nil
		
	case AgentTypeConversational:
		// Use conversational agent with output processor
		c.outputProcessor = NewOutputProcessor(true, true) // Strip thinking blocks and convert to ReAct
		c.agent = agents.NewConversationalAgent(c.llm, c.langchainTools,
			agents.WithMemory(c.memory))
		c.log.Info("Using conversational agent with output processing")
		
	default:
		// Direct mode - no agent needed
		c.log.Info("Using direct LLM mode (no agent)")
		return nil
	}

	if c.agent == nil {
		return fmt.Errorf("failed to create agent")
	}

	c.log.Debug("Creating agent executor")

	// Create executor
	c.executor = agents.NewExecutor(c.agent)

	if c.executor == nil {
		return fmt.Errorf("failed to create agent executor")
	}

	// Configure max iterations if the API supports it
	c.log.Info("Agent executor configured successfully",
		"agent_type", c.agentType,
		"max_iterations", c.config.LangChain.Tools.MaxIterations,
		"tools_available", len(c.langchainTools))

	return nil
}

// determineAgentType selects the appropriate agent type based on model capabilities
func (c *Client) determineAgentType() AgentType {
	if c.agentSelector == nil {
		// Default to conversational if no selector
		return AgentTypeConversational
	}
	
	// For now, use a simplified determination
	// We'll use the selector more intelligently when processing actual queries
	modelInfo := models.GetModelInfo(c.model)
	
	// If model has excellent tool support and is Ollama-compatible, use native
	if modelInfo.ToolCompatibility == models.ToolCompatibilityExcellent {
		if c.isOllamaModel() {
			return AgentTypeOllamaFunctions
		}
	}
	
	// Default to conversational for other tool-capable models
	if modelInfo.ToolCompatibility != models.ToolCompatibilityNone {
		return AgentTypeConversational
	}
	
	// No tool support - use direct mode
	return AgentTypeDirect
}

// isOllamaModel checks if we're using an Ollama model
func (c *Client) isOllamaModel() bool {
	// Check if the LLM is an Ollama instance
	_, ok := c.llm.(*ollama.LLM)
	return ok
}

// SendMessage sends a message using LangChain chains or agents
func (c *Client) SendMessage(ctx context.Context, userInput string) (string, error) {
	c.log.Debug("Processing message", 
		"input_length", len(userInput), 
		"agent_type", c.agentType,
		"has_tools", c.toolRegistry != nil)

	// Determine if this specific query needs tools
	needsTools := false
	if c.agentSelector != nil {
		agentType, needsToolsForQuery := c.agentSelector.SelectAgent(userInput)
		needsTools = needsToolsForQuery
		c.log.Debug("Query analysis", 
			"needs_tools", needsTools,
			"suggested_agent", agentType,
			"configured_agent", c.agentType)
	}

	// Route to appropriate handler based on agent type and query needs
	switch c.agentType {
	case AgentTypeOllamaFunctions:
		if needsTools && c.ollamaToolCaller != nil {
			return c.sendWithNativeTools(ctx, userInput)
		}
		// Fall through to direct mode if no tools needed
		
	case AgentTypeConversational:
		if needsTools && c.executor != nil {
			return c.sendWithAgent(ctx, userInput)
		}
		// Fall through to direct mode if no tools needed
	}

	// Use direct LLM interaction (with tool context if available)
	return c.sendWithChain(ctx, userInput)
}

// sendWithNativeTools uses Ollama's native function calling
func (c *Client) sendWithNativeTools(ctx context.Context, userInput string) (string, error) {
	c.log.Debug("Using native Ollama tool calling")
	
	// Convert input to messages
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, userInput),
	}
	
	// Add memory context if available
	if c.memory != nil {
		memoryVars, err := c.memory.LoadMemoryVariables(ctx, map[string]any{})
		if err == nil {
			if history, ok := memoryVars["history"].(string); ok && history != "" {
				messages = append([]llms.MessageContent{
					llms.TextParts(llms.ChatMessageTypeSystem, fmt.Sprintf("Previous conversation:\n%s", history)),
				}, messages...)
			}
		}
	}
	
	// Call with tools
	response, err := c.ollamaToolCaller.CallWithTools(ctx, messages, c.progressCallback)
	if err != nil {
		c.log.Error("Native tool calling failed", "error", err)
		return "", fmt.Errorf("native tool calling failed: %w", err)
	}
	
	// Save to memory
	if c.memory != nil {
		c.memory.SaveContext(ctx,
			map[string]any{"input": userInput},
			map[string]any{"output": response},
		)
	}
	
	return response, nil
}

// sendWithAgent uses the LangChain agent for autonomous tool calling
func (c *Client) sendWithAgent(ctx context.Context, userInput string) (string, error) {
	c.log.Debug("Using enhanced agent framework for autonomous multi-step reasoning")

	// Enhanced agent execution with ReAct pattern
	result, err := c.executeWithReasoningLoop(ctx, userInput)
	if err != nil {
		// Check if error is due to thinking blocks parsing issue
		if strings.Contains(err.Error(), "unable to parse agent output") && strings.Contains(err.Error(), "<think>") {
			c.log.Error("TOOL EXECUTION FAILED: Agent failed due to thinking blocks, falling back to direct LLM mode (tools will not execute)",
				"error", err,
				"user_input", userInput,
				"fallback_mode", "direct_llm")
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
	c.log.Debug("Starting autonomous reasoning loop", 
		"max_iterations", c.config.LangChain.Tools.MaxIterations,
		"has_output_processor", c.outputProcessor != nil)

	// If we have an output processor, we need to intercept the agent's planning
	if c.outputProcessor != nil && c.agent != nil {
		// Create a wrapped agent that processes outputs
		wrappedAgent := &outputProcessingAgent{
			agent:     c.agent,
			processor: c.outputProcessor,
			log:       c.log,
		}
		
		// Create a temporary executor with the wrapped agent
		tempExecutor := agents.NewExecutor(wrappedAgent)
		
		result, err := tempExecutor.Call(ctx, map[string]any{
			"input": userInput,
		})
		
		if err != nil {
			// Check if it's still a thinking block issue after processing
			if strings.Contains(err.Error(), "unable to parse agent output") {
				c.log.Warn("Agent still failed to parse after processing, trying direct extraction")
				// Try one more time with aggressive extraction
				c.outputProcessor.convertToReAct = true
				result2, err2 := tempExecutor.Call(ctx, map[string]any{
					"input": userInput,
				})
				if err2 == nil {
					result = result2
					err = nil
				}
			}
		}
		
		if err != nil {
			return nil, err
		}
		
		// Process intermediate steps
		if intermediateSteps, ok := result["intermediate_steps"]; ok {
			c.processIntermediateSteps(intermediateSteps)
		}
		
		c.logReasoningInsights(result)
		return result, nil
	}

	// Fallback to standard executor
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

// outputProcessingAgent wraps an agent to process its outputs
type outputProcessingAgent struct {
	agent     agents.Agent
	processor *OutputProcessor
	log       *logger.Logger
}

func (opa *outputProcessingAgent) Plan(ctx context.Context, intermediateSteps []schema.AgentStep, inputs map[string]string) ([]schema.AgentAction, *schema.AgentFinish, error) {
	// First, get the original agent's plan
	actions, finish, err := opa.agent.Plan(ctx, intermediateSteps, inputs)
	
	// If there's an error, try to process the raw output
	if err != nil && strings.Contains(err.Error(), "unable to parse agent output") {
		opa.log.Debug("Agent parse error, attempting output processing", "error", err)
		
		// Try to extract the raw output from the error or context
		// This is a bit hacky but necessary to intercept thinking blocks
		if len(intermediateSteps) > 0 {
			lastStep := intermediateSteps[len(intermediateSteps)-1]
			// AgentStep.Observation is a string, not an interface
			processed := opa.processor.ProcessForAgent(lastStep.Observation)
			opa.log.Debug("Processed observation", 
				"original_len", len(lastStep.Observation),
				"processed_len", len(processed))
			
			// Try to parse again with processed output
			// This would require reimplementing part of the agent's parsing logic
			// For now, we'll return the original error
		}
	}
	
	return actions, finish, err
}

func (opa *outputProcessingAgent) GetInputKeys() []string {
	return opa.agent.GetInputKeys()
}

func (opa *outputProcessingAgent) GetOutputKeys() []string {
	return opa.agent.GetOutputKeys()
}

func (opa *outputProcessingAgent) GetTools() []langchaintools.Tool {
	return opa.agent.GetTools()
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

// StreamMessage provides streaming responses with tool-aware processing
func (c *Client) StreamMessage(ctx context.Context, userInput string, outputChan chan<- string) error {
	c.log.Debug("Starting tool-aware streaming message processing", 
		"input_length", len(userInput), 
		"agent_type", c.agentType,
		"has_tools", c.toolRegistry != nil)

	// Determine if this specific query needs tools using the same logic as SendMessage
	needsTools := false
	if c.agentSelector != nil {
		agentType, needsToolsForQuery := c.agentSelector.SelectAgent(userInput)
		needsTools = needsToolsForQuery
		c.log.Debug("Streaming query analysis", 
			"needs_tools", needsTools,
			"suggested_agent", agentType,
			"configured_agent", c.agentType)
	}

	// Route to appropriate streaming handler based on agent type and query needs
	switch c.agentType {
	case AgentTypeOllamaFunctions:
		if needsTools && c.ollamaToolCaller != nil {
			return c.streamWithNativeTools(ctx, userInput, outputChan)
		}
		// Fall through to direct streaming if no tools needed
		
	case AgentTypeConversational:
		if needsTools && c.executor != nil {
			return c.streamWithAgent(ctx, userInput, outputChan)
		}
		// Fall through to direct streaming if no tools needed
	}

	// Use direct LLM streaming (with tool context if available)
	return c.streamWithDirectLLM(ctx, userInput, outputChan)
}

// streamWithNativeTools streams using Ollama's native function calling
func (c *Client) streamWithNativeTools(ctx context.Context, userInput string, outputChan chan<- string) error {
	c.log.Debug("Using native Ollama tool calling for streaming")
	
	// For streaming with tools, we need to:
	// 1. Get the full response with tool execution first
	// 2. Stream the final response
	
	// This is a compromise - we can't truly stream tool execution, but we can
	// stream the final response after tools are executed
	response, err := c.sendWithNativeTools(ctx, userInput)
	if err != nil {
		return fmt.Errorf("native tool calling failed: %w", err)
	}
	
	// Stream the response in chunks to simulate real streaming
	return c.streamResponse(response, outputChan)
}

// streamWithAgent streams using LangChain agent with output processing
func (c *Client) streamWithAgent(ctx context.Context, userInput string, outputChan chan<- string) error {
	c.log.Debug("Using conversational agent for streaming")
	
	// Similar to native tools - get full response then stream it
	// This is because LangChain agents don't support true streaming with tool execution
	response, err := c.sendWithAgent(ctx, userInput)
	if err != nil {
		return fmt.Errorf("agent call failed: %w", err)
	}
	
	// Stream the response in chunks
	return c.streamResponse(response, outputChan)
}

// streamWithDirectLLM streams using direct LLM without tools
func (c *Client) streamWithDirectLLM(ctx context.Context, userInput string, outputChan chan<- string) error {
	c.log.Debug("Using direct LLM streaming")

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
		fullResponse := strings.Join(allChunks, "")
		c.memory.SaveContext(ctx,
			map[string]any{"input": userInput},
			map[string]any{"output": fullResponse},
		)
	}

	return nil
}

// streamResponse streams a complete response in chunks to simulate streaming
func (c *Client) streamResponse(response string, outputChan chan<- string) error {
	// Clean up thinking blocks and other unwanted content before streaming
	cleanResponse := c.cleanResponseForStreaming(response)
	
	// Split response into words for more natural streaming
	words := strings.Fields(cleanResponse)
	
	for i, word := range words {
		chunk := word
		if i < len(words)-1 {
			chunk += " " // Add space between words
		}
		
		select {
		case outputChan <- chunk:
			// Small delay to simulate natural streaming
			time.Sleep(10 * time.Millisecond)
		default:
			// Channel might be full or closed
			return nil
		}
	}
	
	return nil
}

// cleanResponseForStreaming removes unwanted content from responses before streaming
func (c *Client) cleanResponseForStreaming(response string) string {
	// Remove thinking blocks
	if strings.Contains(response, "<think>") {
		response = regexp.MustCompile(`(?s)<think>.*?</think>`).ReplaceAllString(response, "")
	}
	
	// Clean up extra whitespace
	response = strings.TrimSpace(response)
	
	// Remove empty lines
	lines := strings.Split(response, "\n")
	var cleanLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	
	return strings.Join(cleanLines, "\n")
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
