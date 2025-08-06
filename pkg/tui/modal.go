package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ModalResult represents the result of a modal dialog
type ModalResult int

const (
	ModalResultNone ModalResult = iota
	ModalResultConfirm
	ModalResultCancel
)

// BaseModal provides the foundational modal structure
type BaseModal struct {
	*tview.Flex
	app      *tview.Application
	pageName string
	onClose  func()
}

// NewBaseModal creates a reusable modal container
func NewBaseModal(app *tview.Application, pageName string, onClose func()) *BaseModal {
	return &BaseModal{
		Flex:     tview.NewFlex().SetDirection(tview.FlexRow),
		app:      app,
		pageName: pageName,
		onClose:  onClose,
	}
}

// Show displays the modal with specified dimensions
func (b *BaseModal) Show(pages *tview.Pages, width, height int) {
	// Create centered container
	container := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(b.Flex, height, 0, true).
			AddItem(nil, 0, 1, false), width, 0, true).
		AddItem(nil, 0, 1, false)

	// Remove existing modal and add new one
	pages.RemovePage(b.pageName)
	pages.AddPage(b.pageName, container, true, true)

	// Set focus after a small delay to ensure modal is rendered
	go func() {
		time.Sleep(50 * time.Millisecond)
		b.app.QueueUpdateDraw(func() {
			b.app.SetFocus(b.Flex)
		})
	}()
}

// Close removes the modal from pages
func (b *BaseModal) Close(pages *tview.Pages) {
	pages.RemovePage(b.pageName)
	if b.onClose != nil {
		b.onClose()
	}
}

// Legacy Modal for backward compatibility - will be refactored
type Modal struct {
	*tview.Flex
	title         *tview.TextView
	message       *tview.TextView
	inputField    *tview.InputField
	progressBar   *tview.TextView
	buttons       *tview.Flex
	result        ModalResult
	onClose       func(result ModalResult, input string)
	app           *tview.Application
	inputValue    string
	hasInput      bool
	hasProgress   bool
	progressValue int
}

// NewModal creates a new modal dialog
func NewModal(app *tview.Application, title, message string, onClose func(result ModalResult)) *Modal {
	return newModalInternal(app, title, message, "", false, false, func(result ModalResult, input string) {
		onClose(result)
	})
}

// NewInputModal creates a new modal dialog with an input field
func NewInputModal(app *tview.Application, title, message, placeholder string, onClose func(result ModalResult, input string)) *Modal {
	return newModalInternal(app, title, message, placeholder, true, false, onClose)
}

// NewProgressModal creates a new modal dialog with a progress bar
func NewProgressModal(app *tview.Application, title, message string, onClose func(result ModalResult, input string)) *Modal {
	return newModalInternal(app, title, message, "", false, true, onClose)
}

// NewInputProgressModal creates a new modal dialog with both input field and progress bar
func NewInputProgressModal(app *tview.Application, title, message, placeholder string, onClose func(result ModalResult, input string)) *Modal {
	return newModalInternal(app, title, message, placeholder, true, true, onClose)
}

