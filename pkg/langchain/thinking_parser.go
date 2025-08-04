package langchain

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/logger"
)

// ThinkingOutputParser extracts and handles thinking blocks from LLM output
type ThinkingOutputParser struct {
	showThinking bool
	log          *logger.Logger
}

// ParsedOutput represents the parsed output with separated content
type ParsedOutput struct {
	Content       string
	Thinking      string
	HasThinking   bool
	ToolCalls     []chat.ToolCall
	OriginalText  string
}

// NewThinkingOutputParser creates a new thinking-aware parser
func NewThinkingOutputParser(showThinking bool) *ThinkingOutputParser {
	return &ThinkingOutputParser{
		showThinking: showThinking,
		log:          logger.WithComponent("thinking_parser"),
	}
}

// Parse implements the OutputParser interface
func (p *ThinkingOutputParser) Parse(text string) (ParsedOutput, error) {
	return p.parseOutput(text)
}

// ParseWithPrompt implements the OutputParser interface with prompt context
func (p *ThinkingOutputParser) ParseWithPrompt(text string, prompt string) (ParsedOutput, error) {
	// Could use prompt context for better parsing in the future
	return p.parseOutput(text)
}

// GetFormatInstructions returns instructions for the model
func (p *ThinkingOutputParser) GetFormatInstructions() string {
	return `You may use <think> tags to show your reasoning process. These will be handled appropriately.
Example:
<think>
I need to analyze this problem step by step...
</think>

Your actual response goes here.`
}

// Type returns the parser type
func (p *ThinkingOutputParser) Type() string {
	return "thinking_parser"
}

// parseOutput extracts thinking blocks and content
func (p *ThinkingOutputParser) parseOutput(text string) (ParsedOutput, error) {
	result := ParsedOutput{
		OriginalText: text,
		Content:      text,
	}

	// Extract thinking blocks
	thinkingBlocks, cleanContent := p.extractThinkingBlocks(text)
	
	if len(thinkingBlocks) > 0 {
		result.HasThinking = true
		result.Thinking = strings.Join(thinkingBlocks, "\n\n")
		result.Content = cleanContent
		
		p.log.Debug("Extracted thinking blocks", 
			"thinking_length", len(result.Thinking),
			"content_length", len(result.Content))
	}

	// Extract tool calls if present
	result.ToolCalls = p.extractToolCalls(result.Content)

	return result, nil
}

// extractThinkingBlocks finds and removes thinking blocks from text
func (p *ThinkingOutputParser) extractThinkingBlocks(text string) ([]string, string) {
	var thinkingBlocks []string
	cleanText := text

	// Pattern for <think>...</think> blocks (including multiline)
	thinkPattern := regexp.MustCompile(`(?s)<think>(.*?)</think>`)
	
	matches := thinkPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			thinking := strings.TrimSpace(match[1])
			if thinking != "" {
				thinkingBlocks = append(thinkingBlocks, thinking)
			}
		}
	}

	// Remove thinking blocks from content
	cleanText = thinkPattern.ReplaceAllString(text, "")
	cleanText = strings.TrimSpace(cleanText)

	// Extract LangChain agent "Thought:" patterns
	agentThoughts := p.extractAgentThoughts(cleanText)
	thinkingBlocks = append(thinkingBlocks, agentThoughts...)

	// Also check for alternative thinking patterns
	altPatterns := []string{
		`(?s)<thinking>(.*?)</thinking>`,
		`(?s)\[THINK\](.*?)\[/THINK\]`,
		`(?s)<!-- thinking -->(.*?)<!-- /thinking -->`,
	}

	for _, pattern := range altPatterns {
		altPattern := regexp.MustCompile(pattern)
		altMatches := altPattern.FindAllStringSubmatch(cleanText, -1)
		for _, match := range altMatches {
			if len(match) > 1 {
				thinking := strings.TrimSpace(match[1])
				if thinking != "" {
					thinkingBlocks = append(thinkingBlocks, thinking)
				}
			}
		}
		cleanText = altPattern.ReplaceAllString(cleanText, "")
	}

	// Clean up agent format patterns from the content
	cleanText = p.cleanAgentFormatFromContent(cleanText)

	return thinkingBlocks, strings.TrimSpace(cleanText)
}

