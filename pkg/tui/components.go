package tui

import (
	"strings"
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
	Model          string
	Status         string
	Width          int
	PromptTokens   int
	ResponseTokens int
	ModelAvailable bool
}

func NewStatusBar(width int) StatusBar {
	return StatusBar{
		Model:          "",
		Status:         "Ready",
		Width:          width,
		PromptTokens:   0,
		ResponseTokens: 0,
		ModelAvailable: true,
	}
}

func (sb StatusBar) WithModel(model string) StatusBar {
	return StatusBar{
		Model:          model,
		Status:         sb.Status,
		Width:          sb.Width,
		PromptTokens:   sb.PromptTokens,
		ResponseTokens: sb.ResponseTokens,
		ModelAvailable: sb.ModelAvailable,
	}
}

func (sb StatusBar) WithStatus(status string) StatusBar {
	return StatusBar{
		Model:          sb.Model,
		Status:         status,
		Width:          sb.Width,
		PromptTokens:   sb.PromptTokens,
		ResponseTokens: sb.ResponseTokens,
		ModelAvailable: sb.ModelAvailable,
	}
}

func (sb StatusBar) WithWidth(width int) StatusBar {
	return StatusBar{
		Model:          sb.Model,
		Status:         sb.Status,
		Width:          width,
		PromptTokens:   sb.PromptTokens,
		ResponseTokens: sb.ResponseTokens,
		ModelAvailable: sb.ModelAvailable,
	}
}

func (sb StatusBar) WithTokens(promptTokens, responseTokens int) StatusBar {
	return StatusBar{
		Model:          sb.Model,
		Status:         sb.Status,
		Width:          sb.Width,
		PromptTokens:   promptTokens,
		ResponseTokens: responseTokens,
		ModelAvailable: sb.ModelAvailable,
	}
}