// newModalInternal creates a modal dialog with optional input field and/or progress bar
func newModalInternal(app *tview.Application, title, message, placeholder string, hasInput, hasProgress bool, onClose func(result ModalResult, input string)) *Modal {
	m := &Modal{
		Flex:          tview.NewFlex().SetDirection(tview.FlexRow),
		result:        ModalResultNone,
		onClose:       onClose,
		app:           app,
		hasInput:      hasInput,
		hasProgress:   hasProgress,
		progressValue: 0,
	}

	// Create title text view
	m.title = tview.NewTextView().
		SetText(fmt.Sprintf("[yellow::b]%s[white]", title)).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	m.title.SetBackgroundColor(tcell.GetColor(ColorBase01))
	m.title.SetTextColor(tcell.GetColor(ColorYellow))

	// Create message text view
	m.message = tview.NewTextView().
		SetText(message).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	m.message.SetBackgroundColor(tcell.GetColor(ColorBase01))
	m.message.SetTextColor(tcell.GetColor(ColorBase05))

	// Create buttons
	m.buttons = tview.NewFlex().SetDirection(tview.FlexColumn)

	// Confirm button
	confirmBtn := tview.NewButton("Confirm").
		SetSelectedFunc(func() {
			m.result = ModalResultConfirm
			m.close()
		})
	confirmBtn.SetBackgroundColor(tcell.GetColor(ColorGreen))
	confirmBtn.SetLabelColor(tcell.GetColor(ColorBase00))

	// Cancel button with outline styling when selected
	cancelBtn := tview.NewButton("Cancel").
		SetSelectedFunc(func() {
			m.result = ModalResultCancel
			m.close()
		})
	// Set normal appearance - outline style
	cancelBtn.SetBackgroundColor(tcell.GetColor(ColorBase01))
	cancelBtn.SetLabelColor(tcell.GetColor(ColorRed))
	// Set selected appearance - filled style
	cancelBtn.SetActivatedStyle(tcell.StyleDefault.
		Background(tcell.GetColor(ColorRed)).
		Foreground(tcell.GetColor(ColorBase00)))

	// Add buttons as equal-width columns taking full modal width
	m.buttons.
		AddItem(confirmBtn, 0, 1, true).
		AddItem(cancelBtn, 0, 1, true)

	// Add input field if needed
	if hasInput {
		m.inputField = tview.NewInputField().
			SetPlaceholder(placeholder).
			SetFieldBackgroundColor(tcell.GetColor(ColorBase00)).
			SetFieldTextColor(tcell.GetColor(ColorBase05))
		m.inputField.SetBorder(true).
			SetBorderColor(tcell.GetColor(ColorBlue))
	}

	// Add progress bar if needed
	if hasProgress {
		m.progressBar = tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetDynamicColors(true).
			SetText("[yellow]Ready[white]\n\n[green::b][white::b]░░░░░░░░░░░░░░░░░░░░[white] 0%")
		m.progressBar.SetBorder(true).
			SetBorderColor(tcell.GetColor(ColorBlue)).
			SetTitle("Progress")
		m.progressBar.SetBackgroundColor(tcell.GetColor(ColorBase01))
		m.progressBar.SetTextColor(tcell.GetColor(ColorBase05))
	}

	// Build the modal layout dynamically based on components
	m.Flex.AddItem(nil, 1, 0, false).
		AddItem(m.title, 2, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(m.message, 4, 0, false)

	if hasInput {
		m.Flex.AddItem(nil, 1, 0, false).
			AddItem(m.inputField, 3, 0, false)
	}

	// Progress bar is added dynamically when needed

	m.Flex.AddItem(nil, 2, 0, false).
		AddItem(m.buttons, 3, 0, false).
		AddItem(nil, 1, 0, false)

	// Set background without border
	m.Flex.SetBackgroundColor(tcell.GetColor(ColorBase01))

	// Handle keyboard shortcuts
	m.Flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			// Let the focused button handle Enter key press
			// Don't override button behavior
			if hasInput && m.app.GetFocus() == m.inputField {
				// If input field has focus, treat Enter as Confirm
				m.result = ModalResultConfirm
				m.close()
				return nil
			}
			// Otherwise, let the button handle it
			return event
		case tcell.KeyEscape:
			m.result = ModalResultCancel
			m.close()
			return nil
		case tcell.KeyTab, tcell.KeyBacktab:
			if hasInput {
				// Cycle through input field, confirm button, cancel button
				currentFocus := m.app.GetFocus()
				if currentFocus == m.inputField {
					m.app.SetFocus(confirmBtn)
				} else if currentFocus == confirmBtn {
					m.app.SetFocus(cancelBtn)
				} else {
					m.app.SetFocus(m.inputField)
				}
			} else {
				// Switch focus between buttons
				if m.app.GetFocus() == confirmBtn {
					m.app.SetFocus(cancelBtn)
				} else {
					m.app.SetFocus(confirmBtn)
				}
			}
			return nil
		}
		return event
	})

	return m
}

// Show displays the modal dialog
func (m *Modal) Show(pages *tview.Pages) {
	m.ShowWithSize(pages, 60, 15) // Default size
}

// ShowWithSize displays the modal dialog with specific dimensions
func (m *Modal) ShowWithSize(pages *tview.Pages, width, height int) {
	m.ShowWithSizeAndName(pages, width, height, "modal")
}

