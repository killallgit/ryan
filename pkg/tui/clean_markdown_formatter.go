package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/logger"
)

// CleanMarkdownFormatter provides beautiful, clean markdown formatting without ANSI codes
type CleanMarkdownFormatter struct {
	width int
}

// NewCleanMarkdownFormatter creates a new clean markdown formatter
func NewCleanMarkdownFormatter(width int) *CleanMarkdownFormatter {
	return &CleanMarkdownFormatter{
		width: width,
	}
}

// FormatMarkdown formats markdown content with clean, beautiful styling
func (cmf *CleanMarkdownFormatter) FormatMarkdown(content string) []FormattedLine {
	log := logger.WithComponent("clean_markdown_formatting")
	log.Debug("FormatMarkdown called", "content_length", len(content))

	var formattedLines []FormattedLine
	lines := strings.Split(content, "\n")
	
	inCodeBlock := false
	var codeBlockLines []string
	var codeBlockLang string
	
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// Handle code blocks
		if strings.HasPrefix(trimmedLine, "```") {
			if !inCodeBlock {
				// Starting code block
				inCodeBlock = true
				codeBlockLang = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "```"))
				codeBlockLines = []string{}
				continue
			} else {
				// Ending code block
				inCodeBlock = false
				formattedLines = append(formattedLines, cmf.formatCleanCodeBlock(codeBlockLines, codeBlockLang)...)
				codeBlockLines = nil
				codeBlockLang = ""
				continue
			}
		}
		
		if inCodeBlock {
			codeBlockLines = append(codeBlockLines, line)
			continue
		}
		
		// Handle headers
		if strings.HasPrefix(trimmedLine, "#") {
			formattedLines = append(formattedLines, cmf.formatCleanHeader(trimmedLine)...)
			continue
		}
		
		// Handle lists
		if matched, _ := regexp.MatchString(`^\s*[-*+]\s`, line); matched {
			formattedLines = append(formattedLines, cmf.formatCleanList(line))
			continue
		}
		
		// Handle numbered lists
		if matched, _ := regexp.MatchString(`^\s*\d+\.\s`, line); matched {
			formattedLines = append(formattedLines, cmf.formatCleanNumberedList(line))
			continue
		}
		
		// Handle thinking blocks
		if strings.Contains(strings.ToLower(line), "<think") || strings.Contains(strings.ToLower(line), "<thinking>") {
			// Find the complete thinking block and skip all lines that are part of it
			thinkingContent, endIndex := cmf.extractThinkingBlockWithEndIndex(lines, i)
			if len(thinkingContent) > 0 {
				formattedLines = append(formattedLines, cmf.formatThinkingBlock(thinkingContent)...)
				// Skip all the lines that were part of the thinking block
				i = endIndex
			}
			continue
		}
		
		// Handle inline code
		if strings.Contains(line, "`") {
			formattedLines = append(formattedLines, cmf.formatInlineCode(line))
			continue
		}
		
		// Handle empty lines (preserve some spacing but not excessive)
		if trimmedLine == "" {
			// Only add empty line if the previous line wasn't empty
			if len(formattedLines) > 0 && formattedLines[len(formattedLines)-1].Content != "" {
				formattedLines = append(formattedLines, FormattedLine{
					Content: "",
					Style:   tcell.StyleDefault,
					Indent:  0,
				})
			}
			continue
		}
		
		// Regular text
		formattedLines = append(formattedLines, cmf.formatRegularText(line))
	}
	
	// Handle unclosed code block
	if inCodeBlock && len(codeBlockLines) > 0 {
		formattedLines = append(formattedLines, cmf.formatCleanCodeBlock(codeBlockLines, codeBlockLang)...)
	}
	
	log.Debug("FormatMarkdown result", "formatted_lines_count", len(formattedLines))
	return formattedLines
}

// formatCleanHeader creates clean header formatting
func (cmf *CleanMarkdownFormatter) formatCleanHeader(line string) []FormattedLine {
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
	
	var lines []FormattedLine
	
	// Add some spacing before header (except for level 1)
	if level > 1 {
		lines = append(lines, FormattedLine{
			Content: "",
			Style:   tcell.StyleDefault,
			Indent:  0,
		})
	}
	
	// Format header based on level
	switch level {
	case 1:
		// Main header with bullet
		lines = append(lines, FormattedLine{
			Content: "⏺ " + headerText,
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true),
			Indent:  0,
		})
	case 2:
		// Sub-header with different symbol
		lines = append(lines, FormattedLine{
			Content: "✻ " + headerText,
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true),
			Indent:  0,
		})
	default:
		// Nested headers
		lines = append(lines, FormattedLine{
			Content: "• " + headerText,
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true),
			Indent:  (level - 3) * 2,
		})
	}
	
	return lines
}

