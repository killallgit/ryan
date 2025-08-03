package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
)

// CalculateMessageLines calculates the total number of lines needed to render messages
// using the same logic as RenderMessagesWithSpinnerAndStreaming
func CalculateMessageLines(messages []chat.Message, chatWidth int, streamingThinking bool) int {
	// Build list of message lines with their associated roles for styling
	type MessageLine struct {
		Text       string
		Role       string
		IsThinking bool
		Style      *tcell.Style // Optional specific style for this line
	}

	var allLines []MessageLine
	// Get showThinking from config or use default
	showThinking := true
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Config not initialized, use default
			}
		}()
		if cfg := config.Get(); cfg != nil {
			showThinking = cfg.ShowThinking
		}
	}()

	for i, msg := range messages {
		isLastMessage := i == len(messages)-1

		if msg.Role == chat.RoleAssistant {
			// Check if this is a streaming thinking message (last message + streaming thinking mode)
			if isLastMessage && streamingThinking && showThinking {
				// Handle streaming thinking content - apply thinking styling directly
				thinkingText := "Thinking: " + msg.Content
				thinkingLines := WrapText(thinkingText, chatWidth)
				for _, line := range thinkingLines {
					allLines = append(allLines, MessageLine{
						Text:       line,
						Role:       msg.Role,
						IsThinking: true, // Force thinking styling
						Style:      nil,
					})
				}
			} else {
				// Regular assistant message - use normal ParseThinkingBlock logic
				parsed := ParseThinkingBlock(msg.Content)

				if parsed.HasThinking && showThinking {
					// Add "Thinking: " prefix and format thinking block
					var thinkingText string
					if parsed.ResponseContent != "" {
						// Response is complete, truncate thinking to 3 lines
						thinkingText = "Thinking: " + TruncateThinkingBlock(parsed.ThinkingBlock, 3, chatWidth-10)
					} else {
						// Response not complete, show full thinking block
						thinkingText = "Thinking: " + parsed.ThinkingBlock
					}

					thinkingLines := WrapText(thinkingText, chatWidth)
					for _, line := range thinkingLines {
						allLines = append(allLines, MessageLine{
							Text:       line,
							Role:       msg.Role,
							IsThinking: true,
							Style:      nil,
						})
					}

					// Add separator line between thinking and response
					if parsed.ResponseContent != "" {
						allLines = append(allLines, MessageLine{
							Text:       "",
							Role:       "",
							IsThinking: false,
							Style:      nil,
						})
					}
				}

				// Add response content if present
				var contentToRender string
				if parsed.HasThinking && showThinking {
					contentToRender = parsed.ResponseContent
				} else {
					contentToRender = msg.Content
				}

				if contentToRender != "" {
					// Check if we should use simple formatting for line calculation
					contentTypes := DetectContentTypes(contentToRender)
					if ShouldUseSimpleFormatting(contentTypes) {
						// Use simple formatting to calculate lines
						formatter := NewSimpleFormatter(chatWidth)
						segments := ParseContentSegments(contentToRender)
						formattedLines := formatter.FormatContentSegments(segments)

						for _, formattedLine := range formattedLines {
							// Apply indentation
							content := formattedLine.Content
							if formattedLine.Indent > 0 {
								content = strings.Repeat(" ", formattedLine.Indent) + content
							}

							allLines = append(allLines, MessageLine{
								Text:       content,
								Role:       msg.Role,
								IsThinking: false,
								Style:      &formattedLine.Style,
							})
						}
					} else {
						// Use traditional text wrapping
						contentLines := WrapText(contentToRender, chatWidth)
						for _, line := range contentLines {
							allLines = append(allLines, MessageLine{
								Text:       line,
								Role:       msg.Role,
								IsThinking: false,
								Style:      nil,
							})
						}
					}
				}
			}
		} else {
			// Handle non-assistant messages normally
			contentLines := WrapText(msg.Content, chatWidth)
			for _, line := range contentLines {
				allLines = append(allLines, MessageLine{
					Text:       line,
					Role:       msg.Role,
					IsThinking: false,
				})
			}
		}

		// Add empty line between messages
		allLines = append(allLines, MessageLine{
			Text:       "",
			Role:       "",
			IsThinking: false,
		})
	}

	// Remove trailing empty line
	if len(allLines) > 0 {
		allLines = allLines[:len(allLines)-1]
	}

	return len(allLines)
}

