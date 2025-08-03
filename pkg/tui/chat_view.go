package tui

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
)

// ChatViewMode represents the current interaction mode
type ChatViewMode int

const (
	ModeInput ChatViewMode = iota // Text input mode (default)
	ModeNodes                     // Node selection/navigation mode
)

func (m ChatViewMode) String() string {
	switch m {
	case ModeInput:
		return "Input"
	case ModeNodes:
		return "Select"
	default:
		return "Unknown"
	}
}

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
	helpModal        HelpModal
	downloadCtx      context.Context
	downloadCancel   context.CancelFunc
	pendingMessage   string
	modelsController *controllers.ModelsController

	// Interaction mode
	mode ChatViewMode // Current interaction mode

	// Streaming state
	isStreaming         bool
	streamingContent    string
	currentStreamID     string
	isStreamingThinking bool   // Track if currently streaming thinking content
	thinkingContent     string // Accumulate thinking content separately
	responseContent     string // Accumulate response content separately

	// Early detection buffering
	contentBuffer       string // Buffer for early content type detection
	contentTypeDetected bool   // Whether we've determined the content type
	bufferSize          int    // Current buffer size
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
		helpModal:        NewHelpModal(),
		downloadCtx:      nil,
		downloadCancel:   nil,
		pendingMessage:   "",
		modelsController: modelsController,

		// Initialize interaction mode
		mode: ModeInput, // Start in input mode

		// Initialize streaming state
		isStreaming:         false,
		streamingContent:    "",
		currentStreamID:     "",
		isStreamingThinking: false,
		thinkingContent:     "",
		responseContent:     "",

		// Initialize buffering state
		contentBuffer:       "",
		contentTypeDetected: false,
		bufferSize:          0,
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

	// Use streaming-aware render function that can apply thinking styles during streaming
	spinner := SpinnerComponent{
		IsVisible: cv.alert.IsSpinnerVisible,
		Frame:     cv.alert.SpinnerFrame,
		Text:      cv.alert.SpinnerText,
	}
	RenderMessagesWithStreamingState(screen, cv.messages, messageArea, spinner, cv.isStreamingThinking)
	RenderTokensWithSpinner(screen, alertArea, cv.status.PromptTokens, cv.status.ResponseTokens, cv.alert.IsSpinnerVisible, GetSpinnerFrame(cv.alert.SpinnerFrame))
	RenderInput(screen, cv.input, inputArea)
	RenderStatus(screen, cv.status, statusArea)

	// Render modals on top
	cv.downloadModal.Render(screen, area)
	cv.progressModal.Render(screen, area)
	cv.helpModal.Render(screen, area)
}

