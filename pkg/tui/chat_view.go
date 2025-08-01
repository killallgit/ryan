package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
)

type ChatView struct {
	controller *controllers.ChatController
	input      InputField
	messages   MessageDisplay
	status     StatusBar
	layout     Layout
	screen     tcell.Screen
	alert      AlertDisplay
}

func NewChatView(controller *controllers.ChatController, screen tcell.Screen) *ChatView {
	width, height := screen.Size()

	view := &ChatView{
		controller: controller,
		input:      NewInputField(width),
		messages:   NewMessageDisplay(width, height-5), // -5 for status, input, and alert areas
		status:     NewStatusBar(width).WithModel(controller.GetModel()).WithStatus("Ready"),
		layout:     NewLayout(width, height),
		screen:     screen,
		alert:      NewAlertDisplay(width),
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
	RenderAlert(screen, cv.alert, alertArea)
	RenderInput(screen, cv.input, inputArea)
	RenderStatus(screen, cv.status, statusArea)
}

func (cv *ChatView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool {
	log := logger.WithComponent("chat_view")
	log.Debug("ChatView handling key event", "key", ev.Key(), "rune", ev.Rune())

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
	cv.alert = cv.alert.WithError("Error: " + error.Error.Error())
}

func (cv *ChatView) SyncWithAppState(sending bool) {
	log := logger.WithComponent("chat_view")
	log.Debug("Syncing ChatView state", "app_sending", sending)

	if sending {
		cv.alert = cv.alert.WithSpinner(true, "")
	} else {
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
