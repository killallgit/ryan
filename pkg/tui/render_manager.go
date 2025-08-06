package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/killallgit/ryan/pkg/logger"
)

// ContentType represents the type of content to render
type ContentType int

const (
	ContentTypePlain ContentType = iota
	ContentTypeMarkdown
	ContentTypeCode
	ContentTypeJSON
	ContentTypeThinking
	ContentTypeMixed
)

// RenderManager handles unified rendering of different content types
type RenderManager struct {
	theme *Theme
	width int
	log   *logger.Logger
}

// NewRenderManager creates a new render manager with the given theme
func NewRenderManager(theme *Theme, width int) (*RenderManager, error) {
	rm := &RenderManager{
		theme: theme,
		width: width,
		log:   logger.WithComponent("render_manager"),
	}
	return rm, nil
}

// getTviewColor converts a color to tview format
func (rm *RenderManager) getTviewColor(color string) string {
	return color // tview uses hex colors directly
}

// Render renders content based on its type
func (rm *RenderManager) Render(content string, contentType ContentType, role string) string {
	switch contentType {
	case ContentTypeMarkdown:
		return rm.renderMarkdown(content, role)
	case ContentTypeCode:
		return rm.renderCode(content, "", role)
	case ContentTypeJSON:
		return rm.renderCode(content, "json", role)
	case ContentTypeThinking:
		return rm.renderThinking(content)
	case ContentTypeMixed:
		return rm.renderMixed(content, role)
	default:
		return rm.renderPlain(content, role)
	}
}

// DetectContentType analyzes content and determines its type
func (rm *RenderManager) DetectContentType(content string) ContentType {
	// Check for thinking blocks
	if strings.Contains(content, "<think") || strings.Contains(content, "</think") {
		return ContentTypeMixed
	}

	// Check for JSON first (before markdown, as JSON can contain brackets)
	trimmed := strings.TrimSpace(content)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		return ContentTypeJSON
	}

	// Check for markdown indicators
	if rm.hasMarkdownIndicators(content) {
		// Check if it also has code blocks
		if strings.Contains(content, "```") {
			return ContentTypeMixed
		}
		return ContentTypeMarkdown
	}

	// Check for code patterns
	if rm.hasCodePatterns(content) {
		return ContentTypeCode
	}

	return ContentTypePlain
}

// hasMarkdownIndicators checks if content has markdown formatting
func (rm *RenderManager) hasMarkdownIndicators(content string) bool {
	indicators := []string{
		"# ", "## ", "### ", // Headers
		"- ", "* ", "+ ", // Lists
		"1. ", "2. ", "3. ", // Numbered lists
		"**", "__", // Bold
		"*", "_", // Italic (if not in code)
		"[", "](", // Links
		"```", "`", // Code blocks/inline
		"> ",                // Blockquotes
		"---", "***", "___", // Horizontal rules
		"|", // Tables (check for multiple)
	}

	for _, indicator := range indicators {
		if strings.Contains(content, indicator) {
			// Special handling for tables - need multiple pipes
			if indicator == "|" {
				if strings.Count(content, "|") > 2 {
					return true
				}
			} else {
				return true
			}
		}
	}

	return false
}

