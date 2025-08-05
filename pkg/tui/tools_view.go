package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/tools"
)

// ToolsView displays registered tools and their statistics
type ToolsView struct {
	toolRegistry *tools.Registry
	width        int
	height       int
	scrollY      int
	selectedRow  int
	showDetails  bool
	currentModel string
}

// NewToolsView creates a new tools view
func NewToolsView(toolRegistry *tools.Registry) *ToolsView {
	return &ToolsView{
		toolRegistry: toolRegistry,
		width:        80,
		height:       24,
		scrollY:      0,
		selectedRow:  0,
		showDetails:  false,
		currentModel: "",
	}
}

// SetCurrentModel updates the current model for compatibility display
func (v *ToolsView) SetCurrentModel(model string) {
	v.currentModel = model
}
// Name returns the view name for registration
func (v *ToolsView) Name() string {
	return "tools"
}

// Description returns a description for the command palette
func (v *ToolsView) Description() string {
	return "Tools - View registered tools, usage statistics, and execution status"
}

// Render renders the tools view
func (v *ToolsView) Render(screen tcell.Screen, area Rect) {
	// Clear the area with background
	bgStyle := tcell.StyleDefault.Background(tcell.ColorBlack)
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			screen.SetContent(x, y, ' ', nil, bgStyle)
		}
	}

	if v.toolRegistry == nil {
		v.renderNoRegistry(screen, area)
		return
	}

	// Get all tools and their stats
	allTools := v.toolRegistry.GetTools()
	allStats := v.toolRegistry.GetAllToolStats()

	if len(allTools) == 0 {
		v.renderNoTools(screen, area)
		return
	}

	// Sort tools by name for consistent display
	toolNames := make([]string, 0, len(allTools))
	for name := range allTools {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)

	// Render header
	v.renderHeader(screen, area)

	// Calculate content area (minus header and footer)
	contentArea := Rect{
		X:      area.X,
		Y:      area.Y + 2, // Skip header
		Width:  area.Width,
		Height: area.Height - 4, // Skip header and footer
	}

	if v.showDetails && v.selectedRow < len(toolNames) {
		v.renderToolDetails(screen, contentArea, toolNames[v.selectedRow], allTools, allStats)
	} else {
		v.renderToolsList(screen, contentArea, toolNames, allTools, allStats)
	}

	// Render footer with instructions
	v.renderFooter(screen, area)
}

// renderHeader renders the view header
func (v *ToolsView) renderHeader(screen tcell.Screen, area Rect) {
	title := " Tools Overview "
	titleStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	titleX := area.X + (area.Width-len(title))/2

	for i, ch := range title {
		screen.SetContent(titleX+i, area.Y, ch, nil, titleStyle)
	}

	// Render separator line
	separatorStyle := tcell.StyleDefault.Foreground(tcell.ColorGray).Background(tcell.ColorBlack)
	for x := area.X; x < area.X+area.Width; x++ {
		screen.SetContent(x, area.Y+1, '─', nil, separatorStyle)
	}
}

// renderFooter renders the view footer with instructions
func (v *ToolsView) renderFooter(screen tcell.Screen, area Rect) {
	footerY := area.Y + area.Height - 2
	separatorStyle := tcell.StyleDefault.Foreground(tcell.ColorGray).Background(tcell.ColorBlack)
	instructionStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)

	// Render separator line
	for x := area.X; x < area.X+area.Width; x++ {
		screen.SetContent(x, footerY, '─', nil, separatorStyle)
	}

	// Render instructions
	instructions := "↑/↓/j/k: Navigate | Enter: Details | R: Reset Stats | Escape: Back to chat"
	if v.showDetails {
		instructions = "Escape: Back to list | R: Reset this tool's stats"
	}

	instructionsX := area.X + (area.Width-len(instructions))/2
	for i, ch := range instructions {
		if instructionsX+i < area.X+area.Width {
			screen.SetContent(instructionsX+i, footerY+1, ch, nil, instructionStyle)
		}
	}
}

