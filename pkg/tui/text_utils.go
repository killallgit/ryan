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

// ContentType represents different types of content for formatting
type ContentType int

const (
	ContentTypeText ContentType = iota
	ContentTypeCodeBlock
	ContentTypeInlineCode
	ContentTypeThinking
	ContentTypeHeader
	ContentTypeList
)

// ContentSegment represents a parsed segment of content with its type
type ContentSegment struct {
	Type     ContentType
	Content  string
	Language string // For code blocks
	Level    int    // For headers or list nesting
}

// ParseContentSegments parses content into typed segments for enhanced formatting
func ParseContentSegments(content string) []ContentSegment {
	log := logger.WithComponent("text_utils")
	log.Debug("ParseContentSegments called", "content_length", len(content))

	var segments []ContentSegment
	lines := strings.Split(content, "\n")
	
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		// Check for fenced code blocks (```)
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			// Extract language if specified
			language := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "```"))
			
			// Collect all lines until closing ```
			var codeLines []string
			i++ // Skip the opening ```
			for i < len(lines) {
				if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
					break // Found closing ```
				}
				codeLines = append(codeLines, lines[i])
				i++
			}
			
			segments = append(segments, ContentSegment{
				Type:     ContentTypeCodeBlock,
				Content:  strings.Join(codeLines, "\n"),
				Language: language,
			})
			continue
		}
		
		// Check for headers (# ## ###)
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			level := 0
			trimmed := strings.TrimSpace(line)
			for _, char := range trimmed {
				if char == '#' {
					level++
				} else {
					break
				}
			}
			headerText := strings.TrimSpace(strings.TrimPrefix(trimmed, strings.Repeat("#", level)))
			
			segments = append(segments, ContentSegment{
				Type:    ContentTypeHeader,
				Content: headerText,
				Level:   level,
			})
			continue
		}
		
		// Check for list items (- * +)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
			listText := strings.TrimSpace(trimmed[2:])
			segments = append(segments, ContentSegment{
				Type:    ContentTypeList,
				Content: listText,
				Level:   1, // Could be enhanced to detect nesting level
			})
			continue
		}
		
		// Check for inline code and regular text
		if strings.Contains(line, "`") {
			// Parse inline code within the line
			parseInlineCode(line, &segments)
		} else {
			// Regular text
			if strings.TrimSpace(line) != "" {
				segments = append(segments, ContentSegment{
					Type:    ContentTypeText,
					Content: line,
				})
			}
		}
	}
	
	log.Debug("ParseContentSegments result", "segments_count", len(segments))
	return segments
}

// parseInlineCode parses a line that contains inline code (`code`)
func parseInlineCode(line string, segments *[]ContentSegment) {
	inlineCodeRegex := regexp.MustCompile("`([^`]+)`")
	lastEnd := 0
	
	matches := inlineCodeRegex.FindAllStringSubmatchIndex(line, -1)
	for _, match := range matches {
		// Add text before the code
		if match[0] > lastEnd {
			textBefore := line[lastEnd:match[0]]
			if strings.TrimSpace(textBefore) != "" {
				*segments = append(*segments, ContentSegment{
					Type:    ContentTypeText,
					Content: textBefore,
				})
			}
		}
		
		// Add the inline code
		codeContent := line[match[2]:match[3]] // Group 1 content
		*segments = append(*segments, ContentSegment{
			Type:    ContentTypeInlineCode,
			Content: codeContent,
		})
		
		lastEnd = match[1]
	}
	
	// Add remaining text after last code
	if lastEnd < len(line) {
		textAfter := line[lastEnd:]
		if strings.TrimSpace(textAfter) != "" {
			*segments = append(*segments, ContentSegment{
				Type:    ContentTypeText,
				Content: textAfter,
			})
		}
	}
}

// DetectContentTypes analyzes content and returns detected content types for formatting decisions
func DetectContentTypes(content string) map[ContentType]bool {
	types := make(map[ContentType]bool)
	
	// Check for code blocks
	if strings.Contains(content, "```") {
		types[ContentTypeCodeBlock] = true
	}
	
	// Check for inline code
	if regexp.MustCompile("`[^`]+`").MatchString(content) {
		types[ContentTypeInlineCode] = true
	}
	
	// Check for headers
	if regexp.MustCompile(`(?m)^\s*#+\s`).MatchString(content) {
		types[ContentTypeHeader] = true
	}
	
	// Check for lists
	if regexp.MustCompile(`(?m)^\s*[-*+]\s`).MatchString(content) {
		types[ContentTypeList] = true
	}
	
	// Check for thinking blocks
	if strings.Contains(content, "<think") {
		types[ContentTypeThinking] = true
	}
	
	// Always has text if content is not empty
	if strings.TrimSpace(content) != "" {
		types[ContentTypeText] = true
	}
	
	return types
}
