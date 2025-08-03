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

// EnhancedClient provides full LangChain integration for Ryan
type EnhancedClient struct {
	llm           llms.Model
	memory        schema.Memory
	toolRegistry  *tools.Registry
	langchainTools []langchaintools.Tool
	agent         agents.Agent
	executor      *agents.Executor
	config        *config.Config
	log           *logger.Logger
}

// NewEnhancedClient creates a new LangChain-powered client
func NewEnhancedClient(baseURL, model string, toolRegistry *tools.Registry) (*EnhancedClient, error) {
	cfg := config.Get()
	log := logger.WithComponent("langchain_enhanced")

	// Create Ollama LLM
	llm, err := ollama.New(
		ollama.WithServerURL(baseURL),
		ollama.WithModel(model),
	)
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

	client := &EnhancedClient{
		llm:          llm,
		memory:       mem,
		toolRegistry: toolRegistry,
		config:       cfg,
		log:          log,
	}

	// Initialize tools and agent if enabled
	if cfg.LangChain.Tools.UseAgentFramework && toolRegistry != nil {
		if err := client.initializeAgent(); err != nil {
			log.Warn("Failed to initialize agent, falling back to direct mode", "error", err)
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
func (ec *EnhancedClient) initializeAgent() error {
	if ec.toolRegistry == nil {
		return fmt.Errorf("no tool registry available")
	}

	// Convert Ryan tools to LangChain tools
	ryanTools := ec.toolRegistry.GetTools()
	ec.langchainTools = make([]langchaintools.Tool, 0, len(ryanTools))

	for _, tool := range ryanTools {
		adapter := NewToolAdapter(tool)
		ec.langchainTools = append(ec.langchainTools, adapter)
		ec.log.Debug("Adapted tool", "name", tool.Name())
	}

	// Create conversational agent
	ec.agent = agents.NewConversationalAgent(ec.llm, ec.langchainTools,
		agents.WithMemory(ec.memory))

	// Create executor
	ec.executor = agents.NewExecutor(ec.agent)

	ec.log.Info("LangChain agent initialized", "tools_count", len(ec.langchainTools))
	return nil
}

// SendMessage sends a message using LangChain chains or agents
func (ec *EnhancedClient) SendMessage(ctx context.Context, userInput string) (string, error) {
	ec.log.Debug("Processing message", "input_length", len(userInput), "agent_enabled", ec.executor != nil)

	// Use agent if available and enabled
	if ec.executor != nil && ec.config.LangChain.Tools.UseAgentFramework {
		return ec.sendWithAgent(ctx, userInput)
	}

	// Use conversation chain for direct LLM interaction
	return ec.sendWithChain(ctx, userInput)
}

// sendWithAgent uses the LangChain agent for autonomous tool calling
func (ec *EnhancedClient) sendWithAgent(ctx context.Context, userInput string) (string, error) {
	ec.log.Debug("Using agent framework for message processing")

	result, err := ec.executor.Call(ctx, map[string]any{
		"input": userInput,
	})
	if err != nil {
		return "", fmt.Errorf("agent execution failed: %w", err)
	}

	// Extract the final output
	if output, ok := result["output"].(string); ok {
		return output, nil
	}

	return fmt.Sprintf("%v", result), nil
}

// sendWithChain uses conversation chain for direct LLM interaction
func (ec *EnhancedClient) sendWithChain(ctx context.Context, userInput string) (string, error) {
	ec.log.Debug("Using conversation chain for message processing")

	// Convert input to LangChain message format
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, userInput),
	}

	// Add memory context if available
	if ec.memory != nil {
		memoryVars, err := ec.memory.LoadMemoryVariables(ctx, map[string]any{})
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
	response, err := ec.llm.GenerateContent(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("content generation failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices available")
	}

	result := response.Choices[0].Content

	// Save to memory
	if ec.memory != nil {
		ec.memory.SaveContext(ctx, 
			map[string]any{"input": userInput},
			map[string]any{"output": result},
		)
	}

	return result, nil
}

// StreamMessage provides streaming responses using LangChain's streaming
func (ec *EnhancedClient) StreamMessage(ctx context.Context, userInput string, outputChan chan<- string) error {
	ec.log.Debug("Starting streaming message processing")

	// Convert input to LangChain message format
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, userInput),
	}

	// Add memory context if available
	if ec.memory != nil {
		memoryVars, err := ec.memory.LoadMemoryVariables(ctx, map[string]any{})
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
	_, err := ec.llm.GenerateContent(ctx, messages,
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			select {
			case outputChan <- string(chunk):
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
	if ec.memory != nil {
		// Note: In a real implementation, you'd want to accumulate the full response
		// and save both user input and AI response to memory
		ec.memory.SaveContext(ctx, 
			map[string]any{"input": userInput},
			map[string]any{"output": ""}, // Placeholder - in real implementation, accumulate chunks
		)
	}

	return nil
}

// GetMemory returns the current conversation memory
func (ec *EnhancedClient) GetMemory() schema.Memory {
	return ec.memory
}

// GetTools returns the available LangChain tools
func (ec *EnhancedClient) GetTools() []langchaintools.Tool {
	return ec.langchainTools
}

// ClearMemory clears the conversation memory
func (ec *EnhancedClient) ClearMemory(ctx context.Context) error {
	if ec.memory != nil {
		return ec.memory.Clear(ctx)
	}
	return nil
}

// WithPromptTemplate creates a response using a custom prompt template
func (ec *EnhancedClient) WithPromptTemplate(ctx context.Context, templateStr string, vars map[string]any) (string, error) {
	template := prompts.NewPromptTemplate(templateStr, extractVarNames(vars))
	
	prompt, err := template.Format(vars)
	if err != nil {
		return "", fmt.Errorf("template formatting failed: %w", err)
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	response, err := ec.llm.GenerateContent(ctx, messages)
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