// renderToolsList renders the list of tools with basic stats
func (v *ToolsView) renderToolsList(screen tcell.Screen, area Rect, toolNames []string, allTools map[string]tools.Tool, allStats map[string]*tools.ToolStats) {
	// Render column headers
	headerStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	headerX := area.X + 2

	// Tool Name column (20 chars)
	for i, ch := range "Tool Name" {
		if headerX+i < area.X+area.Width {
			screen.SetContent(headerX+i, area.Y, ch, nil, headerStyle)
		}
	}

	// Description column (35 chars)
	descX := headerX + 22
	for i, ch := range "Description" {
		if descX+i < area.X+area.Width {
			screen.SetContent(descX+i, area.Y, ch, nil, headerStyle)
		}
	}

	// Calls column (8 chars)
	callsX := headerX + 58
	for i, ch := range "Calls" {
		if callsX+i < area.X+area.Width {
			screen.SetContent(callsX+i, area.Y, ch, nil, headerStyle)
		}
	}

	// Success column (8 chars)
	successX := headerX + 67
	for i, ch := range "Success" {
		if successX+i < area.X+area.Width {
			screen.SetContent(successX+i, area.Y, ch, nil, headerStyle)
		}
	}

	// Running column (8 chars)
	runningX := headerX + 76
	for i, ch := range "Running" {
		if runningX+i < area.X+area.Width {
			screen.SetContent(runningX+i, area.Y, ch, nil, headerStyle)
		}
	}

	// Avg Time column
	avgTimeX := headerX + 85
	for i, ch := range "Avg Time" {
		if avgTimeX+i < area.X+area.Width {
			screen.SetContent(avgTimeX+i, area.Y, ch, nil, headerStyle)
		}
	}

	// Compatibility column
	compatX := headerX + 95
	for i, ch := range "Compat" {
		if compatX+i < area.X+area.Width {
			screen.SetContent(compatX+i, area.Y, ch, nil, headerStyle)
		}
	}

	// Render tools
	startY := area.Y + 1
	for i, toolName := range toolNames {
		if i < v.scrollY {
			continue
		}

		rowY := startY + i - v.scrollY
		if rowY >= area.Y+area.Height {
			break
		}

		tool := allTools[toolName]
		stats := allStats[toolName]
		if stats == nil {
			stats = &tools.ToolStats{Name: toolName}
		}

		// Determine row style
		rowStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
		if i == v.selectedRow {
			rowStyle = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorBlue)
		}

		// Tool name (truncate if too long)
		name := toolName
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		for j, ch := range name {
			if headerX+j < area.X+area.Width {
				screen.SetContent(headerX+j, rowY, ch, nil, rowStyle)
			}
		}

		// Description (truncate if too long)
		desc := tool.Description()
		if len(desc) > 35 {
			desc = desc[:32] + "..."
		}
		for j, ch := range desc {
			if descX+j < area.X+area.Width {
				screen.SetContent(descX+j, rowY, ch, nil, rowStyle)
			}
		}

		// Call count
		callsText := fmt.Sprintf("%d", stats.CallCount)
		for j, ch := range callsText {
			if callsX+j < area.X+area.Width {
				screen.SetContent(callsX+j, rowY, ch, nil, rowStyle)
			}
		}

		// Success rate
		successRate := "N/A"
		if stats.CallCount > 0 {
			rate := float64(stats.SuccessCount) / float64(stats.CallCount) * 100
			successRate = fmt.Sprintf("%.1f%%", rate)
		}
		for j, ch := range successRate {
			if successX+j < area.X+area.Width {
				screen.SetContent(successX+j, rowY, ch, nil, rowStyle)
			}
		}

		// Running status
		runningText := "No"
		if stats.IsRunning {
			runningText = fmt.Sprintf("Yes (%d)", stats.CurrentCalls)
		}
		runningStyle := rowStyle
		if stats.IsRunning {
			runningStyle = rowStyle.Foreground(tcell.ColorGreen)
		}
		for j, ch := range runningText {
			if runningX+j < area.X+area.Width {
				screen.SetContent(runningX+j, rowY, ch, nil, runningStyle)
			}
		}

		// Average time
		avgTimeText := "N/A"
		if stats.CallCount > 0 {
			avgTimeText = v.formatDuration(stats.AvgDuration)
		}
		for j, ch := range avgTimeText {
			if avgTimeX+j < area.X+area.Width {
				screen.SetContent(avgTimeX+j, rowY, ch, nil, rowStyle)
			}
		}

		// Compatibility status
		compatText := "Unknown"
		compatStyle := rowStyle
		if v.currentModel != "" {
			compatStatus := v.toolRegistry.GetToolCompatibility(toolName, v.currentModel)
			compatText = compatStatus.String()

			// Color code compatibility status
			switch compatStatus {
			case tools.CompatibilitySupported:
				compatStyle = rowStyle.Foreground(tcell.ColorGreen)
			case tools.CompatibilityUnsupported:
				compatStyle = rowStyle.Foreground(tcell.ColorRed)
			case tools.CompatibilityTesting:
				compatStyle = rowStyle.Foreground(tcell.ColorYellow)
			default:
				compatStyle = rowStyle.Foreground(tcell.ColorGray)
			}
		}
		for j, ch := range compatText {
			if compatX+j < area.X+area.Width {
				screen.SetContent(compatX+j, rowY, ch, nil, compatStyle)
			}
		}

		// Fill the rest of the row
		fillStartX := compatX + len(compatText)
		if fillStartX < avgTimeX+len(avgTimeText) {
			fillStartX = avgTimeX + len(avgTimeText)
		}
		for x := fillStartX; x < area.X+area.Width; x++ {
			screen.SetContent(x, rowY, ' ', nil, rowStyle)
		}
	}
}

