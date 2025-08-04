package tui

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/logger"
)

// EnhancedFormatter provides advanced text formatting capabilities
type EnhancedFormatter struct {
	// Lipgloss styles for different content types
	thinkingBoxStyle lipgloss.Style
	codeBlockStyle   lipgloss.Style
	inlineCodeStyle  lipgloss.Style
	headerStyle      lipgloss.Style
	listStyle        lipgloss.Style

	// Chroma formatter for syntax highlighting
	chromaFormatter chroma.Formatter
	width           int
}

// NewEnhancedFormatter creates a new enhanced formatter with terminal-friendly styling
func NewEnhancedFormatter(width int) *EnhancedFormatter {
	// Create formatter with terminal-safe colors
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	return &EnhancedFormatter{
		width:           width,
		chromaFormatter: formatter,

		// Thinking block style - subtle box with dim colors
		thinkingBoxStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#555555")).
			Padding(0, 1).
			Margin(0, 1).
			Foreground(lipgloss.Color("#888888")).
			Italic(true),

		// Code block style - distinct box for code
		codeBlockStyle: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#FFD700")). // Gold border
			Padding(0, 1).
			Margin(0, 1).
			Background(lipgloss.Color("#1a1a1a")). // Slightly different background
			Foreground(lipgloss.Color("#FFFFFF")),

		// Inline code style - subtle highlighting
		inlineCodeStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#FFB000")). // Amber
			Padding(0, 1),

		// Header style - bold with underline
		headerStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6347")). // Tomato
			MarginTop(1).
			MarginBottom(1),

		// List style - simple indentation
		listStyle: lipgloss.NewStyle().
			MarginLeft(2).
			Foreground(lipgloss.Color("#98FB98")), // Pale green
	}
}

// FormatCodeBlock applies syntax highlighting and boxing to code content
func (ef *EnhancedFormatter) FormatCodeBlock(content, language string) string {
	if content == "" {
		return ""
	}

	log := logger.WithComponent("enhanced_formatting")
	log.Debug("FormatCodeBlock called",
		"content_length", len(content),
		"language", language)

	// Get lexer for the specified language
	var lexer chroma.Lexer
	if language != "" {
		lexer = lexers.Get(language)
	}
	if lexer == nil {
		lexer = lexers.Analyse(content)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}

	// Apply syntax highlighting
	var highlightedContent string
	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		log.Debug("Failed to tokenize code, using plain text", "error", err)
		highlightedContent = content
	} else {
		var buf strings.Builder
		err = ef.chromaFormatter.Format(&buf, styles.Get("monokai"), iterator)
		if err != nil {
			log.Debug("Failed to format code, using plain text", "error", err)
			highlightedContent = content
		} else {
			highlightedContent = buf.String()
		}
	}

	// Apply code block styling
	boxWidth := ef.width - 4 // Account for margins and padding
	if boxWidth < 30 {
		boxWidth = 30 // Minimum width for code
	}

	formatted := ef.codeBlockStyle.Width(boxWidth).Render(highlightedContent)

	log.Debug("FormatCodeBlock result", "formatted_length", len(formatted))
	return formatted
}

// FormatInlineCode applies subtle highlighting to inline code
func (ef *EnhancedFormatter) FormatInlineCode(content string) string {
	if content == "" {
		return ""
	}

	return ef.inlineCodeStyle.Render(content)
}

// FormatHeader applies header styling based on level
func (ef *EnhancedFormatter) FormatHeader(content string, level int) string {
	if content == "" {
		return ""
	}

	// Adjust styling based on header level
	style := ef.headerStyle
	switch level {
	case 1:
		style = style.Underline(true).MarginBottom(1)
	case 2:
		style = style.Foreground(lipgloss.Color("#FFA500")) // Orange
	case 3:
		style = style.Foreground(lipgloss.Color("#FFD700")) // Gold
	default:
		style = style.Foreground(lipgloss.Color("#FFFF99")) // Pale yellow
	}

	return style.Render(content)
}

// FormatList applies list styling with proper indentation
func (ef *EnhancedFormatter) FormatList(content string, level int) string {
	if content == "" {
		return ""
	}

	// Add bullet point and apply styling
	bullet := "•"
	listText := bullet + " " + content

	// Adjust indentation based on nesting level
	style := ef.listStyle.MarginLeft(level * 2)

	return style.Render(listText)
}

// FormatContentSegments formats parsed content segments with appropriate styling
func (ef *EnhancedFormatter) FormatContentSegments(segments []ContentSegment) []string {
	log := logger.WithComponent("enhanced_formatting")
	log.Debug("FormatContentSegments called", "segments_count", len(segments))

	var formattedLines []string

	for _, segment := range segments {
		var formatted string

		switch segment.Type {
		case ContentTypeCodeBlock:
			formatted = ef.FormatCodeBlock(segment.Content, segment.Language)
		case ContentTypeInlineCode:
			formatted = ef.FormatInlineCode(segment.Content)
		case ContentTypeHeader:
			formatted = ef.FormatHeader(segment.Content, segment.Level)
		case ContentTypeList:
			formatted = ef.FormatList(segment.Content, segment.Level)
		case ContentTypeText:
			formatted = segment.Content
		default:
			formatted = segment.Content
		}

		if formatted != "" {
			// Split multi-line formatted content
			lines := strings.Split(formatted, "\n")
			formattedLines = append(formattedLines, lines...)
		}
	}

	log.Debug("FormatContentSegments result", "formatted_lines_count", len(formattedLines))
	return formattedLines
}

// ConvertLipglossToTcell converts Lipgloss styles to tcell styles for terminal rendering
// This is a simplified conversion - Lipgloss handles most rendering internally
func ConvertLipglossToTcell(lipglossOutput string) (string, tcell.Style) {
	// For now, return the Lipgloss output as-is with default style
	// Lipgloss handles ANSI color codes which most terminals support
	return lipglossOutput, tcell.StyleDefault
}

// ShouldUseEnhancedFormatting determines if enhanced formatting should be applied
// based on content types detected
func ShouldUseEnhancedFormatting(contentTypes map[ContentType]bool) bool {
	// Use enhanced formatting if we have code blocks, headers, or complex content
	return contentTypes[ContentTypeCodeBlock] ||
		contentTypes[ContentTypeHeader] ||
		contentTypes[ContentTypeList] ||
		(contentTypes[ContentTypeInlineCode] && len(contentTypes) > 1)
}
