package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
)

type MessageDisplay struct {
	Messages    []chat.Message        // Legacy field for backward compatibility
	NodeManager *MessageNodeManager   // New node-based message management
	Width       int
	Height      int
	Scroll      int
	UseNodes    bool                  // Flag to enable node-based rendering
}

func NewMessageDisplay(width, height int) MessageDisplay {
	return MessageDisplay{
		Messages:    []chat.Message{},
		NodeManager: NewMessageNodeManager(),
		Width:       width,
		Height:      height,
		Scroll:      0,
		UseNodes:    false, // Default to legacy mode for compatibility
	}
}

// NewNodeBasedMessageDisplay creates a new MessageDisplay that uses the node system
func NewNodeBasedMessageDisplay(width, height int) MessageDisplay {
	return MessageDisplay{
		Messages:    []chat.Message{},
		NodeManager: NewMessageNodeManager(),
		Width:       width,
		Height:      height,
		Scroll:      0,
		UseNodes:    true,
	}
}

func (md MessageDisplay) WithMessages(messages []chat.Message) MessageDisplay {
	updated := MessageDisplay{
		Messages:    messages,
		NodeManager: md.NodeManager,
		Width:       md.Width,
		Height:      md.Height,
		Scroll:      md.Scroll,
		UseNodes:    md.UseNodes,
	}
	
	// If using nodes, update the node manager
	if updated.UseNodes && updated.NodeManager != nil {
		updated.NodeManager.SetMessages(messages)
	}
	
	return updated
}

func (md MessageDisplay) WithSize(width, height int) MessageDisplay {
	return MessageDisplay{
		Messages:    md.Messages,
		NodeManager: md.NodeManager,
		Width:       width,
		Height:      height,
		Scroll:      md.Scroll,
		UseNodes:    md.UseNodes,
	}
}

func (md MessageDisplay) WithScroll(scroll int) MessageDisplay {
	return MessageDisplay{
		Messages:    md.Messages,
		NodeManager: md.NodeManager,
		Width:       md.Width,
		Height:      md.Height,
		Scroll:      scroll,
		UseNodes:    md.UseNodes,
	}
}

// EnableNodes enables node-based rendering
func (md MessageDisplay) EnableNodes() MessageDisplay {
	updated := md
	updated.UseNodes = true
	
	// Sync current messages to node manager
	if updated.NodeManager != nil && len(updated.Messages) > 0 {
		updated.NodeManager.SetMessages(updated.Messages)
	}
	
	return updated
}

// DisableNodes disables node-based rendering (fallback to legacy)
func (md MessageDisplay) DisableNodes() MessageDisplay {
	updated := md
	updated.UseNodes = false
	return updated
}

// HandleClick handles mouse clicks for node-based displays
func (md MessageDisplay) HandleClick(x, y int) (string, bool) {
	if !md.UseNodes || md.NodeManager == nil {
		return "", false
	}
	return md.NodeManager.HandleClick(x, y)
}

// HandleKeyEvent handles keyboard events for node-based displays
func (md MessageDisplay) HandleKeyEvent(ev *tcell.EventKey) bool {
	if !md.UseNodes || md.NodeManager == nil {
		return false
	}
	return md.NodeManager.HandleKeyEvent(ev)
}

// MoveFocusUp moves focus to the previous node
func (md MessageDisplay) MoveFocusUp() bool {
	if !md.UseNodes || md.NodeManager == nil {
		return false
	}
	return md.NodeManager.MoveFocusUp()
}

// MoveFocusDown moves focus to the next node
func (md MessageDisplay) MoveFocusDown() bool {
	if !md.UseNodes || md.NodeManager == nil {
		return false
	}
	return md.NodeManager.MoveFocusDown()
}

// GetSelectedNodes returns the IDs of selected nodes
func (md MessageDisplay) GetSelectedNodes() []string {
	if !md.UseNodes || md.NodeManager == nil {
		return []string{}
	}
	return md.NodeManager.GetSelectedNodes()
}

// ClearSelection clears all node selections
func (md MessageDisplay) ClearSelection() {
	if md.UseNodes && md.NodeManager != nil {
		md.NodeManager.ClearSelection()
	}
}

// Streaming support methods

