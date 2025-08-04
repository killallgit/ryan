package tui

import (
	"regexp"
	"strings"
)

// ContentType represents different types of content for formatting
type ContentType int

const (
	ContentTypeText ContentType = iota
	ContentTypeCodeBlock
	ContentTypeInlineCode
	ContentTypeHeader
	ContentTypeList
	ContentTypeThinking
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

	var segments []ContentSegment
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check for thinking blocks (<think> or <thinking>)
		if strings.Contains(strings.ToLower(line), "<think") || strings.Contains(strings.ToLower(line), "<thinking>") {
			// Find the complete thinking block
			var thinkingContent strings.Builder
			var foundEnd bool

			// Accumulate all content until we find the closing tag
			for j := i; j < len(lines); j++ {
				currentLine := lines[j]
				thinkingContent.WriteString(currentLine)
				if j < len(lines)-1 {
					thinkingContent.WriteString("\n")
				}

				if strings.Contains(strings.ToLower(currentLine), "</think>") || strings.Contains(strings.ToLower(currentLine), "</thinking>") {
					foundEnd = true
					i = j // Update loop counter
					break
				}
			}

			if foundEnd {
				// Extract the content between tags
				content := thinkingContent.String()
				// Remove the tags
				thinkRegex := regexp.MustCompile(`(?is)<think(?:ing)?>\s*(.*?)\s*</think(?:ing)?>`)
				if matches := thinkRegex.FindStringSubmatch(content); len(matches) > 1 {
					segments = append(segments, ContentSegment{
						Type:    ContentTypeThinking,
						Content: matches[1],
					})
				}
			}
			continue
		}

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
	if regexp.MustCompile(`(?i)<think(?:ing)?>`).MatchString(content) {
		types[ContentTypeThinking] = true
	}

	// Always has text if content is not empty
	if strings.TrimSpace(content) != "" {
		types[ContentTypeText] = true
	}

	return types
}
