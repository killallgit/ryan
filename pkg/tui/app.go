package tui

import (
	"github.com/rivo/tview"
)

func StartApp() error {
	// Create the tview application
	app := tview.NewApplication()

	// Load and apply the default theme
	theme := DefaultTheme()
	ApplyTheme(theme)

	// Create and load our grid view with theme
	grid := CreateGridView(theme)

	// Set the root and run the application
	if err := app.SetRoot(grid, true).Run(); err != nil {
		return err
	}

	return nil
}
