package tui

import (
	"context"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
)

type ChatView struct {
	controller       *controllers.ChatController
	input            InputField
	messages         MessageDisplay
	status           StatusBar
	layout           Layout
	screen           tcell.Screen
	alert            AlertDisplay
	downloadModal    DownloadPromptModal
	progressModal    ProgressModal
	downloadCtx      context.Context
	downloadCancel   context.CancelFunc
	pendingMessage   string
	modelsController *controllers.ModelsController
}

func NewChatView(controller *controllers.ChatController, modelsController *controllers.ModelsController, screen tcell.Screen) *ChatView {
	width, height := screen.Size()

	view := &ChatView{
		controller:       controller,
		input:            NewInputField(width),
		messages:         NewMessageDisplay(width, height-5), // -5 for status, input, and alert areas
		status:           NewStatusBar(width).WithModel(controller.GetModel()).WithStatus("Ready").WithModelAvailability(true),
		layout:           NewLayout(width, height),
		screen:           screen,
		alert:            NewAlertDisplay(width),
		downloadModal:    NewDownloadPromptModal(),
		progressModal:    NewProgressModal(),
		downloadCtx:      nil,
		downloadCancel:   nil,
		pendingMessage:   "",
		modelsController: modelsController,
	}

	view.updateMessages()
	return view
}

func (cv *ChatView) Name() string {
	return "chat"
}

func (cv *ChatView) Description() string {
	return "Chat with AI"
}

func (cv *ChatView) Render(screen tcell.Screen, area Rect) {
	messageArea, alertArea, inputArea, statusArea := cv.layout.CalculateAreas()

	RenderMessages(screen, cv.messages, messageArea)
	RenderTokensOnly(screen, alertArea, cv.status.PromptTokens, cv.status.ResponseTokens)

	// Create a spinner component for the input field based on current state
	inputSpinner := NewSpinnerComponent()
	if cv.alert.IsSpinnerVisible {
		inputSpinner = inputSpinner.WithVisibility(true)
		inputSpinner = SpinnerComponent{
			IsVisible: true,
			Frame:     cv.alert.SpinnerFrame,
			StartTime: inputSpinner.StartTime,
			Text:      "",
			Style:     tcell.StyleDefault.Foreground(tcell.ColorBlue).Dim(true),
		}
	}

	RenderInputWithSpinner(screen, cv.input, inputArea, inputSpinner)
	RenderStatus(screen, cv.status, statusArea)

	// Render modals on top
	cv.downloadModal.Render(screen, area)
	cv.progressModal.Render(screen, area)
}

