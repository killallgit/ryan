package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/rivo/tview"
)

// ChatView represents the chat interface using tview
type ChatView struct {
	*tview.Flex
	
	// Components
	messages     *tview.TextView
	input        *tview.InputField
	status       *tview.TextView
	
	// State
	controller   ControllerInterface
	app          *tview.Application
	sending      bool
	streaming    bool
	streamID     string
	streamBuffer string
	
	// Callbacks
	onSendMessage func(content string)
}

// NewChatView creates a new chat view
func NewChatView(controller ControllerInterface, app *tview.Application) *ChatView {
	cv := &ChatView{
		Flex:       tview.NewFlex().SetDirection(tview.FlexRow),
		controller: controller,
		app:        app,
		sending:    false,
		streaming:  false,
	}
	
	// Set background color for the entire view
	cv.SetBackgroundColor(GetTcellColor(ColorBase00))
	
	// Create message display
	cv.messages = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true)
	cv.messages.SetBorder(false)
	cv.messages.SetBackgroundColor(GetTcellColor(ColorBase00))
	
	// Create input field with prompt inside
	cv.input = tview.NewInputField().
		SetLabel("> ").
		SetFieldBackgroundColor(GetTcellColor(ColorBase00)).
		SetFieldTextColor(GetTcellColor(ColorBase05)).
		SetLabelColor(GetTcellColor(ColorOrange))
	cv.input.SetBackgroundColor(GetTcellColor(ColorBase00))
	
	cv.input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := cv.input.GetText()
			if text != "" && !cv.sending {
				cv.input.SetText("")
				if cv.onSendMessage != nil {
					cv.onSendMessage(text)
				}
			}
		}
	})
	
	// Create status bar
	cv.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	cv.status.SetBackgroundColor(GetTcellColor(ColorBase00))
	cv.updateStatus()
	
	// Create padded message area with inner padding
	messageContainer := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).                    // Left padding
		AddItem(cv.messages, 0, 1, false).           // Messages content
		AddItem(nil, 2, 0, false)                    // Right padding
	messageContainer.SetBackgroundColor(GetTcellColor(ColorBase00))
	
	// Create padded input area
	inputContainer := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).                    // Left padding
		AddItem(cv.input, 0, 1, true).               // Input content
		AddItem(nil, 2, 0, false)                    // Right padding
	inputContainer.SetBackgroundColor(GetTcellColor(ColorBase00))
	
	// Layout: top padding, messages with padding, gap, input with padding, status
	cv.AddItem(nil, 1, 0, false).                   // Top padding
		AddItem(messageContainer, 0, 1, false).
		AddItem(nil, 1, 0, false).                   // Gap between messages and input
		AddItem(inputContainer, 2, 0, true).         // Input area with more height
		AddItem(cv.status, 1, 0, false)
	
	// Initial message update
	cv.UpdateMessages()
	
	return cv
}

// SetSendMessageHandler sets the callback for sending messages
func (cv *ChatView) SetSendMessageHandler(handler func(content string)) {
	cv.onSendMessage = handler
}

// UpdateMessages updates the message display with current conversation
func (cv *ChatView) UpdateMessages() {
	cv.messages.Clear()
	
	history := cv.controller.GetHistory()
	var output strings.Builder
	
	for i, msg := range history {
		if msg.Role == chat.RoleSystem {
			continue // Skip system messages in display
		}
		
		// Add spacing between messages
		if i > 0 {
			output.WriteString("\n\n")
		}
		
		// Format message based on role with colors only (no labels)
		switch msg.Role {
		case chat.RoleUser, "human":
			output.WriteString(fmt.Sprintf("[#93b56b]%s[-]", msg.Content))
		case chat.RoleAssistant:
			output.WriteString(fmt.Sprintf("[#6b93b5]%s[-]", msg.Content))
		case chat.RoleError:
			output.WriteString(fmt.Sprintf("[#d95f5f]%s[-]", msg.Content))
		default:
			output.WriteString(fmt.Sprintf("[#f5b761]%s[-]", msg.Content))
		}
	}
	
	// Add streaming content if active
	if cv.streaming && cv.streamBuffer != "" {
		output.WriteString("\n\n[#6b93b5]")
		output.WriteString(cv.streamBuffer)
		output.WriteString("[#eb8755]█[-]") // Cursor
	}
	
	cv.messages.SetText(output.String())
	cv.messages.ScrollToEnd()
}

// StartStreaming indicates streaming has started
func (cv *ChatView) StartStreaming(streamID string) {
	cv.streaming = true
	cv.streamID = streamID
	cv.streamBuffer = ""
	cv.updateStatus()
}

// UpdateStreamingContent updates the streaming message content
func (cv *ChatView) UpdateStreamingContent(streamID string, content string) {
	if cv.streamID == streamID {
		cv.streamBuffer = content
		cv.UpdateMessages()
	}
}

// CompleteStreaming marks streaming as complete
func (cv *ChatView) CompleteStreaming(streamID string, finalMessage chat.Message) {
	if cv.streamID == streamID {
		cv.streaming = false
		cv.streamID = ""
		cv.streamBuffer = ""
		cv.updateStatus()
	}
}

// SetSending updates the sending state
func (cv *ChatView) SetSending(sending bool) {
	cv.sending = sending
	cv.input.SetDisabled(sending)
	cv.updateStatus()
}

// updateStatus updates the status bar
func (cv *ChatView) updateStatus() {
	// Model info (right-aligned)
	model := cv.controller.GetModel()
	statusText := fmt.Sprintf("[#f5b761]%s[-]", model)
	
	cv.status.SetTextAlign(tview.AlignRight)
	cv.status.SetText(statusText)
}

// Focus implements tview.Primitive
func (cv *ChatView) Focus(delegate func(p tview.Primitive)) {
	delegate(cv.input)
}

// HasFocus implements tview.Primitive
func (cv *ChatView) HasFocus() bool {
	return cv.input.HasFocus()
}

// InputHandler returns the handler for this primitive
func (cv *ChatView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return cv.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		// Handle special keys
		switch event.Key() {
		case tcell.KeyPgUp:
			cv.messages.ScrollToBeginning()
			return
		case tcell.KeyPgDn:
			cv.messages.ScrollToEnd()
			return
		case tcell.KeyUp:
			// Could implement history navigation here
			return
		case tcell.KeyDown:
			// Could implement history navigation here
			return
		}
		
		// Pass to input field if it has focus
		if cv.input.HasFocus() {
			cv.input.InputHandler()(event, setFocus)
		}
		
		// For unhandled keys (like Escape), let the parent handler deal with them
		// by not returning early - this allows WrapInputHandler to pass them up
	})
}