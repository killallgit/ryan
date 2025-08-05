package tui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/tools"
)

// TableColumn defines a column in the tools table
type TableColumn struct {
	Header    string
	Width     int
	Alignment string // "left", "center", "right"
}

// ToolsTable handles the rendering of the tools table
type ToolsTable struct {
	columns []TableColumn
}

// NewToolsTable creates a new tools table with predefined columns
func NewToolsTable() *ToolsTable {
	return &ToolsTable{
		columns: []TableColumn{
			{Header: "Tool", Width: 20, Alignment: "left"},
			{Header: "Description", Width: 35, Alignment: "left"},
			{Header: "Calls", Width: 8, Alignment: "right"},
			{Header: "Success", Width: 10, Alignment: "right"},
			{Header: "Status", Width: 10, Alignment: "center"},
			{Header: "Avg Time", Width: 10, Alignment: "right"},
			{Header: "Compat", Width: 10, Alignment: "center"},
		},
	}
}

// RenderTable renders the complete tools table
func (t *ToolsTable) RenderTable(screen tcell.Screen, area Rect, toolNames []string, allTools map[string]tools.Tool, allStats map[string]*tools.ToolStats, currentModel string, selectedRow int, scrollY int) {
	// Define styles
	headerStyle := tcell.StyleDefault.Bold(true).Foreground(ColorHighlight).Background(tcell.ColorBlack)
	borderStyle := tcell.StyleDefault.Foreground(ColorBorder)
	selectedStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(ColorMenuSelected)
	normalStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	alternateStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.NewRGBColor(25, 25, 25))

	x := area.X
	y := area.Y

	// Render top border
	t.renderHorizontalBorder(screen, x, y, area.Width, "top", borderStyle)
	y++

	// Render header row
	t.renderHeaderRow(screen, x, y, area.Width, headerStyle, borderStyle)
	y++

	// Render separator line
	t.renderHorizontalBorder(screen, x, y, area.Width, "middle", borderStyle)
	y++

	// Calculate visible rows
	maxVisibleRows := area.Height - 4 // account for borders and header
	visibleRows := 0

	// Render data rows
	for i, toolName := range toolNames {
		if i < scrollY {
			continue
		}

		if visibleRows >= maxVisibleRows {
			break
		}

		tool := allTools[toolName]
		stats := allStats[toolName]
		if stats == nil {
			stats = &tools.ToolStats{Name: toolName}
		}

		// Determine row style
		rowStyle := normalStyle
		if i%2 == 1 {
			rowStyle = alternateStyle
		}
		if i == selectedRow {
			rowStyle = selectedStyle
		}

		t.renderDataRow(screen, x, y, area.Width, tool, stats, currentModel, rowStyle, borderStyle)
		y++
		visibleRows++
	}

	// Fill empty rows if needed
	for visibleRows < maxVisibleRows && y < area.Y+area.Height-1 {
		t.renderEmptyRow(screen, x, y, area.Width, borderStyle)
		y++
		visibleRows++
	}

	// Render bottom border
	if y < area.Y+area.Height {
		t.renderHorizontalBorder(screen, x, y, area.Width, "bottom", borderStyle)
	}
}

// renderHorizontalBorder renders a horizontal border line
func (t *ToolsTable) renderHorizontalBorder(screen tcell.Screen, x, y, width int, position string, style tcell.Style) {
	leftChar := '├'
	rightChar := '┤'
	crossChar := '┼'
	lineChar := '─'

	switch position {
	case "top":
		leftChar = '┌'
		rightChar = '┐'
		crossChar = '┬'
	case "bottom":
		leftChar = '└'
		rightChar = '┘'
		crossChar = '┴'
	}

	screen.SetContent(x, y, leftChar, nil, style)

	currentX := x + 1
	for i, col := range t.columns {
		// Draw line
		for j := 0; j < col.Width; j++ {
			if currentX < x+width-1 {
				screen.SetContent(currentX, y, lineChar, nil, style)
				currentX++
			}
		}

		// Draw cross/separator (except after last column)
		if i < len(t.columns)-1 && currentX < x+width-1 {
			screen.SetContent(currentX, y, crossChar, nil, style)
			currentX++
		}
	}

	// Fill remaining space
	for currentX < x+width-1 {
		screen.SetContent(currentX, y, lineChar, nil, style)
		currentX++
	}

	screen.SetContent(x+width-1, y, rightChar, nil, style)
}