func RenderMessages(screen tcell.Screen, display MessageDisplay, area Rect) {
	RenderMessagesWithSpinner(screen, display, area, SpinnerComponent{})
}

func RenderMessagesWithStreamingState(screen tcell.Screen, display MessageDisplay, area Rect, spinner SpinnerComponent, streamingThinking bool) {
	RenderMessagesWithSpinnerAndStreaming(screen, display, area, spinner, streamingThinking)
}

func RenderMessagesWithSpinnerAndStreaming(screen tcell.Screen, display MessageDisplay, area Rect, spinner SpinnerComponent, streamingThinking bool) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	// Use the area as-is since layout already provides padding
	chatArea := area

	// Build list of message lines with their associated roles for styling
	type MessageLine struct {
		Text       string
		Role       string
		IsThinking bool
		Style      *tcell.Style // Optional specific style for this line
	}

	var allLines []MessageLine
	// Get showThinking from config or use default
	showThinking := true
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Config not initialized, use default
			}
		}()
		if cfg := config.Get(); cfg != nil {
			showThinking = cfg.ShowThinking
		}
	}()

	for i, msg := range display.Messages {
		isLastMessage := i == len(display.Messages)-1

		if msg.Role == chat.RoleAssistant {
			// DEBUG: Log what assistant message is being rendered
			log := logger.WithComponent("render")
			log.Debug("Rendering assistant message",
				"is_last_message", isLastMessage,
				"streaming_thinking", streamingThinking,
				"message_length", len(msg.Content),
				"message_preview", func() string {
					if len(msg.Content) > 100 {
						return msg.Content[:100] + "..."
					}
					return msg.Content
				}())

			// Check if this is a streaming thinking message (last message + streaming thinking mode)
			if isLastMessage && streamingThinking && showThinking {
				// Handle streaming thinking content - apply thinking styling directly
				thinkingText := "Thinking: " + msg.Content
				thinkingLines := WrapText(thinkingText, chatArea.Width)
				for _, line := range thinkingLines {
					allLines = append(allLines, MessageLine{
						Text:       line,
						Role:       msg.Role,
						IsThinking: true, // Force thinking styling
						Style:      nil,
					})
				}
				log.Debug("Rendered streaming thinking message", "lines_added", len(thinkingLines))
			} else {
				// Regular assistant message - use normal ParseThinkingBlock logic
				parsed := ParseThinkingBlock(msg.Content)

				if parsed.HasThinking && showThinking {
					// Add "Thinking: " prefix and format thinking block
					var thinkingText string
					if parsed.ResponseContent != "" {
						// Response is complete, truncate thinking to 3 lines
						thinkingText = "Thinking: " + TruncateThinkingBlock(parsed.ThinkingBlock, 3, chatArea.Width-10)
					} else {
						// Response not complete, show full thinking block
						thinkingText = "Thinking: " + parsed.ThinkingBlock
					}

					thinkingLines := WrapText(thinkingText, chatArea.Width)
					for _, line := range thinkingLines {
						allLines = append(allLines, MessageLine{
							Text:       line,
							Role:       msg.Role,
							IsThinking: true,
							Style:      nil,
						})
					}

					// Add separator line between thinking and response
					if parsed.ResponseContent != "" {
						allLines = append(allLines, MessageLine{
							Text:       "",
							Role:       "",
							IsThinking: false,
							Style:      nil,
						})
					}
				}

				// Add response content if present
				var contentToRender string
				if parsed.HasThinking && showThinking {
					contentToRender = parsed.ResponseContent
				} else {
					contentToRender = msg.Content
				}

				if contentToRender != "" {
					// Check if we should use simple formatting
					contentTypes := DetectContentTypes(contentToRender)
					if ShouldUseSimpleFormatting(contentTypes) {
						// Use simple formatting for complex content
						formatter := NewSimpleFormatter(chatArea.Width)
						segments := ParseContentSegments(contentToRender)
						formattedLines := formatter.FormatContentSegments(segments)

						for _, formattedLine := range formattedLines {
							// Apply indentation
							content := formattedLine.Content
							if formattedLine.Indent > 0 {
								content = strings.Repeat(" ", formattedLine.Indent) + content
							}

							allLines = append(allLines, MessageLine{
								Text:       content,
								Role:       msg.Role,
								IsThinking: false,
								Style:      &formattedLine.Style, // Store the specific style
							})
						}
					} else {
						// Use traditional text wrapping for simple content
						contentLines := WrapText(contentToRender, chatArea.Width)
						log := logger.WithComponent("render")

						firstLine := ""
						if len(contentLines) > 0 {
							firstLine = contentLines[0]
						}
						log.Debug("Rendering assistant message lines",
							"role", msg.Role,
							"content_length", len(contentToRender),
							"width", chatArea.Width,
							"lines", len(contentLines),
							"first_line", firstLine,
							"had_thinking", parsed.HasThinking,
							"content_preview", func() string {
								if len(contentToRender) > 100 {
									return contentToRender[:100] + "..."
								}
								return contentToRender
							}())

						for _, line := range contentLines {
							allLines = append(allLines, MessageLine{
								Text:       line,
								Role:       msg.Role,
								IsThinking: false,
								Style:      nil,
							})
						}
					}
				}
			}
		} else {
			// Handle non-assistant messages normally
			contentLines := WrapText(msg.Content, chatArea.Width)
			log := logger.WithComponent("render")

			firstLine := ""
			if len(contentLines) > 0 {
				firstLine = contentLines[0]
			}
			log.Debug("Rendering message lines",
				"role", msg.Role,
				"content_length", len(msg.Content),
				"width", chatArea.Width,
				"lines", len(contentLines),
				"first_line", firstLine)

			for _, line := range contentLines {
				allLines = append(allLines, MessageLine{
					Text:       line,
					Role:       msg.Role,
					IsThinking: false,
				})
			}
		}

		// Add empty line between messages
		allLines = append(allLines, MessageLine{
			Text:       "",
			Role:       "",
			IsThinking: false,
		})
	}

	// Remove trailing empty line
	if len(allLines) > 0 {
		allLines = allLines[:len(allLines)-1]
	}

	// Calculate visible lines for messages 
	// Note: Spinner is now handled by RenderStatusRow, no need to reserve space here
	availableHeight := chatArea.Height

	// Calculate which lines to show based on scroll
	startLine := display.Scroll
	if startLine < 0 {
		startLine = 0
	}
	if startLine > len(allLines) {
		startLine = len(allLines)
	}

	endLine := startLine + availableHeight
	if endLine > len(allLines) {
		endLine = len(allLines)
	}

	// Ensure valid slice bounds
	if startLine > endLine {
		startLine = endLine
	}

	visibleLines := allLines[startLine:endLine]

	clearArea(screen, area)

	// Render message lines with appropriate styling in the padded area
	for i, msgLine := range visibleLines {
		if i >= availableHeight {
			break
		}

		// Determine style based on custom style or message role and thinking status
		var style tcell.Style
		if msgLine.Style != nil {
			// Use custom style if available
			style = *msgLine.Style
		} else if msgLine.IsThinking {
			// Dimmed italic style for thinking blocks
			style = StyleThinkingText
		} else {
			switch msgLine.Role {
			case chat.RoleUser:
				// User message style
				style = StyleUserText
			case chat.RoleAssistant:
				// Assistant message style
				style = StyleAssistantText
			case chat.RoleSystem:
				// System message style
				style = StyleSystemText
			case chat.RoleError:
				// Error message style
				style = StyleBorderError
			default:
				// Default style for empty lines or unknown roles
				style = tcell.StyleDefault
			}
		}

		renderText(screen, chatArea.X, chatArea.Y+i, msgLine.Text, style)
	}

	// Note: Spinner rendering is now handled by RenderStatusRow in the alert area
	// This eliminates duplicate spinners and provides better status information
}

