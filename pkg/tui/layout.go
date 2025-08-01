package tui

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

func NewRect(x, y, width, height int) Rect {
	return Rect{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
}

func (r Rect) Right() int {
	return r.X + r.Width
}

func (r Rect) Bottom() int {
	return r.Y + r.Height
}

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.Right() && y >= r.Y && y < r.Bottom()
}

func (r Rect) Intersects(other Rect) bool {
	return r.X < other.Right() && r.Right() > other.X &&
		r.Y < other.Bottom() && r.Bottom() > other.Y
}

type Layout struct {
	ScreenWidth  int
	ScreenHeight int
}

func NewLayout(width, height int) Layout {
	return Layout{
		ScreenWidth:  width,
		ScreenHeight: height,
	}
}

func (l Layout) CalculateAreas() (messageArea, alertArea, inputArea, statusArea Rect) {
	statusHeight := 1
	inputHeight := 3
	alertHeight := 1
	alertBottomPadding := 1 // Add small bottom padding for status row
	messageHeight := l.ScreenHeight - statusHeight - inputHeight - alertHeight - alertBottomPadding

	if messageHeight < 1 {
		messageHeight = 1
	}

	// Add horizontal padding (2 characters on each side for more margin)
	padding := 2
	availableWidth := l.ScreenWidth - (2 * padding)
	if availableWidth < 1 {
		availableWidth = l.ScreenWidth
		padding = 0
	}

	// Message area keeps padding for better readability
	messageArea = NewRect(padding, 0, availableWidth, messageHeight)
	// Alert area uses full width to match status bar
	alertArea = NewRect(0, messageHeight, l.ScreenWidth, alertHeight)
	// Input area has padding below alert area
	inputArea = NewRect(0, messageHeight+alertHeight+alertBottomPadding, l.ScreenWidth, inputHeight)
	statusArea = NewRect(0, messageHeight+alertHeight+alertBottomPadding+inputHeight, l.ScreenWidth, statusHeight)

	return messageArea, alertArea, inputArea, statusArea
}

func WrapText(text string, width int) []string {
	if width <= 0 {
		return []string{}
	}

	if text == "" {
		return []string{}
	}

	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	runes := []rune(text)

	for len(runes) > 0 {
		lineLength := width
		if lineLength > len(runes) {
			lineLength = len(runes)
		}

		if lineLength == len(runes) {
			lines = append(lines, string(runes))
			break
		}

		breakPos := lineLength
		for i := lineLength - 1; i >= 0; i-- {
			if runes[i] == ' ' || runes[i] == '\n' {
				breakPos = i
				break
			}
		}

		if breakPos == 0 && lineLength > 0 {
			breakPos = lineLength
		}

		line := string(runes[:breakPos])
		lines = append(lines, line)

		runes = runes[breakPos:]
		for len(runes) > 0 && (runes[0] == ' ' || runes[0] == '\n') {
			runes = runes[1:]
		}
	}

	return lines
}

func CalculateVisibleLines(lines []string, height, scroll int) (visibleLines []string, startLine int) {
	if height <= 0 || len(lines) == 0 {
		return []string{}, 0
	}

	totalLines := len(lines)

	if scroll >= totalLines {
		scroll = totalLines - 1
	}
	if scroll < 0 {
		scroll = 0
	}

	startLine = scroll
	endLine := startLine + height
	if endLine > totalLines {
		endLine = totalLines
	}

	return lines[startLine:endLine], startLine
}
