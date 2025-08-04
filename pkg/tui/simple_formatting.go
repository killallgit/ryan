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

// formatThinkingBlock creates a simple boxed thinking block
func (sf *SimpleFormatter) formatThinkingBlock(content string) []FormattedLine {
	var lines []FormattedLine

	// Ensure minimum width for borders
	minWidth := 20
	borderWidth := sf.width - 4
	if borderWidth < minWidth {
		borderWidth = minWidth
	}

	// Top border - solid thin line
	border := "┌" + strings.Repeat("─", borderWidth) + "┐"
	lines = append(lines, FormattedLine{
		Content: border,
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true),
		Indent:  1,
	})

	// Content with "Thinking:" prefix
	thinkingText := "Thinking: " + strings.TrimSpace(content)
	contentWidth := borderWidth - 2 // Account for padding inside borders
	if contentWidth < 20 {
		contentWidth = 20
	}

	wrappedLines := WrapText(thinkingText, contentWidth)
	for _, line := range wrappedLines {
		paddedLine := "│ " + line + strings.Repeat(" ", contentWidth-len(line)) + " │"
		lines = append(lines, FormattedLine{
			Content: paddedLine,
			Style:   StyleThinkingText,
			Indent:  1,
		})
	}

	// Bottom border - solid thin line
	bottomBorder := "└" + strings.Repeat("─", borderWidth) + "┘"
	lines = append(lines, FormattedLine{
		Content: bottomBorder,
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true),
		Indent:  1,
	})

	return lines
}

// formatCodeBlock creates a simple boxed code block
func (sf *SimpleFormatter) formatCodeBlock(content, language string) []FormattedLine {
	var lines []FormattedLine

	// Ensure minimum width for borders
	minWidth := 20
	borderWidth := sf.width - 4
	if borderWidth < minWidth {
		borderWidth = minWidth
	}

	// Top border with language label if available
	var topBorder string
	if language != "" {
		label := " " + language + " "
		borderLength := borderWidth - len(label)
		if borderLength < 4 {
			borderLength = 4
		}
		leftBorderLen := borderLength / 2
		rightBorderLen := borderLength - leftBorderLen
		if leftBorderLen < 0 {
			leftBorderLen = 0
		}
		if rightBorderLen < 0 {
			rightBorderLen = 0
		}
		leftBorder := strings.Repeat("─", leftBorderLen)
		rightBorder := strings.Repeat("─", rightBorderLen)
		topBorder = "┌" + leftBorder + label + rightBorder + "┐"
	} else {
		topBorder = "┌" + strings.Repeat("─", borderWidth) + "┐"
	}

	lines = append(lines, FormattedLine{
		Content: topBorder,
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true),
		Indent:  1,
	})

	// Content
	contentWidth := borderWidth - 2 // Account for padding inside borders
	if contentWidth < 30 {
		contentWidth = 30
	}

	codeLines := strings.Split(content, "\n")
	for _, codeLine := range codeLines {
		// Don't wrap code lines, just truncate if too long
		if len(codeLine) > contentWidth {
			codeLine = codeLine[:contentWidth-3] + "..."
		}
		paddedLine := "│ " + codeLine + strings.Repeat(" ", contentWidth-len(codeLine)) + " │"
		lines = append(lines, FormattedLine{
			Content: paddedLine,
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
			Indent:  1,
		})
	}

	// Bottom border - solid thin line
	bottomBorder := "└" + strings.Repeat("─", borderWidth) + "┘"
	lines = append(lines, FormattedLine{
		Content: bottomBorder,
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true),
		Indent:  1,
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
		prefix = "━━ "
		// Add underline using characters
		lines = append(lines, FormattedLine{
			Content: prefix + content,
			Style:   headerStyle,
			Indent:  0,
		})
		lines = append(lines, FormattedLine{
			Content: strings.Repeat("━", len(prefix+content)),
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true),
			Indent:  0,
		})
		return lines

	case 2:
		headerStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
		prefix = "── "

	case 3:
		headerStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
		prefix = "─ "

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
	bullet := "•"
	if level > 1 {
		bullet = "◦" // Hollow bullet for nested items
	}

	listText := bullet + " " + content
	wrappedLines := WrapText(listText, sf.width-indent)

	var lines []FormattedLine
	for i, line := range wrappedLines {
		lineIndent := indent
		if i > 0 {
			// Continuation lines get extra indent to align with text
			lineIndent += 2
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
		(contentTypes[ContentTypeInlineCode] && len(contentTypes) > 1)
}
