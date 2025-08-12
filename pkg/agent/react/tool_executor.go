package react

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/tools"
)

// ToolExecutor handles tool execution
type ToolExecutor struct {
	tools   []tools.Tool
	toolMap map[string]tools.Tool
}

// NewToolExecutor creates a new tool executor
func NewToolExecutor(toolList []tools.Tool) *ToolExecutor {
	toolMap := make(map[string]tools.Tool)
	for _, tool := range toolList {
		toolMap[strings.ToLower(tool.Name())] = tool
		// Also map without underscores for flexibility
		toolMap[strings.ToLower(strings.ReplaceAll(tool.Name(), "_", ""))] = tool
	}

	return &ToolExecutor{
		tools:   toolList,
		toolMap: toolMap,
	}
}

// Execute runs the specified tool with the given input
func (te *ToolExecutor) Execute(ctx context.Context, toolName, input string) (string, error) {
	// Normalize tool name
	normalizedName := strings.ToLower(strings.TrimSpace(toolName))

	// Find the tool
	tool, exists := te.toolMap[normalizedName]
	if !exists {
		// Try without underscores
		normalizedName = strings.ReplaceAll(normalizedName, "_", "")
		tool, exists = te.toolMap[normalizedName]
		if !exists {
			return "", fmt.Errorf("tool not found: %s", toolName)
		}
	}

	// Prepare input - try to parse as JSON first
	var toolInput string
	if strings.HasPrefix(strings.TrimSpace(input), "{") {
		// Looks like JSON, use as-is
		toolInput = input
	} else {
		// Convert to JSON if the tool expects it
		// For now, we'll pass simple strings directly
		// Tools that expect JSON will handle conversion
		toolInput = input
	}

	// Execute the tool
	result, err := tool.Call(ctx, toolInput)
	if err != nil {
		return fmt.Sprintf("Error executing %s: %v", toolName, err), nil
	}

	return result, nil
}

// GetToolNames returns the names of available tools
func (te *ToolExecutor) GetToolNames() []string {
	names := make([]string, 0, len(te.tools))
	for _, tool := range te.tools {
		names = append(names, tool.Name())
	}
	return names
}

// GetToolDescriptions returns formatted tool descriptions
func (te *ToolExecutor) GetToolDescriptions() string {
	var descriptions []string
	for _, tool := range te.tools {
		descriptions = append(descriptions, fmt.Sprintf("%s: %s", tool.Name(), tool.Description()))
	}
	return strings.Join(descriptions, "\n")
}

// PrepareJSONInput converts various input formats to JSON
func (te *ToolExecutor) PrepareJSONInput(toolName string, input string) (string, error) {
	// If already JSON, return as-is
	if strings.HasPrefix(strings.TrimSpace(input), "{") {
		return input, nil
	}

	// For simple inputs, create a JSON object with appropriate key
	// This is a simplified version - can be enhanced based on tool requirements
	inputMap := make(map[string]interface{})

	// Check if input has key:value format
	if strings.Contains(input, ":") && !strings.Contains(input, "://") {
		// Parse key:value pairs
		pairs := strings.Split(input, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				inputMap[key] = value
			}
		}
	} else {
		// Single value - use appropriate key based on tool
		switch strings.ToLower(toolName) {
		case "file_read", "fileread":
			inputMap["path"] = input
		case "file_write", "filewrite":
			// For file_write, we'd need path and content
			// This is simplified - real implementation would parse properly
			inputMap["content"] = input
		case "bash":
			inputMap["command"] = input
		case "web_fetch", "webfetch":
			inputMap["url"] = input
		default:
			inputMap["input"] = input
		}
	}

	// Convert to JSON
	jsonBytes, err := json.Marshal(inputMap)
	if err != nil {
		return "", fmt.Errorf("failed to create JSON input: %w", err)
	}

	return string(jsonBytes), nil
}
