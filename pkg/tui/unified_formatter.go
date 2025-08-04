package tui

import (
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// TextFormatter defines the interface for text formatting
type TextFormatter interface {
	Format(content string, width int) []FormattedLine
}

// UnifiedFormatter handles all text formatting with consistent preprocessing and styling
type UnifiedFormatter struct {
	// Configuration options
	enableRichFormatting bool
	enableSyntaxColors   bool
}

// NewUnifiedFormatter creates a new unified text formatter
func NewUnifiedFormatter() *UnifiedFormatter {
	return &UnifiedFormatter{
		enableRichFormatting: true,
		enableSyntaxColors:   true,
	}
}

// NewSimpleUnifiedFormatter creates a formatter with basic formatting only
func NewSimpleUnifiedFormatter() *UnifiedFormatter {
	return &UnifiedFormatter{
		enableRichFormatting: false,
		enableSyntaxColors:   false,
	}
}

// Format processes content through the complete formatting pipeline
func (uf *UnifiedFormatter) Format(content string, width int) []FormattedLine {
	// Step 1: Preprocessing - clean up malformed content
	cleanedContent := uf.preprocessContent(content)

	// Step 2: Content Analysis - detect what types of content we have
	contentTypes := uf.analyzeContent(cleanedContent)

	// Step 3: Segment Parsing - break content into logical chunks
	segments := uf.parseSegments(cleanedContent, contentTypes)

	// Step 4: Formatting - apply appropriate styles and layout
	formattedLines := uf.formatSegments(segments, width)

	return formattedLines
}

// preprocessContent cleans up malformed markdown and normalizes content
func (uf *UnifiedFormatter) preprocessContent(content string) string {
	// Clean up multi-line backtick patterns
	content = uf.cleanBrokenBackticks(content)

	// Normalize whitespace
	content = uf.normalizeWhitespace(content)

	return content
}

// cleanBrokenBackticks handles malformed multi-line backtick patterns
func (uf *UnifiedFormatter) cleanBrokenBackticks(content string) string {
	// Handle multi-line inline code spans
	multiLineCodeRegex := regexp.MustCompile("(?s)`([^`]*?\n[^`]*?)`")
	content = multiLineCodeRegex.ReplaceAllStringFunc(content, func(match string) string {
		codeContent := strings.Trim(match, "`")
		codeContent = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(codeContent), " ")
		if strings.TrimSpace(codeContent) == "" {
			return ""
		}
		return "`" + codeContent + "`"
	})

	// Handle broken backtick patterns: ` followed by newline followed by `
	brokenBacktickPattern := regexp.MustCompile("(?m)`\\s*\n\\s*`")
	content = brokenBacktickPattern.ReplaceAllString(content, " ")

	// Remove standalone backticks on their own lines
	standaloneBackticks := regexp.MustCompile("(?m)^\\s*`+\\s*$")
	content = standaloneBackticks.ReplaceAllString(content, "")

	// Clean up double backticks that are empty or nearly empty
	emptyDoubleBackticks := regexp.MustCompile("``\\s*")
	content = emptyDoubleBackticks.ReplaceAllString(content, "")

	// Handle remaining single backticks with minimal content on their own lines
	minimalBackticks := regexp.MustCompile("(?m)^\\s*`\\s*([^`\\n]{0,3})\\s*`?\\s*$")
	content = minimalBackticks.ReplaceAllStringFunc(content, func(match string) string {
		// If there's actual content, keep it but format it nicely
		cleaned := regexp.MustCompile(`[^a-zA-Z0-9\s]`).ReplaceAllString(match, "")
		cleaned = strings.TrimSpace(cleaned)
		if cleaned != "" && len(cleaned) > 0 {
			return "`" + cleaned + "`"
		}
		return ""
	})

	return content
}

// normalizeWhitespace cleans up excessive whitespace
func (uf *UnifiedFormatter) normalizeWhitespace(content string) string {
	// Remove excessive empty lines
	content = regexp.MustCompile("\n{3,}").ReplaceAllString(content, "\n\n")

	// Clean up trailing whitespace on lines
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	return strings.Join(lines, "\n")
}

// ContentTypeFlags represents detected content types
type ContentTypeFlags struct {
	HasCodeBlocks bool
	HasInlineCode bool
	HasHeaders    bool
	HasLists      bool
	HasThinking   bool
	HasPlainText  bool
}

// analyzeContent detects what types of content are present
func (uf *UnifiedFormatter) analyzeContent(content string) ContentTypeFlags {
	flags := ContentTypeFlags{}

	// Check for code blocks
	if strings.Contains(content, "```") {
		flags.HasCodeBlocks = true
	}

	// Check for inline code (but be more strict than before)
	if uf.hasValidInlineCode(content) {
		flags.HasInlineCode = true
	}

	// Check for headers
	if regexp.MustCompile(`(?m)^\s*#+\s`).MatchString(content) {
		flags.HasHeaders = true
	}

	// Check for lists
	if regexp.MustCompile(`(?m)^\s*[-*+]\s`).MatchString(content) ||
		regexp.MustCompile(`(?m)^\s*\d+\.\s`).MatchString(content) {
		flags.HasLists = true
	}

	// Check for thinking blocks
	if strings.Contains(strings.ToLower(content), "<think") {
		flags.HasThinking = true
	}

	// Always has plain text if we got here
	flags.HasPlainText = true

	return flags
}