// WithStreamingMessage creates a display with both regular and streaming messages
func (md MessageDisplay) WithStreamingMessage(messages []chat.Message, streamingMessage *chat.Message) MessageDisplay {
	updated := MessageDisplay{
		Messages:    messages,
		NodeManager: md.NodeManager,
		Width:       md.Width,
		Height:      md.Height,
		Scroll:      md.Scroll,
		UseNodes:    md.UseNodes,
	}
	
	// If using nodes, update with streaming support
	if updated.UseNodes && updated.NodeManager != nil {
		updated.NodeManager.SetMessagesWithStreaming(messages, streamingMessage)
	} else {
		// Legacy mode: append streaming message to regular messages
		if streamingMessage != nil && streamingMessage.Content != "" {
			messagesWithStreaming := make([]chat.Message, len(messages), len(messages)+1)
			copy(messagesWithStreaming, messages)
			messagesWithStreaming = append(messagesWithStreaming, *streamingMessage)
			updated.Messages = messagesWithStreaming
		}
	}
	
	return updated
}

// UpdateStreamingContent updates the content of the streaming message
func (md MessageDisplay) UpdateStreamingContent(content string) MessageDisplay {
	if !md.UseNodes || md.NodeManager == nil {
		// For legacy mode, this would need to be handled at the chat view level
		return md
	}
	
	streamingNodeID := md.NodeManager.GetStreamingNodeID()
	if streamingNodeID != "" {
		md.NodeManager.UpdateStreamingMessage(streamingNodeID, content)
	}
	
	return md
}

// ClearStreamingMessage removes the streaming message node
func (md MessageDisplay) ClearStreamingMessage() MessageDisplay {
	if !md.UseNodes || md.NodeManager == nil {
		return md
	}
	
	streamingNodeID := md.NodeManager.GetStreamingNodeID()
	if streamingNodeID != "" {
		md.NodeManager.RemoveStreamingMessage(streamingNodeID)
	}
	
	return md
}

// HasStreamingMessage checks if there's currently a streaming message
func (md MessageDisplay) HasStreamingMessage() bool {
	if !md.UseNodes || md.NodeManager == nil {
		return false
	}
	
	return md.NodeManager.HasStreamingMessage()
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
	// Model management view specific fields
	IsModelView bool
	TotalModels int
	TotalSize   int64
}

func NewStatusBar(width int) StatusBar {
	return StatusBar{
		Model:          "",
		Status:         "Ready",
		Width:          width,
		PromptTokens:   0,
		ResponseTokens: 0,
		ModelAvailable: true,
		IsModelView:    false,
		TotalModels:    0,
		TotalSize:      0,
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
		IsModelView:    sb.IsModelView,
		TotalModels:    sb.TotalModels,
		TotalSize:      sb.TotalSize,
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
		IsModelView:    sb.IsModelView,
		TotalModels:    sb.TotalModels,
		TotalSize:      sb.TotalSize,
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
		IsModelView:    sb.IsModelView,
		TotalModels:    sb.TotalModels,
		TotalSize:      sb.TotalSize,
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
		IsModelView:    sb.IsModelView,
		TotalModels:    sb.TotalModels,
		TotalSize:      sb.TotalSize,
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
		IsModelView:    sb.IsModelView,
		TotalModels:    sb.TotalModels,
		TotalSize:      sb.TotalSize,
	}
}

func (sb StatusBar) WithModelViewData(totalModels int, totalSize int64) StatusBar {
	return StatusBar{
		Model:          sb.Model,
		Status:         sb.Status,
		Width:          sb.Width,
		PromptTokens:   sb.PromptTokens,
		ResponseTokens: sb.ResponseTokens,
		ModelAvailable: sb.ModelAvailable,
		IsModelView:    true,
		TotalModels:    totalModels,
		TotalSize:      totalSize,
	}
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
		SpinnerText:      "",
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
		SpinnerFrame:     (ad.SpinnerFrame + 1) % GetSpinnerFrameCount(),
		SpinnerText:      ad.SpinnerText,
		ErrorMessage:     ad.ErrorMessage,
		Width:            ad.Width,
	}
}

func (ad AlertDisplay) GetSpinnerFrame() string {
	if !ad.IsSpinnerVisible {
		return ""
	}
	return GetSpinnerFrame(ad.SpinnerFrame)
}

