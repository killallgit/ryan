package langchain

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/killallgit/ryan/pkg/logger"
)

// OutputProcessor handles preprocessing of LLM outputs for agent compatibility
type OutputProcessor struct {
	stripThinkingBlocks bool
	convertToReAct      bool
	log                 *logger.Logger
}

// NewOutputProcessor creates a new output processor
func NewOutputProcessor(stripThinking, convertReAct bool) *OutputProcessor {
	return &OutputProcessor{
		stripThinkingBlocks: stripThinking,
		convertToReAct:      convertReAct,
		log:                 logger.WithComponent("output_processor"),
	}
}

// ProcessForAgent processes LLM output to make it compatible with agents
func (op *OutputProcessor) ProcessForAgent(output string) string {
	original := output

	// Step 1: Remove thinking blocks if enabled
	if op.stripThinkingBlocks {
		output = op.removeThinkingBlocks(output)
		if output != original {
			op.log.Debug("Removed thinking blocks from output",
				"original_length", len(original),
				"processed_length", len(output))
		}
	}

	// Step 2: Try to extract tool intent and convert to ReAct format
	if op.convertToReAct && op.detectToolIntent(output) {
		converted := op.convertToReActFormat(output)
		if converted != output {
			op.log.Debug("Converted output to ReAct format",
				"original", truncateString(output, 100),
				"converted", truncateString(converted, 100))
			return converted
		}
	}

	return output
}

// ProcessForDisplay processes LLM output for display purposes, preserving thinking blocks
func (op *OutputProcessor) ProcessForDisplay(output string) string {
	// Don't remove thinking blocks for display - they will be formatted by the UI
	// Just clean up extra whitespace and return
	cleanOutput := strings.TrimSpace(output)
	cleanOutput = regexp.MustCompile(`\n{3,}`).ReplaceAllString(cleanOutput, "\n\n")

	op.log.Debug("Processed output for display",
		"original_length", len(output),
		"processed_length", len(cleanOutput))

	return cleanOutput
}

// removeThinkingBlocks removes <think>...</think> and <thinking>...</thinking> blocks
func (op *OutputProcessor) removeThinkingBlocks(output string) string {
	// Remove <think> blocks
	thinkRe := regexp.MustCompile(`(?s)<think>.*?</think>`)
	output = thinkRe.ReplaceAllString(output, "")

	// Remove <thinking> blocks
	thinkingRe := regexp.MustCompile(`(?s)<thinking>.*?</thinking>`)
	output = thinkingRe.ReplaceAllString(output, "")

	// Clean up extra whitespace
	output = strings.TrimSpace(output)
	output = regexp.MustCompile(`\n{3,}`).ReplaceAllString(output, "\n\n")

	return output
}

// detectToolIntent checks if the output indicates tool usage intent
func (op *OutputProcessor) detectToolIntent(output string) bool {
	lowerOutput := strings.ToLower(output)

	// Common patterns indicating tool usage
	toolPatterns := []string{
		"i'll run", "i'll execute", "let me run", "let me execute",
		"i'll use the", "i'll check", "let me check",
		"running the command", "executing", "using the tool",
		"i need to", "i should", "i can help you by",
		"to count the files", "to list the files", "to find",
	}

	for _, pattern := range toolPatterns {
		if strings.Contains(lowerOutput, pattern) {
			return true
		}
	}

	// Check for command-like content
	if op.containsCommand(output) {
		return true
	}

	return false
}

// containsCommand checks if the output contains shell commands or tool calls
func (op *OutputProcessor) containsCommand(output string) bool {
	// Look for command patterns
	commandPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s*\$\s+.+`),                  // $ command
		regexp.MustCompile(`(?m)^\s*>\s+.+`),                   // > command
		regexp.MustCompile(`(?i)(bash|shell|execute):\s*(.+)`), // tool: command
		regexp.MustCompile("`[^`]+\\|[^`]+`"),                  // `command | pipe`
		regexp.MustCompile(`ls\s+.*\|\s*wc\s+-l`),              // specific: ls | wc -l
	}

	for _, pattern := range commandPatterns {
		if pattern.MatchString(output) {
			return true
		}
	}

	return false
}

// convertToReActFormat attempts to convert modern LLM output to ReAct format
func (op *OutputProcessor) convertToReActFormat(output string) string {
	// Try to extract tool and command from various formats
	tool, command := op.extractToolAndCommand(output)

	if tool != "" && command != "" {
		// Format as ReAct
		return fmt.Sprintf("I need to use a tool to help with this task.\n\nAction: %s\nAction Input: %s", tool, command)
	}

	// If we can't convert, return original
	return output
}

// extractToolAndCommand tries to extract tool name and command from output
func (op *OutputProcessor) extractToolAndCommand(output string) (tool, command string) {
	// Pattern 1: "I'll run/execute [the command] X"
	runPattern := regexp.MustCompile(`(?i)(?:i'll|i will|let me|i'm going to)\s+(?:run|execute)\s+(?:the\s+)?(?:command\s+)?["` + "`" + `]?([^"` + "`" + `\n]+)["` + "`" + `]?`)
	if matches := runPattern.FindStringSubmatch(output); len(matches) > 1 {
		return "execute_bash", strings.TrimSpace(matches[1])
	}

	// Pattern 2: Backtick commands
	backtickPattern := regexp.MustCompile("`([^`]+)`")
	if matches := backtickPattern.FindAllStringSubmatch(output, -1); len(matches) > 0 {
		// Look for command-like content
		for _, match := range matches {
			cmd := match[1]
			if strings.Contains(cmd, "|") || strings.Contains(cmd, "ls") ||
				strings.Contains(cmd, "wc") || strings.Contains(cmd, "grep") {
				return "execute_bash", cmd
			}
		}
	}

	// Pattern 3: Direct tool mentions
	toolPattern := regexp.MustCompile(`(?i)(?:using|use|with)\s+(?:the\s+)?(\w+)\s+tool.*?:\s*(.+)`)
	if matches := toolPattern.FindStringSubmatch(output); len(matches) > 2 {
		toolName := strings.ToLower(matches[1])
		command := strings.TrimSpace(matches[2])

		// Map common tool names
		switch toolName {
		case "bash", "shell", "terminal", "command":
			return "execute_bash", command
		case "file", "read":
			return "read_file", command
		case "web", "fetch":
			return "web_fetch", command
		default:
			return toolName, command
		}
	}

	// Pattern 4: Specific for "ls | wc -l" type commands
	if strings.Contains(output, "ls") && strings.Contains(output, "wc") {
		// Extract the actual command
		cmdPattern := regexp.MustCompile(`(ls\s*.*?\|\s*wc\s*-l)`)
		if matches := cmdPattern.FindStringSubmatch(output); len(matches) > 1 {
			return "execute_bash", matches[1]
		}
	}

	return "", ""
}

// CleanToolResponse cleans up tool responses for better readability
func (op *OutputProcessor) CleanToolResponse(response string) string {
	// Remove ANSI escape codes
	ansiPattern := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	response = ansiPattern.ReplaceAllString(response, "")

	// Trim whitespace
	response = strings.TrimSpace(response)

	return response
}