// renderHeaderRow renders the header row with column titles
func (t *ToolsTable) renderHeaderRow(screen tcell.Screen, x, y, width int, headerStyle, borderStyle tcell.Style) {
	screen.SetContent(x, y, '│', nil, borderStyle)

	currentX := x + 1
	for i, col := range t.columns {
		// Render header text
		t.renderCell(screen, currentX, y, col.Width, col.Header, col.Alignment, headerStyle)
		currentX += col.Width

		// Render separator (except after last column)
		if i < len(t.columns)-1 && currentX < x+width-1 {
			screen.SetContent(currentX, y, '│', nil, borderStyle)
			currentX++
		}
	}

	// Fill remaining space
	for currentX < x+width-1 {
		screen.SetContent(currentX, y, ' ', nil, headerStyle)
		currentX++
	}

	screen.SetContent(x+width-1, y, '│', nil, borderStyle)
}

// renderDataRow renders a single data row
func (t *ToolsTable) renderDataRow(screen tcell.Screen, x, y, width int, tool tools.Tool, stats *tools.ToolStats, currentModel string, rowStyle, borderStyle tcell.Style) {
	screen.SetContent(x, y, '│', nil, borderStyle)

	currentX := x + 1

	// Tool name
	name := tool.Name()
	if len(name) > t.columns[0].Width-2 {
		name = name[:t.columns[0].Width-5] + "..."
	}
	t.renderCell(screen, currentX, y, t.columns[0].Width, name, t.columns[0].Alignment, rowStyle)
	currentX += t.columns[0].Width

	screen.SetContent(currentX, y, '│', nil, borderStyle)
	currentX++

	// Description
	desc := tool.Description()
	if len(desc) > t.columns[1].Width-2 {
		desc = desc[:t.columns[1].Width-5] + "..."
	}
	t.renderCell(screen, currentX, y, t.columns[1].Width, desc, t.columns[1].Alignment, rowStyle)
	currentX += t.columns[1].Width

	screen.SetContent(currentX, y, '│', nil, borderStyle)
	currentX++

	// Calls
	callsText := fmt.Sprintf("%d", stats.CallCount)
	t.renderCell(screen, currentX, y, t.columns[2].Width, callsText, t.columns[2].Alignment, rowStyle)
	currentX += t.columns[2].Width

	screen.SetContent(currentX, y, '│', nil, borderStyle)
	currentX++

	// Success rate
	successRate := "N/A"
	successStyle := rowStyle
	if stats.CallCount > 0 {
		rate := float64(stats.SuccessCount) / float64(stats.CallCount) * 100
		successRate = fmt.Sprintf("%.1f%%", rate)
		if rate >= 90 {
			successStyle = rowStyle.Foreground(tcell.ColorGreen)
		} else if rate >= 70 {
			successStyle = rowStyle.Foreground(tcell.ColorYellow)
		} else {
			successStyle = rowStyle.Foreground(tcell.ColorRed)
		}
	}
	t.renderCell(screen, currentX, y, t.columns[3].Width, successRate, t.columns[3].Alignment, successStyle)
	currentX += t.columns[3].Width

	screen.SetContent(currentX, y, '│', nil, borderStyle)
	currentX++

	// Running status
	statusText := "Idle"
	statusStyle := rowStyle.Foreground(tcell.ColorGray)
	if stats.IsRunning {
		statusText = "Running"
		statusStyle = rowStyle.Foreground(tcell.ColorGreen).Bold(true)
		if stats.CurrentCalls > 1 {
			statusText = fmt.Sprintf("Run (%d)", stats.CurrentCalls)
		}
	}
	t.renderCell(screen, currentX, y, t.columns[4].Width, statusText, t.columns[4].Alignment, statusStyle)
	currentX += t.columns[4].Width

	screen.SetContent(currentX, y, '│', nil, borderStyle)
	currentX++

	// Average time
	avgTimeText := "N/A"
	if stats.CallCount > 0 {
		avgTimeText = formatDuration(stats.AvgDuration)
	}
	t.renderCell(screen, currentX, y, t.columns[5].Width, avgTimeText, t.columns[5].Alignment, rowStyle)
	currentX += t.columns[5].Width

	screen.SetContent(currentX, y, '│', nil, borderStyle)
	currentX++

	// Compatibility
	compatText := "Unknown"
	compatStyle := rowStyle.Foreground(tcell.ColorGray)
	if currentModel != "" {
		compatStatus := stats.Compatibility[currentModel]
		switch compatStatus {
		case tools.CompatibilitySupported:
			compatText = "✓ Yes"
			compatStyle = rowStyle.Foreground(tcell.ColorGreen)
		case tools.CompatibilityUnsupported:
			compatText = "✗ No"
			compatStyle = rowStyle.Foreground(tcell.ColorRed)
		case tools.CompatibilityTesting:
			compatText = "⟳ Test"
			compatStyle = rowStyle.Foreground(tcell.ColorYellow)
		default:
			compatText = "? N/A"
		}
	}
	t.renderCell(screen, currentX, y, t.columns[6].Width, compatText, t.columns[6].Alignment, compatStyle)
	currentX += t.columns[6].Width

	// Fill remaining space
	for currentX < x+width-1 {
		screen.SetContent(currentX, y, ' ', nil, rowStyle)
		currentX++
	}

	screen.SetContent(x+width-1, y, '│', nil, borderStyle)
}