// ShowWithSizeAndName displays the modal dialog with specific dimensions and page name
func (m *Modal) ShowWithSizeAndName(pages *tview.Pages, width, height int, pageName string) {
	// Create a centered modal container with specified sizing
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(m.Flex, height, 0, true).
			AddItem(nil, 0, 1, false), width, 0, true).
		AddItem(nil, 0, 1, false)

	// Remove any existing modal first
	pages.RemovePage(pageName)
	// Add as a page with a dim background
	pages.AddPage(pageName, modal, true, true)

	// Set initial focus on the UI thread
	go func() {
		// Small delay to ensure the modal is properly displayed
		time.Sleep(50 * time.Millisecond)
		m.app.QueueUpdateDraw(func() {
			if m.hasInput && m.inputField != nil {
				m.app.SetFocus(m.inputField)
			} else if m.buttons != nil && m.buttons.GetItemCount() > 1 {
				m.app.SetFocus(m.buttons.GetItem(1))
			}
		})
	}()
}

// close closes the modal and calls the callback
func (m *Modal) close() {
	// Get input value if there's an input field
	var inputValue string
	if m.hasInput && m.inputField != nil {
		inputValue = m.inputField.GetText()
	}

	// Call the callback with the result and input
	if m.onClose != nil {
		m.onClose(m.result, inputValue)
	}
}

// Close removes the modal from the pages
func (m *Modal) Close(pages *tview.Pages) {
	pages.RemovePage("modal")
}

// SetMessage updates the modal message
func (m *Modal) SetMessage(message string) {
	m.message.SetText(message)
}

// SetTitle updates the modal title
func (m *Modal) SetTitle(title string) {
	if m.title != nil {
		m.title.SetText(fmt.Sprintf("[yellow::b]%s[white]", title))
	}
}

// GetResult returns the modal result
func (m *Modal) GetResult() ModalResult {
	return m.result
}

// SetProgress updates the progress bar (if present)
func (m *Modal) SetProgress(completed, total int64) {
	if m.progressBar != nil && total > 0 {
		progress := int(completed * 100 / total)
		m.progressValue = progress

		// Create a visual progress bar with filled/empty blocks
		barWidth := 20
		filledBlocks := int(float64(barWidth) * float64(progress) / 100.0)
		emptyBlocks := barWidth - filledBlocks

		progressBar := "[green::b]" + strings.Repeat("█", filledBlocks) + "[white::b]" + strings.Repeat("░", emptyBlocks) + "[white]"

		// Get current label from title
		title := m.progressBar.GetTitle()
		if title == "" {
			title = "Progress"
		}

		text := fmt.Sprintf("[yellow]%s[white]\n\n%s %d%%", title, progressBar, progress)
		m.progressBar.SetText(text)
	}
}

// SetProgressLabel updates the progress bar label
func (m *Modal) SetProgressLabel(label string) {
	if m.progressBar != nil {
		m.progressBar.SetTitle(label)

		// Update the text with the new label
		barWidth := 20
		filledBlocks := int(float64(barWidth) * float64(m.progressValue) / 100.0)
		emptyBlocks := barWidth - filledBlocks

		progressBar := "[green::b]" + strings.Repeat("█", filledBlocks) + "[white::b]" + strings.Repeat("░", emptyBlocks) + "[white]"
		text := fmt.Sprintf("[yellow]%s[white]\n\n%s %d%%", label, progressBar, m.progressValue)
		m.progressBar.SetText(text)
	}
}

// HideButtons hides the confirm/cancel buttons (useful during progress)
func (m *Modal) HideButtons() {
	if m.buttons != nil {
		// Find the buttons in the layout and hide them
		for i := 0; i < m.Flex.GetItemCount(); i++ {
			item := m.Flex.GetItem(i)
			if item == m.buttons {
				m.Flex.RemoveItem(item)
				break
			}
		}
	}
}

// ShowButtons shows the confirm/cancel buttons
func (m *Modal) ShowButtons() {
	if m.buttons != nil {
		// Add buttons back to the layout
		m.Flex.AddItem(nil, 1, 0, false).
			AddItem(m.buttons, 1, 0, true).
			AddItem(nil, 1, 0, false)
	}
}

