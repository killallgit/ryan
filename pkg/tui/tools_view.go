package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/rivo/tview"
)

// ToolsView represents the tools interface using tview
type ToolsView struct {
	*tview.Flex

	// Components
	table  *tview.Table
	status *tview.TextView

	// State
	registry     *tools.Registry
	currentModel string
}

// NewToolsView creates a new tools view
func NewToolsView(registry *tools.Registry) *ToolsView {
	tv := &ToolsView{
		Flex:     tview.NewFlex().SetDirection(tview.FlexRow),
		registry: registry,
	}

	// Create table for tools
	tv.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetSeparator(' ')

	tv.table.SetBorder(false).SetTitle("")
	tv.table.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create headers
	headers := []string{"Tool", "Description", "Status"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.GetColor(ColorYellow)).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetBackgroundColor(tcell.GetColor(ColorBase01)).
			SetExpansion(1)
		tv.table.SetCell(0, col, cell)
	}

	// Create status bar
	tv.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	tv.status.SetBackgroundColor(tcell.GetColor(ColorBase01))

	// Create padded table area
	tableContainer := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).     // Left padding
		AddItem(tv.table, 0, 1, true). // Table content
		AddItem(nil, 2, 0, false)      // Right padding

	// Layout with padding
	tv.AddItem(nil, 1, 0, false). // Top padding
					AddItem(tableContainer, 0, 1, true).
					AddItem(tv.status, 1, 0, false)

	// Load tools
	tv.refreshTools()

	return tv
}

// SetCurrentModel updates the current model
func (tv *ToolsView) SetCurrentModel(model string) {
	tv.currentModel = model
	tv.updateStatus()
}

// refreshTools loads and displays the tool list
func (tv *ToolsView) refreshTools() {
	// Clear existing rows (except header)
	rowCount := tv.table.GetRowCount()
	for i := rowCount - 1; i > 0; i-- {
		tv.table.RemoveRow(i)
	}

	if tv.registry == nil {
		tv.table.SetCell(1, 0, tview.NewTableCell("No tools available").
			SetAlign(tview.AlignCenter).
			SetExpansion(3))
		tv.updateStatus()
		return
	}

	// Get all tools
	allTools := tv.registry.GetTools()

	row := 1
	for _, tool := range allTools {
		// Get tool name
		toolName := fmt.Sprintf("%v", tool) // Convert to string
		if nameable, ok := tool.(interface{ Name() string }); ok {
			toolName = nameable.Name()
		}

		// Alternate row colors
		bgColor := tcell.GetColor(ColorBase00)
		if row%2 == 0 {
			bgColor = tcell.GetColor(ColorBase01)
		}

		// Tool name
		tv.table.SetCell(row, 0, tview.NewTableCell(toolName).
			SetTextColor(tcell.GetColor(ColorCyan)).
			SetAlign(tview.AlignLeft).
			SetBackgroundColor(bgColor).
			SetExpansion(1))

		// Description
		desc := "Tool description"
		if describable, ok := tool.(interface{ Description() string }); ok {
			desc = describable.Description()
		}
		tv.table.SetCell(row, 1, tview.NewTableCell(desc).
			SetTextColor(tcell.GetColor(ColorBase05)).
			SetAlign(tview.AlignLeft).
			SetBackgroundColor(bgColor).
			SetExpansion(2))

		// Status
		status := "[#93b56b]Available[-]"
		tv.table.SetCell(row, 2, tview.NewTableCell(status).
			SetAlign(tview.AlignLeft).
			SetBackgroundColor(bgColor).
			SetExpansion(1))

		row++
	}

	tv.updateStatus()
}

// updateStatus updates the status bar
func (tv *ToolsView) updateStatus() {
	toolCount := 0
	if tv.registry != nil {
		toolCount = len(tv.registry.GetTools())
	}

	status := fmt.Sprintf("[#5c5044]%d tools available | Press Esc to go back[-]", toolCount)

	tv.status.SetTextAlign(tview.AlignCenter)
	tv.status.SetText(status)
}