// hasValidInlineCode checks for well-formed inline code (not broken backticks)
func (uf *UnifiedFormatter) hasValidInlineCode(content string) bool {
	// Look for backticks that have actual content and are well-formed
	inlineCodeRegex := regexp.MustCompile("`[^`\n]{2,}[^`\n]*`")
	return inlineCodeRegex.MatchString(content)
}

// SegmentType represents different types of content segments
type SegmentType int

const (
	SegmentPlainText SegmentType = iota
	SegmentCodeBlock
	SegmentInlineCode
	SegmentHeader
	SegmentList
	SegmentThinking
)

// ContentSegmentNew represents a piece of content with its type and metadata
type ContentSegmentNew struct {
	Type     SegmentType
	Content  string
	Level    int    // For headers (1-6) or list nesting
	Language string // For code blocks
}

// parseSegments breaks content into logical segments for formatting
func (uf *UnifiedFormatter) parseSegments(content string, flags ContentTypeFlags) []ContentSegmentNew {
	var segments []ContentSegmentNew
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Handle code blocks first (they can span multiple lines)
		if strings.HasPrefix(trimmed, "```") {
			language := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			var codeLines []string

			// Collect lines until closing ```
			i++
			for i < len(lines) {
				if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
					break
				}
				codeLines = append(codeLines, lines[i])
				i++
			}

			segments = append(segments, ContentSegmentNew{
				Type:     SegmentCodeBlock,
				Content:  strings.Join(codeLines, "\n"),
				Language: language,
			})
			continue
		}

		// Handle thinking blocks
		if strings.Contains(strings.ToLower(line), "<think") {
			thinkingContent, endIndex := uf.extractThinkingBlock(lines, i)
			if thinkingContent != "" {
				segments = append(segments, ContentSegmentNew{
					Type:    SegmentThinking,
					Content: thinkingContent,
				})
				i = endIndex
				continue
			}
		}

		// Handle headers
		if strings.HasPrefix(trimmed, "#") {
			level := 0
			for _, char := range trimmed {
				if char == '#' {
					level++
				} else {
					break
				}
			}
			headerText := strings.TrimSpace(strings.TrimPrefix(trimmed, strings.Repeat("#", level)))

			segments = append(segments, ContentSegmentNew{
				Type:    SegmentHeader,
				Content: headerText,
				Level:   level,
			})
			continue
		}

		// Handle list items
		if regexp.MustCompile(`^\s*[-*+]\s`).MatchString(line) ||
			regexp.MustCompile(`^\s*\d+\.\s`).MatchString(line) {

			// Calculate nesting level
			level := 0
			for _, char := range line {
				switch char {
				case ' ':
					level++
				case '\t':
					level += 4
				default:
					goto done
				}
			}
		done:

			segments = append(segments, ContentSegmentNew{
				Type:    SegmentList,
				Content: trimmed,
				Level:   level / 2, // Convert spaces to nesting level
			})
			continue
		}

		// Handle inline code within regular text
		if strings.Contains(line, "`") && flags.HasInlineCode {
			segments = append(segments, ContentSegmentNew{
				Type:    SegmentInlineCode,
				Content: line,
			})
			continue
		}

		// Regular text
		if trimmed != "" {
			segments = append(segments, ContentSegmentNew{
				Type:    SegmentPlainText,
				Content: line,
			})
		}
	}

	return segments
}

// extractThinkingBlock extracts thinking content from lines starting at startIndex
func (uf *UnifiedFormatter) extractThinkingBlock(lines []string, startIndex int) (string, int) {
	var content strings.Builder
	endIndex := startIndex

	for i := startIndex; i < len(lines); i++ {
		line := lines[i]
		content.WriteString(line)
		if i < len(lines)-1 {
			content.WriteString("\n")
		}

		if strings.Contains(strings.ToLower(line), "</think>") || strings.Contains(strings.ToLower(line), "</thinking>") {
			endIndex = i
			break
		}
		endIndex = i
	}

	// Extract content between tags
	fullContent := content.String()
	thinkRegex := regexp.MustCompile(`(?is)<think(?:ing)?>\s*(.*?)\s*</think(?:ing)?>`)
	if matches := thinkRegex.FindStringSubmatch(fullContent); len(matches) > 1 {
		return strings.TrimSpace(matches[1]), endIndex
	}

	return "", endIndex
}

