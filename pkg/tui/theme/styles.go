package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Base16 color palette with orange, brown, yellow, and pink tones
// Based on Autumn theme with warm earth tones
var (
	// Base colors (backgrounds and text)
	ColorBase00 = lipgloss.Color("#1a1816") // Dark background
	ColorBase01 = lipgloss.Color("#282420") // Lighter background
	ColorBase02 = lipgloss.Color("#36302a") // Selection background
	ColorBase03 = lipgloss.Color("#5c5044") // Comments, invisibles
	ColorBase04 = lipgloss.Color("#83715f") // Dark foreground
	ColorBase05 = lipgloss.Color("#ab937b") // Default foreground
	ColorBase06 = lipgloss.Color("#d3b597") // Light foreground
	ColorBase07 = lipgloss.Color("#f5d7b9") // Lightest foreground

	// Accent colors (syntax highlighting)
	ColorRed    = lipgloss.Color("#d95f5f") // Errors, deletions
	ColorOrange = lipgloss.Color("#eb8755") // Integers, booleans
	ColorYellow = lipgloss.Color("#f5b761") // Warnings, strings
	ColorGreen  = lipgloss.Color("#93b56b") // Success, additions
	ColorCyan   = lipgloss.Color("#61afaf") // Support, regex
	ColorBlue   = lipgloss.Color("#6b93b5") // Functions, methods
	ColorPurple = lipgloss.Color("#976bb5") // Keywords, storage
	ColorBrown  = lipgloss.Color("#b57f6b") // Deprecated, special

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
	ColorMagenta = lipgloss.Color("#d33682") // Magenta for agent spawning
	ColorViolet  = lipgloss.Color("#6c71c4") // Violet for planning operations
)

// Styles defines the Lipgloss styles for the TUI components
type Styles struct {
	// Layout styles
	ChatBody   lipgloss.Style
	ChatEvents lipgloss.Style
	ChatInput  lipgloss.Style
	ChatFooter lipgloss.Style

	// Input field styles
	InputBorder      lipgloss.Style
	InputPrompt      lipgloss.Style
	InputText        lipgloss.Style
	InputPlaceholder lipgloss.Style
	InputCursor      lipgloss.Style

	// Text styles
	UserMessage      lipgloss.Style
	AssistantMessage lipgloss.Style
	SystemMessage    lipgloss.Style
	ErrorMessage     lipgloss.Style
	InfoMessage      lipgloss.Style
	SuccessMessage   lipgloss.Style

	// General styles
	Focused   lipgloss.Style
	Unfocused lipgloss.Style
}

// DefaultStyles returns the default Lipgloss styles
func DefaultStyles() *Styles {
	return &Styles{
		// Layout styles with backgrounds
		ChatBody: lipgloss.NewStyle().
			Background(ColorSelection).
			Foreground(ColorBase05).
			Align(lipgloss.Center),

		ChatEvents: lipgloss.NewStyle().
			Background(ColorBase02).
			Foreground(ColorInfo).
			Align(lipgloss.Center).
			Height(3),

		ChatInput: lipgloss.NewStyle().
			Background(ColorBase01).
			Padding(1, 2). // Vertical and horizontal padding
			Height(5),

		ChatFooter: lipgloss.NewStyle().
			Background(ColorBase00).
			Foreground(ColorMuted).
			Align(lipgloss.Center).
			Height(2),

		// Input field styles
		InputBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorFocus).
			Padding(1, 2), // Internal padding

		InputPrompt: lipgloss.NewStyle().
			Foreground(ColorFocus).
			Bold(true),

		InputText: lipgloss.NewStyle().
			Foreground(ColorBase05),

		InputPlaceholder: lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true),

		InputCursor: lipgloss.NewStyle().
			Foreground(ColorFocus).
			Background(ColorFocus),

		// Message styles
		UserMessage: lipgloss.NewStyle().
			Foreground(ColorGreen),

		AssistantMessage: lipgloss.NewStyle().
			Foreground(ColorBlue),

		SystemMessage: lipgloss.NewStyle().
			Foreground(ColorPurple),

		ErrorMessage: lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true),

		InfoMessage: lipgloss.NewStyle().
			Foreground(ColorInfo),

		SuccessMessage: lipgloss.NewStyle().
			Foreground(ColorSuccess),

		// Focus states
		Focused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorFocus),

		Unfocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBase03),
	}
}

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)
