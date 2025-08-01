package tui

import (
	"regexp"
	"strings"
)

// ParsedContent represents a message content that has been parsed for thinking blocks
type ParsedContent struct {
	ThinkingBlock   string
	ResponseContent string
	HasThinking     bool
}

// ParseThinkingBlock parses a message content to separate <THINK> blocks from regular content
func ParseThinkingBlock(content string) ParsedContent {
	// Match <THINK>...</THINK> blocks (case insensitive)
	thinkRegex := regexp.MustCompile(`(?i)<THINK>(.*?)</THINK>`)
	matches := thinkRegex.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return ParsedContent{
			ThinkingBlock:   "",
			ResponseContent: content,
			HasThinking:     false,
		}
	}

	// Extract thinking content (combine all THINK blocks if multiple)
	var thinkingParts []string
	for _, match := range matches {
		if len(match) > 1 {
			thinkingParts = append(thinkingParts, strings.TrimSpace(match[1]))
		}
	}

	// Remove THINK blocks from content to get response
	responseContent := thinkRegex.ReplaceAllString(content, "")
	responseContent = strings.TrimSpace(responseContent)

	return ParsedContent{
		ThinkingBlock:   strings.Join(thinkingParts, "\n\n"),
		ResponseContent: responseContent,
		HasThinking:     true,
	}
}

// TruncateThinkingBlock truncates thinking content to a specified number of lines
func TruncateThinkingBlock(content string, maxLines int, width int) string {
	if content == "" {
		return ""
	}

	// Wrap text first to respect line width
	lines := WrapText(content, width)

	if len(lines) <= maxLines {
		return strings.Join(lines, "\n")
	}

	// Take first maxLines-1 lines and add "..."
	truncated := lines[:maxLines-1]
	truncated = append(truncated, "...")

	return strings.Join(truncated, "\n")
}
