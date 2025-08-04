package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/gdamore/tcell/v2"
)

// MarkdownFormatter provides rich markdown formatting using Charm's Glamour
type MarkdownFormatter struct {
	width    int
	renderer *glamour.TermRenderer
}

// NewMarkdownFormatter creates a new markdown formatter using glamour
func NewMarkdownFormatter(width int) (*MarkdownFormatter, error) {
	// Create a glamour renderer with no styling to avoid ANSI codes
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("notty"), // Use plain text style to avoid ANSI codes
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}

	return &MarkdownFormatter{
		width:    width,
		renderer: renderer,
	}, nil
}

// FormatMarkdown formats the entire content as markdown using glamour
func (mf *MarkdownFormatter) FormatMarkdown(content string) ([]FormattedLine, error) {

	// Render the markdown content using glamour
	rendered, err := mf.renderer.Render(content)
	if err != nil {
		// Fallback to simple text formatting
		return mf.fallbackToSimpleText(content), nil
	}

	// Strip any remaining ANSI codes and split into lines
	cleanRendered := mf.stripANSICodes(rendered)
	lines := strings.Split(strings.TrimRight(cleanRendered, "\n"), "\n")

	var formattedLines []FormattedLine
	for _, line := range lines {
		// Clean the line and apply appropriate styling
		cleanLine := strings.TrimRight(line, " \t")

		formattedLines = append(formattedLines, FormattedLine{
			Content: cleanLine,
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
			Indent:  0,
		})
	}

	return formattedLines, nil
}

// stripANSICodes removes ANSI escape sequences from text
func (mf *MarkdownFormatter) stripANSICodes(text string) string {
	// Regex to match ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(text, "")
}

// fallbackToSimpleText provides a simple fallback when glamour fails
func (mf *MarkdownFormatter) fallbackToSimpleText(content string) []FormattedLine {
	var formattedLines []FormattedLine
	lines := WrapText(content, mf.width)

	for _, line := range lines {
		formattedLines = append(formattedLines, FormattedLine{
			Content: line,
			Style:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
			Indent:  0,
		})
	}

	return formattedLines
}