func (ad AlertDisplay) GetDisplayText() string {
	if ad.ErrorMessage != "" {
		return ad.ErrorMessage
	}
	if ad.IsSpinnerVisible {
		// Only return the spinner character, no text
		return ad.GetSpinnerFrame()
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
	borderStyle := StyleBorderError
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := StyleBorderError.Bold(true)
	messageStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)
	instructionStyle := StyleInstruction

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
	borderStyle := StyleBorder
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := StyleHighlight
	promptStyle := StylePrompt
	instructionStyle := StyleDimText

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

type ConfirmationModal struct {
	Visible bool
	Title   string
	Message string
	Width   int
	Height  int
}

func NewConfirmationModal() ConfirmationModal {
	return ConfirmationModal{
		Visible: false,
		Title:   "",
		Message: "",
		Width:   50,
		Height:  8,
	}
}

func (cm ConfirmationModal) Show(title, message string) ConfirmationModal {
	return ConfirmationModal{
		Visible: true,
		Title:   title,
		Message: message,
		Width:   cm.Width,
		Height:  cm.Height,
	}
}

func (cm ConfirmationModal) Hide() ConfirmationModal {
	return ConfirmationModal{
		Visible: false,
		Title:   cm.Title,
		Message: cm.Message,
		Width:   cm.Width,
		Height:  cm.Height,
	}
}

func (cm ConfirmationModal) HandleKeyEvent(ev *tcell.EventKey) (ConfirmationModal, bool, bool) {
	if !cm.Visible {
		return cm, false, false
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		return cm.Hide(), false, false
	case tcell.KeyEnter:
		return cm.Hide(), true, false
	default:
		if ev.Rune() != 0 {
			switch ev.Rune() {
			case 'y', 'Y':
				return cm.Hide(), true, false
			case 'n', 'N':
				return cm.Hide(), false, false
			}
		}
	}
	return cm, false, false
}

func (cm ConfirmationModal) Render(screen tcell.Screen, area Rect) {
	if !cm.Visible {
		return
	}

	// Calculate modal position (centered)
	modalWidth := cm.Width
	modalHeight := cm.Height
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
	borderStyle := StyleBorderError
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := StyleBorderError.Bold(true)
	messageStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)
	instructionStyle := StyleInstruction

	// Render title
	if cm.Title != "" {
		titleX := modalArea.X + (modalArea.Width-len(cm.Title))/2
		if titleX < modalArea.X+1 {
			titleX = modalArea.X + 1
		}
		renderTextWithLimit(screen, titleX, modalArea.Y+1, modalArea.Width-2, cm.Title, titleStyle)
	}

	// Render message (wrap text if needed)
	if cm.Message != "" {
		lines := WrapText(cm.Message, modalArea.Width-4)
		startY := modalArea.Y + 3
		for i, line := range lines {
			if startY+i >= modalArea.Y+modalArea.Height-3 {
				break
			}
			centerX := modalArea.X + (modalArea.Width-len(line))/2
			if centerX < modalArea.X+1 {
				centerX = modalArea.X + 1
			}
			renderTextWithLimit(screen, centerX, startY+i, modalArea.Width-2, line, messageStyle)
		}
	}

	// Render instruction
	instruction := "<enter> to confirm. <esc> to cancel."
	instrX := modalArea.X + (modalArea.Width-len(instruction))/2
	if instrX < modalArea.X+1 {
		instrX = modalArea.X + 1
	}
	renderTextWithLimit(screen, instrX, modalArea.Y+modalArea.Height-2, modalArea.Width-2, instruction, instructionStyle)
}

type DownloadPromptModal struct {
	Visible   bool
	ModelName string
	Width     int
	Height    int
}

func NewDownloadPromptModal() DownloadPromptModal {
	return DownloadPromptModal{
		Visible:   false,
		ModelName: "",
		Width:     60,
		Height:    10,
	}
}

func (dpm DownloadPromptModal) Show(modelName string) DownloadPromptModal {
	return DownloadPromptModal{
		Visible:   true,
		ModelName: modelName,
		Width:     dpm.Width,
		Height:    dpm.Height,
	}
}

func (dpm DownloadPromptModal) Hide() DownloadPromptModal {
	return DownloadPromptModal{
		Visible:   false,
		ModelName: dpm.ModelName,
		Width:     dpm.Width,
		Height:    dpm.Height,
	}
}

