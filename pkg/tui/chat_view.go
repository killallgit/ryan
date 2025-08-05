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
	controller       ControllerInterface
	input            InputField
	messages         MessageDisplay
	status           StatusBar
	layout           Layout
	screen           tcell.Screen
	alert            AlertDisplay     // Legacy field for compatibility
	statusRow        StatusRowDisplay // New enhanced status row
	downloadModal    DownloadPromptModal
	progressModal    ProgressModal
	helpModal        HelpModal
	downloadCtx      context.Context
	downloadCancel   context.CancelFunc
	pendingMessage   string
	modelsController *controllers.ModelsController

	// Interaction mode
	mode ChatViewMode // Current interaction mode

	// Context tree visualization
	// contextTreeView removed - now handled as standalone view in command palette

	// Streaming state
	isStreaming         bool
	streamingContent    string
	currentStreamID     string
	isStreamingThinking bool   // Track if currently streaming thinking content
	thinkingContent     string // Accumulate thinking content separately
	responseContent     string // Accumulate response content separately

	// Stream parser for formatted content
	streamParser     *StreamParser // Parser for handling think blocks and formatting
	lastParsedLength int           // Track how much content we've already parsed
	
	// Streaming renderer for incremental updates
	streamingRenderer *StreamingRenderer
}

