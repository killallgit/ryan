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
	llm            llms.Model
	memory         schema.Memory
	toolRegistry   *tools.Registry
	langchainTools []langchaintools.Tool
	agent          agents.Agent
	executor       *agents.Executor
	config         *config.Config
	log            *logger.Logger
	contextManager *ContextManager
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

	// Create context manager for persistent context
	contextManager, err := NewContextManager("default")
	if err != nil {
		log.Warn("Failed to create context manager", "error", err)
		// Continue without context manager
	}

	client := &Client{
		llm:            llm,
		memory:         mem,
		toolRegistry:   toolRegistry,
		config:         cfg,
		log:            log,
		contextManager: contextManager,
	}

	// Initialize tools and agent (always enabled now)
	if toolRegistry != nil {
		if err := client.initializeAgent(); err != nil {
			log.Warn("Failed to initialize agent, falling back to direct mode", "error", err)
		}
	}

	// Try to load existing context if context manager is available
	if client.contextManager != nil {
		if err := client.loadPersistedContext(context.Background()); err != nil {
			log.Debug("No existing context to load", "error", err)
		}
	}

	return client, nil
}

// ToolAdapter bridges Ryan tools with LangChain tools
type ToolAdapter struct {
	ryanTool tools.Tool
	log      *logger.Logger
}

func NewToolAdapter(ryanTool tools.Tool) *ToolAdapter {
	log := logger.WithComponent("tool_adapter")
	return &ToolAdapter{
		ryanTool: ryanTool,
		log:      log,
	}
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
	if strings.Contains(input, "command:") {
		// For bash tool: "command: docker images | wc -l"
		if cmd := extractValue(input, "command:"); cmd != "" {
			params["command"] = cmd
		}
	} else if strings.Contains(input, "path:") {
		// For file tool: "path: ./README.md"
		if path := extractValue(input, "path:"); path != "" {
			params["path"] = path
		}
	} else {
		// Fallback: use input as command/path
		switch ta.ryanTool.Name() {
		case "execute_bash":
			params["command"] = input
		case "read_file":
			params["path"] = input
		}
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
		c.langchainTools = append(c.langchainTools, adapter)
		c.log.Debug("Adapted tool", "name", tool.Name())
	}

	// Create conversational agent with custom prompt to avoid thinking blocks
	c.agent = agents.NewConversationalAgent(c.llm, c.langchainTools,
		agents.WithMemory(c.memory))

	c.executor = agents.NewExecutor(c.agent)
	// NOTE: Intermediate steps are extracted from result map when available (see SendMessageAsync)

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
	c.log.Debug("Using agent framework for message processing")

	result, err := c.executor.Call(ctx, map[string]any{
		"input": userInput,
	})
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
		return "", fmt.Errorf("agent execution failed: %w", err)
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
		// LOG AGENT OUTPUT DEBUGGING
		c.log.Debug("=== AGENT OUTPUT DEBUG ===")
		c.log.Debug("Agent result keys", "keys", getMapKeys(result))
		for key, value := range result {
			c.log.Debug("Agent result field", "key", key, "value_type", fmt.Sprintf("%T", value), "value", value)
		}
		c.log.Debug("=== END AGENT OUTPUT DEBUG ===")

		// Log raw agent output to check for thinking blocks
		c.log.Debug("Raw agent output", "content", output, "has_think_tags", strings.Contains(output, "<think"))

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

// getMapKeys extracts keys from a map for debugging
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

	// LOG FULL RESPONSE STRUCTURE FOR DEBUGGING
	c.log.Debug("=== FULL RESPONSE STRUCTURE DEBUG ===")
	c.log.Debug("Response choices count", "count", len(response.Choices))

	for i, choice := range response.Choices {
		c.log.Debug("Choice details", "index", i, "content_length", len(choice.Content))
		c.log.Debug("Choice content", "index", i, "content", choice.Content)

		// Log any other fields that might be available in the choice
		c.log.Debug("Choice struct inspection", "index", i, "choice_type", fmt.Sprintf("%T", choice))

		// Try to see if there are additional fields we're missing
		if choice.Content != "" {
			c.log.Debug("Choice has content", "index", i, "has_think_tags", strings.Contains(choice.Content, "<think"))
		}
	}

	// Log the entire response struct to see what else might be available
	c.log.Debug("Full response struct", "response_type", fmt.Sprintf("%T", response))
	c.log.Debug("=== END RESPONSE STRUCTURE DEBUG ===")

	result := response.Choices[0].Content

	// Log raw LLM output to check for thinking blocks
	c.log.Debug("Raw LLM output (chain mode)", "content", result, "has_think_tags", strings.Contains(result, "<think"))

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

			// LOG STREAMING CHUNK DEBUG
			c.log.Debug("=== STREAMING CHUNK DEBUG ===")
			c.log.Debug("Received chunk", "length", len(chunk), "content", chunkStr)
			c.log.Debug("Chunk has thinking", "has_think_tags", strings.Contains(chunkStr, "<think"))
			c.log.Debug("=== END STREAMING CHUNK DEBUG ===")

			select {
			case outputChan <- chunkStr:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}),
	)

	// Log accumulated chunks
	if len(allChunks) > 0 {
		accumulated := strings.Join(allChunks, "")
		c.log.Debug("=== ACCUMULATED STREAMING DEBUG ===")
		c.log.Debug("Total chunks received", "count", len(allChunks))
		c.log.Debug("Accumulated content", "length", len(accumulated), "content", accumulated)
		c.log.Debug("Accumulated has thinking", "has_think_tags", strings.Contains(accumulated, "<think"))
		c.log.Debug("=== END ACCUMULATED STREAMING DEBUG ===")
	}

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

// loadPersistedContext loads conversation context from persistent storage
func (c *Client) loadPersistedContext(ctx context.Context) error {
	if c.contextManager == nil {
		return fmt.Errorf("context manager not available")
	}

	contextData, err := c.contextManager.LoadContext(ctx)
	if err != nil {
		return err // Context doesn't exist yet, which is fine
	}

	c.log.Debug("Loaded persisted context",
		"session_id", contextData.SessionID,
		"message_count", len(contextData.Messages))

	// TODO: Integration with LangChain memory could be enhanced here
	// For now, we just log that we loaded the context
	// In the future, we could restore the conversation to the memory

	return nil
}

// SaveCurrentContext saves the current conversation state to persistent storage
func (c *Client) SaveCurrentContext(ctx context.Context) error {
	if c.contextManager == nil {
		return nil // Silently skip if context manager not available
	}

	// For now, we'll create a simple approach to extract messages
	// This is a simplified implementation - in a full implementation,
	// we'd want to extract actual messages from the memory
	messages := []llms.ChatMessage{} // Placeholder

	return c.contextManager.SaveContext(ctx, messages)
}

// ClearPersistedContext clears the persistent context storage
func (c *Client) ClearPersistedContext() error {
	if c.contextManager == nil {
		return nil
	}

	return c.contextManager.ClearContext()
}
