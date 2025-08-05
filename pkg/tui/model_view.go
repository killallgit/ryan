package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/rivo/tview"
)

// ModelView represents the model management interface using tview
type ModelView struct {
	*tview.Flex
	
	// Components
	table         *tview.Table
	status        *tview.TextView
	
	// Controllers
	modelsController *controllers.ModelsController
	chatController   ControllerInterface
	app              *tview.Application
}

// NewModelView creates a new model view
func NewModelView(modelsController *controllers.ModelsController, chatController ControllerInterface, app *tview.Application) *ModelView {
	mv := &ModelView{
		Flex:             tview.NewFlex().SetDirection(tview.FlexRow),
		modelsController: modelsController,
		chatController:   chatController,
		app:              app,
	}
	
	// Create table for models
	mv.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetSeparator(' ')
	
	mv.table.SetBorder(false).SetTitle("")
	mv.table.SetBackgroundColor(ColorBase00)
	
	// Create headers
	headers := []string{"Name", "Size", "Modified", "Status"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetBackgroundColor(ColorBase01).
			SetExpansion(1)
		mv.table.SetCell(0, col, cell)
	}
	
	// Create status bar
	mv.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	mv.status.SetBackgroundColor(ColorBase01)
	mv.status.SetTextAlign(tview.AlignCenter)
	mv.status.SetText("[#5c5044]Press Enter to select model | d to delete | r to refresh | Esc to go back[-]")
	
	// Create padded table area
	tableContainer := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 2, 0, false).          // Left padding
		AddItem(mv.table, 0, 1, true).      // Table content
		AddItem(nil, 2, 0, false)           // Right padding
	
	// Layout with padding
	mv.AddItem(nil, 1, 0, false).          // Top padding
		AddItem(tableContainer, 0, 1, true).
		AddItem(mv.status, 1, 0, false)
	
	// Setup key bindings
	mv.setupKeyBindings()
	
	// Initial load
	mv.refreshModels()
	
	return mv
}

// setupKeyBindings configures key bindings for the model view
func (mv *ModelView) setupKeyBindings() {
	mv.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			row, _ := mv.table.GetSelection()
			if row > 0 { // Skip header
				cell := mv.table.GetCell(row, 0)
				if cell != nil {
					modelName := cell.Text
					mv.selectModel(modelName)
				}
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'd', 'D':
				row, _ := mv.table.GetSelection()
				if row > 0 {
					cell := mv.table.GetCell(row, 0)
					if cell != nil {
						mv.confirmDelete(cell.Text)
					}
				}
				return nil
			case 'r', 'R':
				mv.refreshModels()
				return nil
			}
		}
		return event
	})
}

// refreshModels loads and displays the model list
func (mv *ModelView) refreshModels() {
	// TODO: Implement actual model loading
	// For now, show placeholder data
	
	// Clear existing rows (except header)
	rowCount := mv.table.GetRowCount()
	for i := rowCount - 1; i > 0; i-- {
		mv.table.RemoveRow(i)
	}
	
	// Add sample models
	models := [][]string{
		{"llama2:latest", "3.8 GB", "2 days ago", "Ready"},
		{"mistral:latest", "4.1 GB", "1 week ago", "Ready"},
		{"codellama:latest", "3.8 GB", "2 weeks ago", "Ready"},
	}
	
	currentModel := mv.chatController.GetModel()
	
	for i, model := range models {
		for col, text := range model {
			cell := tview.NewTableCell(text).
				SetAlign(tview.AlignLeft).
				SetExpansion(1)
			
			// Highlight current model
			if col == 0 && text == currentModel {
				cell.SetTextColor(ColorGreen)
			} else {
				cell.SetTextColor(ColorBase05)
			}
			
			// Alternate row colors
			if i%2 == 0 {
				cell.SetBackgroundColor(ColorBase00)
			} else {
				cell.SetBackgroundColor(ColorBase01)
			}
			
			mv.table.SetCell(i+1, col, cell)
		}
	}
}

// selectModel switches to the selected model
func (mv *ModelView) selectModel(modelName string) {
	if err := mv.chatController.ValidateModel(modelName); err != nil {
		mv.showError(fmt.Sprintf("Invalid model: %v", err))
		return
	}
	
	mv.chatController.SetModel(modelName)
	mv.showSuccess(fmt.Sprintf("Switched to model: %s", modelName))
	mv.refreshModels()
}

// confirmDelete shows a confirmation dialog for model deletion
func (mv *ModelView) confirmDelete(modelName string) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Delete model '%s'?", modelName)).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Delete" {
				mv.deleteModel(modelName)
			}
			mv.app.SetRoot(mv, true)
		})
	
	mv.app.SetRoot(modal, true)
}

// deleteModel deletes the specified model
func (mv *ModelView) deleteModel(modelName string) {
	// TODO: Implement actual deletion
	mv.showSuccess(fmt.Sprintf("Model '%s' deleted", modelName))
	mv.refreshModels()
}

// showError displays an error message
func (mv *ModelView) showError(message string) {
	mv.status.SetText(fmt.Sprintf("[#d95f5f]Error: %s[-]", message))
}

// showSuccess displays a success message
func (mv *ModelView) showSuccess(message string) {
	mv.status.SetText(fmt.Sprintf("[#93b56b]✓ %s[-]", message))
}