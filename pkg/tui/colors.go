package tui

import "github.com/gdamore/tcell/v2"

// Base16 color palette with orange, brown, yellow, and pink tones
// Based on Autumn theme with warm earth tones
var (
	// Base colors (backgrounds and text)
	ColorBase00 = tcell.NewRGBColor(26, 24, 22)    // #1a1816 - Dark background
	ColorBase01 = tcell.NewRGBColor(40, 36, 32)    // #282420 - Lighter background
	ColorBase02 = tcell.NewRGBColor(54, 48, 42)    // #36302a - Selection background
	ColorBase03 = tcell.NewRGBColor(92, 80, 68)    // #5c5044 - Comments, invisibles
	ColorBase04 = tcell.NewRGBColor(131, 113, 95)  // #83715f - Dark foreground
	ColorBase05 = tcell.NewRGBColor(171, 147, 123) // #ab937b - Default foreground
	ColorBase06 = tcell.NewRGBColor(211, 181, 151) // #d3b597 - Light foreground
	ColorBase07 = tcell.NewRGBColor(245, 215, 185) // #f5d7b9 - Lightest foreground

	// Accent colors (syntax highlighting)
	ColorRed    = tcell.NewRGBColor(217, 95, 95)   // #d95f5f - Errors, deletions
	ColorOrange = tcell.NewRGBColor(235, 135, 85)  // #eb8755 - Integers, booleans
	ColorYellow = tcell.NewRGBColor(245, 183, 97)  // #f5b761 - Warnings, strings
	ColorGreen  = tcell.NewRGBColor(147, 181, 107) // #93b56b - Success, additions
	ColorCyan   = tcell.NewRGBColor(97, 175, 175)  // #61afaf - Support, regex
	ColorBlue   = tcell.NewRGBColor(107, 147, 181) // #6b93b5 - Functions, methods
	ColorPurple = tcell.NewRGBColor(151, 107, 181) // #976bb5 - Keywords, storage
	ColorBrown  = tcell.NewRGBColor(181, 127, 107) // #b57f6b - Deprecated, special

	// UI specific colors
	ColorBorder     = ColorBase03
	ColorSelection  = ColorBase02
	ColorFocus      = ColorOrange
	ColorSuccess    = ColorGreen
	ColorWarning    = ColorYellow
	ColorError      = ColorRed
	ColorInfo       = ColorCyan
	ColorMuted      = ColorBase03
	ColorHighlight  = ColorYellow
)

// Theme represents the color theme configuration
type Theme struct {
	Background      tcell.Color
	Foreground      tcell.Color
	Border          tcell.Color
	Selection       tcell.Color
	Focus           tcell.Color
	Success         tcell.Color
	Warning         tcell.Color
	Error           tcell.Color
	Info            tcell.Color
	Muted           tcell.Color
	Highlight       tcell.Color
	UserMessage     tcell.Color
	AssistantMessage tcell.Color
	SystemMessage   tcell.Color
	StreamingCursor tcell.Color
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
		StreamingCursor:  ColorOrange,
	}
}

// ApplyTheme applies the theme to tview defaults
func ApplyTheme(theme *Theme) {
	// Set global tview theme colors
	// This affects the default styling of all components
	tcell.ColorNames["black"] = theme.Background
	tcell.ColorNames["white"] = theme.Foreground
	tcell.ColorNames["blue"] = theme.Info
	tcell.ColorNames["green"] = theme.Success
	tcell.ColorNames["yellow"] = theme.Warning
	tcell.ColorNames["red"] = theme.Error
	tcell.ColorNames["purple"] = ColorPurple
	tcell.ColorNames["cyan"] = ColorCyan
	tcell.ColorNames["orange"] = ColorOrange
}