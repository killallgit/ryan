package langchain

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

// OllamaToolCaller handles native Ollama tool calling
type OllamaToolCaller struct {
	llm          llms.Model
	toolRegistry *tools.Registry
	log          *logger.Logger
}

// NewOllamaToolCaller creates a new Ollama tool caller
func NewOllamaToolCaller(llm llms.Model, toolRegistry *tools.Registry) *OllamaToolCaller {
	return &OllamaToolCaller{
		llm:          llm,
		toolRegistry: toolRegistry,
		log:          logger.WithComponent("ollama_tools"),
	}
}

// CallWithTools makes an LLM call with native Ollama tool support
func (otc *OllamaToolCaller) CallWithTools(ctx context.Context, messages []llms.MessageContent, progressCallback ToolProgressCallback) (string, error) {
	// Format tools for Ollama
	toolDefs := otc.formatToolsForOllama()

	otc.log.Debug("Calling Ollama with native tools",
		"message_count", len(messages),
		"tool_count", len(toolDefs))

	// Create a custom call that includes tools
	// Note: This is a workaround since langchaingo's Ollama doesn't expose tool calling directly
	// We'll need to use the raw API or enhance the wrapper

	// For now, we'll simulate tool calling by injecting tool context
	toolContext := otc.createToolContext()

	// Prepend tool context to messages
	messagesWithContext := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, toolContext),
	}
	messagesWithContext = append(messagesWithContext, messages...)

	// Call LLM
	response, err := otc.llm.GenerateContent(ctx, messagesWithContext)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices")
	}

	content := response.Choices[0].Content

	// Check if response contains tool calls
	if toolCalls := otc.extractToolCalls(content); len(toolCalls) > 0 {
		otc.log.Debug("Detected tool calls in response", "count", len(toolCalls))

		// Execute tools
		toolResults := make([]string, 0, len(toolCalls))
		for _, toolCall := range toolCalls {
			if progressCallback != nil {
				progressCallback(toolCall.Name, toolCall.Input)
			}

			result, err := otc.executeTool(ctx, toolCall)
			if err != nil {
				otc.log.Error("Tool execution failed",
					"tool", toolCall.Name,
					"error", err)
				toolResults = append(toolResults, fmt.Sprintf("Error executing %s: %v", toolCall.Name, err))
			} else {
				toolResults = append(toolResults, result)
			}
		}

		// Create a follow-up call with tool results
		return otc.callWithToolResults(ctx, messagesWithContext, content, toolResults)
	}

	return content, nil
}

// formatToolsForOllama converts tools to Ollama's expected format
func (otc *OllamaToolCaller) formatToolsForOllama() []map[string]any {
	if otc.toolRegistry == nil {
		return nil
	}

	toolDefs := make([]map[string]any, 0)

	for _, tool := range otc.toolRegistry.GetTools() {
		// Convert to Ollama/OpenAI format
		toolDef := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters":  tool.JSONSchema(),
			},
		}
		toolDefs = append(toolDefs, toolDef)
	}

	return toolDefs
}

// createToolContext creates a system message explaining available tools
func (otc *OllamaToolCaller) createToolContext() string {
	if otc.toolRegistry == nil || len(otc.toolRegistry.GetTools()) == 0 {
		return ""
	}

	context := "You have access to the following tools:\n\n"

	for _, tool := range otc.toolRegistry.GetTools() {
		context += fmt.Sprintf("Tool: %s\nDescription: %s\n", tool.Name(), tool.Description())

		// Add parameter info
		if schema := tool.JSONSchema(); schema != nil {
			if props, ok := schema["properties"].(map[string]any); ok {
				context += "Parameters:\n"
				for param, def := range props {
					if paramDef, ok := def.(map[string]any); ok {
						desc := ""
						if d, ok := paramDef["description"].(string); ok {
							desc = d
						}
						context += fmt.Sprintf("  - %s: %s\n", param, desc)
					}
				}
			}
		}
		context += "\n"
	}

	context += `IMPORTANT: To use a tool, you MUST respond with ONLY a JSON object in this exact format (no other text, no thinking blocks, no explanations):
{
  "tool_calls": [
    {
      "name": "tool_name",
      "arguments": {
        "param1": "value1",
        "param2": "value2"
      }
    }
  ]
}

Examples:
- To list files: {"tool_calls":[{"name":"execute_bash","arguments":{"command":"ls -la"}}]}
- To count files: {"tool_calls":[{"name":"execute_bash","arguments":{"command":"ls | wc -l"}}]}

Do NOT add any text before or after the JSON. Do NOT use thinking blocks. Just output the JSON directly.`

	return context
}

// ToolCall represents a tool invocation request
type ToolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Input     string         // For display purposes
}