// hasCodePatterns checks if content looks like code
func (rm *RenderManager) hasCodePatterns(content string) bool {
	codePatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\s*(func|function|def|class|interface|struct|type)\s+\w+`),
		regexp.MustCompile(`^\s*(if|for|while|switch|case)\s*\(`),
		regexp.MustCompile(`^\s*(import|require|include|use)\s+`),
		regexp.MustCompile(`^\s*(var|let|const)\s+\w+\s*=`),
		regexp.MustCompile(`\w+\s*\(\s*.*\s*\)\s*{`),
		regexp.MustCompile(`;\s*$`),
	}

	lines := strings.Split(content, "\n")
	codeLineCount := 0

	for _, line := range lines {
		for _, pattern := range codePatterns {
			if pattern.MatchString(line) {
				codeLineCount++
				break
			}
		}
	}

	// If more than 30% of lines look like code, treat as code
	return float64(codeLineCount)/float64(len(lines)) > 0.3
}

// renderMarkdown renders markdown content using tview formatting
func (rm *RenderManager) renderMarkdown(content string, role string) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// Headers
		if strings.HasPrefix(line, "# ") {
			result.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]", ColorOrange, strings.TrimPrefix(line, "# ")))
		} else if strings.HasPrefix(line, "## ") {
			result.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]", ColorYellow, strings.TrimPrefix(line, "## ")))
		} else if strings.HasPrefix(line, "### ") {
			result.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]", ColorBlue, strings.TrimPrefix(line, "### ")))
		} else if strings.HasPrefix(line, "> ") {
			// Blockquote
			result.WriteString(fmt.Sprintf("[%s]  %s[-]", ColorBase04, strings.TrimPrefix(line, "> ")))
		} else if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "+ ") {
			// Lists
			bullet := strings.Split(line, " ")[0]
			listContent := strings.TrimSpace(strings.TrimPrefix(line, bullet))
			result.WriteString(fmt.Sprintf("[%s]%s[-] %s", ColorCyan, bullet, rm.formatInlineMarkdown(listContent)))
		} else if regexp.MustCompile(`^\d+\. `).MatchString(line) {
			// Numbered lists
			parts := strings.SplitN(line, ". ", 2)
			if len(parts) == 2 {
				result.WriteString(fmt.Sprintf("[%s]%s.[-] %s", ColorCyan, parts[0], rm.formatInlineMarkdown(parts[1])))
			} else {
				result.WriteString(rm.formatInlineMarkdown(line))
			}
		} else if strings.HasPrefix(line, "```") {
			// Skip code block markers, they'll be handled separately
			continue
		} else {
			// Regular text with inline formatting
			result.WriteString(rm.formatInlineMarkdown(line))
		}

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// formatInlineMarkdown handles bold, italic, inline code, and links
func (rm *RenderManager) formatInlineMarkdown(text string) string {
	// Bold text **text**
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	text = boldRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := strings.Trim(match, "*")
		return fmt.Sprintf("[%s::b]%s[-:-:-]", rm.getTviewColor(rm.theme.Foreground), content)
	})

	// Italic text *text*
	italicRegex := regexp.MustCompile(`\*([^*]+)\*`)
	text = italicRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := strings.Trim(match, "*")
		return fmt.Sprintf("[%s::i]%s[-:-:-]", rm.getTviewColor(rm.theme.Foreground), content)
	})

	// Inline code `code`
	codeRegex := regexp.MustCompile("`([^`]+)`")
	text = codeRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := strings.Trim(match, "`")
		return fmt.Sprintf("[%s:%s]%s[-:-]", ColorCyan, ColorBase01, content)
	})

	// Links [text](url)
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	text = linkRegex.ReplaceAllStringFunc(text, func(match string) string {
		linkText := regexp.MustCompile(`\[([^\]]+)\]`).FindStringSubmatch(match)
		if len(linkText) > 1 {
			return fmt.Sprintf("[%s::u]%s[-:-:-]", ColorCyan, linkText[1])
		}
		return match
	})

	return text
}

// renderCode renders code with syntax highlighting using tview colors
func (rm *RenderManager) renderCode(content string, language string, role string) string {
	// Simple syntax highlighting using tview colors
	lines := strings.Split(content, "\n")
	var result strings.Builder

	// Add background for code block
	result.WriteString(fmt.Sprintf("[:%s]", ColorBase01))

	for i, line := range lines {
		highlighted := rm.highlightSyntax(line, language)
		result.WriteString(" " + highlighted + " ") // Add padding
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	result.WriteString("[-:-]") // Reset background
	return result.String()
}

// highlightSyntax provides basic syntax highlighting for common languages
func (rm *RenderManager) highlightSyntax(line string, language string) string {
	switch language {
	case "go", "golang":
		return rm.highlightGo(line)
	case "python", "py":
		return rm.highlightPython(line)
	case "json":
		return rm.highlightJSON(line)
	case "javascript", "js", "typescript", "ts":
		return rm.highlightJS(line)
	default:
		return line
	}
}

// highlightGo provides Go syntax highlighting
func (rm *RenderManager) highlightGo(line string) string {
	// Keywords
	keywords := []string{"func", "var", "const", "type", "struct", "interface", "package", "import", "if", "else", "for", "range", "return", "defer", "go", "chan", "select", "switch", "case", "default"}
	for _, keyword := range keywords {
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(keyword))
		re := regexp.MustCompile(pattern)
		line = re.ReplaceAllStringFunc(line, func(match string) string {
			return fmt.Sprintf("[%s]%s[-]", ColorPurple, match)
		})
	}

	// Types
	types := []string{"string", "int", "int64", "float64", "bool", "error", "byte", "rune"}
	for _, t := range types {
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(t))
		re := regexp.MustCompile(pattern)
		line = re.ReplaceAllStringFunc(line, func(match string) string {
			return fmt.Sprintf("[%s]%s[-]", ColorBlue, match)
		})
	}

	// Strings
	stringRegex := regexp.MustCompile(`"([^"\\]|\\.)*"`)
	line = stringRegex.ReplaceAllStringFunc(line, func(match string) string {
		return fmt.Sprintf("[%s]%s[-]", ColorGreen, match)
	})

	// Comments
	commentRegex := regexp.MustCompile(`//.*$`)
	line = commentRegex.ReplaceAllStringFunc(line, func(match string) string {
		return fmt.Sprintf("[%s]%s[-]", ColorBase03, match)
	})

	return line
}