func (cv *ChatView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool {
	// Handle modal events first
	if cv.helpModal.Visible {
		modal, handled := cv.helpModal.HandleKeyEvent(ev)
		cv.helpModal = modal
		return handled
	}

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

	// Handle help key (?) in any mode
	if cv.handleHelpKey(ev) {
		return true
	}

	// Handle mode switching (Ctrl+N)
	if cv.handleModeSwitch(ev) {
		return true
	}

	// Handle keys based on current mode
	switch cv.mode {
	case ModeInput:
		return cv.handleInputModeKeys(ev, sending)
	case ModeNodes:
		return cv.handleNodeModeKeys(ev, sending)
	default:
		return cv.handleInputModeKeys(ev, sending) // Fallback to input mode
	}
}

func (cv *ChatView) handleHelpKey(ev *tcell.EventKey) bool {
	// ? key to show help modal
	if ev.Rune() == '?' {
		cv.helpModal = cv.helpModal.Show()
		return true
	}
	return false
}

func (cv *ChatView) handleModeSwitch(ev *tcell.EventKey) bool {
	// Ctrl+N to switch modes - use tcell's built-in KeyCtrlN constant
	if ev.Key() == tcell.KeyCtrlN {
		cv.switchMode()
		return true
	}
	return false
}

func (cv *ChatView) switchMode() {
	log := logger.WithComponent("chat_view")

	switch cv.mode {
	case ModeInput:
		cv.mode = ModeNodes
		// Enable node-based rendering if not already enabled
		cv.messages = cv.messages.EnableNodes()
		// Auto-focus first node if none focused
		if cv.messages.NodeManager != nil && cv.messages.NodeManager.GetFocusedNode() == "" {
			cv.messages.MoveFocusDown() // This will focus the first node
		}
		log.Debug("Switched to node selection mode")
	case ModeNodes:
		cv.mode = ModeInput
		// Clear any node focus when switching back to input
		if cv.messages.NodeManager != nil {
			cv.messages.NodeManager.SetFocusedNode("")
		}
		log.Debug("Switched to input mode")
	}

	// Update status bar to show current mode
	cv.updateStatusForMode()
}

func (cv *ChatView) updateStatusForMode() {
	// Update the status to show the current mode with enhanced visual indicators
	var modeText string
	if cv.mode == ModeNodes {
		focusedNode := ""
		if cv.messages.NodeManager != nil {
			focusedNode = cv.messages.NodeManager.GetFocusedNode()
		}

		if focusedNode != "" {
			displayNode := focusedNode
			if len(focusedNode) > 8 {
				displayNode = focusedNode[:8]
			}
			modeText = fmt.Sprintf("🎯 Node Select | Focused: %s | j/k=nav, Tab=expand, Space=select, Esc/i=input", displayNode)
		} else {
			modeText = "🎯 Node Select | j/k=navigate, Tab=expand, Space=select, Esc/i=input"
		}
	} else {
		modeText = "✏️ Input | Ctrl+N=node mode, ?=help"
	}
	cv.status = cv.status.WithStatus(modeText)
}

func (cv *ChatView) handleInputModeKeys(ev *tcell.EventKey, sending bool) bool {
	switch ev.Key() {
	case tcell.KeyEnter:
		if !sending {
			content := cv.sendMessage()
			if content != "" {
				cv.screen.PostEvent(NewChatMessageSendEvent(content))
				// Immediately update the UI to show the user message
				cv.updateMessages()
				cv.scrollToBottom()
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
		// Check for Ctrl+Up for node navigation
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			if cv.handleNodeNavigation(true) {
				return true
			}
		}
		cv.scrollUp()
		return true

	case tcell.KeyDown:
		// Check for Ctrl+Down for node navigation
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			if cv.handleNodeNavigation(false) {
				return true
			}
		}
		cv.scrollDown()
		return true

	case tcell.KeyPgUp:
		cv.pageUp()
		return true

	case tcell.KeyPgDn:
		cv.pageDown()
		return true

	case tcell.KeyTab:
		// Tab to expand/collapse focused node
		if cv.handleNodeToggleExpansion() {
			return true
		}

	default:
		if ev.Rune() != 0 {
			// Check for specific key combinations for node operations
			switch ev.Rune() {
			case ' ':
				// Space to select/deselect focused node
				if ev.Modifiers()&tcell.ModCtrl != 0 {
					if cv.handleNodeToggleSelection() {
						return true
					}
				}
			case 'a', 'A':
				// Ctrl+A to select all nodes
				if ev.Modifiers()&tcell.ModCtrl != 0 {
					if cv.handleSelectAllNodes() {
						return true
					}
				}
			case 'c', 'C':
				// Ctrl+C to clear selection (if not sending)
				if ev.Modifiers()&tcell.ModCtrl != 0 && !sending {
					if cv.handleClearNodeSelection() {
						return true
					}
				}
			}

			cv.input = cv.input.InsertRune(ev.Rune())
			return true
		}
	}

	return false
}

func (cv *ChatView) handleNodeModeKeys(ev *tcell.EventKey, sending bool) bool {
	// In node mode, most keys are for navigation and selection
	switch ev.Key() {
	case tcell.KeyEnter:
		// Enter toggles selection of focused node
		if cv.handleNodeToggleSelection() {
			return true
		}

	case tcell.KeyTab:
		// Tab toggles expansion of focused node
		if cv.handleNodeToggleExpansion() {
			return true
		}

	case tcell.KeyUp:
		// Up arrow moves focus up
		if cv.handleNodeNavigation(true) {
			return true
		}

	case tcell.KeyDown:
		// Down arrow moves focus down
		if cv.handleNodeNavigation(false) {
			return true
		}

	case tcell.KeyPgUp:
		cv.pageUp()
		return true

	case tcell.KeyPgDn:
		cv.pageDown()
		return true

	case tcell.KeyEscape:
		// Escape switches back to input mode
		cv.mode = ModeInput
		if cv.messages.NodeManager != nil {
			cv.messages.NodeManager.SetFocusedNode("")
		}
		cv.updateStatusForMode()
		return true

	default:
		if ev.Rune() != 0 {
			switch ev.Rune() {
			case 'j', 'J':
				// j key moves focus down (vim-style)
				if cv.handleNodeNavigation(false) {
					return true
				}

			case 'k', 'K':
				// k key moves focus up (vim-style)
				if cv.handleNodeNavigation(true) {
					return true
				}

			case ' ':
				// Space toggles selection of focused node
				if cv.handleNodeToggleSelection() {
					return true
				}

			case 'a', 'A':
				// a to select all nodes
				if cv.handleSelectAllNodes() {
					return true
				}

			case 'c', 'C':
				// c to clear selection
				if cv.handleClearNodeSelection() {
					return true
				}

			case 'i', 'I':
				// i to switch to input mode (vim-style)
				cv.mode = ModeInput
				if cv.messages.NodeManager != nil {
					cv.messages.NodeManager.SetFocusedNode("")
				}
				cv.updateStatusForMode()
				return true
			}
		}
	}

	return false
}

func (cv *ChatView) HandleMouseEvent(ev *tcell.EventMouse) bool {
	log := logger.WithComponent("chat_view")

	// Get mouse coordinates
	x, y := ev.Position()
	buttons := ev.Buttons()

	log.Debug("Chat view mouse event", "x", x, "y", y, "buttons", buttons)

	// Only handle left mouse button clicks
	if buttons&tcell.ButtonPrimary == 0 {
		return false
	}

	// Check if the click is in the message area
	messageArea, _, _, _ := cv.layout.CalculateAreas()

	if x >= messageArea.X && x < messageArea.X+messageArea.Width &&
		y >= messageArea.Y && y < messageArea.Y+messageArea.Height {

		// Handle click in message area
		if cv.messages.UseNodes && cv.messages.NodeManager != nil {
			// Use node-based click handling
			nodeID, handled := cv.messages.HandleClick(x, y)
			if handled {
				log.Debug("Node click handled", "node_id", nodeID, "x", x, "y", y)

				// Switch to node mode when a node is clicked
				if cv.mode == ModeInput {
					cv.mode = ModeNodes
					cv.updateStatusForMode()
					log.Debug("Switched to node mode due to mouse click on node")
				}

				// Focus the clicked node
				cv.messages.NodeManager.SetFocusedNode(nodeID)

				// Post node click event for further processing if needed
				cv.screen.PostEvent(NewMessageNodeClickEvent(nodeID, x-messageArea.X, y-messageArea.Y))
				return true
			}
		}

		// For legacy mode or if node handling didn't work,
		// we could implement basic click handling here
		log.Debug("Click in message area not handled by nodes", "x", x, "y", y)
		return true // Still consume the event even if not handled
	}

	// Click was not in message area
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

func (cv *ChatView) updateMessagesWithStreamingThinking() {
	history := cv.controller.GetHistory()

	// If we're streaming and have detected content type, show streaming content
	if cv.isStreaming && cv.contentTypeDetected {
		// Create a copy of history to avoid modifying the original
		messagesWithStreaming := make([]chat.Message, len(history))
		copy(messagesWithStreaming, history)

		// Create properly formatted streaming message with thinking detection
		streamingMessage := cv.createStreamingMessage()

		// Only add the streaming message if it has content
		if streamingMessage.Content != "" {
			messagesWithStreaming = append(messagesWithStreaming, streamingMessage)
		}

		cv.messages = cv.messages.WithMessages(messagesWithStreaming)

		// Auto-scroll to bottom during streaming
		cv.scrollToBottom()
	} else {
		// No streaming or content type not detected yet, show regular messages
		cv.messages = cv.messages.WithMessages(history)
	}
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
	// Account for chat area padding (1 character on each side, 1 line on top)
	paddedWidth := cv.messages.Width - 2
	paddedHeight := cv.messages.Height - 1

	if paddedWidth < 1 {
		paddedWidth = cv.messages.Width // Fall back if too narrow
	}
	if paddedHeight < 1 {
		paddedHeight = cv.messages.Height // Fall back if too short
	}

	// Use the same line calculation logic as the rendering function
	// Check if we're currently streaming thinking content
	streamingThinking := cv.isStreamingThinking
	totalLines := CalculateMessageLines(cv.messages.Messages, paddedWidth, streamingThinking)

	if totalLines > paddedHeight {
		cv.messages = cv.messages.WithScroll(totalLines - paddedHeight)
	} else {
		cv.messages = cv.messages.WithScroll(0)
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

// Streaming Helper Methods

// detectThinkingStart checks if content begins with <think> or <thinking> tags
func (cv *ChatView) detectThinkingStart(content string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(content))
	return strings.HasPrefix(trimmed, "<think>") || strings.HasPrefix(trimmed, "<thinking>")
}

// detectContentTypeFromBuffer analyzes the buffer to determine content type
// Returns true if content type has been determined, false if more buffering needed
func (cv *ChatView) detectContentTypeFromBuffer() bool {
	const minBufferSize = 10 // Need at least 10 chars to detect "<thinking>"

	if cv.bufferSize < minBufferSize && cv.bufferSize < len(cv.streamingContent) {
		// Still need more characters for reliable detection
		return false
	}

	// Check if it starts with thinking tags
	if cv.detectThinkingStart(cv.contentBuffer) {
		cv.isStreamingThinking = true
		// Extract content after the opening tag
		thinkStartRegex := regexp.MustCompile(`(?i)<think(?:ing)?>`)
		cv.thinkingContent = strings.TrimSpace(thinkStartRegex.ReplaceAllString(cv.contentBuffer, ""))
	} else {
		cv.isStreamingThinking = false
		// Not thinking content, treat as regular response
		cv.responseContent = cv.contentBuffer
	}

	cv.contentTypeDetected = true
	return true
}

// processStreamingContent processes the full streaming content and separates thinking from response
func (cv *ChatView) processStreamingContent() {
	fullContent := cv.streamingContent

	// If we haven't detected thinking yet, check for thinking tags at the start
	if !cv.isStreamingThinking && len(cv.thinkingContent) == 0 && len(cv.responseContent) == 0 {
		if cv.detectThinkingStart(fullContent) {
			cv.isStreamingThinking = true
		}
	}

	// Process the content based on current state
	if cv.isStreamingThinking {
		// Check if thinking block ends
		thinkEndRegex := regexp.MustCompile(`(?i)</think(?:ing)?>`)
		if thinkEndRegex.MatchString(fullContent) {
			// Split at the end of thinking block
			parts := thinkEndRegex.Split(fullContent, 2)
			if len(parts) == 2 {
				// Extract thinking content (remove opening tags)
				thinkStartRegex := regexp.MustCompile(`(?i)<think(?:ing)?>`)
				thinkingRaw := thinkStartRegex.ReplaceAllString(parts[0], "")
				cv.thinkingContent = strings.TrimSpace(thinkingRaw)

				// Start response content
				cv.responseContent = strings.TrimSpace(parts[1])
				cv.isStreamingThinking = false
			}
		} else {
			// Still in thinking block, accumulate thinking content
			thinkStartRegex := regexp.MustCompile(`(?i)<think(?:ing)?>`)
			cv.thinkingContent = strings.TrimSpace(thinkStartRegex.ReplaceAllString(fullContent, ""))
		}
	} else {
		// In response mode or no thinking detected
		if len(cv.thinkingContent) == 0 {
			// No thinking content detected, treat as regular response
			cv.responseContent = fullContent
		} else {
			// Already have thinking content, extract response part from full content
			thinkEndRegex := regexp.MustCompile(`(?i)</think(?:ing)?>`)
			if thinkEndRegex.MatchString(fullContent) {
				parts := thinkEndRegex.Split(fullContent, 2)
				if len(parts) == 2 {
					cv.responseContent = strings.TrimSpace(parts[1])
				}
			}
		}
	}
}

// createStreamingMessage creates a properly formatted message for streaming display
func (cv *ChatView) createStreamingMessage() chat.Message {
	var content string

	// If we haven't detected content type yet, don't show anything
	if !cv.contentTypeDetected && cv.isStreaming {
		return chat.Message{
			Role:    chat.RoleAssistant,
			Content: "", // Show nothing while buffering
		}
	}

	if cv.thinkingContent != "" {
		// Format thinking content with proper tags so ParseThinkingBlock can style it correctly
		thinkingWithTags := "<think>" + cv.thinkingContent

		if cv.isStreamingThinking {
			// Still streaming thinking content, add cursor before closing tag
			content = thinkingWithTags + " ▌"
		} else {
			// Thinking complete, close tag and add response if any
			content = thinkingWithTags + "</think>"

			if cv.responseContent != "" {
				// Add response content with cursor if still streaming
				responseContent := cv.responseContent
				if cv.isStreaming {
					responseContent += " ▌"
				}
				content += "\n\n" + responseContent
			}
		}
	} else if cv.responseContent != "" {
		// Only response content (no thinking detected)
		content = cv.responseContent
		if cv.isStreaming {
			content += " ▌"
		}
	} else if cv.isStreamingThinking {
		// Currently streaming thinking content from the beginning
		thinkingRaw := cv.streamingContent
		// Remove any <think> tags that might be in the raw content
		thinkStartRegex := regexp.MustCompile(`(?i)<think(?:ing)?>`)
		thinkingRaw = thinkStartRegex.ReplaceAllString(thinkingRaw, "")
		content = "<think>" + strings.TrimSpace(thinkingRaw) + " ▌"
	} else {
		// Regular content without thinking
		content = cv.streamingContent
		if cv.isStreaming {
			content += " ▌"
		}
	}

	return chat.Message{
		Role:    chat.RoleAssistant,
		Content: content,
	}
}

// Streaming Methods

func (cv *ChatView) HandleStreamStart(streamID, model string) {
	log := logger.WithComponent("chat_view")
	log.Debug("Handling stream start in chat view", "stream_id", streamID, "model", model)

	// Initialize streaming state
	cv.isStreaming = true
	cv.currentStreamID = streamID
	cv.streamingContent = ""
	cv.isStreamingThinking = false
	cv.thinkingContent = ""
	cv.responseContent = ""

	// Initialize buffering state
	cv.contentBuffer = ""
	cv.contentTypeDetected = false
	cv.bufferSize = 0

	// Update status to show streaming
	cv.status = cv.status.WithStatus("Streaming response...")

	// Show spinner with streaming indicator
	cv.alert = cv.alert.WithSpinner(true, "Streaming...")
}

func (cv *ChatView) UpdateStreamingContent(streamID, content string, isComplete bool) {
	log := logger.WithComponent("chat_view")
	log.Debug("Updating streaming content in chat view",
		"stream_id", streamID,
		"content_length", len(content),
		"is_complete", isComplete,
		"content_type_detected", cv.contentTypeDetected,
		"buffer_size", cv.bufferSize)

	// Update basic streaming state
	cv.currentStreamID = streamID
	cv.streamingContent = content
	cv.isStreaming = !isComplete

	// Early detection buffering logic
	if !cv.contentTypeDetected && !isComplete {
		// Still buffering to detect content type
		cv.contentBuffer = content
		cv.bufferSize = len(content)

		// Try to detect content type from buffer
		if cv.detectContentTypeFromBuffer() {
			log.Debug("Content type detected",
				"is_thinking", cv.isStreamingThinking,
				"thinking_content", cv.thinkingContent,
				"response_content", cv.responseContent)
		} else {
			// Still need more content for detection, don't display anything yet
			log.Debug("Still buffering for content type detection", "buffer_size", cv.bufferSize)
			return
		}
	}

	// Content type already detected or stream is complete, process normally
	if cv.contentTypeDetected || isComplete {
		cv.processStreamingContent()

		// Update the message display to show streaming content with proper formatting
		cv.updateMessagesWithStreamingThinking()
	}

	if !isComplete {
		// Update spinner text based on current mode
		spinnerText := "Streaming..."
		if cv.isStreamingThinking {
			spinnerText = "Thinking..."
		}
		cv.alert = cv.alert.WithSpinner(true, spinnerText).NextSpinnerFrame()
	} else {
		// Clear streaming state when complete
		cv.isStreaming = false
		cv.streamingContent = ""
		cv.currentStreamID = ""
		cv.isStreamingThinking = false
		cv.thinkingContent = ""
		cv.responseContent = ""

		// Clear buffering state
		cv.contentBuffer = ""
		cv.contentTypeDetected = false
		cv.bufferSize = 0

		cv.alert = cv.alert.WithSpinner(false, "")
	}
}

func (cv *ChatView) HandleStreamComplete(streamID string, finalMessage chat.Message, totalChunks int, duration time.Duration) {
	log := logger.WithComponent("chat_view")
	log.Debug("Handling stream complete in chat view",
		"stream_id", streamID,
		"total_chunks", totalChunks,
		"duration", duration.String(),
		"final_message_length", len(finalMessage.Content))

	// DEBUG: Log the exact final message content
	log.Debug("Final message details",
		"role", finalMessage.Role,
		"content_length", len(finalMessage.Content),
		"content_preview", func() string {
			if len(finalMessage.Content) > 200 {
				return finalMessage.Content[:200] + "..."
			}
			return finalMessage.Content
		}(),
		"has_thinking_tags", strings.Contains(finalMessage.Content, "<think"),
		"has_response_after_thinking", strings.Contains(finalMessage.Content, "</think>"))

	// Clear streaming state
	cv.isStreaming = false
	cv.streamingContent = ""
	cv.currentStreamID = ""
	cv.isStreamingThinking = false
	cv.thinkingContent = ""
	cv.responseContent = ""

	// Clear buffering state
	cv.contentBuffer = ""
	cv.contentTypeDetected = false
	cv.bufferSize = 0

	// Hide spinner
	cv.alert = cv.alert.WithSpinner(false, "")

	// Update status
	cv.status = cv.status.WithStatus("Ready")

	// Update messages display with final content (no streaming)
	cv.updateMessages()
	cv.scrollToBottom()
}

func (cv *ChatView) HandleStreamError(streamID string, err error) {
	log := logger.WithComponent("chat_view")
	log.Error("Handling stream error in chat view", "stream_id", streamID, "error", err)

	// Clear streaming state
	cv.isStreaming = false
	cv.streamingContent = ""
	cv.currentStreamID = ""
	cv.isStreamingThinking = false
	cv.thinkingContent = ""
	cv.responseContent = ""

	// Clear buffering state
	cv.contentBuffer = ""
	cv.contentTypeDetected = false
	cv.bufferSize = 0

	// Hide spinner
	cv.alert = cv.alert.WithSpinner(false, "")

	// Update status with error
	cv.status = cv.status.WithStatus("Streaming failed: " + err.Error())

	// Update messages display to show error
	cv.updateMessages()
}

func (cv *ChatView) UpdateStreamProgress(streamID string, contentLength, chunkCount int, duration time.Duration) {
	log := logger.WithComponent("chat_view")
	log.Debug("Updating stream progress in chat view",
		"stream_id", streamID,
		"content_length", contentLength,
		"chunk_count", chunkCount,
		"duration", duration.String())

	// Update spinner with progress info for long streams
	if duration > 3*time.Second {
		progressText := fmt.Sprintf("Streaming... %d chars", contentLength)
		cv.alert = cv.alert.WithSpinner(true, progressText).NextSpinnerFrame()
	}
}

func (cv *ChatView) HandleModelChange(ev ModelChangeEvent) {
	log := logger.WithComponent("chat_view")
	log.Debug("Handling ModelChangeEvent in chat view", "model_name", ev.ModelName)

	// Update the status bar to show the new model
	cv.status = cv.status.WithModel(ev.ModelName)
	log.Debug("Updated chat view status bar with new model", "model_name", ev.ModelName)
}

// Node navigation and interaction methods

func (cv *ChatView) handleNodeNavigation(up bool) bool {
	// Only handle if using nodes
	if !cv.messages.UseNodes || cv.messages.NodeManager == nil {
		return false
	}

	log := logger.WithComponent("chat_view")

	var moved bool
	if up {
		moved = cv.messages.MoveFocusUp()
		log.Debug("Node navigation up", "moved", moved)
	} else {
		moved = cv.messages.MoveFocusDown()
		log.Debug("Node navigation down", "moved", moved)
	}

	if moved {
		// Post focus change event
		focusedNodeID := cv.messages.NodeManager.GetFocusedNode()
		cv.screen.PostEvent(NewMessageNodeFocusEvent(focusedNodeID))

		// Update status bar to show focused node
		cv.updateStatusForMode()

		// TODO: Auto-scroll to keep focused node visible
		cv.autoScrollToFocusedNode()
	}

	return moved
}

func (cv *ChatView) handleNodeToggleSelection() bool {
	// Only handle if using nodes
	if !cv.messages.UseNodes || cv.messages.NodeManager == nil {
		return false
	}

	focusedNodeID := cv.messages.NodeManager.GetFocusedNode()
	if focusedNodeID == "" {
		return false
	}

	log := logger.WithComponent("chat_view")
	log.Debug("Toggling selection for focused node", "node_id", focusedNodeID)

	if cv.messages.NodeManager.SelectNode(focusedNodeID) {
		// Get the new selection state
		isSelected := cv.messages.NodeManager.IsNodeSelected(focusedNodeID)
		cv.screen.PostEvent(NewMessageNodeSelectEvent(focusedNodeID, isSelected))
		return true
	}

	return false
}

func (cv *ChatView) handleNodeToggleExpansion() bool {
	// Only handle if using nodes
	if !cv.messages.UseNodes || cv.messages.NodeManager == nil {
		return false
	}

	focusedNodeID := cv.messages.NodeManager.GetFocusedNode()
	if focusedNodeID == "" {
		return false
	}

	log := logger.WithComponent("chat_view")
	log.Debug("Toggling expansion for focused node", "node_id", focusedNodeID)

	if cv.messages.NodeManager.ToggleNodeExpansion(focusedNodeID) {
		// Get the node to check its new state
		if node, exists := cv.messages.NodeManager.GetNode(focusedNodeID); exists {
			cv.screen.PostEvent(NewMessageNodeExpandEvent(focusedNodeID, node.State().Expanded))
		}
		return true
	}

	return false
}

func (cv *ChatView) handleSelectAllNodes() bool {
	// Only handle if using nodes
	if !cv.messages.UseNodes || cv.messages.NodeManager == nil {
		return false
	}

	log := logger.WithComponent("chat_view")
	log.Debug("Selecting all nodes")

	// Get all nodes and select them
	nodes := cv.messages.NodeManager.GetNodes()
	for _, node := range nodes {
		cv.messages.NodeManager.SetNodeSelected(node.ID(), true)
	}

	// Post selection events for all nodes
	for _, node := range nodes {
		cv.screen.PostEvent(NewMessageNodeSelectEvent(node.ID(), true))
	}

	return len(nodes) > 0
}

func (cv *ChatView) handleClearNodeSelection() bool {
	// Only handle if using nodes
	if !cv.messages.UseNodes || cv.messages.NodeManager == nil {
		return false
	}

	log := logger.WithComponent("chat_view")
	log.Debug("Clearing all node selections")

	selectedNodes := cv.messages.GetSelectedNodes()
	cv.messages.ClearSelection()

	// Post deselection events
	for _, nodeID := range selectedNodes {
		cv.screen.PostEvent(NewMessageNodeSelectEvent(nodeID, false))
	}

	return len(selectedNodes) > 0
}

func (cv *ChatView) autoScrollToFocusedNode() {
	// TODO: Implement auto-scrolling to keep focused node visible
	// This would involve calculating the focused node's position and
	// adjusting the scroll offset if needed
	log := logger.WithComponent("chat_view")
	log.Debug("Auto-scroll to focused node requested (not yet implemented)")
}