func RenderMessagesWithSpinner(screen tcell.Screen, display MessageDisplay, area Rect, spinner SpinnerComponent) {
	RenderMessagesWithSpinnerAndStreaming(screen, display, area, spinner, false)
}

func RenderInput(screen tcell.Screen, input InputField, area Rect) {
	RenderInputWithSpinner(screen, input, area, SpinnerComponent{})
}

func RenderInputWithSpinner(screen tcell.Screen, input InputField, area Rect, spinner SpinnerComponent) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	borderStyle := StyleBorder.Background(tcell.ColorDefault)

	if area.Height >= 3 {
		for x := area.X; x < area.X+area.Width; x++ {
			screen.SetContent(x, area.Y, '─', nil, borderStyle)
			screen.SetContent(x, area.Y+2, '─', nil, borderStyle)
		}
		screen.SetContent(area.X, area.Y, '╭', nil, borderStyle)
		screen.SetContent(area.X+area.Width-1, area.Y, '╮', nil, borderStyle)
		screen.SetContent(area.X, area.Y+2, '╰', nil, borderStyle)
		screen.SetContent(area.X+area.Width-1, area.Y+2, '╯', nil, borderStyle)

		screen.SetContent(area.X, area.Y+1, '│', nil, borderStyle)
		screen.SetContent(area.X+area.Width-1, area.Y+1, '│', nil, borderStyle)
	}

	if area.Height >= 2 && area.Width >= 5 { // Need more space for chevron/spinner
		inputY := area.Y + 1
		prefixX := area.X + 1
		inputX := area.X + 3         // Leave space for chevron/spinner and a space
		inputWidth := area.Width - 4 // Account for borders and prefix

		// Render chevron or spinner prefix
		if spinner.IsVisible {
			// Show dimmed blue spinner during processing
			renderText(screen, prefixX, inputY, spinner.GetCurrentFrame(), StyleDimText)
		} else {
			// Show normal chevron when ready
			renderText(screen, prefixX, inputY, ">", StyleUserText)
		}

		// Render input text if we have enough width
		if inputWidth > 0 {
			// Calculate display area for input text with cursor
			inputText := input.Content
			cursor := input.Cursor

			// Ensure cursor is within bounds
			if cursor < 0 {
				cursor = 0
			}
			if cursor > len(inputText) {
				cursor = len(inputText)
			}

			// Calculate visible portion of text that fits in the available width
			visibleStart := 0
			visibleText := inputText

			// If text + cursor position is longer than available width, scroll
			if len(inputText) > inputWidth {
				// Try to center cursor in visible area, but keep it at least partially visible
				if cursor >= inputWidth {
					visibleStart = cursor - inputWidth + 1
					if visibleStart < 0 {
						visibleStart = 0
					}
				}

				// Adjust visible text
				visibleEnd := visibleStart + inputWidth
				if visibleEnd > len(inputText) {
					visibleEnd = len(inputText)
				}
				visibleText = inputText[visibleStart:visibleEnd]
			}

			// Render the visible text
			renderText(screen, inputX, inputY, visibleText, StyleUserText)

			// Render cursor if it's in the visible area
			adjustedCursor := cursor - visibleStart
			if adjustedCursor >= 0 && adjustedCursor <= len(visibleText) && adjustedCursor < inputWidth {
				cursorX := inputX + adjustedCursor
				if cursorX < inputX+inputWidth {
					// Show cursor as inverse character
					var cursorChar rune = ' '
					if adjustedCursor < len(visibleText) {
						cursorChar = rune(visibleText[adjustedCursor])
					}
					cursorStyle := StyleUserText.Reverse(true)
					screen.SetContent(cursorX, inputY, cursorChar, nil, cursorStyle)
				}
			}
		}
	}
}