// highlightPython provides Python syntax highlighting
func (rm *RenderManager) highlightPython(line string) string {
	// Keywords
	keywords := []string{"def", "class", "if", "elif", "else", "for", "while", "return", "import", "from", "as", "try", "except", "finally", "with", "lambda", "and", "or", "not", "in", "is"}
	for _, keyword := range keywords {
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(keyword))
		re := regexp.MustCompile(pattern)
		line = re.ReplaceAllStringFunc(line, func(match string) string {
			return fmt.Sprintf("[%s]%s[-]", ColorPurple, match)
		})
	}

	// Strings (both single and double quotes)
	stringRegex := regexp.MustCompile(`(['"])([^'\\]|\\.)*\1`)
	line = stringRegex.ReplaceAllStringFunc(line, func(match string) string {
		return fmt.Sprintf("[%s]%s[-]", ColorGreen, match)
	})

	// Comments
	commentRegex := regexp.MustCompile(`#.*$`)
	line = commentRegex.ReplaceAllStringFunc(line, func(match string) string {
		return fmt.Sprintf("[%s]%s[-]", ColorBase03, match)
	})

	return line
}

// highlightJS provides JavaScript/TypeScript syntax highlighting
func (rm *RenderManager) highlightJS(line string) string {
	// Keywords
	keywords := []string{"function", "const", "let", "var", "if", "else", "for", "while", "return", "import", "export", "from", "class", "extends", "try", "catch", "finally", "async", "await"}
	for _, keyword := range keywords {
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(keyword))
		re := regexp.MustCompile(pattern)
		line = re.ReplaceAllStringFunc(line, func(match string) string {
			return fmt.Sprintf("[%s]%s[-]", ColorPurple, match)
		})
	}

	// Strings
	stringRegex := regexp.MustCompile("(['\"`])([^'\\\\]|\\\\.)*\\1")
	line = stringRegex.ReplaceAllStringFunc(line, func(match string) string {
		return fmt.Sprintf("[%s]%s[-]", ColorGreen, match)
	})

	// Comments
	commentRegex := regexp.MustCompile(`//.*$`)
	line = commentRegex.ReplaceAllStringFunc(line, func(match string) string {
		return fmt.Sprintf("[%s]%s[-]", ColorBase03, match)
	})

	return line
}

// highlightJSON provides JSON syntax highlighting
func (rm *RenderManager) highlightJSON(line string) string {
	// Keys - look for quoted strings followed by colon (simple approach)
	if strings.Contains(line, ":") {
		// Split on colon to separate keys from values
		colonIndex := strings.Index(line, ":")
		before := line[:colonIndex]
		after := line[colonIndex:]

		// Highlight quoted strings in the key part
		keyRegex := regexp.MustCompile(`"([^"\\]|\\.)*"`)
		before = keyRegex.ReplaceAllStringFunc(before, func(match string) string {
			return fmt.Sprintf("[%s]%s[-]", ColorBlue, match)
		})

		// Highlight values
		// String values
		stringRegex := regexp.MustCompile(`"([^"\\]|\\.)*"`)
		after = stringRegex.ReplaceAllStringFunc(after, func(match string) string {
			return fmt.Sprintf("[%s]%s[-]", ColorGreen, match)
		})

		// Numbers
		numberRegex := regexp.MustCompile(`\b\d+(\.\d+)?\b`)
		after = numberRegex.ReplaceAllStringFunc(after, func(match string) string {
			return fmt.Sprintf("[%s]%s[-]", ColorOrange, match)
		})

		// Booleans and null
		boolRegex := regexp.MustCompile(`\b(true|false|null)\b`)
		after = boolRegex.ReplaceAllStringFunc(after, func(match string) string {
			return fmt.Sprintf("[%s]%s[-]", ColorRed, match)
		})

		return before + after
	}

	// For lines without colons, still highlight strings
	stringRegex := regexp.MustCompile(`"([^"\\]|\\.)*"`)
	line = stringRegex.ReplaceAllStringFunc(line, func(match string) string {
		return fmt.Sprintf("[%s]%s[-]", ColorGreen, match)
	})

	return line
}

