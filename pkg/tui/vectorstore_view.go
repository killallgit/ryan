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
	vv.info.SetBorder(false).SetTitle("")
	vv.info.SetBackgroundColor(tcell.GetColor(ColorBase01))

	// Create collections table
	vv.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetSeparator(' ')
	vv.table.SetBorder(false).SetTitle("")
	vv.table.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Create headers
	headers := []string{"Collection", "Documents", "Embeddings", "Status"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.GetColor(ColorYellow)).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetBackgroundColor(tcell.GetColor(ColorBase01)).
			SetExpansion(1)
		vv.table.SetCell(0, col, cell)
	}

	// Create status bar
	vv.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	vv.status.SetBackgroundColor(tcell.GetColor(ColorBase01))
	vv.status.SetTextAlign(tview.AlignCenter)
	vv.status.SetText("[#5c5044]Press r to refresh | c to create collection | d to delete | Esc to go back[-]")

	// Create padded info area
	infoContainer := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).     // Left padding
		AddItem(vv.info, 0, 1, false). // Info content
		AddItem(nil, 2, 0, false)      // Right padding

	// Create padded table area
	tableContainer := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).     // Left padding
		AddItem(vv.table, 0, 1, true). // Table content
		AddItem(nil, 2, 0, false)      // Right padding

	// Layout with padding
	vv.AddItem(nil, 1, 0, false). // Top padding
					AddItem(infoContainer, 5, 0, false).
					AddItem(tableContainer, 0, 1, true).
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
	info := "[#f5b761]Vector Store Information[-]\n\n"
	info += "Provider: [#61afaf]ChromeM[-]\n"
	info += "Status: [#93b56b]Active[-]\n"
	info += "Persistence: [#61afaf]Enabled[-]\n"
	info += "Embedder: [#61afaf]Local/all-MiniLM-L6-v2[-]"

	vv.info.SetText(info)

	// Clear existing rows (except header)
	rowCount := vv.table.GetRowCount()
	for i := rowCount - 1; i > 0; i-- {
		vv.table.RemoveRow(i)
	}

	// Add sample data
	vv.table.SetCell(1, 0, tview.NewTableCell("default").
		SetTextColor(tcell.GetColor(ColorBase05)).
		SetAlign(tview.AlignLeft).
		SetBackgroundColor(tcell.GetColor(ColorBase00)).
		SetExpansion(1))
	vv.table.SetCell(1, 1, tview.NewTableCell("0").
		SetTextColor(tcell.GetColor(ColorBase05)).
		SetAlign(tview.AlignLeft).
		SetBackgroundColor(tcell.GetColor(ColorBase00)).
		SetExpansion(1))
	vv.table.SetCell(1, 2, tview.NewTableCell("0").
		SetTextColor(tcell.GetColor(ColorBase05)).
		SetAlign(tview.AlignLeft).
		SetBackgroundColor(tcell.GetColor(ColorBase00)).
		SetExpansion(1))
	vv.table.SetCell(1, 3, tview.NewTableCell("[#93b56b]Ready[-]").
		SetAlign(tview.AlignLeft).
		SetBackgroundColor(tcell.GetColor(ColorBase00)).
		SetExpansion(1))
}