// renderToolDetails renders detailed information about a selected tool
func (v *ToolsView) renderToolDetails(screen tcell.Screen, area Rect, toolName string, allTools map[string]tools.Tool, allStats map[string]*tools.ToolStats) {
	tool := allTools[toolName]
	stats := allStats[toolName]
	if stats == nil {
		stats = &tools.ToolStats{Name: toolName}
	}

	y := area.Y
	normalStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	labelStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	valueStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorBlack)

	// Tool name
	v.renderLine(screen, area.X, y, "Tool Name:", labelStyle)
	v.renderLine(screen, area.X+12, y, toolName, valueStyle)
	y++

	// Description
	y++
	v.renderLine(screen, area.X, y, "Description:", labelStyle)
	y++
	desc := tool.Description()
	wrapped := v.wrapText(desc, area.Width-4)
	for _, line := range wrapped {
		v.renderLine(screen, area.X+2, y, line, normalStyle)
		y++
	}

	// Statistics
	y++
	v.renderLine(screen, area.X, y, "Statistics:", labelStyle)
	y++

	v.renderLine(screen, area.X+2, y, fmt.Sprintf("Total Calls: %d", stats.CallCount), normalStyle)
	y++

	v.renderLine(screen, area.X+2, y, fmt.Sprintf("Successful: %d", stats.SuccessCount), normalStyle)
	y++

	v.renderLine(screen, area.X+2, y, fmt.Sprintf("Failed: %d", stats.ErrorCount), normalStyle)
	y++

	successRate := "N/A"
	if stats.CallCount > 0 {
		rate := float64(stats.SuccessCount) / float64(stats.CallCount) * 100
		successRate = fmt.Sprintf("%.1f%%", rate)
	}
	v.renderLine(screen, area.X+2, y, fmt.Sprintf("Success Rate: %s", successRate), normalStyle)
	y++

	currentCallsText := fmt.Sprintf("Currently Running: %d calls", stats.CurrentCalls)
	currentCallsStyle := normalStyle
	if stats.IsRunning {
		currentCallsStyle = normalStyle.Foreground(tcell.ColorGreen)
	}
	v.renderLine(screen, area.X+2, y, currentCallsText, currentCallsStyle)
	y++

	if stats.CallCount > 0 {
		v.renderLine(screen, area.X+2, y, fmt.Sprintf("Average Duration: %s", v.formatDuration(stats.AvgDuration)), normalStyle)
		y++

		v.renderLine(screen, area.X+2, y, fmt.Sprintf("Total Duration: %s", v.formatDuration(stats.TotalDuration)), normalStyle)
		y++

		if !stats.LastCalled.IsZero() {
			lastCalled := stats.LastCalled.Format("2006-01-02 15:04:05")
			v.renderLine(screen, area.X+2, y, fmt.Sprintf("Last Called: %s", lastCalled), normalStyle)
			y++
		}
	}

	// Model Compatibility
	if v.currentModel != "" {
		y++
		v.renderLine(screen, area.X, y, "Model Compatibility:", labelStyle)
		y++

		compatStatus := v.toolRegistry.GetToolCompatibility(toolName, v.currentModel)
		compatText := fmt.Sprintf("Model %s: %s", v.currentModel, compatStatus.String())

		compatStyle := normalStyle
		switch compatStatus {
		case tools.CompatibilitySupported:
			compatStyle = normalStyle.Foreground(tcell.ColorGreen)
		case tools.CompatibilityUnsupported:
			compatStyle = normalStyle.Foreground(tcell.ColorRed)
		case tools.CompatibilityTesting:
			compatStyle = normalStyle.Foreground(tcell.ColorYellow)
		default:
			compatStyle = normalStyle.Foreground(tcell.ColorGray)
		}

		v.renderLine(screen, area.X+2, y, compatText, compatStyle)
		y++

		if lastTested, exists := stats.LastTested[v.currentModel]; exists && !lastTested.IsZero() {
			testedTime := lastTested.Format("2006-01-02 15:04:05")
			v.renderLine(screen, area.X+2, y, fmt.Sprintf("Last Tested: %s", testedTime), normalStyle)
			y++
		}
	}
}