func (sb StatusBar) WithModelAvailability(available bool) StatusBar {
	return StatusBar{
		Model:          sb.Model,
		Status:         sb.Status,
		Width:          sb.Width,
		PromptTokens:   sb.PromptTokens,
		ResponseTokens: sb.ResponseTokens,
		ModelAvailable: available,
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

type AlertDisplay struct {
	IsSpinnerVisible bool
	SpinnerFrame     int
	SpinnerText      string
	ErrorMessage     string
	Width            int
}

func NewAlertDisplay(width int) AlertDisplay {
	return AlertDisplay{
		IsSpinnerVisible: false,
		SpinnerFrame:     0,
		SpinnerText:      "Sending message...",
		ErrorMessage:     "",
		Width:            width,
	}
}

func (ad AlertDisplay) WithSpinner(visible bool, text string) AlertDisplay {
	return AlertDisplay{
		IsSpinnerVisible: visible,
		SpinnerFrame:     ad.SpinnerFrame,
		SpinnerText:      text,
		ErrorMessage:     "", // Clear error when showing spinner
		Width:            ad.Width,
	}
}

func (ad AlertDisplay) WithError(errorMessage string) AlertDisplay {
	return AlertDisplay{
		IsSpinnerVisible: false, // Hide spinner when showing error
		SpinnerFrame:     ad.SpinnerFrame,
		SpinnerText:      ad.SpinnerText,
		ErrorMessage:     errorMessage,
		Width:            ad.Width,
	}
}

func (ad AlertDisplay) Clear() AlertDisplay {
	return AlertDisplay{
		IsSpinnerVisible: false,
		SpinnerFrame:     ad.SpinnerFrame,
		SpinnerText:      ad.SpinnerText,
		ErrorMessage:     "",
		Width:            ad.Width,
	}
}

func (ad AlertDisplay) WithWidth(width int) AlertDisplay {
	return AlertDisplay{
		IsSpinnerVisible: ad.IsSpinnerVisible,
		SpinnerFrame:     ad.SpinnerFrame,
		SpinnerText:      ad.SpinnerText,
		ErrorMessage:     ad.ErrorMessage,
		Width:            width,
	}
}

func (ad AlertDisplay) NextSpinnerFrame() AlertDisplay {
	if !ad.IsSpinnerVisible {
		return ad
	}

	return AlertDisplay{
		IsSpinnerVisible: ad.IsSpinnerVisible,
		SpinnerFrame:     (ad.SpinnerFrame + 1) % len(spinnerFrames),
		SpinnerText:      ad.SpinnerText,
		ErrorMessage:     ad.ErrorMessage,
		Width:            ad.Width,
	}
}

func (ad AlertDisplay) GetSpinnerFrame() string {
	if !ad.IsSpinnerVisible {
		return ""
	}
	return spinnerFrames[ad.SpinnerFrame]
}

func (ad AlertDisplay) GetDisplayText() string {
	if ad.ErrorMessage != "" {
		return ad.ErrorMessage
	}
	if ad.IsSpinnerVisible {
		return ad.GetSpinnerFrame() + " " + ad.SpinnerText
	}
	return ""
}

type ModalDialog struct {
	Visible bool
	Title   string
	Message string
	Width   int
	Height  int
}

func NewModalDialog() ModalDialog {
	return ModalDialog{
		Visible: false,
		Title:   "",
		Message: "",
		Width:   50,
		Height:  8,
	}
}

func (md ModalDialog) WithError(title, message string) ModalDialog {
	return ModalDialog{
		Visible: true,
		Title:   title,
		Message: message,
		Width:   md.Width,
		Height:  md.Height,
	}
}

func (md ModalDialog) Hide() ModalDialog {
	return ModalDialog{
		Visible: false,
		Title:   md.Title,
		Message: md.Message,
		Width:   md.Width,
		Height:  md.Height,
	}
}

func (md ModalDialog) WithSize(width, height int) ModalDialog {
	return ModalDialog{
		Visible: md.Visible,
		Title:   md.Title,
		Message: md.Message,
		Width:   width,
		Height:  height,
	}
}

func (md ModalDialog) Render(screen tcell.Screen, area Rect) {
	if !md.Visible {
		return
	}

	// Calculate modal position (centered)
	modalWidth := md.Width
	modalHeight := md.Height
	if modalWidth > area.Width-4 {
		modalWidth = area.Width - 4
	}
	if modalHeight > area.Height-4 {
		modalHeight = area.Height - 4
	}

	modalX := (area.Width - modalWidth) / 2
	modalY := (area.Height - modalHeight) / 2

	modalArea := Rect{
		X:      modalX,
		Y:      modalY,
		Width:  modalWidth,
		Height: modalHeight,
	}

	// Draw modal background and border
	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorRed)
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)
	messageStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	instructionStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)

	// Render title
	if md.Title != "" {
		titleX := modalArea.X + (modalArea.Width-len(md.Title))/2
		if titleX < modalArea.X+1 {
			titleX = modalArea.X + 1
		}
		renderTextWithLimit(screen, titleX, modalArea.Y+1, modalArea.Width-2, md.Title, titleStyle)
	}

	// Render message (wrap text if needed)
	if md.Message != "" {
		lines := WrapText(md.Message, modalArea.Width-4)
		startY := modalArea.Y + 3
		for i, line := range lines {
			if startY+i >= modalArea.Y+modalArea.Height-3 {
				break
			}
			renderTextWithLimit(screen, modalArea.X+2, startY+i, modalArea.Width-4, line, messageStyle)
		}
	}

	// Render instruction
	instruction := "Press any key to continue"
	instrX := modalArea.X + (modalArea.Width-len(instruction))/2
	if instrX < modalArea.X+1 {
		instrX = modalArea.X + 1
	}
	renderTextWithLimit(screen, instrX, modalArea.Y+modalArea.Height-2, modalArea.Width-2, instruction, instructionStyle)
}

type TextInputModal struct {
	Visible bool
	Title   string
	Prompt  string
	Input   InputField
	Width   int
	Height  int
}

func NewTextInputModal() TextInputModal {
	return TextInputModal{
		Visible: false,
		Title:   "",
		Prompt:  "",
		Input:   NewInputField(40),
		Width:   50,
		Height:  8,
	}
}

func (tim TextInputModal) Show(title, prompt string) TextInputModal {
	return TextInputModal{
		Visible: true,
		Title:   title,
		Prompt:  prompt,
		Input:   NewInputField(40),
		Width:   tim.Width,
		Height:  tim.Height,
	}
}

func (tim TextInputModal) Hide() TextInputModal {
	return TextInputModal{
		Visible: false,
		Title:   tim.Title,
		Prompt:  tim.Prompt,
		Input:   tim.Input.Clear(),
		Width:   tim.Width,
		Height:  tim.Height,
	}
}

func (tim TextInputModal) WithInput(input InputField) TextInputModal {
	return TextInputModal{
		Visible: tim.Visible,
		Title:   tim.Title,
		Prompt:  tim.Prompt,
		Input:   input,
		Width:   tim.Width,
		Height:  tim.Height,
	}
}