// renderThinking renders thinking blocks with special formatting
func (rm *RenderManager) renderThinking(content string) string {
	return fmt.Sprintf("[%s::i]%s[-:-:-]", ColorBase03, content)
}

// renderMixed handles content with multiple types (markdown, code, thinking blocks)
func (rm *RenderManager) renderMixed(content string, role string) string {
	var result strings.Builder

	// Process thinking blocks first
	content = rm.processThinkingBlocks(content, &result)

	// Process code blocks
	codeBlockRegex := regexp.MustCompile("(?s)```(\\w*)\\n(.+?)```")
	content = codeBlockRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := codeBlockRegex.FindStringSubmatch(match)
		if len(matches) >= 3 {
			language := matches[1]
			code := matches[2]
			return rm.renderCode(code, language, role)
		}
		return match
	})

	// Then process remaining content as markdown
	if content != "" {
		rendered := rm.renderMarkdown(content, role)
		result.WriteString(rendered)
	}

	return result.String()
}

// processThinkingBlocks extracts and formats thinking blocks
func (rm *RenderManager) processThinkingBlocks(content string, result *strings.Builder) string {
	thinkPattern := regexp.MustCompile(`(?s)<think(?:ing)?>(.*?)</think(?:ing)?>`)

	lastEnd := 0
	for _, match := range thinkPattern.FindAllStringSubmatchIndex(content, -1) {
		// Add content before the thinking block
		if lastEnd < match[0] {
			beforeContent := content[lastEnd:match[0]]
			if beforeContent != "" {
				result.WriteString(rm.renderMarkdown(beforeContent, "assistant"))
			}
		}

		// Add the thinking block with special formatting
		thinkingContent := content[match[2]:match[3]]
		result.WriteString(rm.renderThinking(thinkingContent))

		lastEnd = match[1]
	}

	// Return remaining content after last thinking block
	if lastEnd < len(content) {
		return content[lastEnd:]
	}

	return ""
}

// renderPlain renders plain text with role-based styling
func (rm *RenderManager) renderPlain(content string, role string) string {
	switch role {
	case "user":
		return fmt.Sprintf("[%s]%s[-]", ColorGreen, content)
	case "assistant":
		return fmt.Sprintf("[%s]%s[-]", ColorBlue, content)
	case "system":
		return fmt.Sprintf("[%s]%s[-]", ColorPurple, content)
	case "error":
		return fmt.Sprintf("[%s::b]%s[-:-:-]", ColorRed, content)
	default:
		return content
	}
}

// RenderToolOutput renders tool output with special formatting
func (rm *RenderManager) RenderToolOutput(toolName string, output string) string {
	header := fmt.Sprintf("[%s]🔧 %s[-]", ColorCyan, toolName)

	// Detect if output is JSON
	if rm.DetectContentType(output) == ContentTypeJSON {
		output = rm.renderCode(output, "json", "tool")
	}

	// Add border and formatting
	return fmt.Sprintf("[%s]│[-] %s\n[%s]│[-] %s", ColorBase03, header, ColorBase03, strings.ReplaceAll(output, "\n", "\n["+ColorBase03+"]│[-] "))
}

// RenderStreamingContent renders content that's being streamed
func (rm *RenderManager) RenderStreamingContent(content string, role string) string {
	// For streaming, we need to be careful about incomplete markdown/code blocks
	// Just apply basic formatting without full markdown parsing

	// Process thinking blocks even in streaming
	var result strings.Builder
	content = rm.processStreamingThinkingBlocks(content, &result)

	if content != "" {
		// Apply role styling without markdown processing to avoid incomplete block issues
		styled := rm.renderPlain(content, role)
		result.WriteString(styled)
	}

	// Add cursor for streaming
	result.WriteString(fmt.Sprintf("[%s]█[-]", ColorOrange))

	return result.String()
}

