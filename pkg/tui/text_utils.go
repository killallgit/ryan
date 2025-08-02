package tui

import (
	"regexp"
	"strings"

	"github.com/killallgit/ryan/pkg/logger"
)

// ParsedContent represents a message content that has been parsed for thinking blocks
type ParsedContent struct {
	ThinkingBlock   string
	ResponseContent string
	HasThinking     bool
}

// ParseThinkingBlock parses a message content to separate <THINK> blocks from regular content
func ParseThinkingBlock(content string) ParsedContent {
	// DEBUG: Log the input content being parsed
	log := logger.WithComponent("text_utils")
	contentPreview := content
	if len(contentPreview) > 150 {
		contentPreview = contentPreview[:150] + "..."
	}
	log.Debug("ParseThinkingBlock called",
		"content_length", len(content),
		"content_preview", contentPreview,
		"has_think_start", strings.Contains(content, "<think"),
		"has_think_end", strings.Contains(content, "</think"))

	thinkRegex := regexp.MustCompile(`(?ims)<think(?:ing)?>(.*?)</think(?:ing)?>`)
	matches := thinkRegex.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		result := ParsedContent{
			ThinkingBlock:   "",
			ResponseContent: strings.TrimSpace(content),
			HasThinking:     false,
		}
		log.Debug("ParseThinkingBlock result - no thinking",
			"response_content_length", len(result.ResponseContent),
			"response_preview", func() string {
				if len(result.ResponseContent) > 100 {
					return result.ResponseContent[:100] + "..."
				}
				return result.ResponseContent
			}())
		return result
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

	result := ParsedContent{
		ThinkingBlock:   strings.Join(thinkingParts, "\n\n"),
		ResponseContent: response,
		HasThinking:     len(thinkingParts) > 0,
	}

	log.Debug("ParseThinkingBlock result - with thinking",
		"thinking_length", len(result.ThinkingBlock),
		"response_content_length", len(result.ResponseContent),
		"response_preview", func() string {
			if len(result.ResponseContent) > 100 {
				return result.ResponseContent[:100] + "..."
			}
			return result.ResponseContent
		}(),
		"thinking_preview", func() string {
			if len(result.ThinkingBlock) > 100 {
				return result.ThinkingBlock[:100] + "..."
			}
			return result.ThinkingBlock
		}())

	return result
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