func (tim TextInputModal) HandleKeyEvent(ev *tcell.EventKey) (TextInputModal, string, bool) {
	if !tim.Visible {
		return tim, "", false
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		return tim.Hide(), "", false
	case tcell.KeyEnter:
		content := strings.TrimSpace(tim.Input.Content)
		return tim.Hide(), content, true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return tim.WithInput(tim.Input.DeleteBackward()), "", false
	case tcell.KeyLeft:
		return tim.WithInput(tim.Input.WithCursor(tim.Input.Cursor - 1)), "", false
	case tcell.KeyRight:
		return tim.WithInput(tim.Input.WithCursor(tim.Input.Cursor + 1)), "", false
	case tcell.KeyHome:
		return tim.WithInput(tim.Input.WithCursor(0)), "", false
	case tcell.KeyEnd:
		return tim.WithInput(tim.Input.WithCursor(len(tim.Input.Content))), "", false
	default:
		if ev.Rune() != 0 {
			return tim.WithInput(tim.Input.InsertRune(ev.Rune())), "", false
		}
	}
	return tim, "", false
}

func (tim TextInputModal) Render(screen tcell.Screen, area Rect) {
	if !tim.Visible {
		return
	}

	// Calculate modal position (centered)
	modalWidth := tim.Width
	modalHeight := tim.Height
	if modalWidth > area.Width-4 {
		modalWidth = area.Width - 4
	}
	if modalHeight > area.Height-4 {
		modalHeight = area.Height - 4
	}

	modalX := (area.Width - modalWidth) / 2
	modalY := (area.Height - modalHeight) / 2

	modalArea := Rect{
		X:      modalX,
		Y:      modalY,
		Width:  modalWidth,
		Height: modalHeight,
	}

	// Draw modal background and border
	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
	promptStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	instructionStyle := tcell.StyleDefault.Foreground(tcell.ColorGray)

	// Render title
	if tim.Title != "" {
		titleX := modalArea.X + (modalArea.Width-len(tim.Title))/2
		if titleX < modalArea.X+1 {
			titleX = modalArea.X + 1
		}
		renderTextWithLimit(screen, titleX, modalArea.Y+1, modalArea.Width-2, tim.Title, titleStyle)
	}

	// Render prompt
	if tim.Prompt != "" {
		renderTextWithLimit(screen, modalArea.X+2, modalArea.Y+3, modalArea.Width-4, tim.Prompt, promptStyle)
	}

	// Render input field
	inputArea := Rect{
		X:      modalArea.X + 2,
		Y:      modalArea.Y + 4,
		Width:  modalArea.Width - 4,
		Height: 1,
	}
	
	// Clear input area and render input text
	for x := inputArea.X; x < inputArea.X+inputArea.Width; x++ {
		screen.SetContent(x, inputArea.Y, ' ', nil, tcell.StyleDefault)
	}
	
	// Render input content
	visibleContent := tim.Input.Content
	cursorPos := tim.Input.Cursor
	
	if len(visibleContent) > inputArea.Width {
		start := 0
		if cursorPos >= inputArea.Width {
			start = cursorPos - inputArea.Width + 1
		}
		end := start + inputArea.Width
		if end > len(visibleContent) {
			end = len(visibleContent)
		}
		visibleContent = visibleContent[start:end]
		cursorPos = cursorPos - start
	}
	
	renderTextWithLimit(screen, inputArea.X, inputArea.Y, inputArea.Width, visibleContent, tcell.StyleDefault)
	
	// Render cursor
	if cursorPos >= 0 && cursorPos <= len(visibleContent) && cursorPos < inputArea.Width {
		cursorStyle := tcell.StyleDefault.Reverse(true)
		if cursorPos < len(visibleContent) {
			r := rune(visibleContent[cursorPos])
			screen.SetContent(inputArea.X+cursorPos, inputArea.Y, r, nil, cursorStyle)
		} else {
			screen.SetContent(inputArea.X+cursorPos, inputArea.Y, ' ', nil, cursorStyle)
		}
	}

	// Render instructions
	instruction := "Enter to confirm, Esc to cancel"
	instrX := modalArea.X + (modalArea.Width-len(instruction))/2
	if instrX < modalArea.X+1 {
		instrX = modalArea.X + 1
	}
	renderTextWithLimit(screen, instrX, modalArea.Y+modalArea.Height-2, modalArea.Width-2, instruction, instructionStyle)
}