// processStreamingThinkingBlocks handles thinking blocks in streaming content
func (rm *RenderManager) processStreamingThinkingBlocks(content string, result *strings.Builder) string {
	// Handle complete thinking blocks
	thinkPattern := regexp.MustCompile(`(?s)<think(?:ing)?>(.*?)</think(?:ing)?>`)

	lastEnd := 0
	for _, match := range thinkPattern.FindAllStringSubmatchIndex(content, -1) {
		// Add content before the thinking block
		if lastEnd < match[0] {
			beforeContent := content[lastEnd:match[0]]
			if beforeContent != "" {
				result.WriteString(beforeContent)
			}
		}

		// Add the thinking block with special formatting
		thinkingContent := content[match[2]:match[3]]
		result.WriteString(rm.renderThinking(thinkingContent))

		lastEnd = match[1]
	}

	// Check for incomplete thinking block at the end
	remaining := content[lastEnd:]
	if strings.Contains(remaining, "<think") && !strings.Contains(remaining, "</think") {
		// We have an incomplete thinking block
		idx := strings.Index(remaining, "<think")
		if idx > 0 {
			result.WriteString(remaining[:idx])
		}
		// Start thinking format for the incomplete block
		result.WriteString(rm.renderThinking(remaining[idx:]))
		return ""
	}

	return remaining
}

// SetWidth updates the render width
func (rm *RenderManager) SetWidth(width int) {
	rm.width = width
}

// RenderList renders a list with consistent formatting
func (rm *RenderManager) RenderList(items []string, selectedIndex int) string {
	var result strings.Builder
	for i, item := range items {
		if i == selectedIndex {
			result.WriteString(fmt.Sprintf("[%s:%s]▸ %s[-:-]", ColorCyan, ColorBase01, item))
		} else {
			result.WriteString(fmt.Sprintf("  %s", item))
		}
		if i < len(items)-1 {
			result.WriteString("\n")
		}
	}
	return result.String()
}

// RenderTable renders table data with consistent formatting
func (rm *RenderManager) RenderTable(headers []string, rows [][]string) string {
	var result strings.Builder

	// Render headers
	for i, header := range headers {
		result.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]", ColorYellow, header))
		if i < len(headers)-1 {
			result.WriteString(" | ")
		}
	}
	result.WriteString("\n")
	result.WriteString(strings.Repeat("─", rm.width))
	result.WriteString("\n")

	// Render rows
	for _, row := range rows {
		for i, cell := range row {
			result.WriteString(cell)
			if i < len(row)-1 {
				result.WriteString(" | ")
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

// RenderStatus renders a status message with appropriate styling
func (rm *RenderManager) RenderStatus(status string, statusType string) string {
	var color string
	switch statusType {
	case "success":
		color = ColorGreen
	case "error":
		color = ColorRed
	case "warning":
		color = ColorYellow
	case "info":
		color = ColorCyan
	default:
		color = ColorBase05
	}
	return fmt.Sprintf("[%s]%s[-]", color, status)
}

// RenderProgress renders a progress bar
func (rm *RenderManager) RenderProgress(current, total int, label string) string {
	if total == 0 {
		return fmt.Sprintf("[%s]%s: --%%[-]", ColorCyan, label)
	}

	percentage := float64(current) / float64(total) * 100
	barWidth := 20
	filled := int(float64(barWidth) * (percentage / 100))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	return fmt.Sprintf("[%s]%s: [%s] %.1f%%[-]", ColorCyan, label, bar, percentage)
}

// RenderTree renders tree-structured data
func (rm *RenderManager) RenderTree(node string, level int, isLast bool, hasChildren bool) string {
	indent := strings.Repeat("  ", level)

	var prefix string
	if level > 0 {
		if isLast {
			prefix = "└─"
		} else {
			prefix = "├─"
		}
	}

	var icon string
	if hasChildren {
		icon = "▸"
	} else {
		icon = "•"
	}

	return fmt.Sprintf("%s[%s]%s %s %s[-]", indent, ColorBase04, prefix, icon, node)
}

// RenderHeader renders a section header
func (rm *RenderManager) RenderHeader(text string, level int) string {
	colors := []string{ColorOrange, ColorYellow, ColorBlue}
	if level < 1 || level > len(colors) {
		level = 1
	}
	return fmt.Sprintf("[%s::b]%s[-:-:-]", colors[level-1], text)
}

// RenderKeyValue renders key-value pairs
func (rm *RenderManager) RenderKeyValue(key, value string) string {
	return fmt.Sprintf("[%s]%s:[-] %s", ColorCyan, key, value)
}

// RenderHighlight highlights specific text within content
func (rm *RenderManager) RenderHighlight(content, highlight string) string {
	if highlight == "" {
		return content
	}
	highlighted := strings.ReplaceAll(content, highlight,
		fmt.Sprintf("[%s:%s]%s[-:-]", ColorYellow, ColorBase02, highlight))
	return highlighted
}