// extractToolCalls extracts tool calls from LLM response
func (otc *OllamaToolCaller) extractToolCalls(content string) []ToolCall {
	otc.log.Debug("Extracting tool calls from content", "content_length", len(content))

	// Strip thinking blocks first if present
	cleanContent := content
	if strings.Contains(cleanContent, "<think>") {
		// Remove thinking blocks
		re := regexp.MustCompile(`(?s)<think>.*?</think>`)
		cleanContent = re.ReplaceAllString(cleanContent, "")
		cleanContent = strings.TrimSpace(cleanContent)
		otc.log.Debug("Removed thinking blocks", "cleaned_length", len(cleanContent))
	}

	// First, try to parse as JSON
	var response struct {
		ToolCalls []ToolCall `json:"tool_calls"`
	}

	// Look for JSON in the response - try multiple approaches
	jsonCandidates := []string{
		cleanContent, // Try the whole cleaned content
	}

	// Also try extracting JSON from the content
	if startIdx := strings.Index(cleanContent, "{"); startIdx >= 0 {
		if endIdx := strings.LastIndex(cleanContent, "}"); endIdx > startIdx {
			jsonStr := cleanContent[startIdx : endIdx+1]
			jsonCandidates = append(jsonCandidates, jsonStr)
		}
	}

	// Try to find compact JSON (single line format)
	lines := strings.Split(cleanContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			jsonCandidates = append(jsonCandidates, line)
		}
	}

	for _, candidate := range jsonCandidates {
		if err := json.Unmarshal([]byte(candidate), &response); err == nil && len(response.ToolCalls) > 0 {
			otc.log.Debug("Successfully parsed JSON tool calls", "count", len(response.ToolCalls))
			// Set display input for each tool call
			for i := range response.ToolCalls {
				if cmd, ok := response.ToolCalls[i].Arguments["command"].(string); ok {
					response.ToolCalls[i].Input = cmd
				} else if path, ok := response.ToolCalls[i].Arguments["path"].(string); ok {
					response.ToolCalls[i].Input = path
				}
				otc.log.Debug("Tool call parsed", "name", response.ToolCalls[i].Name, "input", response.ToolCalls[i].Input)
			}
			return response.ToolCalls
		}
	}

	// Fallback: Try to extract using patterns (for models that don't output proper JSON)
	otc.log.Debug("JSON parsing failed, trying pattern extraction")
	processor := NewOutputProcessor(true, true)
	if tool, command := processor.extractToolAndCommand(cleanContent); tool != "" && command != "" {
		otc.log.Debug("Pattern extraction succeeded", "tool", tool, "command", command)
		return []ToolCall{{
			Name: tool,
			Arguments: map[string]any{
				"command": command,
			},
			Input: command,
		}}
	}

	otc.log.Debug("No tool calls extracted from response")
	return nil
}

// executeTool executes a single tool call
func (otc *OllamaToolCaller) executeTool(ctx context.Context, toolCall ToolCall) (string, error) {
	tool, exists := otc.toolRegistry.Get(toolCall.Name)
	if !exists {
		return "", fmt.Errorf("tool %s not found", toolCall.Name)
	}

	// Execute the tool
	result, err := tool.Execute(ctx, toolCall.Arguments)
	if err != nil {
		return "", err
	}

	if !result.Success {
		return "", fmt.Errorf(result.Error)
	}

	return result.Content, nil
}

// callWithToolResults makes a follow-up call with tool execution results
func (otc *OllamaToolCaller) callWithToolResults(ctx context.Context, originalMessages []llms.MessageContent, _ string, toolResults []string) (string, error) {
	// Extract the original user question from messages
	var userQuestion string
	for _, msg := range originalMessages {
		if msg.Role == llms.ChatMessageTypeHuman {
			for _, part := range msg.Parts {
				if textPart, ok := part.(llms.TextContent); ok {
					userQuestion = textPart.Text
					break
				}
			}
			if userQuestion != "" {
				break
			}
		}
	}

	// Build a comprehensive message with tool results and formatting instructions
	resultMessage := fmt.Sprintf(`I executed tools to answer the user's question: "%s"

Tool Execution Results:
`, userQuestion)

	for i, result := range toolResults {
		resultMessage += fmt.Sprintf("Tool %d Output:\n%s\n\n", i+1, result)
	}

	resultMessage += `IMPORTANT: Please provide a natural, helpful response to the user that:
1. Acknowledges that you executed commands/tools to get this information
2. Presents the results in a clear, formatted way
3. Directly answers their question
4. Uses markdown formatting if appropriate (e.g., code blocks for command output)

Do NOT just return the raw tool output. Format it nicely and explain what it shows.`

	// Create a clean conversation with just the user question and tool results
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, userQuestion),
		llms.TextParts(llms.ChatMessageTypeSystem, resultMessage),
	}

	// Make final call
	response, err := otc.llm.GenerateContent(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("follow-up LLM call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices in follow-up")
	}

	return response.Choices[0].Content, nil
}

// CreateOllamaFunctionsAgent creates an agent that uses native Ollama tool calling
func CreateOllamaFunctionsAgent(llm llms.Model, toolRegistry *tools.Registry, memory schema.Memory) *OllamaFunctionsAgent {
	return &OllamaFunctionsAgent{
		toolCaller: NewOllamaToolCaller(llm, toolRegistry),
		memory:     memory,
		log:        logger.WithComponent("ollama_functions_agent"),
	}
}

// OllamaFunctionsAgent implements an agent using Ollama's native function calling
type OllamaFunctionsAgent struct {
	toolCaller *OllamaToolCaller
	memory     schema.Memory
	log        *logger.Logger
}

// Call executes the agent with native Ollama tool support
func (ofa *OllamaFunctionsAgent) Call(ctx context.Context, inputs map[string]any, progressCallback ToolProgressCallback) (string, error) {
	// Extract user input
	userInput, ok := inputs["input"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'input' in inputs")
	}

	// Build message history from memory
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, userInput),
	}

	// Add memory context if available
	if ofa.memory != nil {
		if memVars, err := ofa.memory.LoadMemoryVariables(ctx, inputs); err == nil {
			if history, ok := memVars["history"].(string); ok && history != "" {
				// Prepend history
				messages = append([]llms.MessageContent{
					llms.TextParts(llms.ChatMessageTypeSystem, fmt.Sprintf("Previous conversation:\n%s", history)),
				}, messages...)
			}
		}
	}

	// Call with tools
	response, err := ofa.toolCaller.CallWithTools(ctx, messages, progressCallback)
	if err != nil {
		return "", err
	}

	// Save to memory
	if ofa.memory != nil {
		ofa.memory.SaveContext(ctx,
			map[string]any{"input": userInput},
			map[string]any{"output": response},
		)
	}

	return response, nil
}
