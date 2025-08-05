package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type FilterableMenuComponent struct {
	options         []MenuOption
	filteredOptions []MenuOption
	selected        int
	width           int
	height          int
	inputText       string
	inputMode       bool
	cursorPos       int
}

func NewFilterableMenuComponent() FilterableMenuComponent {
	return FilterableMenuComponent{
		options:         []MenuOption{},
		filteredOptions: []MenuOption{},
		selected:        0,
		width:           40,
		height:          10,
		inputText:       "",
		inputMode:       true,
		cursorPos:       0,
	}
}

func (fmc FilterableMenuComponent) WithOption(name, description string) FilterableMenuComponent {
	newOptions := make([]MenuOption, len(fmc.options)+1)
	copy(newOptions, fmc.options)
	newOptions[len(fmc.options)] = MenuOption{
		Name:        name,
		Description: description,
	}

	filtered := fmc.filterOptions(newOptions, fmc.inputText)

	return FilterableMenuComponent{
		options:         newOptions,
		filteredOptions: filtered,
		selected:        fmc.selected,
		width:           fmc.width,
		height:          fmc.height,
		inputText:       fmc.inputText,
		inputMode:       fmc.inputMode,
		cursorPos:       fmc.cursorPos,
	}
}

func (fmc FilterableMenuComponent) WithSize(width, height int) FilterableMenuComponent {
	return FilterableMenuComponent{
		options:         fmc.options,
		filteredOptions: fmc.filteredOptions,
		selected:        fmc.selected,
		width:           width,
		height:          height,
		inputText:       fmc.inputText,
		inputMode:       fmc.inputMode,
		cursorPos:       fmc.cursorPos,
	}
}

func (fmc FilterableMenuComponent) filterOptions(options []MenuOption, filter string) []MenuOption {
	if filter == "" {
		return options
	}

	var filtered []MenuOption
	lowerFilter := strings.ToLower(filter)

	for _, option := range options {
		if strings.Contains(strings.ToLower(option.Name), lowerFilter) ||
			strings.Contains(strings.ToLower(option.Description), lowerFilter) {
			filtered = append(filtered, option)
		}
	}

	return filtered
}

func (fmc FilterableMenuComponent) WithInputText(text string) FilterableMenuComponent {
	filtered := fmc.filterOptions(fmc.options, text)

	newSelected := fmc.selected
	if newSelected >= len(filtered) {
		newSelected = 0
	}

	return FilterableMenuComponent{
		options:         fmc.options,
		filteredOptions: filtered,
		selected:        newSelected,
		width:           fmc.width,
		height:          fmc.height,
		inputText:       text,
		inputMode:       fmc.inputMode,
		cursorPos:       len(text),
	}
}

func (fmc FilterableMenuComponent) WithInputMode(inputMode bool) FilterableMenuComponent {
	return FilterableMenuComponent{
		options:         fmc.options,
		filteredOptions: fmc.filteredOptions,
		selected:        fmc.selected,
		width:           fmc.width,
		height:          fmc.height,
		inputText:       fmc.inputText,
		inputMode:       inputMode,
		cursorPos:       fmc.cursorPos,
	}
}

func (fmc FilterableMenuComponent) WithCursorPos(pos int) FilterableMenuComponent {
	if pos < 0 {
		pos = 0
	}
	if pos > len(fmc.inputText) {
		pos = len(fmc.inputText)
	}

	return FilterableMenuComponent{
		options:         fmc.options,
		filteredOptions: fmc.filteredOptions,
		selected:        fmc.selected,
		width:           fmc.width,
		height:          fmc.height,
		inputText:       fmc.inputText,
		inputMode:       fmc.inputMode,
		cursorPos:       pos,
	}
}

func (fmc FilterableMenuComponent) SelectNext() FilterableMenuComponent {
	if len(fmc.filteredOptions) == 0 {
		return fmc
	}

	newSelected := fmc.selected + 1
	if newSelected >= len(fmc.filteredOptions) {
		newSelected = 0
	}

	return FilterableMenuComponent{
		options:         fmc.options,
		filteredOptions: fmc.filteredOptions,
		selected:        newSelected,
		width:           fmc.width,
		height:          fmc.height,
		inputText:       fmc.inputText,
		inputMode:       fmc.inputMode,
		cursorPos:       fmc.cursorPos,
	}
}

func (fmc FilterableMenuComponent) SelectPrevious() FilterableMenuComponent {
	if len(fmc.filteredOptions) == 0 {
		return fmc
	}

	newSelected := fmc.selected - 1
	if newSelected < 0 {
		newSelected = len(fmc.filteredOptions) - 1
	}

	return FilterableMenuComponent{
		options:         fmc.options,
		filteredOptions: fmc.filteredOptions,
		selected:        newSelected,
		width:           fmc.width,
		height:          fmc.height,
		inputText:       fmc.inputText,
		inputMode:       fmc.inputMode,
		cursorPos:       fmc.cursorPos,
	}
}

func (fmc FilterableMenuComponent) GetSelectedOption() string {
	if fmc.selected >= 0 && fmc.selected < len(fmc.filteredOptions) {
		return fmc.filteredOptions[fmc.selected].Name
	}
	return ""
}

func (fmc FilterableMenuComponent) GetInputText() string {
	return fmc.inputText
}

func (fmc FilterableMenuComponent) IsInputMode() bool {
	return fmc.inputMode
}

func (fmc FilterableMenuComponent) GetCursorPos() int {
	return fmc.cursorPos
}

