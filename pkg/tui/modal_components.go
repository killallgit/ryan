package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

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

	// Clear modal background and draw border
	clearArea(screen, modalArea)
	borderStyle := StyleBorderError
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := StyleBorderError.Bold(true)
	messageStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)

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
	renderTextWithLimit(screen, instrX, modalArea.Y+modalArea.Height-2, modalArea.Width-2, instruction, StyleInstruction)
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

	// Clear modal background and draw border
	clearArea(screen, modalArea)
	borderStyle := StyleBorder
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := StyleHighlight
	promptStyle := StylePrompt

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
	renderTextWithLimit(screen, instrX, modalArea.Y+modalArea.Height-2, modalArea.Width-2, instruction, StyleInstruction)
}

type ConfirmationModal struct {
	Visible        bool
	Title          string
	Message        string
	Width          int
	Height         int
	SelectedButton int // 0 = Cancel, 1 = Confirm
}

func NewConfirmationModal() ConfirmationModal {
	return ConfirmationModal{
		Visible:        false,
		Title:          "",
		Message:        "",
		Width:          50,
		Height:         8,
		SelectedButton: 1, // Default to Confirm button
	}
}

func (cm ConfirmationModal) Show(title, message string) ConfirmationModal {
	return ConfirmationModal{
		Visible:        true,
		Title:          title,
		Message:        message,
		Width:          cm.Width,
		Height:         cm.Height,
		SelectedButton: 1, // Default to Confirm button
	}
}

