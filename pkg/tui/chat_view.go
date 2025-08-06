package tui

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/rivo/tview"
)

// ChatView represents the chat interface using tview
type ChatView struct {
	*tview.Flex

	// Components following TUI.md pattern
	messages        *tview.TextView   // MESSAGE_NODES
	statusContainer *tview.Flex       // STATUS_CONTAINER
	spinnerView     *tview.TextView   // SPINNER component
	agentView       *tview.TextView   // AGENT_NAME component
	actionView      *tview.TextView   // ACTION component
	messageView     *tview.TextView   // MESSAGE component
	input           *tview.InputField // CHAT_INPUT
	footer          *tview.Flex       // FOOTER_CONTAINER
	modelView       *tview.TextView   // SELECTED_MODEL

	// State
	controller    ControllerInterface
	app           *tview.Application
	sending       bool
	streaming     bool
	streamID      string
	streamBuffer  string
	spinnerFrame  int
	spinnerFrames []string
	renderManager *RenderManager
	currentState  string     // Current UI state: idle, sending, thinking, streaming, executing, preparing_tools
	currentAgent  string     // Current agent name
	currentAction string     // Current action being performed
	spinnerActive bool       // Track if spinner goroutine is running
	spinnerMutex  sync.Mutex // Protect spinner state

	// Callbacks
	onSendMessage func(content string)
}

// NewChatView creates a new chat view
func NewChatView(controller ControllerInterface, app *tview.Application) *ChatView {
	cv := &ChatView{
		Flex:          tview.NewFlex().SetDirection(tview.FlexRow),
		controller:    controller,
		app:           app,
		sending:       false,
		streaming:     false,
		spinnerFrames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		spinnerFrame:  0,
		currentState:  "idle",
	}

	// Initialize render manager
	theme := DefaultTheme()
	renderManager, err := NewRenderManager(theme, 80) // Default width, will be updated
	if err != nil {
		// Don't panic - fall back to basic rendering
		cv.renderManager = nil
	} else {
		cv.renderManager = renderManager
	}

	// Set background color for the entire view (APP_CONTAINER)
	cv.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create MESSAGE_NODES - scrollable flex column
	cv.messages = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true)
	cv.messages.SetBorder(false)
	cv.messages.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create CHAT_INPUT
	cv.input = tview.NewInputField().
		SetLabel("> ").
		SetFieldBackgroundColor(tcell.GetColor(ColorBase00)).
		SetFieldTextColor(tcell.GetColor(ColorBase05)).
		SetLabelColor(tcell.GetColor(ColorOrange))
	cv.input.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Use SetInputCapture for Enter key handling
	cv.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			text := cv.input.GetText()
			if text != "" && !cv.sending {
				cv.input.SetText("")
				if cv.onSendMessage != nil {
					cv.onSendMessage(text)
				}
				return nil // Consume the event
			}
		}

		return event // Pass through other events
	})

	// Create STATUS_CONTAINER components
	// SPINNER
	cv.spinnerView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false).
		SetWordWrap(false).
		SetScrollable(false)
	cv.spinnerView.SetBackgroundColor(tcell.GetColor(ColorBase00))
	cv.spinnerView.SetTextColor(tcell.GetColor(ColorBase04))

	// AGENT_NAME
	cv.agentView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false).
		SetWordWrap(false).
		SetScrollable(false)
	cv.agentView.SetBackgroundColor(tcell.GetColor(ColorBase00))
	cv.agentView.SetTextColor(tcell.GetColor(ColorCyan))

	// ACTION
	cv.actionView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false).
		SetWordWrap(false).
		SetScrollable(false)
	cv.actionView.SetBackgroundColor(tcell.GetColor(ColorBase00))
	cv.actionView.SetTextColor(tcell.GetColor(ColorBase04))

	// MESSAGE
	cv.messageView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false).
		SetWordWrap(false).
		SetScrollable(false)
	cv.messageView.SetBackgroundColor(tcell.GetColor(ColorBase00))
	cv.messageView.SetTextColor(tcell.GetColor(ColorBase05))

	// SELECTED_MODEL for footer
	cv.modelView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)
	cv.modelView.SetBackgroundColor(tcell.GetColor(ColorBase00))
	cv.modelView.SetTextColor(tcell.GetColor(ColorBase04))
	cv.updateModelView()

	// Create MESSAGE_CONTAINER with padding
	messageContainer := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).         // Left padding
		AddItem(cv.messages, 0, 1, false). // MESSAGE_NODES
		AddItem(nil, 2, 0, false)          // Right padding
	messageContainer.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create STATUS_CONTAINER - flex full width row with left-justified content
	cv.statusContainer = tview.NewFlex().SetDirection(tview.FlexRow)
	statusRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(cv.spinnerView, 2, 0, false). // SPINNER (2 chars wide)
		AddItem(cv.messageView, 0, 1, false). // MESSAGE (takes remaining space, left-justified)
		AddItem(nil, 0, 1, false)             // Spacer to push content left
	cv.statusContainer.AddItem(statusRow, 1, 0, false)
	cv.statusContainer.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create padded STATUS_CONTAINER
	statusContainerPadded := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).                // Left padding
		AddItem(cv.statusContainer, 0, 1, false). // Status content
		AddItem(nil, 2, 0, false)                 // Right padding
	statusContainerPadded.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create CHAT_INPUT_CONTAINER with thin border
	// Create a flex container for the input
	inputFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(cv.input, 1, 0, true)
	inputFlex.SetBorder(true).
		SetBorderColor(tcell.GetColor(ColorBase01)). // Very dim border color
		SetBorderPadding(0, 0, 0, 0).                // No padding inside border
		SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create input container with padding
	inputContainer := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).      // Left padding
		AddItem(inputFlex, 0, 1, true). // Bordered input flex
		AddItem(nil, 2, 0, false)       // Right padding
	inputContainer.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create FOOTER_CONTAINER - flex-row full width
	cv.footer = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).          // Left padding
		AddItem(cv.modelView, 0, 1, false). // SELECTED_MODEL (justified-right)
		AddItem(nil, 2, 0, false)           // Right padding
	cv.footer.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Layout according to TUI.md pattern
	cv.AddItem(nil, 1, 0, false). // Top padding
					AddItem(messageContainer, 0, 1, false).      // MESSAGE_CONTAINER
					AddItem(statusContainerPadded, 1, 0, false). // STATUS_CONTAINER
					AddItem(inputContainer, 3, 0, true).         // CHAT_INPUT_CONTAINER (3 rows for border)
					AddItem(cv.footer, 1, 0, false)              // FOOTER_CONTAINER

	// Initial message update
	cv.UpdateMessages()

	return cv
}