// formatCleanList creates clean list formatting
func (cmf *CleanMarkdownFormatter) formatCleanList(line string) FormattedLine {
	// Extract indentation
	indent := 0
	for _, char := range line {
		if char == ' ' || char == '\t' {
			if char == '\t' {
				indent += 4
			} else {
				indent++
			}
		} else {
			break
		}
	}
	
	// Extract list content
	trimmed := strings.TrimSpace(line)
	content := strings.TrimSpace(trimmed[1:]) // Remove the bullet
	
	// Use clean bullets based on nesting
	bullet := "☐"
	if indent == 0 {
		bullet = "⎿  ☐"
	} else {
		bullet = "☐"
	}
	
	return FormattedLine{
		Content: bullet + " " + content,
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
		Indent:  indent,
	}
}

// formatCleanNumberedList creates clean numbered list formatting
func (cmf *CleanMarkdownFormatter) formatCleanNumberedList(line string) FormattedLine {
	// Extract indentation
	indent := 0
	for _, char := range line {
		if char == ' ' || char == '\t' {
			if char == '\t' {
				indent += 4
			} else {
				indent++
			}
		} else {
			break
		}
	}
	
	// Extract the number and content
	trimmed := strings.TrimSpace(line)
	parts := strings.SplitN(trimmed, ".", 2)
	if len(parts) < 2 {
		return FormattedLine{
			Content: line,
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
			Indent:  0,
		}
	}
	
	number := parts[0]
	content := strings.TrimSpace(parts[1])
	
	return FormattedLine{
		Content: fmt.Sprintf("%s. %s", number, content),
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
		Indent:  indent,
	}
}

// formatCleanCodeBlock creates clean code block formatting
func (cmf *CleanMarkdownFormatter) formatCleanCodeBlock(lines []string, language string) []FormattedLine {
	var formatted []FormattedLine
	
	// Add a clean separator
	formatted = append(formatted, FormattedLine{
		Content: "",
		Style:   tcell.StyleDefault,
		Indent:  0,
	})
	
	for _, line := range lines {
		// Simple indentation for code
		formatted = append(formatted, FormattedLine{
			Content: line,
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true),
			Indent:  4,
		})
	}
	
	// Add spacing after
	formatted = append(formatted, FormattedLine{
		Content: "",
		Style:   tcell.StyleDefault,
		Indent:  0,
	})
	
	return formatted
}

// formatThinkingBlock creates clean thinking block formatting
func (cmf *CleanMarkdownFormatter) formatThinkingBlock(content string) []FormattedLine {
	var lines []FormattedLine
	
	lines = append(lines, FormattedLine{
		Content: "✻ Thinking…",
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true),
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
				Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true),
				Indent:  2,
			})
		}
	}
	
	return lines
}

// formatInlineCode handles inline code formatting
func (cmf *CleanMarkdownFormatter) formatInlineCode(line string) FormattedLine {
	// For now, just render as regular text - inline code styling is complex
	return FormattedLine{
		Content: line,
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
		Indent:  0,
	}
}

// formatRegularText handles regular text
func (cmf *CleanMarkdownFormatter) formatRegularText(line string) FormattedLine {
	// Extract indentation from original line
	indent := 0
	for _, char := range line {
		if char == ' ' || char == '\t' {
			if char == '\t' {
				indent += 4
			} else {
				indent++
			}
		} else {
			break
		}
	}
	
	return FormattedLine{
		Content: strings.TrimSpace(line),
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
		Indent:  indent,
	}
}

// extractThinkingBlockWithEndIndex extracts content from thinking tags and returns the end index
func (cmf *CleanMarkdownFormatter) extractThinkingBlockWithEndIndex(lines []string, startIndex int) (string, int) {
	var content strings.Builder
	endIndex := startIndex
	
	// Look for the complete thinking block
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

// extractThinkingBlock extracts content from thinking tags (legacy function for compatibility)
func (cmf *CleanMarkdownFormatter) extractThinkingBlock(lines []string, startIndex int) string {
	content, _ := cmf.extractThinkingBlockWithEndIndex(lines, startIndex)
	return content
}