func (cm ConfirmationModal) Hide() ConfirmationModal {
	return ConfirmationModal{
		Visible:        false,
		Title:          cm.Title,
		Message:        cm.Message,
		Width:          cm.Width,
		Height:         cm.Height,
		SelectedButton: cm.SelectedButton,
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
		// Enter confirms the selected button
		confirmed := cm.SelectedButton == 1 // 1 = Confirm button
		return cm.Hide(), confirmed, false
	case tcell.KeyTab:
		// Tab cycles between buttons
		newButton := (cm.SelectedButton + 1) % 2
		return ConfirmationModal{
			Visible:        cm.Visible,
			Title:          cm.Title,
			Message:        cm.Message,
			Width:          cm.Width,
			Height:         cm.Height,
			SelectedButton: newButton,
		}, false, false
	case tcell.KeyLeft, tcell.KeyRight:
		// Arrow keys also switch between buttons
		newButton := (cm.SelectedButton + 1) % 2
		return ConfirmationModal{
			Visible:        cm.Visible,
			Title:          cm.Title,
			Message:        cm.Message,
			Width:          cm.Width,
			Height:         cm.Height,
			SelectedButton: newButton,
		}, false, false
	default:
		if ev.Rune() != 0 {
			switch ev.Rune() {
			case 'y', 'Y':
				// Keep Y/N functionality but don't document it
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

	// Clear modal background and draw border
	clearArea(screen, modalArea)
	borderStyle := StyleBorderError
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := StyleBorderError.Bold(true)
	messageStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)

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

	// Render buttons inside the modal border (similar to DownloadPromptModal)
	contentArea := Rect{
		X:      modalArea.X + 1,
		Y:      modalArea.Y + 1,
		Width:  modalArea.Width - 2,
		Height: modalArea.Height - 2,
	}

	buttonY := contentArea.Y + contentArea.Height - 2 // Inside border, from bottom
	buttonSpacing := 2
	availableWidth := contentArea.Width - buttonSpacing
	buttonWidth := availableWidth / 2 // Each button gets half the available width

	// Cancel button (left)
	cancelX := contentArea.X
	cancelStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)
	if cm.SelectedButton == 0 {
		cancelStyle = cancelStyle.Bold(true)
	}

	// Fill button background and render text
	for x := cancelX; x < cancelX+buttonWidth; x++ {
		screen.SetContent(x, buttonY, ' ', nil, tcell.StyleDefault.Background(tcell.ColorDefault))
	}

	// Cancel text (centered in button)
	cancelText := "Cancel"
	cancelTextX := cancelX + (buttonWidth-len(cancelText))/2
	renderTextWithLimit(screen, cancelTextX, buttonY, buttonWidth, cancelText, cancelStyle)

	// Confirm button (right)
	confirmX := contentArea.X + buttonWidth + buttonSpacing
	confirmStyle := StyleBorderError // Use error style for confirm button to match delete action
	if cm.SelectedButton == 1 {
		confirmStyle = confirmStyle.Bold(true)
	}

	// Fill button background and render text
	for x := confirmX; x < confirmX+buttonWidth; x++ {
		screen.SetContent(x, buttonY, ' ', nil, tcell.StyleDefault.Background(tcell.ColorDefault))
	}

	// Confirm text (centered in button)
	confirmText := "Confirm"
	confirmTextX := confirmX + (buttonWidth-len(confirmText))/2
	renderTextWithLimit(screen, confirmTextX, buttonY, buttonWidth, confirmText, confirmStyle)
}

type DownloadPromptModal struct {
	Visible        bool
	ModelName      string
	Width          int
	Height         int
	SelectedButton int // 0 = Cancel, 1 = Download
}

func NewDownloadPromptModal() DownloadPromptModal {
	return DownloadPromptModal{
		Visible:        false,
		ModelName:      "",
		Width:          60,
		Height:         12, // Increased height for buttons
		SelectedButton: 1,  // Default to Download button
	}
}

func (dpm DownloadPromptModal) Show(modelName string) DownloadPromptModal {
	return DownloadPromptModal{
		Visible:        true,
		ModelName:      modelName,
		Width:          dpm.Width,
		Height:         dpm.Height,
		SelectedButton: 1, // Default to Download button
	}
}

func (dpm DownloadPromptModal) Hide() DownloadPromptModal {
	return DownloadPromptModal{
		Visible:        false,
		ModelName:      dpm.ModelName,
		Width:          dpm.Width,
		Height:         dpm.Height,
		SelectedButton: dpm.SelectedButton,
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
		// Enter confirms the selected button
		confirmed := dpm.SelectedButton == 1 // 1 = Download button
		return dpm.Hide(), confirmed, false
	case tcell.KeyTab:
		// Tab cycles between buttons
		newButton := (dpm.SelectedButton + 1) % 2
		return DownloadPromptModal{
			Visible:        dpm.Visible,
			ModelName:      dpm.ModelName,
			Width:          dpm.Width,
			Height:         dpm.Height,
			SelectedButton: newButton,
		}, false, false
	case tcell.KeyLeft, tcell.KeyRight:
		// Arrow keys also switch between buttons
		newButton := (dpm.SelectedButton + 1) % 2
		return DownloadPromptModal{
			Visible:        dpm.Visible,
			ModelName:      dpm.ModelName,
			Width:          dpm.Width,
			Height:         dpm.Height,
			SelectedButton: newButton,
		}, false, false
	default:
		if ev.Rune() != 0 {
			switch ev.Rune() {
			case 'y', 'Y':
				// Keep Y/N functionality but don't document it
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

	// Clear modal background and draw border
	clearArea(screen, modalArea)
	borderStyle := StyleBorder
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := StyleHighlight
	messageStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)
	modelStyle := tcell.StyleDefault.Foreground(ColorModelName).Bold(true)

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

	// Render buttons inside the modal border
	contentArea := Rect{
		X:      modalArea.X + 1,
		Y:      modalArea.Y + 1,
		Width:  modalArea.Width - 2,
		Height: modalArea.Height - 2,
	}

	buttonY := contentArea.Y + contentArea.Height - 2 // Inside border, from bottom
	buttonSpacing := 2
	availableWidth := contentArea.Width - buttonSpacing
	buttonWidth := availableWidth / 2 // Each button gets half the available width

	// Cancel button (left)
	cancelX := contentArea.X
	cancelStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)
	if dpm.SelectedButton == 0 {
		cancelStyle = cancelStyle.Bold(true)
	}

	// Fill button background and render text
	for x := cancelX; x < cancelX+buttonWidth; x++ {
		screen.SetContent(x, buttonY, ' ', nil, tcell.StyleDefault.Background(tcell.ColorDefault))
	}

	// Cancel text (centered in button)
	cancelText := "Cancel"
	cancelTextX := cancelX + (buttonWidth-len(cancelText))/2
	renderTextWithLimit(screen, cancelTextX, buttonY, buttonWidth, cancelText, cancelStyle)

	// Download button (right)
	downloadX := contentArea.X + buttonWidth + buttonSpacing
	downloadStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)
	if dpm.SelectedButton == 1 {
		downloadStyle = downloadStyle.Bold(true)
	}

	// Fill button background and render text
	for x := downloadX; x < downloadX+buttonWidth; x++ {
		screen.SetContent(x, buttonY, ' ', nil, tcell.StyleDefault.Background(tcell.ColorDefault))
	}

	// Download text (centered in button)
	downloadText := "Download"
	downloadTextX := downloadX + (buttonWidth-len(downloadText))/2
	renderTextWithLimit(screen, downloadTextX, buttonY, buttonWidth, downloadText, downloadStyle)
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

	// Clear modal background and draw border
	clearArea(screen, modalArea)
	borderStyle := tcell.StyleDefault.Foreground(ColorProgressBar)
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := tcell.StyleDefault.Foreground(ColorProgressBar).Bold(true)
	modelStyle := tcell.StyleDefault.Foreground(ColorModelName).Bold(true)
	statusStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)
	progressStyle := tcell.StyleDefault.Background(ColorProgressBar).Foreground(ColorMenuNormal)

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
		renderTextWithLimit(screen, instrX, modalArea.Y+modalArea.Height-2, modalArea.Width-2, instruction, StyleInstruction)
	}
}