func (cv *ChatView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool {
	log := logger.WithComponent("chat_view")
	log.Debug("ChatView handling key event", "key", ev.Key(), "rune", ev.Rune())

	// Handle modal events first
	if cv.progressModal.Visible {
		modal, cancel := cv.progressModal.HandleKeyEvent(ev)
		cv.progressModal = modal
		if cancel && cv.downloadCancel != nil {
			cv.downloadCancel()
		}
		return true
	}

	if cv.downloadModal.Visible {
		modal, confirmed, _ := cv.downloadModal.HandleKeyEvent(ev)
		cv.downloadModal = modal
		if confirmed {
			cv.startModelDownload(cv.downloadModal.ModelName)
		}
		return true
	}

	switch ev.Key() {
	case tcell.KeyEnter:
		log.Debug("Enter key pressed for message send", "sending", sending, "input_content", cv.input.Content)
		if !sending {
			content := cv.sendMessage()
			if content != "" {
				cv.screen.PostEvent(NewChatMessageSendEvent(content))
			}
		}
		return true

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		cv.input = cv.input.DeleteBackward()
		return true

	case tcell.KeyLeft:
		cv.input = cv.input.WithCursor(cv.input.Cursor - 1)
		return true

	case tcell.KeyRight:
		cv.input = cv.input.WithCursor(cv.input.Cursor + 1)
		return true

	case tcell.KeyHome:
		cv.input = cv.input.WithCursor(0)
		return true

	case tcell.KeyEnd:
		cv.input = cv.input.WithCursor(len(cv.input.Content))
		return true

	case tcell.KeyUp:
		cv.scrollUp()
		return true

	case tcell.KeyDown:
		cv.scrollDown()
		return true

	case tcell.KeyPgUp:
		cv.pageUp()
		return true

	case tcell.KeyPgDn:
		cv.pageDown()
		return true

	default:
		if ev.Rune() != 0 {
			cv.input = cv.input.InsertRune(ev.Rune())
			return true
		}
	}

	return false
}

func (cv *ChatView) HandleResize(width, height int) {
	cv.layout = NewLayout(width, height)
	cv.input = cv.input.WithWidth(width)
	cv.messages = cv.messages.WithSize(width, height-5) // -5 for status, input, and alert areas
	cv.status = cv.status.WithWidth(width)
	cv.alert = cv.alert.WithWidth(width)
}

func (cv *ChatView) sendMessage() string {
	log := logger.WithComponent("chat_view")
	content := strings.TrimSpace(cv.input.Content)
	log.Debug("sendMessage called", "content", content, "length", len(content))

	if content == "" {
		log.Debug("Empty message, skipping send")
		return ""
	}

	// Check if current model is available
	currentModel := cv.controller.GetModel()
	if err := cv.controller.ValidateModel(currentModel); err != nil {
		log.Debug("Model not available, showing download prompt", "model", currentModel, "error", err)
		// Store the message to send after download
		cv.pendingMessage = content
		cv.input = cv.input.Clear()
		cv.downloadModal = cv.downloadModal.Show(currentModel)
		return ""
	}

	cv.input = cv.input.Clear()
	log.Debug("Message content prepared for send", "content", content)

	return content
}

func (cv *ChatView) HandleMessageResponse(response MessageResponseEvent) {
	cv.status = cv.status.WithStatus("Ready")
	cv.alert = cv.alert.Clear()

	// Update token information
	promptTokens, responseTokens := cv.controller.GetTokenUsage()
	cv.status = cv.status.WithTokens(promptTokens, responseTokens)

	cv.updateMessages()
	cv.scrollToBottom()
}

func (cv *ChatView) HandleMessageError(error MessageErrorEvent) {
	cv.status = cv.status.WithStatus("Ready") // Keep status simple
	// Don't set alert error - only show error in chat messages
	cv.alert = cv.alert.Clear()

	// Update messages to show the error message that was added to conversation
	cv.updateMessages()
	cv.scrollToBottom()
}

func (cv *ChatView) SyncWithAppState(sending bool) {
	log := logger.WithComponent("chat_view")
	log.Debug("Syncing ChatView state", "app_sending", sending)

	if sending {
		cv.alert = cv.alert.WithSpinner(true, "")
	} else {
		// Always clear alert since errors only show in chat messages now
		cv.alert = cv.alert.Clear()
	}
}

func (cv *ChatView) UpdateSpinnerFrame() {
	cv.alert = cv.alert.NextSpinnerFrame()
}

func (cv *ChatView) updateMessages() {
	history := cv.controller.GetHistory()
	cv.messages = cv.messages.WithMessages(history)
}

func (cv *ChatView) scrollUp() {
	if cv.messages.Scroll > 0 {
		cv.messages = cv.messages.WithScroll(cv.messages.Scroll - 1)
	}
}

func (cv *ChatView) scrollDown() {
	cv.messages = cv.messages.WithScroll(cv.messages.Scroll + 1)
}

func (cv *ChatView) pageUp() {
	newScroll := cv.messages.Scroll - cv.messages.Height
	if newScroll < 0 {
		newScroll = 0
	}
	cv.messages = cv.messages.WithScroll(newScroll)
}

func (cv *ChatView) pageDown() {
	newScroll := cv.messages.Scroll + cv.messages.Height
	cv.messages = cv.messages.WithScroll(newScroll)
}

func (cv *ChatView) scrollToBottom() {
	var totalLines int
	// Account for chat area padding (1 character on each side, 1 line on top)
	paddedWidth := cv.messages.Width - 2
	paddedHeight := cv.messages.Height - 1

	if paddedWidth < 1 {
		paddedWidth = cv.messages.Width // Fall back if too narrow
	}
	if paddedHeight < 1 {
		paddedHeight = cv.messages.Height // Fall back if too short
	}

	for _, msg := range cv.messages.Messages {
		lines := WrapText(msg.Content, paddedWidth)
		totalLines += len(lines) + 1 // +1 for empty line between messages
	}

	// Remove the trailing empty line
	if totalLines > 0 {
		totalLines -= 1
	}

	if totalLines > paddedHeight {
		cv.messages = cv.messages.WithScroll(totalLines - paddedHeight)
	}
}

func (cv *ChatView) startModelDownload(modelName string) {
	log := logger.WithComponent("chat_view")
	log.Debug("Starting model download from chat view", "model_name", modelName)

	// Create cancellable context
	cv.downloadCtx, cv.downloadCancel = context.WithCancel(context.Background())

	// Show progress modal
	cv.progressModal = cv.progressModal.Show("Downloading Model", modelName, "Preparing download...", true)

	// Start download in goroutine
	go func() {
		err := cv.modelsController.PullWithProgress(cv.downloadCtx, modelName, func(status string, completed, total int64) {
			// Calculate progress
			progress := 0.0
			if total > 0 {
				progress = float64(completed) / float64(total)
			}

			// Post progress event
			cv.screen.PostEvent(NewModelDownloadProgressEvent(modelName, status, progress))
		})

		if err != nil {
			if err == context.Canceled {
				log.Debug("Model download cancelled in chat view", "model_name", modelName)
				cv.screen.PostEvent(NewModelDownloadErrorEvent(modelName, err))
			} else {
				log.Error("Model download failed in chat view", "model_name", modelName, "error", err)
				cv.screen.PostEvent(NewModelDownloadErrorEvent(modelName, err))
			}
		} else {
			log.Debug("Model download completed successfully in chat view", "model_name", modelName)
			cv.screen.PostEvent(NewModelDownloadCompleteEvent(modelName))
		}
	}()
}

func (cv *ChatView) HandleModelDownloadProgress(ev ModelDownloadProgressEvent) {
	log := logger.WithComponent("chat_view")
	log.Debug("Handling ModelDownloadProgressEvent in chat view", "model", ev.ModelName, "status", ev.Status, "progress", ev.Progress)

	cv.progressModal = cv.progressModal.WithProgress(ev.Progress, ev.Status).NextSpinnerFrame()
}

func (cv *ChatView) HandleModelDownloadComplete(ev ModelDownloadCompleteEvent) {
	log := logger.WithComponent("chat_view")
	log.Debug("Handling ModelDownloadCompleteEvent in chat view", "model", ev.ModelName)

	// Hide progress modal
	cv.progressModal = cv.progressModal.Hide()
	cv.downloadCtx = nil
	cv.downloadCancel = nil

	// Update status
	cv.status = cv.status.WithStatus("Model downloaded successfully: " + ev.ModelName)

	// Set as current model
	cv.controller.SetModel(ev.ModelName)
	cv.status = cv.status.WithModel(ev.ModelName)

	// Send the pending message if we have one
	if cv.pendingMessage != "" {
		log.Debug("Sending pending message after download", "message", cv.pendingMessage)
		cv.screen.PostEvent(NewChatMessageSendEvent(cv.pendingMessage))
		cv.pendingMessage = ""
	}
}

func (cv *ChatView) HandleModelDownloadError(ev ModelDownloadErrorEvent) {
	log := logger.WithComponent("chat_view")
	log.Error("Handling ModelDownloadErrorEvent in chat view", "model", ev.ModelName, "error", ev.Error)

	// Hide progress modal
	cv.progressModal = cv.progressModal.Hide()
	cv.downloadCtx = nil
	cv.downloadCancel = nil

	// Clear pending message since download failed
	cv.pendingMessage = ""

	// Update status with error
	if ev.Error == context.Canceled {
		cv.status = cv.status.WithStatus("Model download cancelled: " + ev.ModelName)
	} else {
		cv.status = cv.status.WithStatus("Model download failed: " + ev.Error.Error())
	}
}
