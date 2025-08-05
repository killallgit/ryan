package tui

import (
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

// CommandPalette is an enhanced menu with search/filter capabilities
type CommandPalette struct {
	allOptions      []MenuOption // All available options
	filteredOptions []MenuOption // Options matching current filter
	selected        int          // Selected index in filtered options
	width           int
	height          int
	filterText      string // Current filter/search text
	cursorPos       int    // Cursor position in filter text
	showInput       bool   // Whether to show the input field
}

// NewCommandPalette creates a new command palette
func NewCommandPalette() CommandPalette {
	return CommandPalette{
		allOptions:      []MenuOption{},
		filteredOptions: []MenuOption{},
		selected:        0,
		width:           60,
		height:          12,
		filterText:      "",
		cursorPos:       0,
		showInput:       true,
	}
}

// WithOption adds an option to the command palette
func (cp CommandPalette) WithOption(name, description string) CommandPalette {
	newOption := MenuOption{
		Name:        name,
		Description: description,
	}

	newAllOptions := make([]MenuOption, len(cp.allOptions)+1)
	copy(newAllOptions, cp.allOptions)
	newAllOptions[len(cp.allOptions)] = newOption

	// Recalculate filtered options
	filteredOptions := cp.filterOptions(newAllOptions, cp.filterText)

	return CommandPalette{
		allOptions:      newAllOptions,
		filteredOptions: filteredOptions,
		selected:        cp.selected, // Keep current selection if still valid
		width:           cp.width,
		height:          cp.height,
		filterText:      cp.filterText,
		cursorPos:       cp.cursorPos,
		showInput:       cp.showInput,
	}
}

// WithSize updates the dimensions
func (cp CommandPalette) WithSize(width, height int) CommandPalette {
	return CommandPalette{
		allOptions:      cp.allOptions,
		filteredOptions: cp.filteredOptions,
		selected:        cp.selected,
		width:           width,
		height:          height,
		filterText:      cp.filterText,
		cursorPos:       cp.cursorPos,
		showInput:       cp.showInput,
	}
}

// WithFilterText updates the filter text and recalculates filtered options
func (cp CommandPalette) WithFilterText(text string) CommandPalette {
	filteredOptions := cp.filterOptions(cp.allOptions, text)

	// Reset selection to 0 when filter changes
	selected := 0
	if len(filteredOptions) == 0 {
		selected = -1 // No selection if no options
	}

	return CommandPalette{
		allOptions:      cp.allOptions,
		filteredOptions: filteredOptions,
		selected:        selected,
		width:           cp.width,
		height:          cp.height,
		filterText:      text,
		cursorPos:       len(text), // Move cursor to end
		showInput:       cp.showInput,
	}
}

// WithCursorPos updates the cursor position
func (cp CommandPalette) WithCursorPos(pos int) CommandPalette {
	if pos < 0 {
		pos = 0
	}
	if pos > len(cp.filterText) {
		pos = len(cp.filterText)
	}

	return CommandPalette{
		allOptions:      cp.allOptions,
		filteredOptions: cp.filteredOptions,
		selected:        cp.selected,
		width:           cp.width,
		height:          cp.height,
		filterText:      cp.filterText,
		cursorPos:       pos,
		showInput:       cp.showInput,
	}
}

// SelectNext moves selection to next option
func (cp CommandPalette) SelectNext() CommandPalette {
	if len(cp.filteredOptions) == 0 {
		return cp
	}

	newSelected := cp.selected + 1
	if newSelected >= len(cp.filteredOptions) {
		newSelected = 0
	}

	return CommandPalette{
		allOptions:      cp.allOptions,
		filteredOptions: cp.filteredOptions,
		selected:        newSelected,
		width:           cp.width,
		height:          cp.height,
		filterText:      cp.filterText,
		cursorPos:       cp.cursorPos,
		showInput:       cp.showInput,
	}
}

// SelectPrevious moves selection to previous option
func (cp CommandPalette) SelectPrevious() CommandPalette {
	if len(cp.filteredOptions) == 0 {
		return cp
	}

	newSelected := cp.selected - 1
	if newSelected < 0 {
		newSelected = len(cp.filteredOptions) - 1
	}

	return CommandPalette{
		allOptions:      cp.allOptions,
		filteredOptions: cp.filteredOptions,
		selected:        newSelected,
		width:           cp.width,
		height:          cp.height,
		filterText:      cp.filterText,
		cursorPos:       cp.cursorPos,
		showInput:       cp.showInput,
	}
}

// GetSelectedOption returns the currently selected option name
func (cp CommandPalette) GetSelectedOption() string {
	if cp.selected >= 0 && cp.selected < len(cp.filteredOptions) {
		return cp.filteredOptions[cp.selected].Name
	}
	return ""
}

// GetFilterText returns the current filter text
func (cp CommandPalette) GetFilterText() string {
	return cp.filterText
}

// HandleKeyEvent processes keyboard input for the command palette
func (cp CommandPalette) HandleKeyEvent(ev *tcell.EventKey) (CommandPalette, bool, bool) {
	// Returns: (newPalette, handled, shouldClose)

	switch ev.Key() {
	case tcell.KeyEscape:
		return cp, true, true

	case tcell.KeyEnter:
		// Select current option
		return cp, true, true

	case tcell.KeyTab:
		// Tab to select without closing (alternative selection method)
		return cp.SelectNext(), true, false

	case tcell.KeyBacktab: // Shift+Tab
		return cp.SelectPrevious(), true, false

	case tcell.KeyUp:
		return cp.SelectPrevious(), true, false

	case tcell.KeyDown:
		return cp.SelectNext(), true, false

	case tcell.KeyLeft:
		return cp.WithCursorPos(cp.cursorPos - 1), true, false

	case tcell.KeyRight:
		return cp.WithCursorPos(cp.cursorPos + 1), true, false

	case tcell.KeyHome:
		return cp.WithCursorPos(0), true, false

	case tcell.KeyEnd:
		return cp.WithCursorPos(len(cp.filterText)), true, false

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if cp.cursorPos > 0 {
			newText := cp.filterText[:cp.cursorPos-1] + cp.filterText[cp.cursorPos:]
			return cp.WithFilterText(newText).WithCursorPos(cp.cursorPos - 1), true, false
		}
		return cp, true, false

	case tcell.KeyDelete:
		if cp.cursorPos < len(cp.filterText) {
			newText := cp.filterText[:cp.cursorPos] + cp.filterText[cp.cursorPos+1:]
			return cp.WithFilterText(newText), true, false
		}
		return cp, true, false

	case tcell.KeyCtrlA:
		return cp.WithCursorPos(0), true, false

	case tcell.KeyCtrlE:
		return cp.WithCursorPos(len(cp.filterText)), true, false

	case tcell.KeyCtrlU:
		// Clear line
		return cp.WithFilterText("").WithCursorPos(0), true, false

	default:
		// Handle character input
		if ev.Rune() != 0 && unicode.IsPrint(ev.Rune()) {
			newText := cp.filterText[:cp.cursorPos] + string(ev.Rune()) + cp.filterText[cp.cursorPos:]
			return cp.WithFilterText(newText).WithCursorPos(cp.cursorPos + 1), true, false
		}
	}

	return cp, false, false
}

// filterOptions filters the options based on the search text
func (cp CommandPalette) filterOptions(options []MenuOption, filterText string) []MenuOption {
	if filterText == "" {
		// Return all options if no filter
		result := make([]MenuOption, len(options))
		copy(result, options)
		return result
	}

	filterLower := strings.ToLower(filterText)
	var filtered []MenuOption

	for _, option := range options {
		// Check if filter matches name or description (case insensitive)
		nameMatch := strings.Contains(strings.ToLower(option.Name), filterLower)
		descMatch := strings.Contains(strings.ToLower(option.Description), filterLower)

		if nameMatch || descMatch {
			filtered = append(filtered, option)
		}
	}

	return filtered
}

// Render draws the command palette
func (cp CommandPalette) Render(screen tcell.Screen, area Rect) {
	if area.Width < 8 || area.Height < 4 {
		return
	}

	borderStyle := StyleDimText
	selectedStyle := StyleMenuSelected
	normalStyle := StyleMenuNormal
	inputStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)

	// Clear the background first to make modal opaque
	clearArea(screen, area)
	drawBorder(screen, area, borderStyle)

	currentY := area.Y + 1

	// Render title
	title := "Command Palette"
	titleStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	titleX := area.X + (area.Width-len(title))/2
	renderTextWithLimit(screen, titleX, currentY, area.Width-2, title, titleStyle)
	currentY++

	// Render search input field if enabled
	if cp.showInput {
		// Input field border/label
		searchLabel := "Search: "
		renderTextWithLimit(screen, area.X+2, currentY, len(searchLabel), searchLabel, normalStyle)

		// Input field area
		inputX := area.X + 2 + len(searchLabel)
		inputWidth := area.Width - 4 - len(searchLabel)

		// Render input text
		displayText := cp.filterText
		if len(displayText) > inputWidth {
			// Scroll text if too long
			start := len(displayText) - inputWidth + 1
			if start < 0 {
				start = 0
			}
			displayText = displayText[start:]
		}

		renderTextWithLimit(screen, inputX, currentY, inputWidth, displayText, inputStyle)

		// Render cursor
		cursorX := inputX + cp.cursorPos
		if cp.cursorPos >= len(displayText) {
			cursorX = inputX + len(displayText)
		}
		if cursorX < inputX+inputWidth {
			cursorStyle := inputStyle.Reverse(true)
			if cursorX < inputX+len(displayText) {
				// Cursor over character
				char, _, _, _ := screen.GetContent(cursorX, currentY)
				screen.SetContent(cursorX, currentY, char, nil, cursorStyle)
			} else {
				// Cursor at end
				screen.SetContent(cursorX, currentY, ' ', nil, cursorStyle)
			}
		}

		currentY += 2 // Skip line after input
	}

	// Show filter stats
	if cp.filterText != "" {
		stats := ""
		if len(cp.filteredOptions) == 0 {
			stats = "No matches"
		} else {
			stats = "Matches: " + string(rune('0'+len(cp.filteredOptions)))
			if len(cp.filteredOptions) >= 10 {
				stats = "Matches: " + string(rune('0'+len(cp.filteredOptions)/10)) + string(rune('0'+len(cp.filteredOptions)%10))
			}
		}
		statsStyle := tcell.StyleDefault.Foreground(tcell.ColorGray).Background(tcell.ColorBlack)
		renderTextWithLimit(screen, area.X+2, currentY, area.Width-4, stats, statsStyle)
		currentY++
	}

	// Render filtered options
	maxVisibleOptions := area.Y + area.Height - currentY - 1
	if maxVisibleOptions < 1 {
		return
	}

	if len(cp.filteredOptions) == 0 {
		// Show "no results" message
		noResultsMsg := "No matching commands"
		if cp.filterText == "" {
			noResultsMsg = "No commands available"
		}
		noResultsStyle := tcell.StyleDefault.Foreground(tcell.ColorGray).Background(tcell.ColorBlack)
		msgX := area.X + (area.Width-len(noResultsMsg))/2
		renderTextWithLimit(screen, msgX, currentY+1, area.Width-2, noResultsMsg, noResultsStyle)
		return
	}

	// Calculate scroll offset to keep selected item visible
	scrollOffset := 0
	if cp.selected >= maxVisibleOptions {
		scrollOffset = cp.selected - maxVisibleOptions + 1
	}

	// Render visible options
	for i := scrollOffset; i < len(cp.filteredOptions) && currentY < area.Y+area.Height-1; i++ {
		option := cp.filteredOptions[i]

		style := normalStyle
		if i == cp.selected {
			style = selectedStyle
		}

		optionText := option.Description
		maxTextWidth := area.Width - 4
		if len(optionText) > maxTextWidth && maxTextWidth > 3 {
			optionText = optionText[:maxTextWidth-3] + "..."
		}

		// Fill the entire row with the background color for selected item
		for x := area.X + 1; x < area.X+area.Width-1; x++ {
			char := ' '
			textIndex := x - (area.X + 2)
			if textIndex >= 0 && textIndex < len([]rune(optionText)) {
				optionTextRunes := []rune(optionText)
				char = optionTextRunes[textIndex]
			}
			screen.SetContent(x, currentY, char, nil, style)
		}

		currentY++
	}

	// Show scroll indicators if needed
	if scrollOffset > 0 {
		screen.SetContent(area.X+area.Width-2, area.Y+3, '▲', nil, borderStyle)
	}
	if scrollOffset+maxVisibleOptions < len(cp.filteredOptions) {
		screen.SetContent(area.X+area.Width-2, area.Y+area.Height-2, '▼', nil, borderStyle)
	}
}

// GetOptionCount returns the number of filtered options
func (cp CommandPalette) GetOptionCount() int {
	return len(cp.filteredOptions)
}

// IsEmpty returns true if there are no options
func (cp CommandPalette) IsEmpty() bool {
	return len(cp.allOptions) == 0
}
