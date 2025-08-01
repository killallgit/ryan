package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/spf13/viper"
)

func RenderMessages(screen tcell.Screen, display MessageDisplay, area Rect) {
	RenderMessagesWithSpinner(screen, display, area, SpinnerComponent{})
}

func RenderMessagesWithSpinner(screen tcell.Screen, display MessageDisplay, area Rect, spinner SpinnerComponent) {
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
	showThinking := viper.GetBool("show_thinking")

	for _, msg := range display.Messages {
		if msg.Role == chat.RoleAssistant {
			// Parse thinking blocks for assistant messages
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
			if parsed.ResponseContent != "" {
				contentLines := WrapText(parsed.ResponseContent, chatArea.Width)
				if len(contentLines) == 0 {
					contentLines = []string{""}
				}

				for _, line := range contentLines {
					allLines = append(allLines, MessageLine{
						Text:       line,
						Role:       msg.Role,
						IsThinking: false,
					})
				}
			}
		} else {
			// Handle non-assistant messages normally
			contentLines := WrapText(msg.Content, chatArea.Width)
			if len(contentLines) == 0 {
				contentLines = []string{""}
			}

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
		availableHeight = chatArea.Height - 1 // Reserve bottom line for spinner
	}

	// Calculate which lines are visible based on scroll
	startLine := display.Scroll
	if startLine >= len(allLines) {
		startLine = len(allLines) - 1
	}
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
			// Dimmed white style for thinking blocks
			style = tcell.StyleDefault.Foreground(tcell.ColorWhite).Dim(true)
		} else {
			switch msgLine.Role {
			case chat.RoleUser:
				// Dimmed style for user messages
				style = tcell.StyleDefault.Foreground(tcell.ColorGray)
			case chat.RoleAssistant:
				// Normal style for assistant messages
				style = tcell.StyleDefault
			case chat.RoleSystem:
				// Normal style for system messages
				style = tcell.StyleDefault
			case chat.RoleError:
				// Red style for error messages
				style = tcell.StyleDefault.Foreground(tcell.NewRGBColor(220, 50, 47))
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
			// Render spinner text with horizontal padding
			renderText(screen, area.X, spinnerY, spinner.GetDisplayText(), spinner.Style)
		}
	}
}

func RenderInput(screen tcell.Screen, input InputField, area Rect) {
	RenderInputWithSpinner(screen, input, area, SpinnerComponent{})
}

func RenderInputWithSpinner(screen tcell.Screen, input InputField, area Rect, spinner SpinnerComponent) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDefault)

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
			spinnerStyle := tcell.StyleDefault.Foreground(tcell.ColorBlue).Dim(true)
			renderText(screen, prefixX, inputY, spinner.GetCurrentFrame(), spinnerStyle)
		} else {
			// Show dimmed yellow chevron when not processing
			chevronStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Dim(true)
			renderText(screen, prefixX, inputY, ">", chevronStyle)
		}

		visibleContent := input.Content
		cursorPos := input.Cursor

		if len(visibleContent) > inputWidth {
			start := 0
			if cursorPos >= inputWidth {
				start = cursorPos - inputWidth + 1
			}
			end := start + inputWidth
			if end > len(visibleContent) {
				end = len(visibleContent)
			}
			visibleContent = visibleContent[start:end]
			cursorPos = cursorPos - start
		}

		renderText(screen, inputX, inputY, visibleContent, tcell.StyleDefault)

		if cursorPos >= 0 && cursorPos <= len(visibleContent) && cursorPos < inputWidth {
			cursorStyle := tcell.StyleDefault.Reverse(true)
			if cursorPos < len(visibleContent) {
				r := rune(visibleContent[cursorPos])
				screen.SetContent(inputX+cursorPos, inputY, r, nil, cursorStyle)
			} else {
				screen.SetContent(inputX+cursorPos, inputY, ' ', nil, cursorStyle)
			}
		}
	}
}

