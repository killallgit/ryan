package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

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
	activityView *tview.TextView // New component for activity tree
	spinnerView  *tview.TextView // Spinner/status view above input

	// State
	controller    ControllerInterface
	app           *tview.Application
	sending       bool
	streaming     bool
	streamID      string
	streamBuffer  string
	activityTree  string // Current activity tree text
	spinnerFrame  int
	spinnerFrames []string
	renderManager *RenderManager // Add render manager
	currentState  string         // Current UI state: idle, sending, thinking, streaming, executing, preparing_tools

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

	// Set background color for the entire view
	cv.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create message display
	cv.messages = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true)
	cv.messages.SetBorder(false)
	cv.messages.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create input field with prompt inside
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

	// Create activity indicator view
	cv.activityView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false).
		SetWordWrap(false).
		SetScrollable(false)
	cv.activityView.SetBackgroundColor(tcell.GetColor(ColorBase00))
	cv.activityView.SetTextColor(tcell.GetColor(ColorBase04))

	// Create spinner/status view above input
	cv.spinnerView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false).
		SetWordWrap(false).
		SetScrollable(false)
	cv.spinnerView.SetBackgroundColor(tcell.GetColor(ColorBase00))
	cv.spinnerView.SetTextColor(tcell.GetColor(ColorBase04))

	// Create status bar
	cv.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	cv.status.SetBackgroundColor(tcell.GetColor(ColorBase00))
	cv.updateStatus()

	// Create padded message area with inner padding
	messageContainer := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).         // Left padding
		AddItem(cv.messages, 0, 1, false). // Messages content
		AddItem(nil, 2, 0, false)          // Right padding
	messageContainer.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create padded activity area
	activityContainer := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).             // Left padding
		AddItem(cv.activityView, 0, 1, false). // Activity content
		AddItem(nil, 2, 0, false)              // Right padding
	activityContainer.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create padded spinner area
	spinnerContainer := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).            // Left padding
		AddItem(cv.spinnerView, 0, 1, false). // Spinner content
		AddItem(nil, 2, 0, false)             // Right padding
	spinnerContainer.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create padded input area
	inputContainer := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).     // Left padding
		AddItem(cv.input, 0, 1, true). // Input content
		AddItem(nil, 2, 0, false)      // Right padding
	inputContainer.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Layout: top padding, messages, activity indicator, spinner, input, status
	cv.AddItem(nil, 1, 0, false). // Top padding
					AddItem(messageContainer, 0, 1, false).  // Messages take most space
					AddItem(activityContainer, 0, 0, false). // Activity indicator (dynamic height)
					AddItem(spinnerContainer, 1, 0, false).  // Spinner/status above input
					AddItem(inputContainer, 2, 0, true).     // Input area with more height
					AddItem(cv.status, 1, 0, false)          // Status bar

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
	cv.streaming = true
	cv.streamID = streamID
	cv.streamBuffer = ""
	cv.updateStatus()
	cv.startSpinner()
}

