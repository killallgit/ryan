package tui

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

var spinnerFrames = []string{"░", "▒", "▓", "█"}

type SpinnerComponent struct {
	IsVisible bool
	Frame     int
	StartTime time.Time
	Text      string
	Style     tcell.Style
}

func NewSpinnerComponent() SpinnerComponent {
	return SpinnerComponent{
		IsVisible: false,
		Frame:     0,
		StartTime: time.Now(),
		Text:      "",
		Style:     tcell.StyleDefault.Foreground(tcell.ColorGray),
	}
}

func (sc SpinnerComponent) WithVisibility(visible bool) SpinnerComponent {
	spinner := SpinnerComponent{
		IsVisible: visible,
		Frame:     sc.Frame,
		StartTime: sc.StartTime,
		Text:      sc.Text,
		Style:     sc.Style,
	}

	// Reset animation when becoming visible
	if visible && !sc.IsVisible {
		spinner.StartTime = time.Now()
		spinner.Frame = 0
	}

	return spinner
}

func (sc SpinnerComponent) WithText(text string) SpinnerComponent {
	return SpinnerComponent{
		IsVisible: sc.IsVisible,
		Frame:     sc.Frame,
		StartTime: sc.StartTime,
		Text:      text,
		Style:     sc.Style,
	}
}

func (sc SpinnerComponent) NextFrame() SpinnerComponent {
	if !sc.IsVisible {
		return sc
	}

	return SpinnerComponent{
		IsVisible: sc.IsVisible,
		Frame:     (sc.Frame + 1) % len(spinnerFrames),
		StartTime: sc.StartTime,
		Text:      sc.Text,
		Style:     sc.Style,
	}
}

func (sc SpinnerComponent) GetCurrentFrame() string {
	if !sc.IsVisible {
		return ""
	}
	return spinnerFrames[sc.Frame]
}

func (sc SpinnerComponent) GetDisplayText() string {
	if !sc.IsVisible {
		return ""
	}
	// Only return the spinner character, no text
	return sc.GetCurrentFrame()
}

// GetSpinnerFrameCount returns the total number of spinner frames
func GetSpinnerFrameCount() int {
	return len(spinnerFrames)
}

// GetSpinnerFrame returns the spinner character at the given frame index
func GetSpinnerFrame(frame int) string {
	if frame < 0 || frame >= len(spinnerFrames) {
		return ""
	}
	return spinnerFrames[frame]
}