// extractAgentThoughts extracts thinking content from LangChain agent format
func (p *ThinkingOutputParser) extractAgentThoughts(text string) []string {
	var thoughts []string
	lines := strings.Split(text, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Extract "Thought:" lines
		if strings.HasPrefix(line, "Thought:") {
			thought := strings.TrimPrefix(line, "Thought:")
			thought = strings.TrimSpace(thought)
			
			// Skip common agent framework thoughts
			if thought != "" && 
			   !strings.Contains(thought, "I need to use a tool") &&
			   !strings.Contains(thought, "Do I need to use a tool") &&
			   !strings.Contains(thought, "should be one of") {
				thoughts = append(thoughts, thought)
			}
		}
	}
	
	return thoughts
}

// cleanAgentFormatFromContent removes agent format artifacts from content
func (p *ThinkingOutputParser) cleanAgentFormatFromContent(text string) string {
	lines := strings.Split(text, "\n")
	var cleanLines []string
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip agent format lines but preserve AI response
		if strings.HasPrefix(trimmed, "Thought:") ||
		   strings.HasPrefix(trimmed, "Action:") ||
		   strings.HasPrefix(trimmed, "Action Input:") ||
		   strings.HasPrefix(trimmed, "Observation:") {
			continue
		}
		
		// Handle AI: prefix - extract the content but keep it
		if strings.HasPrefix(trimmed, "AI:") {
			aiContent := strings.TrimSpace(strings.TrimPrefix(trimmed, "AI:"))
			if aiContent != "" {
				cleanLines = append(cleanLines, aiContent)
			}
			continue
		}
		
		cleanLines = append(cleanLines, line)
	}
	
	return strings.TrimSpace(strings.Join(cleanLines, "\n"))
}