func RenderStatus(screen tcell.Screen, status StatusBar, area Rect) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	// Create status indicator (green circle for ready, hollow for other states)
	var statusIndicator string
	indicatorStyle := StyleDimText
	if status.Status == "Ready" {
		statusIndicator = "●" // Filled circle for ready
		indicatorStyle = tcell.StyleDefault.Foreground(tcell.ColorGreen)
	} else {
		statusIndicator = "○" // Hollow circle for other states
	}

	// Format the status text with model name only (indicator rendered separately)
	statusText := status.Model

	// Add token information if available (just total count)
	if status.PromptTokens > 0 || status.ResponseTokens > 0 {
		totalTokens := status.PromptTokens + status.ResponseTokens
		tokenText := fmt.Sprintf(" %d", totalTokens)
		statusText += tokenText
	}

	// Calculate position for right-aligned text (include space for indicator + space)
	textLen := len(statusText) + 2 // +2 for indicator and space
	startX := area.X + area.Width - textLen
	if startX < area.X {
		// Text is too long, truncate from the left
		startX = area.X
		if textLen > area.Width {
			// Show the end of the text (keep model and tokens visible)
			statusText = "..." + statusText[len(statusText)-(area.Width-5):]
		}
	}

	// Render the indicator with appropriate style
	// Convert string to runes and get the first rune to handle Unicode properly
	runes := []rune(statusIndicator)
	if len(runes) > 0 {
		screen.SetContent(startX, area.Y, runes[0], nil, indicatorStyle)
	}

	// Render the model name and tokens after the indicator
	if len(statusText) > 0 {
		renderText(screen, startX+2, area.Y, statusText, StyleDimText) // Render model name + tokens
	}
}

