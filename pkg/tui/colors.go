package tui

import "github.com/gdamore/tcell/v2"

// Color constants for the 1980s-inspired theme using muted pastels and base16 colors
var (
	// Primary text colors
	ColorUserText      = tcell.NewRGBColor(255, 176, 0)   // Warm amber/orange - for user messages
	ColorAssistantText = tcell.NewRGBColor(0, 255, 135)   // Mint green - for assistant messages
	ColorSystemText    = tcell.NewRGBColor(255, 255, 128) // Pale yellow - for system messages
	ColorToolText      = tcell.NewRGBColor(255, 128, 255) // Soft magenta - for tool outputs

	// UI element colors
	ColorBorder        = tcell.NewRGBColor(255, 215, 0)   // Gold - for borders
	ColorBorderActive  = tcell.NewRGBColor(255, 165, 0)   // Orange - for active borders
	ColorBorderError   = tcell.NewRGBColor(255, 99, 71)   // Tomato - for error borders
	ColorBorderSuccess = tcell.NewRGBColor(50, 205, 50)   // Lime green - for success borders
	ColorHeaderText    = tcell.NewRGBColor(175, 175, 175) // Light gray - for muted headers

	// Background colors
	ColorBackground         = tcell.ColorBlack              // Classic terminal black
	ColorBackgroundSelected = tcell.NewRGBColor(64, 64, 64) // Dark gray - for selections
	ColorBackgroundHover    = tcell.NewRGBColor(48, 48, 48) // Slightly lighter than black

	// Text style colors
	ColorDimText     = tcell.NewRGBColor(169, 169, 169) // Dark gray - for dim/secondary text
	ColorHighlight   = tcell.NewRGBColor(255, 255, 0)   // Pure yellow - for highlights
	ColorPrompt      = tcell.NewRGBColor(255, 192, 203) // Pink - for prompts
	ColorInstruction = tcell.NewRGBColor(176, 224, 230) // Powder blue - for instructions

	// Status colors
	ColorStatusReady   = tcell.NewRGBColor(144, 238, 144) // Light green - ready status
	ColorStatusBusy    = tcell.NewRGBColor(255, 218, 185) // Peach - busy/loading status
	ColorStatusError   = tcell.NewRGBColor(255, 182, 193) // Light pink - error status
	ColorStatusOffline = tcell.NewRGBColor(211, 211, 211) // Light gray - offline status

	// Progress bar colors
	ColorProgressBar   = tcell.NewRGBColor(0, 191, 255)   // Deep sky blue
	ColorProgressBarBg = tcell.NewRGBColor(105, 105, 105) // Dim gray

	// Menu colors
	ColorMenuSelected = tcell.NewRGBColor(255, 140, 0)   // Dark orange
	ColorMenuNormal   = tcell.NewRGBColor(255, 228, 181) // Moccasin
	ColorMenuShortcut = tcell.NewRGBColor(255, 20, 147)  // Deep pink

	// Token/metric colors
	ColorTokenCount = tcell.NewRGBColor(127, 255, 212) // Aquamarine
	ColorMetric     = tcell.NewRGBColor(176, 196, 222) // Light steel blue

	// Model-specific colors
	ColorModelName     = tcell.NewRGBColor(218, 112, 214) // Orchid
	ColorModelRunning  = tcell.NewRGBColor(50, 255, 50)   // Bright green
	ColorModelOffline  = tcell.NewRGBColor(255, 99, 71)   // Tomato
	ColorModelSelected = tcell.NewRGBColor(221, 160, 221) // Plum - soft lavender for list selection
	ColorModelCurrent  = tcell.NewRGBColor(255, 215, 0)   // Gold - bright color for current model name
)

// Style presets combining colors with text attributes
var (
	StyleUserText      = tcell.StyleDefault.Foreground(ColorUserText)
	StyleAssistantText = tcell.StyleDefault.Foreground(ColorAssistantText)
	StyleSystemText    = tcell.StyleDefault.Foreground(ColorSystemText)
	StyleToolText      = tcell.StyleDefault.Foreground(ColorToolText)

	StyleBorder       = tcell.StyleDefault.Foreground(ColorBorder)
	StyleBorderActive = tcell.StyleDefault.Foreground(ColorBorderActive)
	StyleBorderError  = tcell.StyleDefault.Foreground(ColorBorderError)
	StyleHeaderText   = tcell.StyleDefault.Foreground(ColorHeaderText).Bold(true)

	StyleDimText     = tcell.StyleDefault.Foreground(ColorDimText).Dim(true)
	StyleHighlight   = tcell.StyleDefault.Foreground(ColorHighlight).Bold(true)
	StylePrompt      = tcell.StyleDefault.Foreground(ColorPrompt)
	StyleInstruction = tcell.StyleDefault.Foreground(ColorInstruction)

	StyleMenuSelected = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(ColorMenuSelected)
	StyleMenuNormal   = tcell.StyleDefault.Foreground(ColorMenuNormal)

	StyleStatusReady   = tcell.StyleDefault.Foreground(ColorStatusReady)
	StyleStatusBusy    = tcell.StyleDefault.Foreground(ColorStatusBusy)
	StyleStatusError   = tcell.StyleDefault.Foreground(ColorStatusError)
	StyleStatusOffline = tcell.StyleDefault.Foreground(ColorStatusOffline).StrikeThrough(true)

	StyleTokenCount    = tcell.StyleDefault.Foreground(ColorTokenCount)
	StyleMetric        = tcell.StyleDefault.Foreground(ColorMetric)
	StyleModelCurrent  = tcell.StyleDefault.Foreground(ColorModelCurrent).Bold(true)
)