func (dpm DownloadPromptModal) HandleKeyEvent(ev *tcell.EventKey) (DownloadPromptModal, bool, bool) {
	if !dpm.Visible {
		return dpm, false, false
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		return dpm.Hide(), false, false
	case tcell.KeyEnter:
		return dpm.Hide(), true, false
	default:
		if ev.Rune() != 0 {
			switch ev.Rune() {
			case 'y', 'Y':
				return dpm.Hide(), true, false
			case 'n', 'N':
				return dpm.Hide(), false, false
			}
		}
	}
	return dpm, false, false
}

func (dpm DownloadPromptModal) Render(screen tcell.Screen, area Rect) {
	if !dpm.Visible {
		return
	}

	// Calculate modal position (centered)
	modalWidth := dpm.Width
	modalHeight := dpm.Height
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
	borderStyle := StyleBorder
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := StyleHighlight
	messageStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)
	modelStyle := tcell.StyleDefault.Foreground(ColorModelName).Bold(true)
	instructionStyle := StyleDimText

	// Render title
	title := "Download Model"
	titleX := modalArea.X + (modalArea.Width-len(title))/2
	if titleX < modalArea.X+1 {
		titleX = modalArea.X + 1
	}
	renderTextWithLimit(screen, titleX, modalArea.Y+1, modalArea.Width-2, title, titleStyle)

	// Render message
	message1 := "The model you selected is not available locally:"
	renderTextWithLimit(screen, modalArea.X+2, modalArea.Y+3, modalArea.Width-4, message1, messageStyle)

	// Render model name (centered and highlighted)
	modelNameX := modalArea.X + (modalArea.Width-len(dpm.ModelName))/2
	if modelNameX < modalArea.X+1 {
		modelNameX = modalArea.X + 1
	}
	renderTextWithLimit(screen, modelNameX, modalArea.Y+5, modalArea.Width-2, dpm.ModelName, modelStyle)

	// Render prompt
	message2 := "Would you like to download it now?"
	message2X := modalArea.X + (modalArea.Width-len(message2))/2
	if message2X < modalArea.X+1 {
		message2X = modalArea.X + 1
	}
	renderTextWithLimit(screen, message2X, modalArea.Y+6, modalArea.Width-2, message2, messageStyle)

	// Render instructions
	instruction := "[Y]es / [N]o / [Esc] to cancel"
	instrX := modalArea.X + (modalArea.Width-len(instruction))/2
	if instrX < modalArea.X+1 {
		instrX = modalArea.X + 1
	}
	renderTextWithLimit(screen, instrX, modalArea.Y+modalArea.Height-2, modalArea.Width-2, instruction, instructionStyle)
}

type ProgressModal struct {
	Visible     bool
	Title       string
	ModelName   string
	Status      string
	Progress    float64
	Spinner     SpinnerComponent
	Cancellable bool
	Width       int
	Height      int
}

func NewProgressModal() ProgressModal {
	return ProgressModal{
		Visible:     false,
		Title:       "",
		ModelName:   "",
		Status:      "",
		Progress:    0.0,
		Spinner:     NewSpinnerComponent(),
		Cancellable: true,
		Width:       60,
		Height:      10,
	}
}

func (pm ProgressModal) Show(title, modelName, status string, cancellable bool) ProgressModal {
	return ProgressModal{
		Visible:     true,
		Title:       title,
		ModelName:   modelName,
		Status:      status,
		Progress:    pm.Progress,
		Spinner:     pm.Spinner.WithVisibility(true),
		Cancellable: cancellable,
		Width:       pm.Width,
		Height:      pm.Height,
	}
}

func (pm ProgressModal) Hide() ProgressModal {
	return ProgressModal{
		Visible:     false,
		Title:       pm.Title,
		ModelName:   pm.ModelName,
		Status:      pm.Status,
		Progress:    pm.Progress,
		Spinner:     pm.Spinner.WithVisibility(false),
		Cancellable: pm.Cancellable,
		Width:       pm.Width,
		Height:      pm.Height,
	}
}

func (pm ProgressModal) WithProgress(progress float64, status string) ProgressModal {
	return ProgressModal{
		Visible:     pm.Visible,
		Title:       pm.Title,
		ModelName:   pm.ModelName,
		Status:      status,
		Progress:    progress,
		Spinner:     pm.Spinner,
		Cancellable: pm.Cancellable,
		Width:       pm.Width,
		Height:      pm.Height,
	}
}