// formatSegments applies formatting and styling to segments
func (uf *UnifiedFormatter) formatSegments(segments []ContentSegmentNew, width int) []FormattedLine {
	var formattedLines []FormattedLine
	lastWasList := false

	for _, segment := range segments {
		switch segment.Type {
		case SegmentCodeBlock:
			formattedLines = append(formattedLines, uf.formatCodeBlock(segment.Content, segment.Language)...)
			lastWasList = false

		case SegmentInlineCode:
			formattedLines = append(formattedLines, uf.formatInlineCode(segment.Content))
			lastWasList = false

		case SegmentHeader:
			formattedLines = append(formattedLines, uf.formatHeader(segment.Content, segment.Level)...)
			lastWasList = false

		case SegmentList:
			formattedLines = append(formattedLines, uf.formatListItem(segment.Content, segment.Level))
			lastWasList = true

		case SegmentThinking:
			formattedLines = append(formattedLines, uf.formatThinkingBlock(segment.Content)...)
			lastWasList = false

		case SegmentPlainText:
			// Don't add empty lines between consecutive list items
			if segment.Content == "" && lastWasList {
				lastWasList = false
				continue
			}

			lines := WrapText(segment.Content, width)
			for _, line := range lines {
				formattedLines = append(formattedLines, FormattedLine{
					Content: line,
					Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
					Indent:  0,
				})
			}
			lastWasList = false
		}
	}

	return formattedLines
}

// formatCodeBlock formats a code block with clean styling
func (uf *UnifiedFormatter) formatCodeBlock(content, _ string) []FormattedLine {
	var formatted []FormattedLine

	// Add spacing before code block
	formatted = append(formatted, FormattedLine{
		Content: "",
		Style:   tcell.StyleDefault,
		Indent:  0,
	})

	// Add code lines with indentation
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		formatted = append(formatted, FormattedLine{
			Content: line,
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true),
			Indent:  4,
		})
	}

	// Add spacing after code block
	formatted = append(formatted, FormattedLine{
		Content: "",
		Style:   tcell.StyleDefault,
		Indent:  0,
	})

	return formatted
}

// formatInlineCode formats a line containing inline code
func (uf *UnifiedFormatter) formatInlineCode(line string) FormattedLine {
	// Replace backticks with brackets for better terminal display
	inlineCodeRegex := regexp.MustCompile("`([^`]+)`")
	formattedContent := inlineCodeRegex.ReplaceAllString(line, "[$1]")

	return FormattedLine{
		Content: formattedContent,
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
		Indent:  0,
	}
}

// formatHeader formats a header with clean styling
func (uf *UnifiedFormatter) formatHeader(content string, level int) []FormattedLine {
	var lines []FormattedLine

	// Add spacing before header (except for level 1)
	if level > 1 {
		lines = append(lines, FormattedLine{
			Content: "",
			Style:   tcell.StyleDefault,
			Indent:  0,
		})
	}

	// Format header based on level
	var symbol string
	switch level {
	case 1:
		symbol = "⏺"
	case 2:
		symbol = "✻"
	default:
		symbol = "•"
	}

	lines = append(lines, FormattedLine{
		Content: symbol + " " + content,
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true),
		Indent:  max(0, (level-3)*2),
	})

	return lines
}

// formatListItem formats a list item with clean styling
func (uf *UnifiedFormatter) formatListItem(content string, level int) FormattedLine {
	// Extract list content (remove bullet/number)
	listContent := content
	if regexp.MustCompile(`^\s*[-*+]\s`).MatchString(content) {
		listContent = regexp.MustCompile(`^\s*[-*+]\s`).ReplaceAllString(content, "")
	} else if regexp.MustCompile(`^\s*\d+\.\s`).MatchString(content) {
		parts := regexp.MustCompile(`^\s*(\d+)\.\s`).FindStringSubmatch(content)
		if len(parts) > 1 {
			listContent = strings.TrimSpace(content[len(parts[0]):])
		}
	}

	// Use clean bullet style
	bullet := "☐"
	if level == 0 {
		bullet = "⎿  ☐"
	}

	return FormattedLine{
		Content: bullet + " " + listContent,
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
		Indent:  level * 2,
	}
}

// formatThinkingBlock formats thinking content with clean styling
func (uf *UnifiedFormatter) formatThinkingBlock(content string) []FormattedLine {
	var lines []FormattedLine

	lines = append(lines, FormattedLine{
		Content: "✻ Thinking…",
		Style:   StyleThinkingText,
		Indent:  0,
	})

	lines = append(lines, FormattedLine{
		Content: "",
		Style:   tcell.StyleDefault,
		Indent:  0,
	})

	// Format thinking content with indentation
	contentLines := strings.Split(content, "\n")
	for _, contentLine := range contentLines {
		if strings.TrimSpace(contentLine) != "" {
			lines = append(lines, FormattedLine{
				Content: strings.TrimSpace(contentLine),
				Style:   StyleThinkingText,
				Indent:  2,
			})
		}
	}

	return lines
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
