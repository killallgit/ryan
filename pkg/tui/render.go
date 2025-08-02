package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
)

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
						})
					}

					// Add separator line between thinking and response
					if parsed.ResponseContent != "" {
						allLines = append(allLines, MessageLine{
							Text:       "",
							Role:       "",
							IsThinking: false,
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
						"first_line", firstLine)

					for _, line := range contentLines {
						allLines = append(allLines, MessageLine{
							Text:       line,
							Role:       msg.Role,
							IsThinking: false,
						})
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

	// Calculate visible lines for messages (leave space for spinner if visible)
	availableHeight := chatArea.Height
	if spinner.IsVisible {
		availableHeight -= 1
	}

	// Calculate which lines to show based on scroll
	startLine := display.Scroll
	if startLine < 0 {
		startLine = 0
	}

	endLine := startLine + availableHeight
	if endLine > len(allLines) {
		endLine = len(allLines)
	}

	visibleLines := allLines[startLine:endLine]

	clearArea(screen, area)

	// Render message lines with appropriate styling in the padded area
	for i, msgLine := range visibleLines {
		if i >= availableHeight {
			break
		}

		// Determine style based on message role and thinking status
		var style tcell.Style
		if msgLine.IsThinking {
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

	// Render spinner at the bottom if visible (use original area for full width)
	if spinner.IsVisible {
		spinnerY := area.Y + area.Height - 1
		if spinnerY >= area.Y && spinnerY < area.Y+area.Height {
			// Clear the spinner line first
			for x := area.X; x < area.X+area.Width; x++ {
				screen.SetContent(x, spinnerY, ' ', nil, tcell.StyleDefault)
			}

			// Render spinner text
			spinnerText := fmt.Sprintf(" %s %s", spinner.GetCurrentFrame(), spinner.Text)
			renderText(screen, area.X, spinnerY, spinnerText, StyleDimText)
		}
	}
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

	// Format the status text with indicator and model name
	statusText := fmt.Sprintf("%s %s", statusIndicator, status.Model)

	// Add token information if available (just total count)
	if status.PromptTokens > 0 || status.ResponseTokens > 0 {
		totalTokens := status.PromptTokens + status.ResponseTokens
		tokenText := fmt.Sprintf(" %d", totalTokens)
		statusText += tokenText
	}

	// Calculate position for right-aligned text
	textLen := len(statusText)
	startX := area.X + area.Width - textLen
	if startX < area.X {
		// Text is too long, truncate from the left
		startX = area.X
		if textLen > area.Width {
			// Show the end of the text (keep model and tokens visible)
			statusText = "..." + statusText[textLen-area.Width+3:]
		}
	}

	// Render the indicator with appropriate style
	screen.SetContent(startX, area.Y, rune(statusIndicator[0]), nil, indicatorStyle)

	// Render the rest of the text
	if len(statusText) > 1 {
		renderText(screen, startX+2, area.Y, statusText[2:], StyleDimText) // Skip the indicator and space
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
	fullText := tokenText

	if spinnerVisible {
		spinnerText := fmt.Sprintf(" %s", spinnerFrame)
		fullText = tokenText + spinnerText
	}

	// Right-align the full text
	startX := area.X + area.Width - len(fullText)
	if startX < area.X {
		startX = area.X
		// Truncate if necessary
		if len(fullText) > area.Width {
			fullText = fullText[:area.Width]
		}
	}

	if spinnerVisible {
		// Render token text first in dim style
		renderText(screen, startX, area.Y, tokenText, StyleDimText)
		// Then render spinner
		spinnerX := startX + len(tokenText)
		if spinnerX < area.X+area.Width {
			renderText(screen, spinnerX, area.Y, fmt.Sprintf(" %s", spinnerFrame), StyleDimText)
		}
	} else {
		renderText(screen, startX, area.Y, tokenText, StyleDimText)
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