// formatThinkingText formats thinking blocks with italics and dim styling, removing the XML tags
func (cv *ChatView) formatThinkingText(content string) string {
	// Regex to match thinking blocks
	thinkingRegex := regexp.MustCompile(`(?s)<think(?:ing)?>\s*(.*?)\s*</think(?:ing)?>`)

	// Replace thinking blocks with formatted text, stripping the XML tags
	formatted := thinkingRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the thinking content
		submatch := thinkingRegex.FindStringSubmatch(match)
		if len(submatch) > 1 {
			thinkingContent := strings.TrimSpace(submatch[1])
			// Format with muted color and italic-like styling using underline, no XML tags
			return fmt.Sprintf("[#5c5044::u]%s[-:-:-]", thinkingContent)
		}
		return match
	})

	return formatted
}

// formatStreamingThinkingText formats thinking text during streaming, stripping XML tags
func (cv *ChatView) formatStreamingThinkingText(content string) string {
	// First, check if we have complete thinking blocks and process those
	if strings.Contains(content, "<think") && (strings.Contains(content, "</think>") || strings.Contains(content, "</thinking>")) {
		return cv.formatThinkingText(content) // Use complete block formatting which strips tags
	}

	// For streaming content, we need to strip tags and format thinking content in real-time
	var result strings.Builder
	insideThinking := false
	i := 0

	for i < len(content) {
		// Check for opening thinking tags
		if i < len(content)-5 && (strings.HasPrefix(content[i:], "<think>") || strings.HasPrefix(content[i:], "<thinking>")) {
			// Skip the opening tag
			if strings.HasPrefix(content[i:], "<thinking>") {
				i += 10 // Skip "<thinking>"
			} else {
				i += 7 // Skip "<think>"
			}
			insideThinking = true
			result.WriteString("[#5c5044::u]") // Start thinking format
			continue
		} else if strings.HasPrefix(content[i:], "<think") {
			// Partial opening tag - likely at end of stream chunk
			insideThinking = true
			result.WriteString("[#5c5044::u]") // Start thinking format
			// Skip what we can see of the tag
			for i < len(content) && content[i] != '>' {
				i++
			}
			if i < len(content) && content[i] == '>' {
				i++ // Skip the '>'
			}
			continue
		}

		// Check for closing thinking tags
		if i < len(content)-7 && (strings.HasPrefix(content[i:], "</think>") || strings.HasPrefix(content[i:], "</thinking>")) {
			// Skip the closing tag
			if insideThinking {
				result.WriteString("[-:-:-]") // End thinking format
				insideThinking = false
			}
			if strings.HasPrefix(content[i:], "</thinking>") {
				i += 11 // Skip "</thinking>"
			} else {
				i += 8 // Skip "</think>"
			}
			continue
		} else if strings.HasPrefix(content[i:], "</think") {
			// Partial closing tag - likely at end of stream chunk
			if insideThinking {
				result.WriteString("[-:-:-]") // End thinking format
				insideThinking = false
			}
			// Skip the rest of the content since it's a partial tag
			break
		}

		// Add regular character
		result.WriteByte(content[i])
		i++
	}

	// If we're still inside thinking at the end, close the formatting
	if insideThinking {
		result.WriteString("[-:-:-]")
	}

	return result.String()
}

