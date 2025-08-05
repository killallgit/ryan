package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/models"
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
	mv.table.SetBackgroundColor(GetTcellColor(ColorBase00))
	
	// Create headers
	headers := []string{"Name", "Size", "Parameters", "Quantization", "Tools", "Status"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(GetTcellColor(ColorYellow)).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetBackgroundColor(GetTcellColor(ColorBase01)).
			SetExpansion(1)
		mv.table.SetCell(0, col, cell)
	}
	
	// Create status bar
	mv.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	mv.status.SetBackgroundColor(GetTcellColor(ColorBase01))
	mv.status.SetTextAlign(tview.AlignCenter)
	mv.status.SetText("[#5c5044]Enter: select | d: delete | r: refresh | Esc: back | Tools: [#93b56b]Excellent[-] [#61afaf]Good[-] [#f5b761]Basic[-] [#d95f5f]None[-]")
	
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
	log := logger.WithComponent("model_view")
	
	// Clear existing rows (except header)
	rowCount := mv.table.GetRowCount()
	for i := rowCount - 1; i > 0; i-- {
		mv.table.RemoveRow(i)
	}
	
	// Get models from Ollama via the controller
	response, err := mv.modelsController.Tags()
	if err != nil {
		log.Error("Failed to get models from Ollama", "error", err)
		mv.showError(fmt.Sprintf("Failed to load models: %v", err))
		return
	}
	
	if len(response.Models) == 0 {
		// Show empty state spanning all columns
		cell := tview.NewTableCell("No models found. Pull a model first.").
			SetAlign(tview.AlignCenter).
			SetTextColor(GetTcellColor(ColorMuted)).
			SetBackgroundColor(GetTcellColor(ColorBase00)).
			SetExpansion(6) // Updated to span 6 columns
		mv.table.SetCell(1, 0, cell)
		return
	}
	
	currentModel := mv.chatController.GetModel()
	
	// Add models to table
	for i, model := range response.Models {
		// Convert size to human readable format
		sizeGB := float64(model.Size) / (1024 * 1024 * 1024)
		sizeStr := fmt.Sprintf("%.1f GB", sizeGB)
		
		// Get parameter size and quantization
		paramSize := model.Details.ParameterSize
		if paramSize == "" {
			paramSize = "Unknown"
		}
		
		quantization := model.Details.QuantizationLevel
		if quantization == "" {
			quantization = "Unknown"
		}
		
		// Get tool compatibility info
		modelInfo := models.GetModelInfo(model.Name)
		toolsSupport := modelInfo.ToolCompatibility.String()
		if modelInfo.RecommendedForTools {
			toolsSupport += " ✓"
		}
		
		// Determine status (could be enhanced to check if model is loaded)
		status := "Available"
		if model.Name == currentModel {
			status = "Current"
		}
		
		// Create table cells
		modelData := []string{model.Name, sizeStr, paramSize, quantization, toolsSupport, status}
		
		for col, text := range modelData {
			cell := tview.NewTableCell(text).
				SetAlign(tview.AlignLeft).
				SetExpansion(1)
			
			// Color coding based on column content
			if col == 0 && model.Name == currentModel {
				// Current model name in green
				cell.SetTextColor(GetTcellColor(ColorGreen))
			} else if col == 4 { // Tools column
				// Color code tool compatibility
				switch modelInfo.ToolCompatibility {
				case models.ToolCompatibilityExcellent:
					cell.SetTextColor(GetTcellColor(ColorGreen))
				case models.ToolCompatibilityGood:
					cell.SetTextColor(GetTcellColor(ColorCyan))
				case models.ToolCompatibilityBasic:
					cell.SetTextColor(GetTcellColor(ColorYellow))
				case models.ToolCompatibilityNone:
					cell.SetTextColor(GetTcellColor(ColorRed))
				default:
					cell.SetTextColor(GetTcellColor(ColorMuted))
				}
			} else if col == 5 && status == "Current" {
				// Current status in green
				cell.SetTextColor(GetTcellColor(ColorGreen))
			} else {
				cell.SetTextColor(GetTcellColor(ColorBase05))
			}
			
			// Alternate row colors
			if i%2 == 0 {
				cell.SetBackgroundColor(GetTcellColor(ColorBase00))
			} else {
				cell.SetBackgroundColor(GetTcellColor(ColorBase01))
			}
			
			mv.table.SetCell(i+1, col, cell)
		}
	}
	
	mv.showSuccess(fmt.Sprintf("Loaded %d models", len(response.Models)))
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
	log := logger.WithComponent("model_view")
	
	// Don't allow deleting the current model
	if modelName == mv.chatController.GetModel() {
		mv.showError("Cannot delete the currently selected model")
		return
	}
	
	log.Debug("Deleting model", "model", modelName)
	
	err := mv.modelsController.Delete(modelName)
	if err != nil {
		log.Error("Failed to delete model", "model", modelName, "error", err)
		mv.showError(fmt.Sprintf("Failed to delete model: %v", err))
		return
	}
	
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