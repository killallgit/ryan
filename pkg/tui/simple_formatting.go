package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/logger"
)

// SimpleFormatter provides basic, readable text formatting without complex ANSI codes
type SimpleFormatter struct {
	width int
}

// FormattedLine represents a line with its content and tcell style
type FormattedLine struct {
	Content string
	Style   tcell.Style
	Indent  int // Number of spaces to indent
}

// NewSimpleFormatter creates a new simple formatter
func NewSimpleFormatter(width int) *SimpleFormatter {
	return &SimpleFormatter{
		width: width,
	}
}

// FormatContentSegments formats parsed content segments with simple, readable styling
func (sf *SimpleFormatter) FormatContentSegments(segments []ContentSegment) []FormattedLine {
	log := logger.WithComponent("simple_formatting")
	log.Debug("FormatContentSegments called", "segments_count", len(segments))

	var formattedLines []FormattedLine

	for i, segment := range segments {
		switch segment.Type {

		case ContentTypeCodeBlock:
			// Simple code block with border
			formattedLines = append(formattedLines, sf.formatCodeBlock(segment.Content, segment.Language)...)

		case ContentTypeInlineCode:
			// Inline code with different styling - keep on same line
			formattedLines = append(formattedLines, FormattedLine{
				Content: "`" + segment.Content + "`",
				Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
				Indent:  0,
			})

		case ContentTypeHeader:
			// Header with appropriate styling
			formattedLines = append(formattedLines, sf.formatHeader(segment.Content, segment.Level)...)

		case ContentTypeList:
			// List item with bullet and indentation
			formattedLines = append(formattedLines, sf.formatListItem(segment.Content, segment.Level)...)

		case ContentTypeThinking:
			// Format thinking block with dim italic style
			formattedLines = append(formattedLines, sf.formatThinkingBlock(segment.Content)...)

		case ContentTypeText:
			// Regular text
			textLines := WrapText(segment.Content, sf.width)
			for _, line := range textLines {
				if strings.TrimSpace(line) != "" {
					formattedLines = append(formattedLines, FormattedLine{
						Content: line,
						Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
						Indent:  0,
					})
				}
			}
		}

		// Add spacing between different content types (except inline code)
		if i < len(segments)-1 && segment.Type != ContentTypeInlineCode &&
			segments[i+1].Type != ContentTypeInlineCode {
			formattedLines = append(formattedLines, FormattedLine{
				Content: "",
				Style:   tcell.StyleDefault,
				Indent:  0,
			})
		}
	}

	log.Debug("FormatContentSegments result", "formatted_lines_count", len(formattedLines))
	return formattedLines
}

// formatThinkingBlock creates thinking content with clean styling
func (sf *SimpleFormatter) formatThinkingBlock(content string) []FormattedLine {
	var lines []FormattedLine

	// Add thinking header with clean symbol
	lines = append(lines, FormattedLine{
		Content: "✻ Thinking…",
		Style:   StyleThinkingText,
		Indent:  0,
	})

	// Add empty line for spacing
	lines = append(lines, FormattedLine{
		Content: "",
		Style:   tcell.StyleDefault,
		Indent:  0,
	})

	// Format thinking content with clean indentation
	wrappedLines := WrapText(strings.TrimSpace(content), sf.width-4)
	for _, line := range wrappedLines {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, FormattedLine{
				Content: strings.TrimSpace(line),
				Style:   StyleThinkingText,
				Indent:  2,
			})
		}
	}

	return lines
}

// formatCodeBlock creates a clean code block without heavy borders
func (sf *SimpleFormatter) formatCodeBlock(content, language string) []FormattedLine {
	var lines []FormattedLine

	// Add spacing before code block
	lines = append(lines, FormattedLine{
		Content: "",
		Style:   tcell.StyleDefault,
		Indent:  0,
	})

	// Add language label if available
	if language != "" {
		lines = append(lines, FormattedLine{
			Content: language + ":",
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true),
			Indent:  0,
		})
	}

	// Format code lines with simple indentation
	codeLines := strings.Split(content, "\n")
	for _, codeLine := range codeLines {
		lines = append(lines, FormattedLine{
			Content: codeLine,
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true),
			Indent:  4,
		})
	}

	// Add spacing after code block
	lines = append(lines, FormattedLine{
		Content: "",
		Style:   tcell.StyleDefault,
		Indent:  0,
	})

	return lines
}

// formatHeader creates a styled header
func (sf *SimpleFormatter) formatHeader(content string, level int) []FormattedLine {
	var lines []FormattedLine

	// Add some spacing before header
	lines = append(lines, FormattedLine{
		Content: "",
		Style:   tcell.StyleDefault,
		Indent:  0,
	})

	// Header text with level-appropriate styling
	var headerStyle tcell.Style
	var prefix string

	switch level {
	case 1:
		headerStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
		prefix = "⏺ "
		// No underline for cleaner look
		lines = append(lines, FormattedLine{
			Content: prefix + content,
			Style:   headerStyle,
			Indent:  0,
		})
		return lines

	case 2:
		headerStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
		prefix = "✻ "

	case 3:
		headerStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
		prefix = "• "

	default:
		headerStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
		prefix = "• "
	}

	lines = append(lines, FormattedLine{
		Content: prefix + content,
		Style:   headerStyle,
		Indent:  0,
	})

	return lines
}

// formatListItem creates a styled list item
func (sf *SimpleFormatter) formatListItem(content string, level int) []FormattedLine {
	indent := (level - 1) * 2
	bullet := "☐"
	if level == 1 {
		bullet = "⎿  ☐" // Top level with continuation symbol
	} else {
		bullet = "☐" // Nested items
	}

	listText := bullet + " " + content
	wrappedLines := WrapText(listText, sf.width-indent)

	var lines []FormattedLine
	for i, line := range wrappedLines {
		lineIndent := indent
		if i > 0 {
			// Continuation lines get extra indent to align with text
			lineIndent += len(bullet) + 1
		}

		lines = append(lines, FormattedLine{
			Content: line,
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
			Indent:  lineIndent,
		})
	}

	return lines
}

// ShouldUseSimpleFormatting determines if simple formatting should be applied
func ShouldUseSimpleFormatting(contentTypes map[ContentType]bool) bool {
	// Use simple formatting for any complex content
	return contentTypes[ContentTypeCodeBlock] ||
		contentTypes[ContentTypeHeader] ||
		contentTypes[ContentTypeList] ||
		contentTypes[ContentTypeThinking] ||
		(contentTypes[ContentTypeInlineCode] && len(contentTypes) > 1)
}
