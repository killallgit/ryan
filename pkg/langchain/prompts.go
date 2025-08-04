package langchain

import (
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/prompts"
)

const (
	// AgentSystemPrompt instructs the agent to show thinking and use tools
	AgentSystemPrompt = `You are a helpful AI assistant with access to tools. You should:

1. Show your reasoning process using <think>...</think> tags when the user has enabled thinking display
2. Use available tools when they can help answer questions
3. Be thorough in your analysis and responses

When you need to use a tool, follow this format:
Thought: I need to use a tool to help with this request.
Action: tool_name
Action Input: input_for_tool
Observation: [tool result will be inserted here]

You can use thinking blocks to show your reasoning:
<think>
Let me think through this step by step...
- First, I need to understand what the user is asking
- Then, I should consider which tools might be helpful
- Finally, I'll formulate my response
</think>

After using tools or thinking, provide your final response.`

	// ConversationalSuffix is the template for continuing conversations
	ConversationalSuffix = `Previous conversation history:
{{.history}}

New input: {{.input}}
{{.agent_scratchpad}}`

	// ThinkingEnabledPrefix adds thinking encouragement when show_thinking is true
	ThinkingEnabledPrefix = `You should show your thought process using <think>...</think> tags since thinking display is enabled.

`
)

// CreateAgentPrompt creates a custom agent prompt template
func CreateAgentPrompt(showThinking bool, toolNames []string) prompts.PromptTemplate {
	systemPrompt := AgentSystemPrompt
	
	if showThinking {
		systemPrompt = ThinkingEnabledPrefix + systemPrompt
	}
	
	if len(toolNames) > 0 {
		toolList := strings.Join(toolNames, ", ")
		systemPrompt += fmt.Sprintf("\n\nAvailable tools: %s", toolList)
	}
	
	// Create the full prompt template
	fullTemplate := systemPrompt + "\n\n" + ConversationalSuffix
	
	return prompts.NewPromptTemplate(
		fullTemplate,
		[]string{"history", "input", "agent_scratchpad"},
	)
}

// CreateThinkingPrompt creates a prompt specifically for thinking-enabled responses
func CreateThinkingPrompt(userInput string, showThinking bool) prompts.PromptTemplate {
	var template string
	
	if showThinking {
		template = `You should show your reasoning process using <think>...</think> tags.

<think>
Let me analyze this request step by step...
</think>

User: {{.input}}

Please provide a thoughtful response, showing your reasoning process in thinking tags.`
	} else {
		template = `User: {{.input}}

Please provide a helpful response.`
	}
	
	return prompts.NewPromptTemplate(template, []string{"input"})
}

// FormatAgentOutput formats agent output to preserve thinking blocks and tool usage
func FormatAgentOutput(result map[string]any, showThinking bool) string {
	var output strings.Builder
	
	// Extract and format agent scratchpad for tool usage visibility
	scratchpadContent := ""
	if scratchpad, ok := result["agent_scratchpad"].(string); ok {
		scratchpadContent = scratchpad
	}
	
	// Add thinking content if available and enabled
	if showThinking {
		if thinking, ok := result["thinking"].(string); ok && thinking != "" {
			output.WriteString(fmt.Sprintf("<think>\n%s\n</think>\n\n", thinking))
		}
		
		if scratchpadContent != "" {
			// Parse scratchpad for thinking content
			thoughts := extractThoughts(scratchpadContent)
			if thoughts != "" {
				output.WriteString(fmt.Sprintf("<think>\n%s\n</think>\n\n", thoughts))
			}
		}
	}
	
	// Always show tool usage (even if thinking is disabled) for transparency
	toolUsage := extractToolUsage(scratchpadContent)
	if toolUsage != "" {
		output.WriteString(fmt.Sprintf("**Tool Usage:**\n%s\n\n", toolUsage))
	}
	
	// Add intermediate steps if available
	if steps, ok := result["intermediate_steps"]; ok {
		formattedSteps := formatIntermediateSteps(steps, showThinking)
		if formattedSteps != "" {
			output.WriteString(formattedSteps)
		}
	}
	
	// Add the main output
	if mainOutput, ok := result["output"].(string); ok {
		// Clean the main output of agent format artifacts
		cleanOutput := cleanAgentFormatFromOutput(mainOutput)
		output.WriteString(cleanOutput)
	}
	
	return output.String()
}

