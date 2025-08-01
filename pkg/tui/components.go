package tui

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
)

type MessageDisplay struct {
	Messages []chat.Message
	Width    int
	Height   int
	Scroll   int
}

func NewMessageDisplay(width, height int) MessageDisplay {
	return MessageDisplay{
		Messages: []chat.Message{},
		Width:    width,
		Height:   height,
		Scroll:   0,
	}
}

func (md MessageDisplay) WithMessages(messages []chat.Message) MessageDisplay {
	return MessageDisplay{
		Messages: messages,
		Width:    md.Width,
		Height:   md.Height,
		Scroll:   md.Scroll,
	}
}

func (md MessageDisplay) WithSize(width, height int) MessageDisplay {
	return MessageDisplay{
		Messages: md.Messages,
		Width:    width,
		Height:   height,
		Scroll:   md.Scroll,
	}
}

func (md MessageDisplay) WithScroll(scroll int) MessageDisplay {
	return MessageDisplay{
		Messages: md.Messages,
		Width:    md.Width,
		Height:   md.Height,
		Scroll:   scroll,
	}
}

type InputField struct {
	Content string
	Cursor  int
	Width   int
}

func NewInputField(width int) InputField {
	return InputField{
		Content: "",
		Cursor:  0,
		Width:   width,
	}
}

func (inf InputField) WithContent(content string) InputField {
	cursor := inf.Cursor
	if cursor > len(content) {
		cursor = len(content)
	}
	return InputField{
		Content: content,
		Cursor:  cursor,
		Width:   inf.Width,
	}
}

func (inf InputField) WithCursor(cursor int) InputField {
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(inf.Content) {
		cursor = len(inf.Content)
	}
	return InputField{
		Content: inf.Content,
		Cursor:  cursor,
		Width:   inf.Width,
	}
}

func (inf InputField) WithWidth(width int) InputField {
	return InputField{
		Content: inf.Content,
		Cursor:  inf.Cursor,
		Width:   width,
	}
}

func (inf InputField) InsertRune(r rune) InputField {
	content := inf.Content
	left := content[:inf.Cursor]
	right := content[inf.Cursor:]
	newContent := left + string(r) + right
	
	return InputField{
		Content: newContent,
		Cursor:  inf.Cursor + 1,
		Width:   inf.Width,
	}
}

func (inf InputField) DeleteBackward() InputField {
	if inf.Cursor == 0 {
		return inf
	}
	
	content := inf.Content
	left := content[:inf.Cursor-1]
	right := content[inf.Cursor:]
	
	return InputField{
		Content: left + right,
		Cursor:  inf.Cursor - 1,
		Width:   inf.Width,
	}
}

func (inf InputField) Clear() InputField {
	return InputField{
		Content: "",
		Cursor:  0,
		Width:   inf.Width,
	}
}

type StatusBar struct {
	Model  string
	Status string
	Width  int
}

func NewStatusBar(width int) StatusBar {
	return StatusBar{
		Model:  "",
		Status: "Ready",
		Width:  width,
	}
}

func (sb StatusBar) WithModel(model string) StatusBar {
	return StatusBar{
		Model:  model,
		Status: sb.Status,
		Width:  sb.Width,
	}
}

func (sb StatusBar) WithStatus(status string) StatusBar {
	return StatusBar{
		Model:  sb.Model,
		Status: status,
		Width:  sb.Width,
	}
}

func (sb StatusBar) WithWidth(width int) StatusBar {
	return StatusBar{
		Model:  sb.Model,
		Status: sb.Status,
		Width:  width,
	}
}

type SpinnerComponent struct {
	IsVisible bool
	Frame     int
	StartTime time.Time
	Text      string
	Style     tcell.Style
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧"}

func NewSpinnerComponent() SpinnerComponent {
	return SpinnerComponent{
		IsVisible: false,
		Frame:     0,
		StartTime: time.Now(),
		Text:      "Sending message...",
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
	return sc.GetCurrentFrame() + " " + sc.Text
}