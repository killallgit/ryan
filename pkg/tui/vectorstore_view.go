package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// VectorStoreView represents the vector store interface using tview
type VectorStoreView struct {
	*tview.Flex
	
	// Components
	info   *tview.TextView
	table  *tview.Table
	status *tview.TextView
}

// NewVectorStoreView creates a new vector store view
func NewVectorStoreView() *VectorStoreView {
	vv := &VectorStoreView{
		Flex: tview.NewFlex().SetDirection(tview.FlexRow),
	}
	
	// Create info panel
	vv.info = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	vv.info.SetBorder(true).SetTitle("Vector Store Status")
	
	// Create collections table
	vv.table = tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)
	vv.table.SetBorder(true).SetTitle("Collections")
	
	// Create headers
	headers := []string{"Collection", "Documents", "Embeddings", "Status"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		vv.table.SetCell(0, col, cell)
	}
	
	// Create status bar
	vv.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	vv.status.SetText("[dim]Press r to refresh | c to create collection | d to delete | Esc to go back[white]")
	
	// Layout
	vv.AddItem(vv.info, 5, 0, false).
		AddItem(vv.table, 0, 1, true).
		AddItem(vv.status, 1, 0, false)
	
	// Load initial data
	vv.refresh()
	
	// Setup key bindings
	vv.setupKeyBindings()
	
	return vv
}

// setupKeyBindings configures key bindings for the vector store view
func (vv *VectorStoreView) setupKeyBindings() {
	vv.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'r', 'R':
			vv.refresh()
			return nil
		case 'c', 'C':
			// TODO: Implement create collection
			return nil
		case 'd', 'D':
			// TODO: Implement delete collection
			return nil
		}
		return event
	})
}

// refresh updates the vector store information
func (vv *VectorStoreView) refresh() {
	// Update info panel
	info := "[yellow]Vector Store Information[white]\n\n"
	info += "Provider: [cyan]ChromeM[white]\n"
	info += "Status: [green]Active[white]\n"
	info += "Persistence: [cyan]Enabled[white]\n"
	info += "Embedder: [cyan]Local/all-MiniLM-L6-v2[white]"
	
	vv.info.SetText(info)
	
	// Clear existing rows (except header)
	rowCount := vv.table.GetRowCount()
	for i := rowCount - 1; i > 0; i-- {
		vv.table.RemoveRow(i)
	}
	
	// Add sample data
	vv.table.SetCell(1, 0, tview.NewTableCell("default"))
	vv.table.SetCell(1, 1, tview.NewTableCell("0"))
	vv.table.SetCell(1, 2, tview.NewTableCell("0"))
	vv.table.SetCell(1, 3, tview.NewTableCell("[green]Ready[white]"))
}