func (fmc FilterableMenuComponent) AddChar(char rune) FilterableMenuComponent {
	runes := []rune(fmc.inputText)
	newRunes := make([]rune, len(runes)+1)

	copy(newRunes[:fmc.cursorPos], runes[:fmc.cursorPos])
	newRunes[fmc.cursorPos] = char
	copy(newRunes[fmc.cursorPos+1:], runes[fmc.cursorPos:])

	newText := string(newRunes)
	return fmc.WithInputText(newText).WithCursorPos(fmc.cursorPos + 1)
}

func (fmc FilterableMenuComponent) DeleteChar() FilterableMenuComponent {
	if fmc.cursorPos == 0 || len(fmc.inputText) == 0 {
		return fmc
	}

	runes := []rune(fmc.inputText)
	newRunes := make([]rune, len(runes)-1)

	copy(newRunes[:fmc.cursorPos-1], runes[:fmc.cursorPos-1])
	copy(newRunes[fmc.cursorPos-1:], runes[fmc.cursorPos:])

	newText := string(newRunes)
	return fmc.WithInputText(newText).WithCursorPos(fmc.cursorPos - 1)
}

func (fmc FilterableMenuComponent) MoveCursorLeft() FilterableMenuComponent {
	return fmc.WithCursorPos(fmc.cursorPos - 1)
}

func (fmc FilterableMenuComponent) MoveCursorRight() FilterableMenuComponent {
	return fmc.WithCursorPos(fmc.cursorPos + 1)
}

func (fmc FilterableMenuComponent) Render(screen tcell.Screen, area Rect) {
	if area.Width < 4 || area.Height < 4 {
		return
	}

	borderStyle := StyleDimText
	selectedStyle := StyleMenuSelected
	normalStyle := StyleMenuNormal
	inputStyle := tcell.StyleDefault.Foreground(ColorUserText).Background(tcell.ColorBlack)

	clearArea(screen, area)
	drawBorder(screen, area, borderStyle)

	// Render input field at the top
	inputY := area.Y + 1
	inputPrompt := "> "

	for i, r := range []rune(inputPrompt) {
		if area.X+1+i < area.X+area.Width-1 {
			screen.SetContent(area.X+1+i, inputY, r, nil, StylePrompt)
		}
	}

	inputStartX := area.X + 1 + len(inputPrompt)
	inputText := []rune(fmc.inputText)
	maxInputWidth := area.Width - 4 - len(inputPrompt)

	for i := 0; i < maxInputWidth; i++ {
		x := inputStartX + i
		if x >= area.X+area.Width-1 {
			break
		}

		var char rune = ' '
		style := inputStyle

		if i < len(inputText) {
			char = inputText[i]
		}

		if fmc.inputMode && i == fmc.cursorPos {
			style = style.Background(ColorBackgroundSelected)
		}

		screen.SetContent(x, inputY, char, nil, style)
	}

	// Render filtered options starting from line 3 (skipping input line and separator)
	startY := area.Y + 3
	maxVisibleOptions := 5 // Fixed to show only 5 items
	totalOptions := len(fmc.filteredOptions)

	// Calculate scroll offset to keep selected item visible
	scrollOffset := 0
	if fmc.selected >= maxVisibleOptions {
		scrollOffset = fmc.selected - maxVisibleOptions + 1
	}

	// Render up to 5 visible options
	visibleCount := 0
	for i := scrollOffset; i < totalOptions && visibleCount < maxVisibleOptions; i++ {
		option := fmc.filteredOptions[i]
		y := startY + visibleCount

		if y >= area.Y+area.Height-1 {
			break
		}

		style := normalStyle
		if i == fmc.selected && !fmc.inputMode {
			style = selectedStyle
		}

		optionText := option.Description
		maxTextWidth := area.Width - 4
		if maxTextWidth > 3 && len(optionText) > maxTextWidth {
			cutoffWidth := maxTextWidth - 3
			if cutoffWidth > 0 && cutoffWidth < len(optionText) {
				optionText = optionText[:cutoffWidth] + "..."
			}
		}

		for x := area.X + 1; x < area.X+area.Width-1; x++ {
			char := ' '
			textIndex := x - (area.X + 2)
			if textIndex >= 0 && textIndex < len([]rune(optionText)) {
				optionTextRunes := []rune(optionText)
				char = optionTextRunes[textIndex]
			}
			screen.SetContent(x, y, char, nil, style)
		}
		visibleCount++
	}

	// Show "more items" indicator if there are more options than visible
	if totalOptions > maxVisibleOptions {
		moreIndicatorY := startY + visibleCount
		if moreIndicatorY < area.Y+area.Height-1 {
			// Show downward arrow and count
			moreText := fmt.Sprintf("↓ %d more", totalOptions-maxVisibleOptions)
			moreStyle := tcell.StyleDefault.Foreground(ColorDimText).Dim(true)

			// Center the indicator
			indicatorX := area.X + (area.Width-len(moreText))/2
			for i := range len(moreText) {
				ch := rune(moreText[i])
				if indicatorX+i < area.X+area.Width-1 {
					screen.SetContent(indicatorX+i, moreIndicatorY, ch, nil, moreStyle)
				}
			}
		}
	}

	// Draw separator line between input and options
	separatorY := area.Y + 2
	for x := area.X + 1; x < area.X+area.Width-1; x++ {
		screen.SetContent(x, separatorY, '─', nil, borderStyle)
	}
}