// renderEmptyRow renders an empty row
func (t *ToolsTable) renderEmptyRow(screen tcell.Screen, x, y, width int, borderStyle tcell.Style) {
	screen.SetContent(x, y, '│', nil, borderStyle)

	currentX := x + 1
	for i, col := range t.columns {
		// Fill with spaces
		for j := 0; j < col.Width; j++ {
			if currentX < x+width-1 {
				screen.SetContent(currentX, y, ' ', nil, tcell.StyleDefault)
				currentX++
			}
		}

		// Render separator (except after last column)
		if i < len(t.columns)-1 && currentX < x+width-1 {
			screen.SetContent(currentX, y, '│', nil, borderStyle)
			currentX++
		}
	}

	// Fill remaining space
	for currentX < x+width-1 {
		screen.SetContent(currentX, y, ' ', nil, tcell.StyleDefault)
		currentX++
	}

	screen.SetContent(x+width-1, y, '│', nil, borderStyle)
}

// renderCell renders text within a cell with proper alignment and padding
func (t *ToolsTable) renderCell(screen tcell.Screen, x, y, width int, text, alignment string, style tcell.Style) {
	// Add padding
	paddedWidth := width - 2
	if paddedWidth < 1 {
		return
	}

	// Truncate if needed
	runes := []rune(text)
	if len(runes) > paddedWidth {
		runes = runes[:paddedWidth]
	}

	// Calculate padding based on alignment
	textLen := len(runes)
	leftPad := 1
	rightPad := paddedWidth - textLen + 1

	switch alignment {
	case "center":
		leftPad = (paddedWidth-textLen)/2 + 1
		rightPad = paddedWidth - textLen - leftPad + 2
	case "right":
		leftPad = paddedWidth - textLen + 1
		rightPad = 1
	}

	// Render left padding
	for i := 0; i < leftPad; i++ {
		if x+i < x+width {
			screen.SetContent(x+i, y, ' ', nil, style)
		}
	}

	// Render text
	for i, r := range runes {
		if x+leftPad+i < x+width {
			screen.SetContent(x+leftPad+i, y, r, nil, style)
		}
	}

	// Render right padding
	for i := 0; i < rightPad; i++ {
		if x+leftPad+textLen+i < x+width {
			screen.SetContent(x+leftPad+textLen+i, y, ' ', nil, style)
		}
	}
}

// formatDuration formats a duration for display in the table
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dμs", d.Microseconds())
	} else if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}