// SetSendMessageHandler sets the callback for sending messages
func (cv *ChatView) SetSendMessageHandler(handler func(content string)) {
	cv.onSendMessage = handler
}

// UpdateMessages updates the message display with current conversation
func (cv *ChatView) UpdateMessages() {
	log := logger.WithComponent("chat_view")
	log.Debug("UpdateMessages called",
		"streaming", cv.streaming,
		"stream_buffer_length", len(cv.streamBuffer))

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

		// Use render manager if available
		if cv.renderManager != nil {
			// Detect content type and render appropriately
			contentType := cv.renderManager.DetectContentType(msg.Content)
			// Convert chat.MessageRole to string for render manager
			roleStr := "assistant"
			switch msg.Role {
			case chat.RoleUser:
				roleStr = "user"
			case chat.RoleAssistant:
				roleStr = "assistant"
			case chat.RoleSystem:
				roleStr = "system"
			case chat.RoleError:
				roleStr = "error"
			}

			// Add error handling for render manager
			func() {
				defer func() {
					if r := recover(); r != nil {
						// If render manager panics, fall back to basic rendering
						output.WriteString(fmt.Sprintf("[%s]%s[-]", cv.getRoleColor(msg.Role), msg.Content))
					}
				}()
				rendered := cv.renderManager.Render(msg.Content, contentType, roleStr)
				output.WriteString(rendered)
			}()
		} else {
			// Fallback to basic rendering
			switch msg.Role {
			case chat.RoleUser, "human":
				output.WriteString(fmt.Sprintf("[#93b56b]%s[-]", msg.Content))
			case chat.RoleAssistant:
				// Format thinking text in assistant messages and apply assistant color to rest
				formattedContent := cv.formatThinkingText(msg.Content)
				output.WriteString(fmt.Sprintf("[#6b93b5]%s[-]", formattedContent))
			case chat.RoleError:
				output.WriteString(fmt.Sprintf("[#d95f5f]%s[-]", msg.Content))
			default:
				output.WriteString(fmt.Sprintf("[#f5b761]%s[-]", msg.Content))
			}
		}
	}

	// Add streaming content if active
	if cv.streaming && cv.streamBuffer != "" {
		output.WriteString("\n\n")
		if cv.renderManager != nil {
			// Use render manager for streaming content with error handling
			func() {
				defer func() {
					if r := recover(); r != nil {
						// If render manager panics, fall back to basic streaming rendering
						formattedStreamContent := cv.formatStreamingThinkingText(cv.streamBuffer)
						output.WriteString(fmt.Sprintf("[#6b93b5]%s[-]", formattedStreamContent))
						output.WriteString("[#eb8755]█[-]") // Cursor
					}
				}()
				rendered := cv.renderManager.RenderStreamingContent(cv.streamBuffer, "assistant")
				output.WriteString(rendered)
			}()
		} else {
			// Fallback to basic streaming rendering
			formattedStreamContent := cv.formatStreamingThinkingText(cv.streamBuffer)
			output.WriteString(fmt.Sprintf("[#6b93b5]%s[-]", formattedStreamContent))
			output.WriteString("[#eb8755]█[-]") // Cursor
		}
	}

	cv.messages.SetText(output.String())
	cv.messages.ScrollToEnd()
}

