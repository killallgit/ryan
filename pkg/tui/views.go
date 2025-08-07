package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// CreateGridView creates a basic 3-row grid layout with header, body, and footer
func CreateGridView(theme *Theme) *tview.Grid {
	// Create header with centered text and theme colors
	header := tview.NewTextView().
		SetText("HEADER").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.GetColor(theme.Info))

	// Create body with centered text and theme colors
	body := tview.NewTextView().
		SetText("BODY").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.GetColor(theme.Foreground))

	// Create footer with centered text and theme colors
	footer := tview.NewTextView().
		SetText("FOOTER").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.GetColor(theme.Warning))

	// Create grid with 3 rows:
	// - Row 0: Header (height 3)
	// - Row 1: Body (height 0 = fill remaining space)
	// - Row 2: Footer (height 3)
	grid := tview.NewGrid().
		SetRows(3, 0, 3).
		SetColumns(0).
		AddItem(header, 0, 0, 1, 1, 0, 0, false).
		AddItem(body, 1, 0, 1, 1, 0, 0, false).
		AddItem(footer, 2, 0, 1, 1, 0, 0, false)

	// Apply background color to the grid
	grid.SetBackgroundColor(tcell.GetColor(theme.Background))

	return grid
}
