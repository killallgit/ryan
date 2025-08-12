package views

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/ollama"
)

// updateTable rebuilds the table with current model state
func (v *ModelsView) updateTable() {
	// Convert models to table rows with simplified columns
	// Add "Pull model" row at the top
	rows := make([]table.Row, 0, len(v.models)+1)
	rows = append(rows, table.Row{
		"→ Pull model",
		"",
		"",
		"",
	})
	// Add downloading model if it's not in the list yet
	downloadingModelExists := false
	if v.downloadActive && v.downloadingModel != "" {
		for _, model := range v.models {
			if model.Name == v.downloadingModel {
				downloadingModelExists = true
				break
			}
		}

		// If downloading model doesn't exist in current models, add it as a temporary entry
		if !downloadingModelExists {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
			spinnerChar := v.spinnerChars[v.spinnerFrame%len(v.spinnerChars)]
			modelName := spinnerChar + " " + v.downloadingModel
			row := table.Row{
				dimStyle.Render(modelName),
				dimStyle.Render("Downloading..."),
				dimStyle.Render(""),
				dimStyle.Render(""),
			}
			rows = append(rows, row)
		}
	}

	for _, model := range v.models {
		modelName := model.Name
		size := ollama.FormatSize(model.Size)
		params := model.Details.ParameterSize
		modified := model.ModifiedAt.Format("2006-01-02 15:04")

		// Add spinner and dim if this model is being downloaded
		if v.downloadActive && v.downloadingModel == model.Name {
			spinnerChar := v.spinnerChars[v.spinnerFrame%len(v.spinnerChars)]
			modelName = spinnerChar + " " + model.Name
			// Apply dimmed styling to all fields for downloading model
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
			modelName = dimStyle.Render(modelName)
			size = dimStyle.Render(size)
			params = dimStyle.Render(params)
			modified = dimStyle.Render(modified)
		}

		row := table.Row{
			modelName,
			size,
			params,
			modified,
		}
		rows = append(rows, row)
	}
	v.table.SetRows(rows)
}

// getModelAtCursor returns information about what's at the current cursor position
func (v *ModelsView) getModelAtCursor(cursor int) (isDownloading bool, modelName string, modelIndex int) {
	if cursor == 0 {
		// "Pull model" row
		return false, "", -1
	}

	currentRow := 1 // Start after "Pull model" row

	// Check if there's a temporary downloading model
	if v.downloadActive && v.downloadingModel != "" {
		downloadingModelExists := false
		for _, model := range v.models {
			if model.Name == v.downloadingModel {
				downloadingModelExists = true
				break
			}
		}

		if !downloadingModelExists {
			// There's a temporary downloading model at row 1
			if cursor == currentRow {
				return true, v.downloadingModel, -1
			}
			currentRow++
		}
	}

	// Check regular models
	for i, model := range v.models {
		if cursor == currentRow {
			isDownloadingThis := v.downloadActive && v.downloadingModel == model.Name
			return isDownloadingThis, model.Name, i
		}
		currentRow++
	}

	return false, "", -1
}

// handleEnterKey processes Enter key press based on cursor position
func (v *ModelsView) handleEnterKey() (tea.Model, tea.Cmd) {
	cursor := v.table.Cursor()
	if cursor == 0 {
		// "Pull model" row selected - show download modal
		v.modalType = ModalDownload
		v.textInput.Focus()
		v.textInput.SetValue("")
	} else {
		// Check what's at the cursor position
		isDownloading, modelName, modelIndex := v.getModelAtCursor(cursor)

		if isDownloading {
			// Show download progress modal for downloading model
			v.modalType = ModalDownload
			logger.Debug("Opening progress modal for downloading model: %s", modelName)
		} else if modelIndex >= 0 && modelIndex < len(v.models) {
			// Regular model selected - show details modal
			v.selectedModel = &v.models[modelIndex]
			v.modalType = ModalDetails
		}
	}
	return v, nil
}
