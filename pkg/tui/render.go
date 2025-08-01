package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
)

func RenderMessages(screen tcell.Screen, display MessageDisplay, area Rect) {
	RenderMessagesWithSpinner(screen, display, area, SpinnerComponent{})
}

func RenderMessagesWithSpinner(screen tcell.Screen, display MessageDisplay, area Rect, spinner SpinnerComponent) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}
	
	var allLines []string
	
	for _, msg := range display.Messages {
		roleLabel := getRoleLabel(msg.Role)
		timestamp := msg.Timestamp.Format("15:04")
		prefix := fmt.Sprintf("[%s] %s: ", timestamp, roleLabel)
		
		contentLines := WrapText(msg.Content, area.Width-len(prefix))
		if len(contentLines) == 0 {
			contentLines = []string{""}
		}
		
		allLines = append(allLines, prefix+contentLines[0])
		for i := 1; i < len(contentLines); i++ {
			padding := strings.Repeat(" ", len(prefix))
			allLines = append(allLines, padding+contentLines[i])
		}
		
		allLines = append(allLines, "")
	}
	
	if len(allLines) > 0 {
		allLines = allLines[:len(allLines)-1]
	}
	
	// Calculate visible lines for messages (leave space for spinner if visible)
	availableHeight := area.Height
	if spinner.IsVisible {
		availableHeight = area.Height - 1 // Reserve bottom line for spinner
	}
	
	visibleLines, _ := CalculateVisibleLines(allLines, availableHeight, display.Scroll)
	
	clearArea(screen, area)
	
	// Render message lines
	for i, line := range visibleLines {
		if i >= availableHeight {
			break
		}
		renderText(screen, area.X, area.Y+i, line, tcell.StyleDefault)
	}
	
	// Render spinner at the bottom if visible
	if spinner.IsVisible {
		spinnerY := area.Y + area.Height - 1
		if spinnerY >= area.Y && spinnerY < area.Y + area.Height {
			// Clear the spinner line first
			for x := area.X; x < area.X + area.Width; x++ {
				screen.SetContent(x, spinnerY, ' ', nil, tcell.StyleDefault)
			}
			// Render spinner text
			renderText(screen, area.X, spinnerY, spinner.GetDisplayText(), spinner.Style)
		}
	}
}

func RenderInput(screen tcell.Screen, input InputField, area Rect) {
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
		screen.SetContent(area.X, area.Y, '┌', nil, borderStyle)
		screen.SetContent(area.X+area.Width-1, area.Y, '┐', nil, borderStyle)
		screen.SetContent(area.X, area.Y+2, '└', nil, borderStyle)
		screen.SetContent(area.X+area.Width-1, area.Y+2, '┘', nil, borderStyle)
		
		screen.SetContent(area.X, area.Y+1, '│', nil, borderStyle)
		screen.SetContent(area.X+area.Width-1, area.Y+1, '│', nil, borderStyle)
	}
	
	if area.Height >= 2 && area.Width >= 3 {
		inputY := area.Y + 1
		inputX := area.X + 1
		inputWidth := area.Width - 2
		
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
	
	statusStyle := tcell.StyleDefault.Foreground(tcell.ColorSilver).Background(tcell.ColorDarkGray)
	
	statusText := fmt.Sprintf(" Model: %s | Status: %s ", status.Model, status.Status)
	
	for x := area.X; x < area.X+area.Width; x++ {
		screen.SetContent(x, area.Y, ' ', nil, statusStyle)
	}
	
	if len(statusText) <= area.Width {
		for i, r := range statusText {
			if i >= area.Width {
				break
			}
			screen.SetContent(area.X+i, area.Y, r, nil, statusStyle)
		}
	} else {
		truncated := statusText[:area.Width-3] + "..."
		for i, r := range truncated {
			screen.SetContent(area.X+i, area.Y, r, nil, statusStyle)
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

func getRoleLabel(role string) string {
	switch role {
	case chat.RoleUser:
		return "You"
	case chat.RoleAssistant:
		return "Assistant"
	case chat.RoleSystem:
		return "System"
	default:
		return role
	}
}