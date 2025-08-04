package tools

import (
	"fmt"
)

// ConvertToProvider converts a tool definition to a provider-specific format
func ConvertToProvider(tool Tool, provider string) (map[string]any, error) {
	switch provider {
	case "openai", "ollama":
		return convertToOpenAI(tool), nil
	case "anthropic":
		return convertToAnthropic(tool), nil
	case "mcp":
		return convertToMCP(tool), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// convertToOpenAI converts a tool to OpenAI/Ollama format
// Format: {"type": "function", "function": {"name": ..., "description": ..., "parameters": ...}}
func convertToOpenAI(tool Tool) map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  tool.JSONSchema(),
		},
	}
}

// convertToAnthropic converts a tool to Anthropic format
// Format: {"name": ..., "description": ..., "input_schema": ...}
func convertToAnthropic(tool Tool) map[string]any {
	return map[string]any{
		"name":         tool.Name(),
		"description":  tool.Description(),
		"input_schema": tool.JSONSchema(),
	}
}

// convertToMCP converts a tool to MCP format
// Format: Similar to Anthropic but with additional MCP-specific metadata
func convertToMCP(tool Tool) map[string]any {
	return map[string]any{
		"name":        tool.Name(),
		"description": tool.Description(),
		"inputSchema": tool.JSONSchema(), // MCP uses camelCase
		"type":        "tool",
	}
}

// ConvertToolResult converts a ToolResult to provider-specific format
func ConvertToolResult(result ToolResult, provider string) (map[string]any, error) {
	switch provider {
	case "openai", "ollama":
		return convertResultToOpenAI(result), nil
	case "anthropic":
		return convertResultToAnthropic(result), nil
	case "mcp":
		return convertResultToMCP(result), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// convertResultToOpenAI converts ToolResult to OpenAI format
func convertResultToOpenAI(result ToolResult) map[string]any {
	if result.Success {
		return map[string]any{
			"content": result.Content,
			"role":    "tool",
		}
	}

	return map[string]any{
		"content": result.Error,
		"role":    "tool",
		"error":   true,
	}
}

// convertResultToAnthropic converts ToolResult to Anthropic format
func convertResultToAnthropic(result ToolResult) map[string]any {
	anthropicResult := map[string]any{
		"type":    "tool_result",
		"content": result.Content,
	}

	if !result.Success {
		anthropicResult["is_error"] = true
		anthropicResult["content"] = result.Error
	}

	return anthropicResult
}

// convertResultToMCP converts ToolResult to MCP format
func convertResultToMCP(result ToolResult) map[string]any {
	mcpResult := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": result.Content,
			},
		},
	}

	if !result.Success {
		mcpResult["isError"] = true
		mcpResult["content"] = []map[string]any{
			{
				"type": "text",
				"text": result.Error,
			},
		}
	}

	return mcpResult
}

// BatchConvertToProvider converts multiple tools to provider-specific format
func BatchConvertToProvider(tools []Tool, provider string) ([]map[string]any, error) {
	definitions := make([]map[string]any, 0, len(tools))

	for _, tool := range tools {
		definition, err := ConvertToProvider(tool, provider)
		if err != nil {
			return nil, fmt.Errorf("failed to convert tool %s: %w", tool.Name(), err)
		}
		definitions = append(definitions, definition)
	}

	return definitions, nil
}