func NewChatView(controller ControllerInterface, modelsController *controllers.ModelsController, screen tcell.Screen) *ChatView {
	width, height := screen.Size()

	// Context tree will be initialized on demand
	// We'll get it from the conversation when needed

	view := &ChatView{
		controller:       controller,
		input:            NewInputField(width),
		messages:         NewMessageDisplay(width, height-5), // -5 for status, input, and alert areas
		status:           NewStatusBar(width).WithModel(controller.GetModel()).WithStatus("Ready").WithModelAvailability(true),
		layout:           NewLayout(width, height),
		screen:           screen,
		alert:            NewAlertDisplay(width),
		statusRow:        NewStatusRowDisplay(width),
		downloadModal:    NewDownloadPromptModal(),
		progressModal:    NewProgressModal(),
		helpModal:        NewHelpModal(),
		downloadCtx:      nil,
		downloadCancel:   nil,
		pendingMessage:   "",
		modelsController: modelsController,

		// Initialize interaction mode
		mode: ModeInput, // Start in input mode

		// Initialize context tree view
		// contextTreeView removed - now handled as standalone view

		// Initialize streaming state
		isStreaming:         false,
		streamingContent:    "",
		currentStreamID:     "",
		isStreamingThinking: false,
		thinkingContent:     "",
		responseContent:     "",

		// Initialize stream parser
		streamParser: NewStreamParser(),
		
		// Initialize streaming renderer
		streamingRenderer: NewStreamingRenderer(),
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
	
	// Check if this is a streaming update that can use incremental rendering
	isStreamingUpdate := cv.isStreaming && cv.streamingRenderer != nil
	
	if isStreamingUpdate {
		// Use incremental rendering for streaming content
		RenderMessagesWithIncrementalStreaming(
			screen, 
			cv.messages, 
			messageArea, 
			spinner, 
			cv.isStreamingThinking,
			cv.streamingRenderer,
			true, // This is a streaming update
		)
	} else {
		// Full render for non-streaming updates
		RenderMessagesWithStreamingState(screen, cv.messages, messageArea, spinner, cv.isStreamingThinking)
		// Reset streaming renderer when not streaming
		if cv.streamingRenderer != nil && !cv.isStreaming {
			cv.streamingRenderer.Reset()
		}
	}

	// Update status row with current token count and render it
	// Get the most current token count from both status and controller
	statusTokens := cv.status.PromptTokens + cv.status.ResponseTokens
	promptTokens, responseTokens := cv.controller.GetTokenUsage()
	controllerTokens := promptTokens + responseTokens

	// Use whichever is higher (more recent)
	totalTokens := statusTokens
	if controllerTokens > statusTokens {
		totalTokens = controllerTokens
	}

	cv.statusRow = cv.statusRow.WithTokens(totalTokens).UpdateDuration()
	RenderStatusRow(screen, alertArea, cv.statusRow)

	RenderInput(screen, cv.input, inputArea)
	RenderStatus(screen, cv.status, statusArea)

	// Render context tree view if visible
	// Context tree view now rendered as standalone view via command palette

	// Render modals on top
	cv.downloadModal.Render(screen, area)
	cv.progressModal.Render(screen, area)
	cv.helpModal.Render(screen, area)
}

func (cv *ChatView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool {
	// Context tree keys now handled by standalone view via command palette

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
	// F2 key to show help modal
	if ev.Key() == tcell.KeyF2 {
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
		modeText = "✏️ Input | Ctrl+N=node mode, F2=help"
	}
	cv.status = cv.status.WithStatus(modeText)
}

func (cv *ChatView) handleInputModeKeys(ev *tcell.EventKey, sending bool) bool {
	switch ev.Key() {
	case tcell.KeyEnter:
		if !sending {
			content := cv.sendMessage()
			if content != "" {
				// Clean thinking blocks from all assistant messages when user sends a new message
				cv.controller.CleanThinkingBlocks()
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

	// Note: Ctrl+T was moved to command palette. Context tree is now accessed via Ctrl+P -> Context Tree

	case tcell.KeyCtrlB:
		// Ctrl+B to branch from current message
		if cv.handleBranchContext() {
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
			case 't', 'T':
				// Alt+T to toggle context tree
				if ev.Modifiers()&tcell.ModAlt != 0 {
					// Context tree view now accessed via command palette (Ctrl+P)
					return true
				}
			}

			cv.input = cv.input.InsertRune(ev.Rune())
			return true
		}
	}

	return false
}

func (cv *ChatView) handleNodeModeKeys(ev *tcell.EventKey, _ bool) bool {
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

	// Update context tree view if it exists
	// Context tree view sizing now handled by standalone view
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
		cv.statusRow = cv.statusRow.WithSpinner(true, "Sending...")
	} else {
		// Always clear alert since errors only show in chat messages now
		cv.alert = cv.alert.Clear()
		cv.statusRow = cv.statusRow.ClearSpinnerOnly() // Preserve token count
	}
}

func (cv *ChatView) UpdateSpinnerFrame() {
	cv.alert = cv.alert.NextSpinnerFrame()
	cv.statusRow = cv.statusRow.NextSpinnerFrame()
}

func (cv *ChatView) updateMessages() {
	history := cv.controller.GetHistory()
	// Filter out system messages - they should not be displayed to the user
	var filteredHistory []chat.Message
	for _, msg := range history {
		if msg.Role != chat.RoleSystem {
			filteredHistory = append(filteredHistory, msg)
		}
	}
	cv.messages = cv.messages.WithMessages(filteredHistory)

	// Update context tree if it exists
	// Context tree view updates now handled by standalone view
}

func (cv *ChatView) updateMessagesWithStreamingThinking() {
	history := cv.controller.GetHistory()
	// Filter out system messages - they should not be displayed to the user
	var filteredHistory []chat.Message
	for _, msg := range history {
		if msg.Role != chat.RoleSystem {
			filteredHistory = append(filteredHistory, msg)
		}
	}
	history = filteredHistory

	// If we're streaming, show streaming content
	if cv.isStreaming {
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

	// Add a buffer of 3-4 lines when spinner is visible to prevent crowding
	bufferLines := 0
	if cv.alert.IsSpinnerVisible || cv.statusRow.IsSpinnerVisible || cv.isStreaming {
		bufferLines = 4 // Keep 4 lines of buffer above spinner
	}

	// Calculate the effective visible height (minus buffer)
	effectiveHeight := paddedHeight - bufferLines
	if effectiveHeight < 1 {
		effectiveHeight = 1 // Ensure we always have at least 1 line visible
	}

	if totalLines > effectiveHeight {
		cv.messages = cv.messages.WithScroll(totalLines - effectiveHeight)
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
		var lastProgress float64 = 0.0
		err := cv.modelsController.PullWithProgress(cv.downloadCtx, modelName, func(status string, completed, total int64) {
			// Calculate progress
			progress := 0.0
			if total > 0 {
				progress = float64(completed) / float64(total)
			}

			// Smooth out progress updates - only update if progress is actually advancing
			// This prevents the modal from jumping back to 0% during different download phases
			if progress > lastProgress || status == "success" {
				lastProgress = progress
				// Post progress event
				cv.screen.PostEvent(NewModelDownloadProgressEvent(modelName, status, progress))
			}
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

	// Force screen refresh to update UI immediately (hide modal)
	cv.screen.Show()

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

	// Force screen refresh to update UI immediately (hide modal)
	cv.screen.Show()
}

// Streaming Helper Methods

// detectThinkingStart checks if content begins with <think> or <thinking> tags
func (cv *ChatView) detectThinkingStart(content string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(content))
	return strings.HasPrefix(trimmed, "<think>") || strings.HasPrefix(trimmed, "<thinking>")
}

// processStreamingContent processes the full streaming content and separates thinking from response
func (cv *ChatView) processStreamingContent() {
	// Extract only the new chunk since last parse
	newChunk := ""
	if len(cv.streamingContent) > cv.lastParsedLength {
		newChunk = cv.streamingContent[cv.lastParsedLength:]
		cv.lastParsedLength = len(cv.streamingContent)
	}

	// Parse only the new chunk
	if newChunk != "" {
		segments := cv.streamParser.ParseChunk(newChunk)

		// Process segments to update thinking/response state
		for _, segment := range segments {
			// The parser handles the formatting, we just need to track state
			if segment.Format == FormatTypeThink {
				cv.isStreamingThinking = true
			}
		}
	}

	// Update streaming thinking state
	cv.isStreamingThinking = cv.streamParser.IsInThinkBlock()

	// Now reconstruct the full content with proper separation
	// We need to re-parse the entire content to get the proper separation
	cv.streamParser.Reset()
	cv.lastParsedLength = 0

	segments := cv.streamParser.ParseChunk(cv.streamingContent)
	cv.lastParsedLength = len(cv.streamingContent)

	// Reset thinking and response content
	cv.thinkingContent = ""
	cv.responseContent = ""

	// Accumulate content based on segment types
	var thinkingBuilder strings.Builder
	var responseBuilder strings.Builder
	var inThinkContent bool

	for _, segment := range segments {
		// Skip tag content (the actual <think> tags)
		if segment.Content == "<think>" || segment.Content == "<thinking>" ||
			segment.Content == "</think>" || segment.Content == "</thinking>" {
			if strings.HasPrefix(segment.Content, "<") && !strings.HasPrefix(segment.Content, "</") {
				inThinkContent = true
			} else if strings.HasPrefix(segment.Content, "</") {
				inThinkContent = false
			}
			continue
		}

		// Accumulate content based on format type
		if segment.Format == FormatTypeThink || inThinkContent {
			thinkingBuilder.WriteString(segment.Content)
		} else {
			responseBuilder.WriteString(segment.Content)
		}
	}

	cv.thinkingContent = strings.TrimSpace(thinkingBuilder.String())
	cv.responseContent = strings.TrimSpace(responseBuilder.String())
}

// createStreamingMessage creates a properly formatted message for streaming display
func (cv *ChatView) createStreamingMessage() chat.Message {
	var content string

	if cv.thinkingContent != "" {
		// Format thinking content with proper tags so ParseThinkingBlock can style it correctly
		thinkingWithTags := "<think>" + cv.thinkingContent

		if cv.isStreamingThinking {
			// Still streaming thinking content, add cursor and close tag for proper formatting
			content = thinkingWithTags + " ▌</think>"
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
		content = "<think>" + strings.TrimSpace(thinkingRaw) + " ▌</think>"
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

	// Reset stream parser for new stream
	cv.streamParser.Reset()
	cv.lastParsedLength = 0
	
	// Reset streaming renderer for new stream
	if cv.streamingRenderer != nil {
		cv.streamingRenderer.Reset()
	}

	// Update status to show streaming
	cv.status = cv.status.WithStatus("Streaming response...")

	// Initialize status row with current token count and streaming spinner
	promptTokens, responseTokens := cv.controller.GetTokenUsage()
	totalTokens := promptTokens + responseTokens
	cv.statusRow = cv.statusRow.WithSpinner(true, "Streaming...").WithTokens(totalTokens)

	// Show alert spinner
	cv.alert = cv.alert.WithSpinner(true, "Streaming...")
}

func (cv *ChatView) UpdateStreamingContent(streamID, content string, isComplete bool) {
	log := logger.WithComponent("chat_view")
	log.Debug("Updating streaming content in chat view",
		"stream_id", streamID,
		"content_length", len(content),
		"is_complete", isComplete)

	// Update basic streaming state
	cv.currentStreamID = streamID
	cv.streamingContent = content
	cv.isStreaming = !isComplete

	// Process content immediately without buffering delay
	cv.processStreamingContent()

	// Update the message display to show streaming content with proper formatting
	cv.updateMessagesWithStreamingThinking()

	if !isComplete {
		// Update spinner text based on current mode
		spinnerText := "Streaming..."
		if cv.isStreamingThinking {
			spinnerText = "Thinking..."
		}
		cv.alert = cv.alert.WithSpinner(true, spinnerText).NextSpinnerFrame()
		cv.statusRow = cv.statusRow.WithSpinner(true, spinnerText).NextSpinnerFrame()
	} else {
		// Clear streaming state when complete
		cv.isStreaming = false
		cv.streamingContent = ""
		cv.currentStreamID = ""
		cv.isStreamingThinking = false
		cv.thinkingContent = ""
		cv.responseContent = ""

		// Reset stream parser for next message
		cv.streamParser.Reset()
		cv.lastParsedLength = 0

		cv.alert = cv.alert.WithSpinner(false, "")
		cv.statusRow = cv.statusRow.ClearSpinnerOnly() // Preserve token count
	}
}

func (cv *ChatView) HandleStreamComplete(streamID string, finalMessage chat.Message, totalChunks int, duration time.Duration) {
	log := logger.WithComponent("chat_view")
	log.Debug("Handling stream complete in chat view",
		"stream_id", streamID,
		"total_chunks", totalChunks,
		"duration", duration.String(),
		"final_message_length", len(finalMessage.Content))

	// Clear streaming state
	cv.isStreaming = false
	cv.streamingContent = ""
	cv.currentStreamID = ""
	cv.isStreamingThinking = false
	cv.thinkingContent = ""
	cv.responseContent = ""

	// Hide spinner
	cv.alert = cv.alert.WithSpinner(false, "")
	cv.statusRow = cv.statusRow.ClearSpinnerOnly() // Preserve token count

	// Update status
	cv.status = cv.status.WithStatus("Ready")

	// Update token information
	promptTokens, responseTokens := cv.controller.GetTokenUsage()
	cv.status = cv.status.WithTokens(promptTokens, responseTokens)
	// Note: Token counts are currently 0 due to LangChain Go not exposing usage info

	// Save history with original thinking blocks before cleaning for display
	if saver, ok := cv.controller.(interface{ SaveHistoryToDisk() error }); ok {
		if err := saver.SaveHistoryToDisk(); err != nil {
			// Don't fail the UI update, just log the error
			// TODO: Add proper logging here
		}
	}

	// Clean thinking blocks from the final assistant response for display only
	cv.controller.CleanThinkingBlocks()

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

	// Hide spinner
	cv.alert = cv.alert.WithSpinner(false, "")
	cv.statusRow = cv.statusRow.ClearSpinnerOnly() // Preserve token count

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
		cv.statusRow = cv.statusRow.WithSpinner(true, progressText).WithDuration(duration).NextSpinnerFrame()
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

// Context tree methods

// initializeContextTreeView method removed - context tree is now a standalone view

func (cv *ChatView) handleBranchContext() bool {
	log := logger.WithComponent("chat_view")

	// Check if we have a focused message node to branch from
	if !cv.messages.UseNodes || cv.messages.NodeManager == nil {
		log.Debug("Cannot branch - not using nodes")
		return false
	}

	focusedNodeID := cv.messages.NodeManager.GetFocusedNode()
	if focusedNodeID == "" {
		log.Debug("Cannot branch - no focused node")
		return false
	}

	// Get the message associated with this node
	node, exists := cv.messages.NodeManager.GetNode(focusedNodeID)
	if !exists {
		log.Debug("Cannot branch - node not found", "node_id", focusedNodeID)
		return false
	}

	// Extract message ID from node
	// Node IDs are typically in format "msg-<messageID>"
	messageID := strings.TrimPrefix(node.ID(), "msg-")

	log.Debug("Branching from message", "message_id", messageID, "node_id", focusedNodeID)

	// Post a branch context event that the controller can handle
	cv.screen.PostEvent(NewBranchContextEvent(messageID))

	// Update status to show branching
	cv.status = cv.status.WithStatus(fmt.Sprintf("Branching from message %s...", messageID[:8]))

	return true
}

// handleContextTreeKeys method removed - context tree is now a standalone view