// ShowProgress shows the progress bar in the modal
func (m *Modal) ShowProgress() {
	if m.hasProgress && m.progressBar != nil {
		// Find the position before buttons and insert progress bar
		buttonIndex := -1
		for i := 0; i < m.Flex.GetItemCount(); i++ {
			if m.Flex.GetItem(i) == m.buttons {
				buttonIndex = i
				break
			}
		}

		if buttonIndex > 0 {
			// Remove buttons temporarily
			m.Flex.RemoveItem(m.buttons)
			// Remove the spacer before buttons
			if buttonIndex > 1 {
				spacerItem := m.Flex.GetItem(buttonIndex - 1)
				m.Flex.RemoveItem(spacerItem)
			}

			// Add progress bar with spacing
			m.Flex.AddItem(nil, 1, 0, false).
				AddItem(m.progressBar, 4, 0, false).
				AddItem(nil, 2, 0, false).
				AddItem(m.buttons, 3, 0, false).
				AddItem(nil, 1, 0, false)
		}
	}
}

// HideProgress hides the progress bar from the modal
func (m *Modal) HideProgress() {
	if m.progressBar != nil {
		// Remove progress bar from layout
		for i := 0; i < m.Flex.GetItemCount(); i++ {
			if m.Flex.GetItem(i) == m.progressBar {
				m.Flex.RemoveItem(m.progressBar)
				break
			}
		}
	}
}

// OllamaSetupModal creates a modal specifically for Ollama setup
func OllamaSetupModal(app *tview.Application, issue string, onClose func(result ModalResult)) *Modal {
	var message string
	var title string

	switch issue {
	case "not_running":
		title = "Ollama Not Running"
		message = "Ollama service is not running.\n\nWould you like to start it?"
	case "no_model":
		title = "No Model Available"
		message = "No Ollama model is available.\n\nWould you like to download the default model?"
	case "download_model":
		title = "Download Model"
		message = "Downloading default model...\n\nThis may take a few minutes."
	default:
		title = "Ollama Setup"
		message = issue
	}

	return NewModal(app, title, message, onClose)
}

// OllamaURLInputModal creates a modal for entering Ollama URL
func OllamaURLInputModal(app *tview.Application, currentURL string, onClose func(result ModalResult, url string)) *Modal {
	title := "Ollama Server URL"
	message := fmt.Sprintf("Ollama service is not available at: %s\n\nPlease enter the URL for your Ollama server:", currentURL)
	placeholder := currentURL
	if placeholder == "" {
		placeholder = "http://localhost:11434"
	}

	modal := NewInputModal(app, title, message, placeholder, onClose)
	// Pre-populate with current URL
	if modal.inputField != nil && currentURL != "" {
		modal.inputField.SetText(currentURL)
	}
	return modal
}

// DownloadModal represents a modal for model downloads
type DownloadModal struct {
	*BaseModal
	title       *tview.TextView
	message     *tview.TextView
	inputField  *tview.InputField
	progressBar *tview.TextView
	buttons     *tview.Flex
	confirmBtn  *tview.Button
	cancelBtn   *tview.Button
	onResult    func(result ModalResult, modelName string)
}

