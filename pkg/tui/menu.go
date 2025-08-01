package tui

import "github.com/gdamore/tcell/v2"

type MenuOption struct {
	Name        string
	Description string
}

type MenuComponent struct {
	options  []MenuOption
	selected int
	width    int
	height   int
}

func NewMenuComponent() MenuComponent {
	return MenuComponent{
		options:  []MenuOption{},
		selected: 0,
		width:    40,
		height:   10,
	}
}

func (mc MenuComponent) WithOption(name, description string) MenuComponent {
	newOptions := make([]MenuOption, len(mc.options)+1)
	copy(newOptions, mc.options)
	newOptions[len(mc.options)] = MenuOption{
		Name:        name,
		Description: description,
	}
	
	return MenuComponent{
		options:  newOptions,
		selected: mc.selected,
		width:    mc.width,
		height:   mc.height,
	}
}

func (mc MenuComponent) WithSize(width, height int) MenuComponent {
	return MenuComponent{
		options:  mc.options,
		selected: mc.selected,
		width:    width,
		height:   height,
	}
}

func (mc MenuComponent) SelectNext() MenuComponent {
	newSelected := mc.selected + 1
	if newSelected >= len(mc.options) {
		newSelected = 0
	}
	
	return MenuComponent{
		options:  mc.options,
		selected: newSelected,
		width:    mc.width,
		height:   mc.height,
	}
}

func (mc MenuComponent) SelectPrevious() MenuComponent {
	newSelected := mc.selected - 1
	if newSelected < 0 {
		newSelected = len(mc.options) - 1
	}
	
	return MenuComponent{
		options:  mc.options,
		selected: newSelected,
		width:    mc.width,
		height:   mc.height,
	}
}

func (mc MenuComponent) GetSelectedOption() string {
	if mc.selected >= 0 && mc.selected < len(mc.options) {
		return mc.options[mc.selected].Name
	}
	return ""
}

func (mc MenuComponent) GetOptionByIndex(index int) string {
	if index >= 0 && index < len(mc.options) {
		return mc.options[index].Name
	}
	return ""
}

func (mc MenuComponent) Render(screen tcell.Screen, area Rect) {
	if len(mc.options) == 0 || area.Width < 4 || area.Height < 4 {
		return
	}
	
	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	selectedStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorOrange)
	normalStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
	
	drawBorder(screen, area, borderStyle)
	
	titleText := "Select View"
	titleX := area.X + (area.Width-len(titleText))/2
	if titleX < area.X+1 {
		titleX = area.X + 1
	}
	renderTextWithLimit(screen, titleX, area.Y+1, len(titleText), titleText, titleStyle)
	
	startY := area.Y + 3
	for i, option := range mc.options {
		if startY+i >= area.Y+area.Height-1 {
			break
		}
		
		style := normalStyle
		if i == mc.selected {
			style = selectedStyle
		}
		
		optionText := option.Description
		if len(optionText) > area.Width-6 {
			optionText = optionText[:area.Width-9] + "..."
		}
		
		numberText := string(rune('1' + i))
		fullText := numberText + ". " + optionText
		
		// Fill the entire row with the background color for selected item
		for x := area.X + 1; x < area.X + area.Width - 1; x++ {
			char := ' '
			textIndex := x - (area.X + 2)
			if textIndex >= 0 && textIndex < len([]rune(fullText)) {
				fullTextRunes := []rune(fullText)
				char = fullTextRunes[textIndex]
			}
			screen.SetContent(x, startY+i, char, nil, style)
		}
	}
	
	instructionText := "Use ↑↓ or 1-9, Enter to select, Esc to cancel"
	if len(instructionText) > area.Width-4 {
		instructionText = "↑↓ or 1-9, Enter, Esc"
	}
	instrX := area.X + (area.Width-len(instructionText))/2
	if instrX < area.X+1 {
		instrX = area.X + 1
	}
	renderTextWithLimit(screen, instrX, area.Y+area.Height-2, len(instructionText), instructionText, normalStyle)
}

func drawBorder(screen tcell.Screen, area Rect, style tcell.Style) {
	for x := area.X; x < area.X+area.Width; x++ {
		screen.SetContent(x, area.Y, '─', nil, style)
		screen.SetContent(x, area.Y+area.Height-1, '─', nil, style)
	}
	
	for y := area.Y; y < area.Y+area.Height; y++ {
		screen.SetContent(area.X, y, '│', nil, style)
		screen.SetContent(area.X+area.Width-1, y, '│', nil, style)
	}
	
	screen.SetContent(area.X, area.Y, '┌', nil, style)
	screen.SetContent(area.X+area.Width-1, area.Y, '┐', nil, style)
	screen.SetContent(area.X, area.Y+area.Height-1, '└', nil, style)
	screen.SetContent(area.X+area.Width-1, area.Y+area.Height-1, '┘', nil, style)
	
}

func renderTextWithLimit(screen tcell.Screen, x, y, maxWidth int, text string, style tcell.Style) {
	runes := []rune(text)
	for i, r := range runes {
		if i >= maxWidth {
			break
		}
		screen.SetContent(x+i, y, r, nil, style)
	}
}