package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestColorConstants(t *testing.T) {
	// Test that all color constants are properly defined as hex strings
	assert.NotEmpty(t, ColorBase00)
	assert.NotEmpty(t, ColorBase01)
	assert.NotEmpty(t, ColorBase02)
	assert.NotEmpty(t, ColorBase03)
	assert.NotEmpty(t, ColorBase04)
	assert.NotEmpty(t, ColorBase05)
	assert.NotEmpty(t, ColorBase06)
	assert.NotEmpty(t, ColorBase07)

	// Test accent colors
	assert.NotEmpty(t, ColorRed)
	assert.NotEmpty(t, ColorOrange)
	assert.NotEmpty(t, ColorYellow)
	assert.NotEmpty(t, ColorGreen)
	assert.NotEmpty(t, ColorCyan)
	assert.NotEmpty(t, ColorBlue)
	assert.NotEmpty(t, ColorPurple)
	assert.NotEmpty(t, ColorBrown)

	// Test that colors are valid hex format
	assert.True(t, ColorBase00[0] == '#')
	assert.True(t, ColorRed[0] == '#')
	assert.Len(t, ColorBase00, 7) // #RRGGBB format
	assert.Len(t, ColorRed, 7)    // #RRGGBB format

	// Test UI specific colors
	assert.Equal(t, ColorBase03, ColorBorder)
	assert.Equal(t, ColorBase02, ColorSelection)
	assert.Equal(t, ColorOrange, ColorFocus)
	assert.Equal(t, ColorGreen, ColorSuccess)
	assert.Equal(t, ColorYellow, ColorWarning)
	assert.Equal(t, ColorRed, ColorError)
	assert.Equal(t, ColorCyan, ColorInfo)
	assert.Equal(t, ColorBase03, ColorMuted)
	assert.Equal(t, ColorYellow, ColorHighlight)
}

func TestTheme(t *testing.T) {
	theme := &Theme{
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

	assert.Equal(t, ColorBase00, theme.Background)
	assert.Equal(t, ColorBase05, theme.Foreground)
	assert.Equal(t, ColorBase03, theme.Border)
	assert.Equal(t, ColorBase02, theme.Selection)
	assert.Equal(t, ColorFocus, theme.Focus)
	assert.Equal(t, ColorSuccess, theme.Success)
	assert.Equal(t, ColorWarning, theme.Warning)
	assert.Equal(t, ColorError, theme.Error)
	assert.Equal(t, ColorInfo, theme.Info)
	assert.Equal(t, ColorMuted, theme.Muted)
	assert.Equal(t, ColorHighlight, theme.Highlight)
	assert.Equal(t, ColorGreen, theme.UserMessage)
	assert.Equal(t, ColorBlue, theme.AssistantMessage)
	assert.Equal(t, ColorPurple, theme.SystemMessage)
	assert.Equal(t, ColorOrange, theme.StreamingCursor)
}

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	assert.NotNil(t, theme)
	assert.Equal(t, ColorBase00, theme.Background)
	assert.Equal(t, ColorBase05, theme.Foreground)
	assert.Equal(t, ColorBase03, theme.Border)
	assert.Equal(t, ColorBase02, theme.Selection)
	assert.Equal(t, ColorFocus, theme.Focus)
	assert.Equal(t, ColorSuccess, theme.Success)
	assert.Equal(t, ColorWarning, theme.Warning)
	assert.Equal(t, ColorError, theme.Error)
	assert.Equal(t, ColorInfo, theme.Info)
	assert.Equal(t, ColorMuted, theme.Muted)
	assert.Equal(t, ColorHighlight, theme.Highlight)
	assert.Equal(t, ColorGreen, theme.UserMessage)
	assert.Equal(t, ColorBlue, theme.AssistantMessage)
	assert.Equal(t, ColorPurple, theme.SystemMessage)
	assert.Equal(t, ColorOrange, theme.StreamingCursor)
}

func TestApplyTheme(t *testing.T) {
	theme := DefaultTheme()

	// Store original values to restore later
	originalPrimitiveBackground := tview.Styles.PrimitiveBackgroundColor
	originalBorderColor := tview.Styles.BorderColor
	originalTitleColor := tview.Styles.TitleColor

	defer func() {
		// Restore original values
		tview.Styles.PrimitiveBackgroundColor = originalPrimitiveBackground
		tview.Styles.BorderColor = originalBorderColor
		tview.Styles.TitleColor = originalTitleColor
	}()

	// Apply the theme
	ApplyTheme(theme)

	// Verify that tview styles were updated using tcell.GetColor
	assert.Equal(t, tcell.GetColor(theme.Background), tview.Styles.PrimitiveBackgroundColor)
	assert.Equal(t, tcell.GetColor(theme.Background), tview.Styles.ContrastBackgroundColor)
	assert.Equal(t, tcell.GetColor(theme.Background), tview.Styles.MoreContrastBackgroundColor)
	assert.Equal(t, tcell.GetColor(theme.Border), tview.Styles.BorderColor)
	assert.Equal(t, tcell.GetColor(theme.Foreground), tview.Styles.TitleColor)
	assert.Equal(t, tcell.GetColor(theme.Border), tview.Styles.GraphicsColor)
}

func TestHexColors(t *testing.T) {
	// Test that hex colors can be converted to tcell.Color correctly
	// We test by converting and ensuring they're different from defaults

	colorStrings := []string{
		ColorBase00, ColorBase01, ColorBase02, ColorBase03,
		ColorBase04, ColorBase05, ColorBase06, ColorBase07,
		ColorRed, ColorOrange, ColorYellow, ColorGreen,
		ColorCyan, ColorBlue, ColorPurple, ColorBrown,
	}

	// All colors should convert to valid tcell.Color and be different from default
	for _, colorStr := range colorStrings {
		color := tcell.GetColor(colorStr)
		assert.NotEqual(t, tcell.ColorDefault, color)
	}

	// Test a few specific colors to ensure they're unique
	assert.NotEqual(t, ColorBase00, ColorBase07) // Dark vs light
	assert.NotEqual(t, ColorRed, ColorGreen)     // Different accent colors
	assert.NotEqual(t, ColorBlue, ColorYellow)   // Different accent colors
}

func TestThemeConsistency(t *testing.T) {
	theme := DefaultTheme()

	// Test that message colors are consistent with accent colors
	assert.Equal(t, ColorGreen, theme.UserMessage)
	assert.Equal(t, ColorBlue, theme.AssistantMessage)
	assert.Equal(t, ColorPurple, theme.SystemMessage)
	assert.Equal(t, ColorOrange, theme.StreamingCursor)

	// Test that UI colors are consistent
	assert.Equal(t, ColorSuccess, theme.Success)
	assert.Equal(t, ColorWarning, theme.Warning)
	assert.Equal(t, ColorError, theme.Error)
}
