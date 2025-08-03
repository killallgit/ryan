package chat

import (
	"regexp"
	"strings"
)

// ParsedMessage represents a message content that has been parsed for thinking blocks
type ParsedMessage struct {
	ThinkingContent string
	ResponseContent string
	HasThinking     bool
}

// ParseMessageThinking parses a message content to separate <think> blocks from regular content
func ParseMessageThinking(content string) ParsedMessage {
	thinkRegex := regexp.MustCompile(`(?ims)<think(?:ing)?>(.*?)</think(?:ing)?>`)
	matches := thinkRegex.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return ParsedMessage{
			ThinkingContent: "",
			ResponseContent: strings.TrimSpace(content),
			HasThinking:     false,
		}
	}

	// Extract thinking content (combine all THINK blocks if multiple)
	var thinkingParts []string
	for _, m := range matches {
		if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
			thinkingParts = append(thinkingParts, strings.TrimSpace(m[1]))
		}
	}

	// Remove THINK blocks from content to get response
	response := strings.TrimSpace(thinkRegex.ReplaceAllString(content, ""))

	return ParsedMessage{
		ThinkingContent: strings.Join(thinkingParts, "\n\n"),
		ResponseContent: response,
		HasThinking:     len(thinkingParts) > 0,
	}
}

// ExtractResponseContent extracts only the response content from a message, removing thinking blocks
func ExtractResponseContent(content string) string {
	parsed := ParseMessageThinking(content)
	return parsed.ResponseContent
}