func RenderStatus(screen tcell.Screen, status StatusBar, area Rect) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	if status.IsModelView {
		// Simplified format for model management page
		statusStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		totalSizeGB := float64(status.TotalSize) / (1024 * 1024 * 1024)
		statusText := fmt.Sprintf(" models: %d | size: %.1f GB ", status.TotalModels, totalSizeGB)

		// Fill entire row with background
		for x := area.X; x < area.X+area.Width; x++ {
			screen.SetContent(x, area.Y, ' ', nil, statusStyle)
		}

		// Right-justify the status text
		if len(statusText) <= area.Width {
			startX := area.X + area.Width - len(statusText)
			for i, r := range statusText {
				screen.SetContent(startX+i, area.Y, r, nil, statusStyle)
			}
		}
	} else {
		// Chat view format with reorganized layout
		readyStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen).Dim(true)
		modelStyle := tcell.StyleDefault.Foreground(tcell.ColorGray) // Dim white
		if !status.ModelAvailable {
			modelStyle = tcell.StyleDefault.Foreground(tcell.ColorRed).StrikeThrough(true)
		}

		// Left-justified Ready text
		readyText := status.Status
		for i, r := range readyText {
			if area.X+i < area.X+area.Width {
				screen.SetContent(area.X+i, area.Y, r, nil, readyStyle)
			}
		}

		// Right-justified model name
		modelText := status.Model
		if len(modelText) > 0 {
			modelStartX := area.X + area.Width - len(modelText)
			if modelStartX > area.X+len(readyText)+2 { // Ensure spacing
				for i, r := range modelText {
					screen.SetContent(modelStartX+i, area.Y, r, nil, modelStyle)
				}
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

func RenderAlert(screen tcell.Screen, alert AlertDisplay, area Rect) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	displayText := alert.GetDisplayText()
	if displayText == "" {
		return
	}

	// Determine style based on content type
	var style tcell.Style
	if alert.ErrorMessage != "" {
		// Base16 red color for errors
		style = tcell.StyleDefault.Foreground(tcell.NewRGBColor(220, 50, 47))
	} else {
		// Default gray for spinner
		style = tcell.StyleDefault.Foreground(tcell.ColorGray)
	}

	// Render left-justified
	renderText(screen, area.X, area.Y, displayText, style)
}

func RenderAlertWithTokens(screen tcell.Screen, alert AlertDisplay, area Rect, promptTokens, responseTokens int) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	// Render spinner or error on the left
	displayText := alert.GetDisplayText()
	if displayText != "" {
		// Determine style based on content type
		var style tcell.Style
		if alert.ErrorMessage != "" {
			// Base16 red color for errors
			style = tcell.StyleDefault.Foreground(tcell.NewRGBColor(220, 50, 47))
		} else {
			// Default gray for spinner
			style = tcell.StyleDefault.Foreground(tcell.ColorGray)
		}
		// Render left-justified
		renderText(screen, area.X, area.Y, displayText, style)
	}

	// Render token display on the right if tokens are present
	totalTokens := promptTokens + responseTokens
	if totalTokens > 0 {
		tokenStyle := tcell.StyleDefault.Foreground(tcell.ColorBlue).Dim(true) // Dim blue
		tokenText := fmt.Sprintf("%d", totalTokens)

		// Right-justify the token text
		tokenStartX := area.X + area.Width - len(tokenText)
		if tokenStartX > area.X+len(displayText)+2 { // Ensure spacing
			for i, r := range tokenText {
				screen.SetContent(tokenStartX+i, area.Y, r, nil, tokenStyle)
			}
		}
	}
}

func RenderTokensOnly(screen tcell.Screen, area Rect, promptTokens, responseTokens int) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	// Render token display on the right if tokens are present
	totalTokens := promptTokens + responseTokens
	if totalTokens > 0 {
		tokenStyle := tcell.StyleDefault.Foreground(tcell.ColorBlue).Dim(true) // Dim blue
		tokenText := fmt.Sprintf("%d", totalTokens)

		// Right-justify the token text
		tokenStartX := area.X + area.Width - len(tokenText)
		if tokenStartX >= area.X {
			for i, r := range tokenText {
				screen.SetContent(tokenStartX+i, area.Y, r, nil, tokenStyle)
			}
		}
	}
}