// UpdateStreamingContent updates the streaming message content
func (cv *ChatView) UpdateStreamingContent(streamID string, content string) {
	if cv.streamID == streamID {
		// Check for tool mode marker
		if content == "<<<TOOL_MODE>>>" {
			cv.currentState = "preparing_tools"
			cv.updateSpinnerView()
			return // Don't add marker to buffer
		}

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

	// Start or stop spinner animation
	if sending {
		cv.currentState = "sending"
		cv.startSpinner()
	} else {
		cv.currentState = "idle"
	}
	cv.updateSpinnerView()
	cv.updateStatus()
}

// updateStatus updates the status bar
func (cv *ChatView) updateStatus() {
	// Model info (right-aligned)
	model := cv.controller.GetModel()
	statusText := ""

	// Add status indicators
	if cv.sending {
		spinner := cv.spinnerFrames[cv.spinnerFrame]
		statusText = fmt.Sprintf("[#93b56b]%s Sending...[-] ", spinner)
	} else if cv.streaming {
		spinner := cv.spinnerFrames[cv.spinnerFrame]
		statusText = fmt.Sprintf("[#6b93b5]%s Streaming...[-] ", spinner)
	}

	statusText += fmt.Sprintf("[#f5b761]%s[-]", model)

	cv.status.SetTextAlign(tview.AlignRight)
	cv.status.SetText(statusText)
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

// UpdateActivityTree updates the activity tree display
func (cv *ChatView) UpdateActivityTree(treeText string) {
	cv.activityTree = treeText

	// Update the activity view
	cv.activityView.Clear()
	if treeText != "" {
		// Apply color formatting to the tree text
		formattedTree := cv.formatActivityTree(treeText)
		cv.activityView.SetText(formattedTree)
		// Dynamically resize the activity container based on content
		lines := strings.Count(treeText, "\n") + 1
		if lines > 5 {
			lines = 5 // Cap at 5 lines max
		}
		// The activity container is the 3rd item (index 2) in the main flex
		// 0: top padding, 1: messages, 2: activity, 3: gap, 4: input, 5: status
		if cv.GetItemCount() > 2 {
			cv.ResizeItem(cv.GetItem(2), lines, 0)
		}
	} else {
		// Hide when no activity by setting height to 0
		if cv.GetItemCount() > 2 {
			cv.ResizeItem(cv.GetItem(2), 0, 0)
		}
	}
}

// ClearActivityTree clears the activity tree display
func (cv *ChatView) ClearActivityTree() {
	cv.UpdateActivityTree("")
}

// startSpinner starts the spinner animation
func (cv *ChatView) startSpinner() {
	go func() {
		for cv.sending || cv.streaming || cv.currentState != "idle" {
			time.Sleep(100 * time.Millisecond)
			cv.spinnerFrame = (cv.spinnerFrame + 1) % len(cv.spinnerFrames)
			cv.app.QueueUpdateDraw(func() {
				cv.updateSpinnerView()
				cv.updateStatus()
			})
		}
	}()
}

// formatActivityTree applies color formatting to the activity tree text
func (cv *ChatView) formatActivityTree(tree string) string {
	if tree == "" {
		return ""
	}

	// Apply basic coloring
	lines := strings.Split(tree, "\n")
	for i, line := range lines {
		// Color the agent names
		if strings.Contains(line, "Assistant") {
			lines[i] = strings.Replace(line, "Assistant", "[#6b93b5]Assistant[-]", 1)
		}
		if strings.Contains(line, "ChatController") {
			lines[i] = strings.Replace(line, "ChatController", "[#93b56b]ChatController[-]", 1)
		}

		// Color the status indicators
		lines[i] = strings.Replace(lines[i], "●", "[#93b56b]●[-]", -1) // Active - green
		lines[i] = strings.Replace(lines[i], "○", "[#f5b761]○[-]", -1) // Pending - yellow
		lines[i] = strings.Replace(lines[i], "✗", "[#d95f5f]✗[-]", -1) // Error - red
		lines[i] = strings.Replace(lines[i], "✓", "[#93b56b]✓[-]", -1) // Complete - green

		// Color the tree structure
		lines[i] = strings.Replace(lines[i], "├──", "[#5c5044]├──[-]", -1)
		lines[i] = strings.Replace(lines[i], "└──", "[#5c5044]└──[-]", -1)
		lines[i] = strings.Replace(lines[i], "│", "[#5c5044]│[-]", -1)
	}

	return strings.Join(lines, "\n")
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

// updateSpinnerView updates the spinner/status view above the input
func (cv *ChatView) updateSpinnerView() {
	if cv.currentState == "idle" {
		cv.spinnerView.SetText("")
		return
	}

	spinner := cv.spinnerFrames[cv.spinnerFrame]
	var statusText string

	switch cv.currentState {
	case "sending":
		statusText = fmt.Sprintf("[#93b56b]%s[-] [#f5b761]Sending message...[-]", spinner)
	case "thinking":
		statusText = fmt.Sprintf("[#976bb5]%s[-] [#976bb5]Thinking...[-]", spinner)
	case "streaming":
		statusText = fmt.Sprintf("[#6b93b5]%s[-] [#6b93b5]Streaming response...[-]", spinner)
	case "preparing_tools":
		statusText = fmt.Sprintf("[#f5b761]%s[-] [#f5b761]Using tool agent (non-streaming mode)...[-]", spinner)
	case "executing":
		toolName := cv.streamID
		if toolName != "" {
			statusText = fmt.Sprintf("[#d95f5f]%s[-] [#d95f5f]Executing %s...[-]", spinner, toolName)
		} else {
			statusText = fmt.Sprintf("[#d95f5f]%s[-] [#d95f5f]Executing tool...[-]", spinner)
		}
	default:
		statusText = ""
	}

	cv.spinnerView.SetText(statusText)
}