// extractToolUsage extracts and formats tool usage information
func extractToolUsage(scratchpad string) string {
	if scratchpad == "" {
		return ""
	}
	
	var toolUsages []string
	lines := strings.Split(scratchpad, "\n")
	
	var currentTool string
	var currentInput string
	var currentObservation string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "Action:") {
			currentTool = strings.TrimSpace(strings.TrimPrefix(line, "Action:"))
		} else if strings.HasPrefix(line, "Action Input:") {
			currentInput = strings.TrimSpace(strings.TrimPrefix(line, "Action Input:"))
		} else if strings.HasPrefix(line, "Observation:") {
			currentObservation = strings.TrimSpace(strings.TrimPrefix(line, "Observation:"))
			
			// Format complete tool usage
			if currentTool != "" {
				usage := fmt.Sprintf("🔧 **%s**", currentTool)
				if currentInput != "" {
					usage += fmt.Sprintf("\n   Input: `%s`", currentInput)
				}
				if currentObservation != "" {
					// Truncate long observations
					obs := currentObservation
					if len(obs) > 200 {
						obs = obs[:200] + "..."
					}
					usage += fmt.Sprintf("\n   Result: %s", obs)
				}
				
				toolUsages = append(toolUsages, usage)
				
				// Reset for next tool
				currentTool = ""
				currentInput = ""
				currentObservation = ""
			}
		}
	}
	
	return strings.Join(toolUsages, "\n\n")
}

// formatIntermediateSteps formats intermediate steps for display
func formatIntermediateSteps(steps any, showThinking bool) string {
	stepSlice, ok := steps.([]any)
	if !ok || len(stepSlice) == 0 {
		return ""
	}
	
	var output strings.Builder
	
	if showThinking {
		output.WriteString("<think>\nAgent reasoning steps:\n")
		for i, step := range stepSlice {
			output.WriteString(fmt.Sprintf("Step %d: %v\n", i+1, step))
		}
		output.WriteString("</think>\n\n")
	} else {
		// Even without thinking, show a summary of steps taken
		output.WriteString(fmt.Sprintf("*Executed %d reasoning steps*\n\n", len(stepSlice)))
	}
	
	return output.String()
}

// cleanAgentFormatFromOutput removes agent format artifacts from the final output
func cleanAgentFormatFromOutput(output string) string {
	lines := strings.Split(output, "\n")
	var cleanLines []string
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip agent format lines
		if strings.HasPrefix(trimmed, "Thought:") ||
		   strings.HasPrefix(trimmed, "Action:") ||
		   strings.HasPrefix(trimmed, "Action Input:") ||
		   strings.HasPrefix(trimmed, "Observation:") {
			continue
		}
		
		// Keep AI: prefix lines but clean them
		if strings.HasPrefix(trimmed, "AI:") {
			cleaned := strings.TrimSpace(strings.TrimPrefix(trimmed, "AI:"))
			if cleaned != "" {
				cleanLines = append(cleanLines, cleaned)
			}
			continue
		}
		
		cleanLines = append(cleanLines, line)
	}
	
	result := strings.Join(cleanLines, "\n")
	return strings.TrimSpace(result)
}

// extractThoughts extracts thinking content from agent scratchpad
func extractThoughts(scratchpad string) string {
	var thoughts []string
	lines := strings.Split(scratchpad, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Thought:") {
			thought := strings.TrimPrefix(line, "Thought:")
			thought = strings.TrimSpace(thought)
			if thought != "" && !strings.Contains(thought, "I need to use a tool") {
				thoughts = append(thoughts, thought)
			}
		}
	}
	
	return strings.Join(thoughts, "\n")
}

// GetToolNames extracts tool names from a tool list
func GetToolNames(tools []interface{}) []string {
	var names []string
	for _, tool := range tools {
		if t, ok := tool.(interface{ Name() string }); ok {
			names = append(names, t.Name())
		}
	}
	return names
}