func RenderTokensOnly(screen tcell.Screen, area Rect, promptTokens, responseTokens int) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	tokenText := fmt.Sprintf("Tokens: %d/%d", promptTokens, responseTokens)

	// Right-align the token text
	startX := area.X + area.Width - len(tokenText)
	if startX < area.X {
		startX = area.X
		// Truncate if necessary
		if len(tokenText) > area.Width {
			tokenText = tokenText[:area.Width]
		}
	}

	renderText(screen, startX, area.Y, tokenText, StyleTokenCount)
}

func RenderTokensWithSpinner(screen tcell.Screen, area Rect, promptTokens, responseTokens int, spinnerVisible bool, spinnerFrame string) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	// Show only total token count as a single number
	totalTokens := promptTokens + responseTokens
	tokenText := fmt.Sprintf("%d", totalTokens)

	// Right-align the token text
	startX := area.X + area.Width - len(tokenText)
	if startX < area.X {
		startX = area.X
		// Truncate if necessary
		if len(tokenText) > area.Width {
			tokenText = tokenText[:area.Width]
		}
	}

	// Always render just the token text, no spinner
	renderText(screen, startX, area.Y, tokenText, StyleDimText)
}

// RenderStatusRow renders the new enhanced status row with format:
// <SPINNER> <FEEDBACK_TEXT> (<DURATION> | <NUM_TOKENS> | <bold>esc</bold> to interject)
func RenderStatusRow(screen tcell.Screen, area Rect, statusRow StatusRowDisplay) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	x := area.X

	// 1. Spinner (if visible)
	if statusRow.IsSpinnerVisible {
		spinnerFrame := GetSpinnerFrame(statusRow.SpinnerFrame)
		renderText(screen, x, area.Y, spinnerFrame, StyleDimText)
		x += len(spinnerFrame) + 1 // Add space after spinner
	}

	// 2. Feedback text
	if statusRow.FeedbackText != "" {
		// Calculate available width for feedback text
		remainingWidth := area.Width - (x - area.X)
		feedbackText := statusRow.FeedbackText
		
		// Reserve space for the status info in parentheses (estimate ~30 chars)
		maxFeedbackWidth := remainingWidth - 35
		if maxFeedbackWidth > 0 && len(feedbackText) > maxFeedbackWidth {
			feedbackText = feedbackText[:maxFeedbackWidth-3] + "..."
		}
		
		renderText(screen, x, area.Y, feedbackText, tcell.StyleDefault)
		x += len(feedbackText) + 1 // Add space after feedback text
	}

	// 3. Build status info in parentheses: (DURATION | NUM_TOKENS | esc to interject)
	var statusInfo []string
	
	// Duration
	if statusRow.CurrentDuration > 0 {
		duration := statusRow.CurrentDuration
		if statusRow.IsSpinnerVisible && !statusRow.StartTime.IsZero() {
			duration = time.Since(statusRow.StartTime)
		}
		
		// Format duration nicely
		var durationStr string
		if duration < time.Second {
			durationStr = fmt.Sprintf("%.0fms", float64(duration.Nanoseconds())/1e6)
		} else if duration < time.Minute {
			durationStr = fmt.Sprintf("%.1fs", duration.Seconds())
		} else {
			durationStr = fmt.Sprintf("%.1fm", duration.Minutes())
		}
		statusInfo = append(statusInfo, durationStr)
	}
	
	// Token count
	if statusRow.TokenCount > 0 {
		statusInfo = append(statusInfo, fmt.Sprintf("%d", statusRow.TokenCount))
	}
	
	// "esc to interject" (only when spinner is visible)
	if statusRow.IsSpinnerVisible {
		statusInfo = append(statusInfo, "esc to interject")
	}
	
	// Render status info if we have any
	if len(statusInfo) > 0 {
		statusText := "(" + strings.Join(statusInfo, " | ") + ")"
		
		// Calculate position (try to right-align, but ensure it fits)
		textWidth := len(statusText)
		availableWidth := area.Width - (x - area.X)
		
		if textWidth <= availableWidth {
			// Can fit the status text
			statusX := x
			if availableWidth > textWidth {
				// Right-align within available space
				statusX = area.X + area.Width - textWidth
			}
			
			// Render with proper styling - make "esc" bold if present
			if statusRow.IsSpinnerVisible && strings.Contains(statusText, "esc to interject") {
				// Split the text to make "esc" bold
				beforeEsc := strings.Split(statusText, "esc to interject")[0]
				afterEsc := strings.Split(statusText, "esc to interject")[1]
				
				// Render before "esc"
				renderText(screen, statusX, area.Y, beforeEsc, StyleDimText)
				escX := statusX + len(beforeEsc)
				
				// Render "esc" in bold
				renderText(screen, escX, area.Y, "esc", StyleDimText.Bold(true))
				
				// Render rest
				renderText(screen, escX+3, area.Y, " to interject"+afterEsc, StyleDimText)
			} else {
				// Regular rendering
				renderText(screen, statusX, area.Y, statusText, StyleDimText)
			}
		}
	}
}