// renderLine renders a line of text at the specified position
func (v *ToolsView) renderLine(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	for i, ch := range text {
		screen.SetContent(x+i, y, ch, nil, style)
	}
}

// wrapText wraps text to the specified width
func (v *ToolsView) wrapText(text string, width int) []string {
	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// formatDuration formats a duration for display
func (v *ToolsView) formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dμs", d.Microseconds())
	} else if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// renderNoRegistry shows a message when no tool registry is available
func (v *ToolsView) renderNoRegistry(screen tcell.Screen, area Rect) {
	message := "No Tool Registry Available"
	messageStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack).Bold(true)

	messageX := area.X + (area.Width-len(message))/2
	messageY := area.Y + area.Height/2

	for i, ch := range message {
		screen.SetContent(messageX+i, messageY, ch, nil, messageStyle)
	}
}

// renderNoTools shows a message when no tools are registered
func (v *ToolsView) renderNoTools(screen tcell.Screen, area Rect) {
	message := "No Tools Registered"
	subMessage := "Tools will appear here once they are registered with the system"

	messageStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack).Bold(true)
	subMessageStyle := tcell.StyleDefault.Foreground(tcell.ColorGray).Background(tcell.ColorBlack)

	messageX := area.X + (area.Width-len(message))/2
	messageY := area.Y + area.Height/2 - 1
	subMessageX := area.X + (area.Width-len(subMessage))/2
	subMessageY := messageY + 2

	for i, ch := range message {
		screen.SetContent(messageX+i, messageY, ch, nil, messageStyle)
	}

	for i, ch := range subMessage {
		screen.SetContent(subMessageX+i, subMessageY, ch, nil, subMessageStyle)
	}
}

// HandleKeyEvent processes keyboard input
func (v *ToolsView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool {
	if v.toolRegistry == nil {
		return false
	}

	allTools := v.toolRegistry.GetTools()
	toolNames := make([]string, 0, len(allTools))
	for name := range allTools {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)

	switch ev.Key() {
	case tcell.KeyEscape:
		if v.showDetails {
			v.showDetails = false
			return true
		}
		return false // Let view manager handle switching back

	case tcell.KeyUp:
		if !v.showDetails && len(toolNames) > 0 {
			if v.selectedRow > 0 {
				v.selectedRow--
				v.ensureVisible()
			}
		}
		return true

	case tcell.KeyDown:
		if !v.showDetails && len(toolNames) > 0 {
			if v.selectedRow < len(toolNames)-1 {
				v.selectedRow++
				v.ensureVisible()
			}
		}
		return true

	case tcell.KeyEnter:
		if !v.showDetails && len(toolNames) > 0 && v.selectedRow < len(toolNames) {
			v.showDetails = true
		}
		return true
	}

	switch ev.Rune() {
	case 'r', 'R':
		if v.showDetails && len(toolNames) > 0 && v.selectedRow < len(toolNames) {
			// Reset stats for selected tool
			toolName := toolNames[v.selectedRow]
			v.toolRegistry.ResetToolStats(toolName)
		} else if !v.showDetails {
			// Reset all stats
			v.toolRegistry.ResetAllToolStats()
		}
		return true
	case 'j', 'J':
		if !v.showDetails && len(toolNames) > 0 {
			if v.selectedRow < len(toolNames)-1 {
				v.selectedRow++
				v.ensureVisible()
			}
		}
		return true
	case 'k', 'K':
		if !v.showDetails && len(toolNames) > 0 {
			if v.selectedRow > 0 {
				v.selectedRow--
				v.ensureVisible()
			}
		}
		return true
	}

	return false
}

// ensureVisible adjusts scroll to keep selected row visible
func (v *ToolsView) ensureVisible() {
	visibleRows := v.height - 6 // Account for header, footer, and spacing
	if v.selectedRow < v.scrollY {
		v.scrollY = v.selectedRow
	} else if v.selectedRow >= v.scrollY+visibleRows {
		v.scrollY = v.selectedRow - visibleRows + 1
	}
}

// HandleResize updates the view size
func (v *ToolsView) HandleResize(width, height int) {
	v.width = width
	v.height = height
}