// NewDownloadModal creates a modal for model downloads
func NewDownloadModal(app *tview.Application, defaultModel string, onResult func(result ModalResult, modelName string)) *DownloadModal {
	dm := &DownloadModal{
		BaseModal: NewBaseModal(app, "download-modal", nil),
		onResult:  onResult,
	}

	// Create title
	dm.title = tview.NewTextView().
		SetText("[yellow::b]Download Ollama Model[white]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	dm.title.SetBackgroundColor(tcell.GetColor(ColorBase01))

	// Create message
	dm.message = tview.NewTextView().
		SetText("No Ollama model is available.\n\nEnter the model name to download:").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	dm.message.SetBackgroundColor(tcell.GetColor(ColorBase01))

	// Create input field
	dm.inputField = tview.NewInputField().
		SetPlaceholder(defaultModel).
		SetText(defaultModel).
		SetFieldBackgroundColor(tcell.GetColor(ColorBase00))
	dm.inputField.SetBorder(true).SetBorderColor(tcell.GetColor(ColorBlue))

	// Create progress bar (hidden initially)
	dm.progressBar = tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText("[yellow]Ready[white]\n\n[green::b][white::b]░░░░░░░░░░░░░░░░░░░░[white] 0%")
	dm.progressBar.SetBorder(true).
		SetBorderColor(tcell.GetColor(ColorBlue)).
		SetTitle("Progress")

	// Create buttons
	dm.createButtons()

	// Layout the modal
	dm.layoutComponents()

	// Set background
	dm.Flex.SetBackgroundColor(tcell.GetColor(ColorBase01))

	// Setup input handling
	dm.setupInputHandling()

	return dm
}

// Show displays the download modal with proper focus
func (dm *DownloadModal) Show(pages *tview.Pages, width, height int) {
	// Call base show
	dm.BaseModal.Show(pages, width, height)

	// Set initial focus to input field
	go func() {
		time.Sleep(100 * time.Millisecond) // Slightly longer delay
		dm.app.QueueUpdateDraw(func() {
			dm.app.SetFocus(dm.inputField)
		})
	}()
}

// createButtons creates the confirm/cancel buttons
func (dm *DownloadModal) createButtons() {
	dm.buttons = tview.NewFlex().SetDirection(tview.FlexColumn)

	confirmBtn := tview.NewButton("Download").
		SetSelectedFunc(func() {
			if dm.onResult != nil {
				modelName := dm.inputField.GetText()
				dm.onResult(ModalResultConfirm, modelName)
			}
		})
	confirmBtn.SetBackgroundColor(tcell.GetColor(ColorGreen))
	confirmBtn.SetLabelColor(tcell.GetColor(ColorBase00))

	cancelBtn := tview.NewButton("Cancel").
		SetSelectedFunc(func() {
			if dm.onResult != nil {
				dm.onResult(ModalResultCancel, "")
			}
		})
	cancelBtn.SetBackgroundColor(tcell.GetColor(ColorBase01))
	cancelBtn.SetLabelColor(tcell.GetColor(ColorRed))

	dm.buttons.AddItem(confirmBtn, 0, 1, true).
		AddItem(cancelBtn, 0, 1, true)

	// Store references for focus management
	dm.confirmBtn = confirmBtn
	dm.cancelBtn = cancelBtn
}

// layoutComponents arranges the modal components
func (dm *DownloadModal) layoutComponents() {
	dm.Flex.AddItem(nil, 1, 0, false). // top padding
						AddItem(dm.title, 2, 0, false).      // title
						AddItem(nil, 1, 0, false).           // spacing
						AddItem(dm.message, 4, 0, false).    // message
						AddItem(nil, 1, 0, false).           // spacing
						AddItem(dm.inputField, 3, 0, false). // input
						AddItem(nil, 2, 0, false).           // spacing
						AddItem(dm.buttons, 3, 0, false).    // buttons
						AddItem(nil, 1, 0, false)            // bottom padding
}

// setupInputHandling configures keyboard navigation for the modal
func (dm *DownloadModal) setupInputHandling() {
	dm.Flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			// Handle Enter key based on current focus
			currentFocus := dm.app.GetFocus()
			if currentFocus == dm.inputField {
				// Enter in input field = confirm
				if dm.onResult != nil {
					modelName := dm.inputField.GetText()
					dm.onResult(ModalResultConfirm, modelName)
				}
				return nil
			}
			// Let buttons handle their own Enter
			return event
		case tcell.KeyEscape:
			// Escape = cancel
			if dm.onResult != nil {
				dm.onResult(ModalResultCancel, "")
			}
			return nil
		case tcell.KeyTab, tcell.KeyBacktab:
			// Tab navigation between input and buttons
			currentFocus := dm.app.GetFocus()
			if event.Key() == tcell.KeyTab {
				if currentFocus == dm.inputField {
					dm.app.SetFocus(dm.confirmBtn)
				} else if currentFocus == dm.confirmBtn {
					dm.app.SetFocus(dm.cancelBtn)
				} else {
					dm.app.SetFocus(dm.inputField)
				}
			} else { // BackTab
				if currentFocus == dm.inputField {
					dm.app.SetFocus(dm.cancelBtn)
				} else if currentFocus == dm.cancelBtn {
					dm.app.SetFocus(dm.confirmBtn)
				} else {
					dm.app.SetFocus(dm.inputField)
				}
			}
			return nil
		}
		return event
	})
}

