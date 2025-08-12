package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/logger"
)

// startDelete returns a command to start deleting a model
func (v ModelsView) startDelete(modelName string) tea.Cmd {
	return func() tea.Msg {
		err := v.apiClient.DeleteModel(modelName)
		return deleteCompleteMsg{success: err == nil, err: err}
	}
}

// handleDeleteComplete processes deletion completion
func (v *ModelsView) handleDeleteComplete(msg deleteCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.success {
		// Refresh models list
		v.loading = true
		return v, v.fetchModels()
	} else {
		v.err = msg.err
	}
	return v, nil
}

// handleDeleteConfirmation processes delete modal key inputs
func (v *ModelsView) handleDeleteConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.modalType = ModalNone
		v.modelToDelete = ""
	case "enter", "y", "Y":
		// Confirm delete
		if v.modelToDelete != "" {
			modelName := v.modelToDelete
			v.modalType = ModalNone
			v.modelToDelete = ""
			return v, v.startDelete(modelName)
		}
	case "n", "N":
		v.modalType = ModalNone
		v.modelToDelete = ""
	}
	return v, nil
}

// handleDeleteFromList initiates delete confirmation or cancels download
func (v *ModelsView) handleDeleteFromList() (tea.Model, tea.Cmd) {
	cursor := v.table.Cursor()
	if cursor > 0 {
		isDownloading, modelName, modelIndex := v.getModelAtCursor(cursor)

		if isDownloading {
			// Cancel the active download
			logger.Debug("Cancelling download of model: %s", modelName)
			v.downloadActive = false
			v.progressChan = nil
			v.errorChan = nil
			v.downloadingModel = ""
			v.progressPercent = 0
			v.progressStatus = ""
			// Update table to remove downloading model
			v.updateTable()
		} else if modelIndex >= 0 && modelIndex < len(v.models) {
			// Show delete confirmation for non-downloading models
			v.modelToDelete = modelName
			v.modalType = ModalDelete
		}
	}
	return v, nil
}
