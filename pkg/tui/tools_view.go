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
	table    *tview.Table
	status   *tview.TextView
	
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
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)
	
	tv.table.SetBorder(true).SetTitle("Available Tools")
	
	// Create headers
	headers := []string{"Tool", "Description", "Status"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		tv.table.SetCell(0, col, cell)
	}
	
	// Create status bar
	tv.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	
	// Layout
	tv.AddItem(tv.table, 0, 1, true).
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
		
		// Tool name
		tv.table.SetCell(row, 0, tview.NewTableCell(toolName).
			SetTextColor(tcell.ColorLightCyan))
		
		// Description
		desc := "Tool description"
		if describable, ok := tool.(interface{ Description() string }); ok {
			desc = describable.Description()
		}
		tv.table.SetCell(row, 1, tview.NewTableCell(desc).
			SetExpansion(2))
		
		// Status
		status := "[green]Available[white]"
		tv.table.SetCell(row, 2, tview.NewTableCell(status))
		
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
	
	status := fmt.Sprintf("[yellow]Model:[white] %s | ", tv.currentModel)
	status += fmt.Sprintf("[cyan]Tools:[white] %d available", toolCount)
	status += " | [dim]Press Esc to go back[white]"
	
	tv.status.SetText(status)
}