// extractToolCalls attempts to extract tool call patterns from content
func (p *ThinkingOutputParser) extractToolCalls(content string) []chat.ToolCall {
	var toolCalls []chat.ToolCall

	// Extract LangChain agent format tool calls first
	agentToolCalls := p.extractAgentToolCalls(content)
	toolCalls = append(toolCalls, agentToolCalls...)

	// Pattern for tool calls in various formats
	// Format 1: Action: tool_name\nAction Input: {...}
	actionPattern := regexp.MustCompile(`Action:\s*(\w+)\s*\nAction Input:\s*({[^}]+}|\S+)`)
	
	matches := actionPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 2 {
			toolName := match[1]
			inputStr := match[2]
			
			// Try to parse as JSON or create simple argument map
			args := p.parseToolArguments(inputStr)
			
			toolCall := chat.ToolCall{
				Function: chat.ToolFunction{
					Name:      toolName,
					Arguments: args,
				},
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	// Format 2: Function calls like execute_bash(command="ls -la")
	funcPattern := regexp.MustCompile(`(\w+)\(([^)]+)\)`)
	funcMatches := funcPattern.FindAllStringSubmatch(content, -1)
	
	for _, match := range funcMatches {
		if len(match) > 2 && p.isKnownTool(match[1]) {
			toolName := match[1]
			argsStr := match[2]
			
			args := p.parseInlineArguments(argsStr)
			
			toolCall := chat.ToolCall{
				Function: chat.ToolFunction{
					Name:      toolName,
					Arguments: args,
				},
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	return toolCalls
}

// extractAgentToolCalls extracts tool calls from LangChain agent format
func (p *ThinkingOutputParser) extractAgentToolCalls(content string) []chat.ToolCall {
	var toolCalls []chat.ToolCall
	lines := strings.Split(content, "\n")
	
	var currentAction string
	var currentInput string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "Action:") {
			currentAction = strings.TrimSpace(strings.TrimPrefix(line, "Action:"))
		} else if strings.HasPrefix(line, "Action Input:") {
			currentInput = strings.TrimSpace(strings.TrimPrefix(line, "Action Input:"))
			
			// If we have both action and input, create a tool call
			if currentAction != "" && currentInput != "" {
				args := p.parseToolArguments(currentInput)
				
				toolCall := chat.ToolCall{
					Function: chat.ToolFunction{
						Name:      currentAction,
						Arguments: args,
					},
				}
				toolCalls = append(toolCalls, toolCall)
				
				// Reset for next tool call
				currentAction = ""
				currentInput = ""
			}
		} else if strings.HasPrefix(line, "Observation:") {
			// Extract observation result for context
			observation := strings.TrimSpace(strings.TrimPrefix(line, "Observation:"))
			
			// If we have a recent tool call, we could add the observation as metadata
			if len(toolCalls) > 0 {
				lastCall := &toolCalls[len(toolCalls)-1]
				if lastCall.Function.Arguments == nil {
					lastCall.Function.Arguments = make(map[string]any)
				}
				lastCall.Function.Arguments["_observation"] = observation
			}
		}
	}
	
	return toolCalls
}

// parseToolArguments attempts to parse tool arguments
func (p *ThinkingOutputParser) parseToolArguments(input string) map[string]any {
	args := make(map[string]any)

	// Try to parse as key=value pairs
	if strings.Contains(input, "=") {
		pairs := strings.Split(input, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
				args[key] = value
			}
		}
	} else {
		// Single argument, assume it's the main parameter
		args["input"] = strings.Trim(input, `"'{}`)
	}

	return args
}

// parseInlineArguments parses function-style arguments
func (p *ThinkingOutputParser) parseInlineArguments(argsStr string) map[string]any {
	args := make(map[string]any)

	// Parse key="value" or key='value' patterns
	kvPattern := regexp.MustCompile(`(\w+)\s*=\s*["']([^"']+)["']`)
	matches := kvPattern.FindAllStringSubmatch(argsStr, -1)
	
	for _, match := range matches {
		if len(match) > 2 {
			args[match[1]] = match[2]
		}
	}

	// If no key-value pairs found, treat as single positional argument
	if len(args) == 0 {
		args["input"] = strings.Trim(argsStr, `"'`)
	}

	return args
}

// isKnownTool checks if a function name is a known tool
func (p *ThinkingOutputParser) isKnownTool(name string) bool {
	knownTools := []string{
		"execute_bash", "read_file", "write_file", "edit_file",
		"search", "browse", "calculate", "get_weather",
	}
	
	for _, tool := range knownTools {
		if name == tool {
			return true
		}
	}
	return false
}

// FormatResponse formats the parsed output based on settings
func (p *ThinkingOutputParser) FormatResponse(parsed ParsedOutput) string {
	if !parsed.HasThinking || !p.showThinking {
		return parsed.Content
	}

	// Include thinking in the response
	return fmt.Sprintf("<think>\n%s\n</think>\n\n%s", parsed.Thinking, parsed.Content)
}

// CreateMessage creates a chat message from parsed output
func (p *ThinkingOutputParser) CreateMessage(parsed ParsedOutput, role string) chat.Message {
	msg := chat.Message{
		Role:    role,
		Content: parsed.Content,
	}

	if parsed.HasThinking {
		msg = msg.WithThinking(parsed.Thinking, p.showThinking)
	}

	if len(parsed.ToolCalls) > 0 {
		msg.ToolCalls = parsed.ToolCalls
	}

	return msg
}

// StreamingThinkingParser handles thinking blocks in streaming responses
type StreamingThinkingParser struct {
	parser           *ThinkingOutputParser
	buffer           strings.Builder
	inThinkingBlock  bool
	thinkingBuffer   strings.Builder
	contentBuffer    strings.Builder
	agentBuffer      strings.Builder
	inAgentThought   bool
	inAgentAction    bool
	currentTool      string
	log              *logger.Logger
}

// NewStreamingThinkingParser creates a streaming parser
func NewStreamingThinkingParser(showThinking bool) *StreamingThinkingParser {
	return &StreamingThinkingParser{
		parser: NewThinkingOutputParser(showThinking),
		log:    logger.WithComponent("streaming_thinking_parser"),
	}
}

// ProcessChunk processes a streaming chunk
func (sp *StreamingThinkingParser) ProcessChunk(chunk string) (content string, thinking string, isComplete bool) {
	sp.buffer.WriteString(chunk)
	sp.agentBuffer.WriteString(chunk)
	
	// Check for thinking block markers (original format)
	if strings.Contains(chunk, "<think>") {
		sp.inThinkingBlock = true
	}
	
	if strings.Contains(chunk, "</think>") {
		sp.inThinkingBlock = false
		// Extract complete thinking block
		fullText := sp.buffer.String()
		blocks, clean := sp.parser.extractThinkingBlocks(fullText)
		if len(blocks) > 0 {
			thinking = strings.Join(blocks, "\n")
		}
		content = clean
		isComplete = true
		return
	}

	// Check for agent format markers
	agentContent := sp.processAgentFormat(chunk)
	if agentContent.HasUpdate {
		if agentContent.Thinking != "" {
			thinking = agentContent.Thinking
		}
		if agentContent.ToolUsage != "" {
			content = agentContent.ToolUsage
			return content, thinking, false
		}
		if agentContent.IsComplete {
			isComplete = true
			content = agentContent.FinalContent
			return content, thinking, isComplete
		}
	}

	// If in thinking block, accumulate in thinking buffer
	if sp.inThinkingBlock {
		sp.thinkingBuffer.WriteString(chunk)
		return "", "", false
	}

	// Otherwise, it's content
	sp.contentBuffer.WriteString(chunk)
	return chunk, "", false
}

// AgentChunkResult represents processed agent chunk information  
type AgentChunkResult struct {
	HasUpdate    bool
	Thinking     string
	ToolUsage    string
	FinalContent string
	IsComplete   bool
}

// processAgentFormat processes agent format chunks
func (sp *StreamingThinkingParser) processAgentFormat(chunk string) AgentChunkResult {
	result := AgentChunkResult{}
	
	// Look for agent format patterns in the accumulated agent buffer
	agentText := sp.agentBuffer.String()
	lines := strings.Split(agentText, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "Thought:") {
			sp.inAgentThought = true
			thought := strings.TrimSpace(strings.TrimPrefix(line, "Thought:"))
			if thought != "" && 
			   !strings.Contains(thought, "I need to use a tool") &&
			   !strings.Contains(thought, "Do I need to use a tool") {
				result.HasUpdate = true
				result.Thinking = thought
			}
		} else if strings.HasPrefix(line, "Action:") {
			sp.inAgentAction = true
			sp.currentTool = strings.TrimSpace(strings.TrimPrefix(line, "Action:"))
			result.HasUpdate = true
			result.ToolUsage = fmt.Sprintf("🔧 Using tool: **%s**", sp.currentTool)
		} else if strings.HasPrefix(line, "Action Input:") {
			input := strings.TrimSpace(strings.TrimPrefix(line, "Action Input:"))
			if sp.currentTool != "" && input != "" {
				result.HasUpdate = true
				result.ToolUsage = fmt.Sprintf("🔧 **%s**\n   Input: `%s`", sp.currentTool, input)
			}
		} else if strings.HasPrefix(line, "Observation:") {
			observation := strings.TrimSpace(strings.TrimPrefix(line, "Observation:"))
			if observation != "" {
				// Truncate long observations for streaming
				if len(observation) > 100 {
					observation = observation[:100] + "..."
				}
				result.HasUpdate = true
				result.ToolUsage = fmt.Sprintf("   Result: %s", observation)
			}
		} else if strings.HasPrefix(line, "AI:") {
			// Final AI response
			finalResponse := strings.TrimSpace(strings.TrimPrefix(line, "AI:"))
			if finalResponse != "" {
				result.HasUpdate = true
				result.IsComplete = true
				result.FinalContent = finalResponse
			}
		}
	}
	
	return result
}

// Finalize completes parsing of accumulated chunks
func (sp *StreamingThinkingParser) Finalize() ParsedOutput {
	fullText := sp.buffer.String()
	parsed, _ := sp.parser.Parse(fullText)
	return parsed
}

// Reset clears the streaming parser state
func (sp *StreamingThinkingParser) Reset() {
	sp.buffer.Reset()
	sp.thinkingBuffer.Reset()
	sp.contentBuffer.Reset()
	sp.agentBuffer.Reset()
	sp.inThinkingBlock = false
	sp.inAgentThought = false
	sp.inAgentAction = false
	sp.currentTool = ""
}