// HelpModal displays keyboard shortcuts and help information
type HelpModal struct {
	Visible bool
	Width   int
	Height  int
}

func NewHelpModal() HelpModal {
	return HelpModal{
		Visible: false,
		Width:   70,
		Height:  25,
	}
}

func (hm HelpModal) Show() HelpModal {
	return HelpModal{
		Visible: true,
		Width:   hm.Width,
		Height:  hm.Height,
	}
}

func (hm HelpModal) Hide() HelpModal {
	return HelpModal{
		Visible: false,
		Width:   hm.Width,
		Height:  hm.Height,
	}
}

func (hm HelpModal) HandleKeyEvent(ev *tcell.EventKey) (HelpModal, bool) {
	if !hm.Visible {
		return hm, false
	}

	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyF2:
		return hm.Hide(), true
	default:
		if ev.Rune() != 0 {
			switch ev.Rune() {
			case 'q', 'Q':
				return hm.Hide(), true
			}
		}
	}
	return hm, false
}

func (hm HelpModal) Render(screen tcell.Screen, area Rect) {
	if !hm.Visible {
		return
	}

	// Calculate modal position (centered)
	modalWidth := hm.Width
	modalHeight := hm.Height
	if modalWidth > area.Width-4 {
		modalWidth = area.Width - 4
	}
	if modalHeight > area.Height-4 {
		modalHeight = area.Height - 4
	}

	startX := area.X + (area.Width-modalWidth)/2
	startY := area.Y + (area.Height-modalHeight)/2

	modalArea := Rect{
		X:      startX,
		Y:      startY,
		Width:  modalWidth,
		Height: modalHeight,
	}

	// Clear modal background and draw border (same as other modals)
	clearArea(screen, modalArea)
	borderStyle := StyleBorder
	drawBorder(screen, modalArea, borderStyle)

	// Styles
	titleStyle := StyleHighlight
	contentStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)

	// Help content
	helpLines := []string{
		"RYAN - Keyboard Shortcuts & Commands",
		"",
		"GENERAL:",
		"  F2          Show/hide this help",
		"  Esc         Cancel/quit current operation/modal",
		"  Tab         Switch between chat and model views",
		"  Ctrl+C      Quit application",
		"  Ctrl+R      Reset/clear conversation",
		"",
		"CHAT VIEW - INPUT MODE (✏️):",
		"  Enter       Send message",
		"  Ctrl+N      Switch to node selection mode",
		"  ↑/↓         Scroll chat history",
		"  PgUp/PgDn   Page up/down in chat",
		"  Esc         Interrupt streaming response",
		"",
		"CHAT VIEW - NODE SELECTION MODE (🎯):",
		"  j/k         Navigate up/down between nodes (vim-style)",
		"  ↑/↓         Navigate up/down between nodes (arrow keys)",
		"  Tab         Expand/collapse focused node",
		"  Space       Select/deselect focused node",
		"  Enter       Toggle selection of focused node",
		"  a           Select all nodes",
		"  c           Clear all selections",
		"  i/Esc       Return to input mode",
		"  Click       Focus node and switch to node mode",
		"",
		"MODEL MANAGEMENT VIEW:",
		"  ↑/↓         Navigate model list",
		"  j/k         Navigate model list (vim-style)",
		"  Enter       Select/download/switch to model",
		"  d           Show model details",
		"  Ctrl+D      Delete selected model",
		"  r           Refresh model list",
		"  Home/End    Jump to first/last model",
		"  PgUp/PgDn   Page up/down in model list",
		"",
		"MODALS:",
		"  Tab/←/→     Navigate between buttons",
		"  Enter       Confirm selected button",
		"  Esc         Close modal/cancel",
		"  y/n         Quick yes/no in confirmation modals",
		"",
		"Press F2, Q, or Esc to close this help",
	}

	// Render title
	title := helpLines[0]
	titleX := modalArea.X + (modalArea.Width-len(title))/2
	if titleX < modalArea.X+1 {
		titleX = modalArea.X + 1
	}
	renderTextWithLimit(screen, titleX, modalArea.Y+1, modalArea.Width-2, title, titleStyle)

	// Render help lines
	startLineY := modalArea.Y + 3
	for i, line := range helpLines[1:] {
		if startLineY+i >= modalArea.Y+modalArea.Height-2 {
			break // Don't overflow modal
		}

		style := contentStyle
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
			// Indent command lines slightly differently
			style = contentStyle
		} else if strings.HasSuffix(line, ":") {
			// Section headers
			style = titleStyle
		}

		renderTextWithLimit(screen, modalArea.X+2, startLineY+i, modalArea.Width-4, line, style)
	}
}