// ShowProgress converts the modal to progress mode
func (dm *DownloadModal) ShowProgress() {
	// Clear and rebuild layout for progress mode
	dm.Flex.Clear()
	dm.Flex.AddItem(nil, 1, 0, false). // top padding
						AddItem(dm.title, 2, 0, false).       // title
						AddItem(nil, 1, 0, false).            // spacing
						AddItem(dm.message, 2, 0, false).     // shorter message
						AddItem(nil, 1, 0, false).            // spacing
						AddItem(dm.progressBar, 4, 0, false). // progress bar
						AddItem(nil, 2, 0, false)             // bottom padding
}

// SetProgress updates the progress display
func (dm *DownloadModal) SetProgress(status string, completed, total int64) {
	if total > 0 {
		progress := int(completed * 100 / total)
		barWidth := 20
		filledBlocks := int(float64(barWidth) * float64(progress) / 100.0)
		emptyBlocks := barWidth - filledBlocks

		progressBar := "[green::b]" + strings.Repeat("█", filledBlocks) + "[white::b]" + strings.Repeat("░", emptyBlocks) + "[white]"
		text := fmt.Sprintf("[yellow]%s[white]\n\n%s %d%%", status, progressBar, progress)
		dm.progressBar.SetText(text)

		// Update message with model name and progress
		if dm.inputField != nil {
			modelName := dm.inputField.GetText()
			dm.message.SetText(fmt.Sprintf("Downloading: %s\n%s (%d%%)", modelName, status, progress))
		}
	}
}

// ErrorModal represents a modal for displaying errors
type ErrorModal struct {
	*BaseModal
	title   *tview.TextView
	message *tview.TextView
}

// NewErrorModal creates a modal for displaying errors
func NewErrorModal(app *tview.Application, title, errorMessage string, onClose func()) *ErrorModal {
	em := &ErrorModal{
		BaseModal: NewBaseModal(app, "error-modal", onClose),
	}

	// Create title
	em.title = tview.NewTextView().
		SetText(fmt.Sprintf("[red::b]%s[white]", title)).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	em.title.SetBackgroundColor(tcell.GetColor(ColorBase01))

	// Create message with wrapping and flexible space
	em.message = tview.NewTextView().
		SetText(errorMessage + "\n\n[yellow]Press any key to exit[white]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetWrap(true)
	em.message.SetBackgroundColor(tcell.GetColor(ColorBase01))

	// Layout components
	em.Flex.AddItem(nil, 1, 0, false). // top padding
						AddItem(em.title, 2, 0, false).  // title
						AddItem(nil, 1, 0, false).       // spacing
						AddItem(em.message, 0, 1, true). // flexible message area
						AddItem(nil, 1, 0, false)        // bottom padding

	// Set background
	em.Flex.SetBackgroundColor(tcell.GetColor(ColorBase01))

	// Set up any-key-to-quit behavior
	em.Flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if em.onClose != nil {
			em.onClose()
		}
		return nil
	})

	return em
}

// OllamaModelDownloadModal creates a modal for model downloads with input and progress
func OllamaModelDownloadModal(app *tview.Application, defaultModel string, onClose func(result ModalResult, modelName string)) *Modal {
	title := "Download Ollama Model"
	message := "No Ollama model is available.\n\nEnter the model name to download:"
	placeholder := defaultModel
	if placeholder == "" {
		placeholder = "qwen3:latest"
	}

	modal := NewInputProgressModal(app, title, message, placeholder, onClose)
	// Pre-populate with default model
	if modal.inputField != nil && defaultModel != "" {
		modal.inputField.SetText(defaultModel)
	}
	return modal
}

// ResizeForError reconfigures the modal layout to give the message maximum flexible space
func (m *Modal) ResizeForError() {
	// Clear the current layout
	m.Flex.Clear()

	// Rebuild with flexible message area
	m.Flex.AddItem(nil, 1, 0, false). // top padding
						AddItem(m.title, 2, 0, false).  // fixed title
						AddItem(nil, 1, 0, false).      // spacing
						AddItem(m.message, 0, 1, true). // flexible message area (takes remaining space)
						AddItem(nil, 1, 0, false)       // bottom padding

	// Make sure message can wrap text and use full width
	m.message.SetWrap(true)
	m.message.SetTextAlign(tview.AlignLeft) // Left align for better readability of error text
}