// StartStreaming indicates streaming has started
func (cv *ChatView) StartStreaming(streamID string) {
	log := logger.WithComponent("chat_view")
	log.Info("StartStreaming called",
		"streamID", streamID,
		"was_streaming", cv.streaming,
		"was_sending", cv.sending)

	cv.streaming = true
	cv.streamID = streamID
	cv.streamBuffer = ""
	log.Debug("Set streaming state",
		"streaming", cv.streaming,
		"streamID", cv.streamID)

	cv.updateStatus()
	cv.startSpinner()
	log.Debug("Started spinner and updated status")
}

// UpdateStreamingContent updates the streaming message content
func (cv *ChatView) UpdateStreamingContent(streamID string, content string) {
	log := logger.WithComponent("chat_view")
	log.Debug("UpdateStreamingContent called",
		"streamID", streamID,
		"current_streamID", cv.streamID,
		"content_length", len(content),
		"is_tool_mode", content == "<<<TOOL_MODE>>>")

	if cv.streamID == streamID {
		// Check for tool mode marker
		if content == "<<<TOOL_MODE>>>" {
			cv.currentState = "preparing_tools"
			cv.updateSpinnerView()
			log.Debug("Tool mode marker detected, updating state")
			return // Don't add marker to buffer
		}

		cv.streamBuffer = content
		log.Debug("Updated stream buffer", "buffer_length", len(cv.streamBuffer))
		cv.UpdateMessages()
	} else {
		log.Warn("StreamID mismatch", "expected", cv.streamID, "got", streamID)
	}
}

// CompleteStreaming marks streaming as complete
func (cv *ChatView) CompleteStreaming(streamID string, finalMessage chat.Message) {
	log := logger.WithComponent("chat_view")
	log.Info("CompleteStreaming called",
		"streamID", streamID,
		"current_streamID", cv.streamID,
		"final_message_role", finalMessage.Role,
		"final_message_length", len(finalMessage.Content))

	if cv.streamID == streamID {
		cv.streaming = false
		cv.streamID = ""
		cv.streamBuffer = ""
		log.Debug("Cleared streaming state")
		cv.updateStatus()
	} else {
		log.Warn("StreamID mismatch in complete", "expected", cv.streamID, "got", streamID)
	}
}

// SetSending updates the sending state
func (cv *ChatView) SetSending(sending bool) {
	log := logger.WithComponent("chat_view")
	log.Info("SetSending called",
		"new_sending", sending,
		"was_sending", cv.sending,
		"is_streaming", cv.streaming)

	cv.sending = sending
	cv.input.SetDisabled(sending)
	cv.updateStatus()

	// Start or stop spinner animation
	if sending {
		cv.currentState = "sending"
		log.Debug("Starting spinner from SetSending")
		cv.startSpinner()
	} else {
		cv.currentState = "idle"
		log.Debug("SetSending(false) - spinner will stop naturally")
	}
	cv.updateSpinnerView()
	cv.updateStatus()
}

