package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Base16 color palette with orange, brown, yellow, and pink tones
// Based on Autumn theme with warm earth tones
// Using hex color strings compatible with tview
var (
	// Base colors (backgrounds and text)
	ColorBase00 = "#1a1816" // Dark background
	ColorBase01 = "#282420" // Lighter background
	ColorBase02 = "#36302a" // Selection background
	ColorBase03 = "#5c5044" // Comments, invisibles
	ColorBase04 = "#83715f" // Dark foreground
	ColorBase05 = "#ab937b" // Default foreground
	ColorBase06 = "#d3b597" // Light foreground
	ColorBase07 = "#f5d7b9" // Lightest foreground

	// Accent colors (syntax highlighting)
	ColorRed    = "#d95f5f" // Errors, deletions
	ColorOrange = "#eb8755" // Integers, booleans
	ColorYellow = "#f5b761" // Warnings, strings
	ColorGreen  = "#93b56b" // Success, additions
	ColorCyan   = "#61afaf" // Support, regex
	ColorBlue   = "#6b93b5" // Functions, methods
	ColorPurple = "#976bb5" // Keywords, storage
	ColorBrown  = "#b57f6b" // Deprecated, special

	// UI specific colors
	ColorBorder    = ColorBase03
	ColorSelection = ColorBase02
	ColorFocus     = ColorOrange
	ColorSuccess   = ColorGreen
	ColorWarning   = ColorYellow
	ColorError     = ColorRed
	ColorInfo      = ColorCyan
	ColorMuted     = ColorBase03
	ColorHighlight = ColorYellow

	// Additional colors for activity indicators
	ColorMagenta = "#d33682" // Magenta for agent spawning
	ColorViolet  = "#6c71c4" // Violet for planning operations
)

// Theme represents the color theme configuration
type Theme struct {
	Background       string
	Foreground       string
	Border           string
	Selection        string
	Focus            string
	Success          string
	Warning          string
	Error            string
	Info             string
	Muted            string
	Highlight        string
	UserMessage      string
	AssistantMessage string
	SystemMessage    string
	ThinkingText     string
	StreamingCursor  string
}

// DefaultTheme returns the default warm base16 theme
func DefaultTheme() *Theme {
	return &Theme{
		Background:       ColorBase00,
		Foreground:       ColorBase05,
		Border:           ColorBase03,
		Selection:        ColorBase02,
		Focus:            ColorFocus,
		Success:          ColorSuccess,
		Warning:          ColorWarning,
		Error:            ColorError,
		Info:             ColorInfo,
		Muted:            ColorMuted,
		Highlight:        ColorHighlight,
		UserMessage:      ColorGreen,
		AssistantMessage: ColorBlue,
		SystemMessage:    ColorPurple,
		ThinkingText:     ColorMuted,
		StreamingCursor:  ColorOrange,
	}
}

// ApplyTheme applies the theme to tview defaults using tcell color functions
func ApplyTheme(theme *Theme) {
	// Use tcell.GetColor to parse hex color strings (with # prefix)
	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(theme.Background)
	tview.Styles.ContrastBackgroundColor = tcell.GetColor(theme.Background)
	tview.Styles.MoreContrastBackgroundColor = tcell.GetColor(theme.Background)
	tview.Styles.BorderColor = tcell.GetColor(theme.Border)
	tview.Styles.TitleColor = tcell.GetColor(theme.Foreground)
	tview.Styles.GraphicsColor = tcell.GetColor(theme.Border)
}