func clearArea(screen tcell.Screen, area Rect) {
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
	}
}

func renderText(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, style)
	}
}

// Node-based rendering functions

// RenderMessagesWithNodes renders messages using the node-based system
func RenderMessagesWithNodes(screen tcell.Screen, display MessageDisplay, area Rect) {
	RenderMessagesWithNodesAndSpinner(screen, display, area, SpinnerComponent{})
}

// RenderMessagesWithNodesAndSpinner renders messages using the node-based system with spinner support
func RenderMessagesWithNodesAndSpinner(screen tcell.Screen, display MessageDisplay, area Rect, spinner SpinnerComponent) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	// If not using nodes, fall back to legacy rendering
	if !display.UseNodes || display.NodeManager == nil {
		RenderMessagesWithSpinner(screen, display, area, spinner)
		return
	}

	clearArea(screen, area)

	// Get nodes from the node manager
	nodes := display.NodeManager.GetNodes()
	if len(nodes) == 0 {
		return
	}

	// Calculate available height (accounting for spinner)
	availableHeight := area.Height
	if spinner.IsVisible {
		availableHeight -= 1
	}

	// Calculate total content height and determine which nodes to render
	var visibleNodes []MessageNode
	var nodeYPositions []int
	currentY := 0
	startY := area.Y

	// Apply scroll offset
	scrollOffset := display.Scroll

	for i, node := range nodes {
		nodeHeight := node.CalculateHeight(area.Width)

		// Check if this node is visible after scrolling
		nodeEndY := currentY + nodeHeight
		if nodeEndY > scrollOffset && currentY < scrollOffset+availableHeight {
			// Node is at least partially visible
			adjustedY := startY + (currentY - scrollOffset)

			// Only include if it fits in the display area
			if adjustedY < startY+availableHeight {
				visibleNodes = append(visibleNodes, node)
				nodeYPositions = append(nodeYPositions, adjustedY)
			}
		}

		currentY += nodeHeight

		// Add spacing between nodes (except after the last one)
		if i < len(nodes)-1 {
			currentY += 1
		}
	}

	// Render each visible node
	for i, node := range visibleNodes {
		nodeY := nodeYPositions[i]
		nodeHeight := node.CalculateHeight(area.Width)

		// Calculate the area for this node
		nodeArea := Rect{
			X:      area.X,
			Y:      nodeY,
			Width:  area.Width,
			Height: nodeHeight,
		}

		// Clip to available area
		if nodeArea.Y+nodeArea.Height > startY+availableHeight {
			nodeArea.Height = (startY + availableHeight) - nodeArea.Y
		}

		if nodeArea.Height > 0 {
			// Update node bounds for click handling
			nodeBounds := NodeBounds{
				X:      nodeArea.X,
				Y:      nodeArea.Y,
				Width:  nodeArea.Width,
				Height: nodeArea.Height,
			}
			display.NodeManager.UpdateNodeBounds(node.ID(), nodeBounds)

			// Render the node
			renderedLines := node.Render(nodeArea, node.State())

			// Render each line of the node
			for lineIndex, renderedLine := range renderedLines {
				lineY := nodeArea.Y + lineIndex
				if lineY >= startY && lineY < startY+availableHeight {
					// Apply indentation
					lineX := nodeArea.X + renderedLine.Indent

					// Ensure we don't exceed the area width
					maxWidth := nodeArea.Width - renderedLine.Indent
					if maxWidth > 0 {
						text := renderedLine.Text
						if len(text) > maxWidth {
							text = text[:maxWidth]
						}
						renderText(screen, lineX, lineY, text, renderedLine.Style)
					}
				}
			}
		}
	}

	// Note: Spinner rendering is now handled by RenderStatusRow in the alert area
	// This eliminates duplicate spinners and provides better status information
}

// CalculateNodesHeight calculates the total height needed for node-based rendering
func CalculateNodesHeight(display MessageDisplay, width int) int {
	if !display.UseNodes || display.NodeManager == nil {
		// Fall back to legacy calculation
		return CalculateMessageLines(display.Messages, width, false)
	}

	return display.NodeManager.CalculateTotalHeight(width)
}

// RenderWithNodeDetection automatically chooses between node and legacy rendering
func RenderWithNodeDetection(screen tcell.Screen, display MessageDisplay, area Rect, spinner SpinnerComponent) {
	if display.UseNodes && display.NodeManager != nil {
		RenderMessagesWithNodesAndSpinner(screen, display, area, spinner)
	} else {
		RenderMessagesWithSpinner(screen, display, area, spinner)
	}
}
