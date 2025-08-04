package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
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
	// Pre-process content to handle multi-line inline code patterns
	content = cmf.preprocessInlineCode(content)

	var formattedLines []FormattedLine
	lines := strings.Split(content, "\n")

	inCodeBlock := false
	var codeBlockLines []string
	var codeBlockLang string
	lastWasList := false

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
			lastWasList = false
			continue
		}

		// Handle lists
		if matched, _ := regexp.MatchString(`^\s*[-*+]\s`, line); matched {
			formattedLines = append(formattedLines, cmf.formatCleanList(line))
			lastWasList = true
			continue
		}

		// Handle numbered lists
		if matched, _ := regexp.MatchString(`^\s*\d+\.\s`, line); matched {
			formattedLines = append(formattedLines, cmf.formatCleanNumberedList(line))
			lastWasList = true
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
			lastWasList = false
			continue
		}

		// Handle inline code
		if strings.Contains(line, "`") {
			formattedLines = append(formattedLines, cmf.formatInlineCode(line))
			lastWasList = false
			continue
		}

		// Handle empty lines (preserve some spacing but not excessive)
		if trimmedLine == "" {
			// Don't add empty lines between consecutive list items
			if lastWasList {
				lastWasList = false
				continue
			}
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
		lastWasList = false
	}

	// Handle unclosed code block
	if inCodeBlock && len(codeBlockLines) > 0 {
		formattedLines = append(formattedLines, cmf.formatCleanCodeBlock(codeBlockLines, codeBlockLang)...)
	}

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
func (cmf *CleanMarkdownFormatter) formatCleanCodeBlock(lines []string, _ string) []FormattedLine {
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

	// Replace single backticks with styled inline code
	inlineCodeRegex := regexp.MustCompile("`([^`]+)`")
	formattedContent := inlineCodeRegex.ReplaceAllStringFunc(line, func(match string) string {
		// Extract code content (remove backticks)
		codeContent := strings.Trim(match, "`")
		// Style inline code with brackets and dim styling
		return "[" + codeContent + "]"
	})

	return FormattedLine{
		Content: strings.TrimSpace(formattedContent),
		Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true),
		Indent:  indent,
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

// preprocessInlineCode handles multi-line inline code patterns and cleans them up
func (cmf *CleanMarkdownFormatter) preprocessInlineCode(content string) string {
	// Handle the specific case where backticks are malformed across lines
	// This is a more aggressive approach to clean up broken markdown

	// First, handle obvious multi-line inline code spans
	multiLineCodeRegex := regexp.MustCompile("(?s)`([^`]*?\n[^`]*?)`")
	content = multiLineCodeRegex.ReplaceAllStringFunc(content, func(match string) string {
		codeContent := strings.Trim(match, "`")
		codeContent = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(codeContent), " ")
		if strings.TrimSpace(codeContent) == "" {
			return ""
		}
		return "[" + codeContent + "]"
	})

	// Handle the common broken pattern: ` followed by newline followed by `
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
			return "[" + cleaned + "]"
		}
		return ""
	})

	// Final cleanup: remove excessive empty lines that might have been created
	content = regexp.MustCompile("\n{3,}").ReplaceAllString(content, "\n\n")

	return content
}