// updateModelView updates the SELECTED_MODEL in footer
func (cv *ChatView) updateModelView() {
	if cv.modelView == nil {
		return
	}
	model := cv.controller.GetModel()
	cv.modelView.SetText(fmt.Sprintf("[#f5b761]%s[-]", model))
}

// updateStatusComponents updates all STATUS_CONTAINER components
func (cv *ChatView) updateStatusComponents() {
	// Update spinner
	if cv.sending || cv.streaming || cv.currentState != "idle" {
		spinner := cv.spinnerFrames[cv.spinnerFrame]
		cv.spinnerView.SetText(fmt.Sprintf("[#93b56b]%s[-]", spinner))
	} else {
		cv.spinnerView.SetText("")
	}

	// Build combined status message (left-justified next to spinner)
	var statusParts []string

	// Add primary status
	if cv.sending {
		statusParts = append(statusParts, "[#f5b761]Sending...[-]")
	} else if cv.streaming {
		statusParts = append(statusParts, "[#6b93b5]Streaming...[-]")
	} else if cv.currentState == "thinking" {
		statusParts = append(statusParts, "[#976bb5]Thinking...[-]")
	} else if cv.currentState == "executing" {
		if cv.currentAction != "" {
			statusParts = append(statusParts, fmt.Sprintf("[#d95f5f]Executing %s...[-]", cv.currentAction))
		} else {
			statusParts = append(statusParts, "[#d95f5f]Executing tools...[-]")
		}
	} else if cv.currentState == "preparing_tools" {
		statusParts = append(statusParts, "[#f5b761]Preparing tools...[-]")
	}

	// Add agent info if present
	if cv.currentAgent != "" && cv.currentState != "idle" {
		statusParts = append(statusParts, fmt.Sprintf("[#6b93b5](%s)[-]", cv.currentAgent))
	}

	// Combine all parts with a space
	cv.messageView.SetText(strings.Join(statusParts, " "))

	// Note: agentView and actionView are no longer displayed separately
	cv.agentView.SetText("")
	cv.actionView.SetText("")
}

func (cv *ChatView) updateStatus() {
	cv.updateStatusComponents()
	cv.updateModelView()
}

// Focus implements tview.Primitive
func (cv *ChatView) Focus(delegate func(p tview.Primitive)) {
	// Let the Flex handle focus delegation to the focusable input
	cv.Flex.Focus(delegate)
}

// HasFocus implements tview.Primitive
func (cv *ChatView) HasFocus() bool {
	return cv.input.HasFocus()
}

// InputHandler returns the handler for this primitive with global shortcuts
func (cv *ChatView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		// Handle global shortcuts first
		switch event.Key() {
		case tcell.KeyCtrlC:
			// Quit application
			if cv.app != nil {
				cv.app.Stop()
			}
			return
		case tcell.KeyCtrlP:
			// Could implement view switcher here if needed
			return
		case tcell.KeyEscape:
			// Could implement view navigation here if needed
			return
		}

		// Handle Ctrl+1 through Ctrl+5 for view switching
		if event.Key() >= tcell.KeyCtrlA && event.Key() <= tcell.KeyCtrlE {
			// Could implement view switching here if needed
			return
		}

		// For all other events, delegate to the default Flex handler
		cv.Flex.InputHandler()(event, setFocus)
	}
}

// UpdateActivityTree updates the activity display (now part of status)
func (cv *ChatView) UpdateActivityTree(treeText string) {
	// Parse the tree text to extract agent and action info
	if treeText != "" {
		lines := strings.Split(treeText, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Assistant") || strings.Contains(line, "Agent") {
				// Extract agent name
				if strings.Contains(line, "●") { // Active agent
					parts := strings.Split(line, " ")
					for _, part := range parts {
						if part != "" && part != "●" && !strings.Contains(part, "─") {
							cv.currentAgent = strings.TrimSpace(part)
							break
						}
					}
				}
			} else if strings.Contains(line, "├──") || strings.Contains(line, "└──") {
				// Extract action
				actionStart := strings.LastIndex(line, "──") + 2
				if actionStart < len(line) {
					cv.currentAction = strings.TrimSpace(line[actionStart:])
				}
			}
		}
	} else {
		cv.currentAgent = ""
		cv.currentAction = ""
	}
	cv.updateStatusComponents()
}