func (pm ProgressModal) NextSpinnerFrame() ProgressModal {
	return ProgressModal{
		Visible:     pm.Visible,
		Title:       pm.Title,
		ModelName:   pm.ModelName,
		Status:      pm.Status,
		Progress:    pm.Progress,
		Spinner:     pm.Spinner.NextFrame(),
		Cancellable: pm.Cancellable,
		Width:       pm.Width,
		Height:      pm.Height,
	}
}

func (pm ProgressModal) HandleKeyEvent(ev *tcell.EventKey) (ProgressModal, bool) {
	if !pm.Visible || !pm.Cancellable {
		return pm, false
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		return pm.Hide(), true
	default:
		if ev.Rune() != 0 {
			switch ev.Rune() {
			case 'c', 'C':
				return pm.Hide(), true
			}
		}
	}
	return pm, false
}

func (pm ProgressModal) Render(screen tcell.Screen, area Rect) {
	if !pm.Visible {
		return
	}

	// Calculate modal position (centered)
	modalWidth := pm.Width
	modalHeight := pm.Height
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
	borderStyle := tcell.StyleDefault.Foreground(ColorProgressBar)
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := tcell.StyleDefault.Foreground(ColorProgressBar).Bold(true)
	modelStyle := tcell.StyleDefault.Foreground(ColorModelName).Bold(true)
	statusStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)
	progressStyle := tcell.StyleDefault.Background(ColorProgressBar).Foreground(ColorMenuNormal)
	instructionStyle := StyleDimText

	// Render title with spinner
	spinnerChar := pm.Spinner.GetCurrentFrame()
	titleWithSpinner := spinnerChar + " " + pm.Title
	titleX := modalArea.X + (modalArea.Width-len(titleWithSpinner))/2
	if titleX < modalArea.X+1 {
		titleX = modalArea.X + 1
	}
	renderTextWithLimit(screen, titleX, modalArea.Y+1, modalArea.Width-2, titleWithSpinner, titleStyle)

	// Render model name
	if pm.ModelName != "" {
		modelNameX := modalArea.X + (modalArea.Width-len(pm.ModelName))/2
		if modelNameX < modalArea.X+1 {
			modelNameX = modalArea.X + 1
		}
		renderTextWithLimit(screen, modelNameX, modalArea.Y+3, modalArea.Width-2, pm.ModelName, modelStyle)
	}

	// Render progress bar
	progressBarWidth := modalArea.Width - 6
	if progressBarWidth > 0 {
		progressFilled := int(pm.Progress * float64(progressBarWidth))
		if progressFilled > progressBarWidth {
			progressFilled = progressBarWidth
		}

		progressY := modalArea.Y + 5
		progressX := modalArea.X + 3

		// Draw progress bar background
		for i := 0; i < progressBarWidth; i++ {
			screen.SetContent(progressX+i, progressY, '░', nil, tcell.StyleDefault.Foreground(ColorProgressBarBg))
		}

		// Draw progress bar fill
		for i := 0; i < progressFilled; i++ {
			screen.SetContent(progressX+i, progressY, '█', nil, progressStyle)
		}

		// Draw percentage
		percentage := fmt.Sprintf("%.1f%%", pm.Progress*100)
		percentX := modalArea.X + (modalArea.Width-len(percentage))/2
		if percentX < modalArea.X+1 {
			percentX = modalArea.X + 1
		}
		renderTextWithLimit(screen, percentX, modalArea.Y+6, modalArea.Width-2, percentage, statusStyle)
	}

	// Render status
	if pm.Status != "" {
		statusX := modalArea.X + (modalArea.Width-len(pm.Status))/2
		if statusX < modalArea.X+1 {
			statusX = modalArea.X + 1
		}
		renderTextWithLimit(screen, statusX, modalArea.Y+7, modalArea.Width-2, pm.Status, statusStyle)
	}

	// Render cancellation instruction
	if pm.Cancellable {
		instruction := "[Esc] or [C] to cancel"
		instrX := modalArea.X + (modalArea.Width-len(instruction))/2
		if instrX < modalArea.X+1 {
			instrX = modalArea.X + 1
		}
		renderTextWithLimit(screen, instrX, modalArea.Y+modalArea.Height-2, modalArea.Width-2, instruction, instructionStyle)
	}
}