// ClearActivityTree clears the activity tree display
func (cv *ChatView) ClearActivityTree() {
	cv.UpdateActivityTree("")
}

// startSpinner starts the spinner animation
func (cv *ChatView) startSpinner() {
	cv.spinnerMutex.Lock()
	defer cv.spinnerMutex.Unlock()

	// Don't start if already running
	if cv.spinnerActive {
		return
	}

	log := logger.WithComponent("chat_view_spinner")
	log.Info("startSpinner called",
		"sending", cv.sending,
		"streaming", cv.streaming)

	cv.spinnerActive = true

	go func() {
		iterations := 0
		for {
			cv.spinnerMutex.Lock()
			shouldContinue := cv.sending || cv.streaming || cv.currentState != "idle"
			cv.spinnerMutex.Unlock()

			if !shouldContinue {
				break
			}

			iterations++
			time.Sleep(100 * time.Millisecond)

			cv.spinnerMutex.Lock()
			cv.spinnerFrame = (cv.spinnerFrame + 1) % len(cv.spinnerFrames)
			cv.spinnerMutex.Unlock()

			cv.app.QueueUpdateDraw(func() {
				cv.updateSpinnerView()
				cv.updateStatus()
			})

			// Log every 10 iterations (1 second)
			if iterations%10 == 0 {
				log.Debug("Spinner still running",
					"iterations", iterations,
					"sending", cv.sending,
					"streaming", cv.streaming,
					"current_state", cv.currentState,
					"frame", cv.spinnerFrame)
			}
		}

		cv.spinnerMutex.Lock()
		cv.spinnerActive = false
		cv.spinnerMutex.Unlock()

		log.Info("Spinner stopped",
			"total_iterations", iterations,
			"final_sending", cv.sending,
			"final_streaming", cv.streaming,
			"final_state", cv.currentState)
	}()
}

// OnResize handles terminal resize events
func (cv *ChatView) OnResize(width, height int) {
	if cv.renderManager != nil {
		// Update render manager width for proper text wrapping
		cv.renderManager.SetWidth(width - 4) // Account for padding
	}
}

// getRoleColor returns the color for a given message role
func (cv *ChatView) getRoleColor(role string) string {
	switch role {
	case chat.RoleUser:
		return "#93b56b"
	case chat.RoleAssistant:
		return "#6b93b5"
	case chat.RoleError:
		return "#d95f5f"
	case chat.RoleSystem:
		return "#976bb5"
	default:
		return "#f5b761"
	}
}

// SetStreaming updates the streaming state
func (cv *ChatView) SetStreaming(streaming bool, streamID string) {
	cv.streaming = streaming
	cv.streamID = streamID
	if streaming {
		cv.currentState = "streaming"
		cv.startSpinner()
		cv.streamBuffer = ""
	} else if cv.currentState == "streaming" {
		cv.currentState = "idle"
	}
	cv.updateSpinnerView()
	cv.updateStatus()
}

// SetThinking sets the thinking state
func (cv *ChatView) SetThinking(thinking bool) {
	if thinking {
		cv.currentState = "thinking"
		cv.startSpinner()
	} else if cv.currentState == "thinking" {
		cv.currentState = "idle"
	}
	cv.updateSpinnerView()
}

// SetExecuting sets the executing state (for tool execution)
func (cv *ChatView) SetExecuting(executing bool, toolName string) {
	if executing {
		cv.currentState = "executing"
		cv.startSpinner()
		// Store tool name in streamID temporarily for display
		cv.streamID = toolName
	} else if cv.currentState == "executing" {
		cv.currentState = "idle"
		cv.streamID = ""
	}
	cv.updateSpinnerView()
}

// updateSpinnerView updates the spinner component in STATUS_CONTAINER
func (cv *ChatView) updateSpinnerView() {
	// This is now handled by updateStatusComponents
	cv.updateStatusComponents()
}
