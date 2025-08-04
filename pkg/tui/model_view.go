package tui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/models"
	"github.com/spf13/viper"
)

type ModelView struct {
	controller         *controllers.ModelsController
	chatController     controllers.ChatControllerInterface
	modelList          ModelListDisplay
	modelStats         ModelStatsDisplay
	status             StatusBar
	layout             Layout
	loading            bool
	screen             tcell.Screen
	showStats          bool
	pullModal          TextInputModal
	confirmationModal  ConfirmationModal
	downloadModal      DownloadPromptModal
	progressModal      ProgressModal
	downloadCtx        context.Context
	downloadCancel     context.CancelFunc
	showingDetails     bool
	detailsModel       ModelInfo
	selectAfterRefresh string // Model name to select after next refresh
}

func NewModelView(controller *controllers.ModelsController, chatController controllers.ChatControllerInterface, screen tcell.Screen) *ModelView {
	width, height := screen.Size()

	log := logger.WithComponent("model_view")
	log.Debug("Creating new ModelView", "width", width, "height", height)

	view := &ModelView{
		controller:         controller,
		chatController:     chatController,
		modelList:          NewModelListDisplay(width, height-6),
		modelStats:         NewModelStatsDisplay(width, 4),
		status:             NewStatusBar(width).WithStatus("Ready").WithModelViewData(0, 0),
		layout:             NewLayout(width, height),
		loading:            false,
		screen:             screen,
		showStats:          true,
		pullModal:          NewTextInputModal(),
		confirmationModal:  NewConfirmationModal(),
		downloadModal:      NewDownloadPromptModal(),
		progressModal:      NewProgressModal(),
		downloadCtx:        nil,
		downloadCancel:     nil,
		showingDetails:     false,
		detailsModel:       ModelInfo{},
		selectAfterRefresh: "",
	}

	// Don't auto-refresh on creation - wait until view becomes active
	log.Debug("ModelView created, deferring model refresh until activation")
	return view
}

func (mv *ModelView) Name() string {
	return "models"
}

func (mv *ModelView) Description() string {
	return "Model Management"
}

func (mv *ModelView) Activate() {
	log := logger.WithComponent("model_view")
	log.Debug("ModelView activated, starting model refresh")
	mv.refreshModels()
}

func (mv *ModelView) Render(screen tcell.Screen, area Rect) {
	// If showing details, render details view instead
	if mv.showingDetails {
		mv.renderModelDetailsView(screen, area)
		return
	}

	helpHeight := 1
	modelInfoHeight := 1 // Height for model count and selected model info

	listArea := Rect{
		X:      area.X,
		Y:      area.Y,
		Width:  area.Width,
		Height: area.Height - helpHeight - modelInfoHeight - 1, // -1 for spacing
	}

	modelInfoArea := Rect{
		X:      area.X,
		Y:      listArea.Y + listArea.Height,
		Width:  area.Width,
		Height: modelInfoHeight,
	}

	helpArea := Rect{
		X:      area.X,
		Y:      area.Y + area.Height - 1, // Bottom row
		Width:  area.Width,
		Height: helpHeight,
	}

	currentModel := mv.chatController.GetModel()
	RenderModelListWithCurrentModel(screen, mv.modelList, listArea, currentModel)

	// Render model info row (count on left, selected model on right)
	mv.renderModelInfo(screen, modelInfoArea)

	// Render help text
	mv.renderHelpText(screen, helpArea)

	mv.pullModal.Render(screen, area)
	mv.confirmationModal.Render(screen, area)
	mv.downloadModal.Render(screen, area)
	mv.progressModal.Render(screen, area)
}

func (mv *ModelView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool {
	if mv.loading {
		return true // Consume all events while loading
	}

	// If showing details view, handle only escape key and consume all other events
	if mv.showingDetails {
		if ev.Key() == tcell.KeyEscape {
			mv.showingDetails = false
			mv.screen.Show()
			return true // Escape key handled, event consumed
		}
		// Consume all other events while in details view
		return true
	}

	// Handle modal events first
	if mv.progressModal.Visible {
		modal, cancel := mv.progressModal.HandleKeyEvent(ev)
		mv.progressModal = modal
		if cancel && mv.downloadCancel != nil {
			mv.downloadCancel()
		}
		return true
	}

	if mv.downloadModal.Visible {
		modal, confirmed, _ := mv.downloadModal.HandleKeyEvent(ev)
		mv.downloadModal = modal
		if confirmed {
			mv.startModelDownload(mv.downloadModal.ModelName)
		}
		return true
	}

	if mv.confirmationModal.Visible {
		modal, confirmed, _ := mv.confirmationModal.HandleKeyEvent(ev)
		mv.confirmationModal = modal
		if confirmed {
			mv.deleteModel()
		}
		return true
	}

	if mv.pullModal.Visible {
		modal, modelName, confirmed := mv.pullModal.HandleKeyEvent(ev)
		mv.pullModal = modal
		if confirmed && modelName != "" {
			mv.pullModel(modelName)
		}
		return true
	}

	switch ev.Key() {
	case tcell.KeyUp:
		mv.selectPrevious()
		return true

	case tcell.KeyDown, tcell.KeyCtrlN:
		mv.selectNext()
		return true

	case tcell.KeyPgUp:
		mv.pageUp()
		return true

	case tcell.KeyPgDn:
		mv.pageDown()
		return true

	case tcell.KeyHome:
		mv.selectFirst()
		return true

	case tcell.KeyEnd:
		mv.selectLast()
		return true

	case tcell.KeyEnter:
		mv.changeModel()
		return true

	case tcell.KeyCtrlD:
		mv.showDeleteConfirmation()
		return true

	default:
		if ev.Rune() != 0 {
			switch ev.Rune() {
			case 'j', 'J':
				mv.selectNext()
				return true

			case 'k', 'K':
				mv.selectPrevious()
				return true

			case 'd', 'D':
				mv.showModelDetails()
				return true

			case 'r', 'R':
				mv.refreshModels()
				return true

			case 'n', 'N':
				mv.showPullModal()
				return true

			case 'q', 'Q':
				// Let the app handle quit
				return false
			}
		}
	}

	return false
}

func (mv *ModelView) HandleResize(width, height int) {
	mv.layout = NewLayout(width, height)

	helpHeight := 1
	modelInfoHeight := 1

	mv.modelList = mv.modelList.WithSize(width, height-helpHeight-modelInfoHeight-1)
	mv.status = mv.status.WithWidth(width)
}

func (mv *ModelView) refreshModels() {
	log := logger.WithComponent("model_view")
	log.Debug("Starting model refresh")

	mv.loading = true
	mv.status = mv.status.WithStatus("Loading models...")

	go func() {
		log.Debug("Calling controller.Tags() for model list")
		response, err := mv.controller.Tags()
		if err != nil {
			log.Error("Failed to get model tags", "error", err)
			mv.screen.PostEvent(NewModelErrorEvent(err))
			return
		}
		log.Debug("Successfully retrieved model tags", "model_count", len(response.Models))

		// Get running models first to determine status
		var runningModels []RunningModelInfo
		runningModelNames := make(map[string]bool)
		log.Debug("Calling controller.Ps() for running models")
		psResponse, err := mv.controller.Ps()
		if err == nil {
			runningModels = convertOllamaPsToRunningModelInfo(psResponse.Models)
			for _, running := range runningModels {
				runningModelNames[running.Name] = true
			}
			log.Debug("Successfully retrieved running models", "running_count", len(runningModels))
		} else {
			log.Warn("Failed to get running models", "error", err)
		}

		// Convert models and set running status
		downloadedModels := convertOllamaModelsToModelInfoWithStatus(response.Models, runningModelNames)

		// Get available models and merge with downloaded ones
		availableModels := createAvailableModelInfo()
		downloadedModelNames := make(map[string]bool)
		for _, model := range downloadedModels {
			downloadedModelNames[model.Name] = true
			log.Debug("Downloaded model found", "name", model.Name)
		}

		// Filter out available models that are already downloaded
		var filteredAvailable []ModelInfo
		for _, available := range availableModels {
			if !isModelAlreadyDownloaded(available.Name, downloadedModelNames) {
				filteredAvailable = append(filteredAvailable, available)
				log.Debug("Available model added to list", "name", available.Name)
			} else {
				log.Debug("Available model filtered out (already downloaded)", "name", available.Name)
			}
		}

		// Combine downloaded and available models
		models := append(downloadedModels, filteredAvailable...)

		// Sort models: downloaded first, then available, alphabetically within each group
		sortModelsByDownloadStatus(models)
		log.Debug("Converted models to UI format", "downloaded_models", len(downloadedModels), "available_models", len(filteredAvailable), "total_models", len(models))

		log.Debug("Posting ModelListUpdateEvent")
		mv.screen.PostEvent(NewModelListUpdateEvent(models))

		totalSize := int64(0)
		for _, model := range models {
			totalSize += model.Size
		}

		stats := ModelStats{
			TotalModels:   len(models),
			RunningModels: runningModels,
			TotalSize:     totalSize,
		}
		log.Debug("Posting ModelStatsUpdateEvent", "total_size", totalSize)
		mv.screen.PostEvent(NewModelStatsUpdateEvent(stats))

		log.Debug("Model refresh completed successfully")
	}()
}

func (mv *ModelView) HandleModelListUpdate(ev ModelListUpdateEvent) {
	log := logger.WithComponent("model_view")
	log.Debug("Handling ModelListUpdateEvent", "model_count", len(ev.Models))

	mv.loading = false
	mv.modelList = mv.modelList.WithModels(ev.Models)

	// If we have a model to select after refresh, find and select it
	if mv.selectAfterRefresh != "" {
		for i, model := range ev.Models {
			if model.Name == mv.selectAfterRefresh {
				mv.modelList = mv.modelList.WithSelection(i)
				mv.ensureSelectionVisible()
				log.Debug("Selected newly downloaded model in UI", "model", mv.selectAfterRefresh, "index", i)
				break
			}
		}
		// Clear the selection flag
		mv.selectAfterRefresh = ""
	}

	// Check if current model is available
	currentModel := mv.chatController.GetModel()
	isAvailable := false
	for _, model := range ev.Models {
		if model.Name == currentModel {
			isAvailable = true
			break
		}
	}

	mv.status = mv.status.WithStatus("Ready").WithModelAvailability(isAvailable)
	log.Debug("ModelListUpdate completed", "current_model", currentModel, "available", isAvailable)
}

func (mv *ModelView) HandleModelStatsUpdate(ev ModelStatsUpdateEvent) {
	log := logger.WithComponent("model_view")
	log.Debug("Handling ModelStatsUpdateEvent",
		"total_models", ev.Stats.TotalModels,
		"running_models", len(ev.Stats.RunningModels),
		"total_size", ev.Stats.TotalSize)

	mv.modelStats = mv.modelStats.WithStats(ev.Stats)
	// Update status bar with model count and size
	mv.status = mv.status.WithModelViewData(ev.Stats.TotalModels, ev.Stats.TotalSize)
}

func (mv *ModelView) HandleModelError(ev ModelErrorEvent) {
	log := logger.WithComponent("model_view")
	log.Error("Handling ModelErrorEvent", "error", ev.Error)

	mv.loading = false
	mv.status = mv.status.WithStatus("Error: " + ev.Error.Error() + " - Press 'r' to retry")
}

func (mv *ModelView) HandleModelDeleted(ev ModelDeletedEvent) {
	log := logger.WithComponent("model_view")
	log.Debug("Handling ModelDeletedEvent", "model_name", ev.ModelName)

	mv.loading = false
	mv.status = mv.status.WithStatus("Model deleted: " + ev.ModelName)

	// Instead of full refresh, update the deleted model in place
	mv.markModelAsAvailable(ev.ModelName)
}

// markModelAsAvailable finds a model by name and marks it as available for download
func (mv *ModelView) markModelAsAvailable(modelName string) {
	log := logger.WithComponent("model_view")

	// Find the model in the current list
	models := mv.modelList.Models
	var wasSelected bool = false
	var originalIndex int = -1

	for i, model := range models {
		if model.Name == modelName {
			originalIndex = i
			wasSelected = (i == mv.modelList.Selected)

			// Mark as not downloaded and update properties
			models[i].IsDownloaded = false
			models[i].IsRunning = false
			models[i].Size = 0 // Reset size since it's not downloaded

			// Get estimated size from available models info
			availableModels := createAvailableModelInfo()
			for _, available := range availableModels {
				if available.Name == modelName {
					models[i].Size = available.Size
					models[i].ParameterSize = available.ParameterSize
					models[i].QuantizationLevel = available.QuantizationLevel
					break
				}
			}

			log.Debug("Marked model as available for download", "model", modelName)
			break
		}
	}

	if originalIndex >= 0 {
		// Sort the models to move the newly available model to the correct position
		sortModelsByDownloadStatus(models)

		// If the deleted model was selected, find its new position and select it
		newSelection := mv.modelList.Selected
		if wasSelected {
			for i, model := range models {
				if model.Name == modelName {
					newSelection = i
					break
				}
			}
		}

		// Update the model list with the modified models and preserve selection
		mv.modelList = mv.modelList.WithModels(models).WithSelection(newSelection)
		mv.ensureSelectionVisible()

		// Trigger screen refresh to update the display
		mv.screen.Show()
		return
	}

	// If model wasn't found in current list, add it as available
	availableModels := createAvailableModelInfo()
	for _, available := range availableModels {
		if available.Name == modelName {
			// Add the model back to the list as available
			updatedModels := append(models, available)

			// Sort the models to place the newly added model in the correct position
			sortModelsByDownloadStatus(updatedModels)

			mv.modelList = mv.modelList.WithModels(updatedModels)
			log.Debug("Added deleted model back to list as available", "model", modelName)

			// Trigger screen refresh
			mv.screen.Show()
			return
		}
	}

	log.Warn("Could not find deleted model to mark as available", "model", modelName)
}

func (mv *ModelView) selectNext() {
	if len(mv.modelList.Models) == 0 {
		return
	}

	newSelected := mv.modelList.Selected + 1
	if newSelected >= len(mv.modelList.Models) {
		newSelected = 0
	}

	mv.modelList = mv.modelList.WithSelection(newSelected)
	mv.ensureSelectionVisible()
}

func (mv *ModelView) selectPrevious() {
	if len(mv.modelList.Models) == 0 {
		return
	}

	newSelected := mv.modelList.Selected - 1
	if newSelected < 0 {
		newSelected = len(mv.modelList.Models) - 1
	}

	mv.modelList = mv.modelList.WithSelection(newSelected)
	mv.ensureSelectionVisible()
}

func (mv *ModelView) selectFirst() {
	if len(mv.modelList.Models) == 0 {
		return
	}

	mv.modelList = mv.modelList.WithSelection(0).WithScroll(0)
}

func (mv *ModelView) selectLast() {
	if len(mv.modelList.Models) == 0 {
		return
	}

	lastIndex := len(mv.modelList.Models) - 1
	mv.modelList = mv.modelList.WithSelection(lastIndex)
	mv.ensureSelectionVisible()
}

func (mv *ModelView) pageUp() {
	newScroll := mv.modelList.Scroll - mv.modelList.Height
	if newScroll < 0 {
		newScroll = 0
	}
	mv.modelList = mv.modelList.WithScroll(newScroll)

	newSelected := mv.modelList.Selected - mv.modelList.Height
	if newSelected < mv.modelList.Scroll {
		newSelected = mv.modelList.Scroll
	}
	if newSelected < 0 {
		newSelected = 0
	}
	mv.modelList = mv.modelList.WithSelection(newSelected)
}

func (mv *ModelView) pageDown() {
	maxScroll := len(mv.modelList.Models) - mv.modelList.Height
	if maxScroll < 0 {
		maxScroll = 0
	}

	newScroll := mv.modelList.Scroll + mv.modelList.Height
	if newScroll > maxScroll {
		newScroll = maxScroll
	}
	mv.modelList = mv.modelList.WithScroll(newScroll)

	newSelected := mv.modelList.Selected + mv.modelList.Height
	maxSelected := len(mv.modelList.Models) - 1
	if newSelected > maxSelected {
		newSelected = maxSelected
	}
	mv.modelList = mv.modelList.WithSelection(newSelected)
}

func (mv *ModelView) ensureSelectionVisible() {
	if mv.modelList.Selected < mv.modelList.Scroll {
		mv.modelList = mv.modelList.WithScroll(mv.modelList.Selected)
	} else if mv.modelList.Selected >= mv.modelList.Scroll+mv.modelList.Height {
		newScroll := mv.modelList.Selected - mv.modelList.Height + 1
		if newScroll < 0 {
			newScroll = 0
		}
		mv.modelList = mv.modelList.WithScroll(newScroll)
	}
}

func (mv *ModelView) changeModel() {
	if len(mv.modelList.Models) == 0 || mv.modelList.Selected < 0 || mv.modelList.Selected >= len(mv.modelList.Models) {
		return
	}

	selectedModel := mv.modelList.Models[mv.modelList.Selected]
	log := logger.WithComponent("model_view")
	log.Debug("Changing model", "selected_model", selectedModel.Name, "is_downloaded", selectedModel.IsDownloaded)

	// Check if model is downloaded
	if !selectedModel.IsDownloaded {
		log.Debug("Model not downloaded, showing download prompt", "model", selectedModel.Name)
		mv.downloadModal = mv.downloadModal.Show(selectedModel.Name)
		return
	}

	// Update the chat controller's model
	mv.chatController.SetModel(selectedModel.Name)

	// Update the configuration and save it
	viper.Set("ollama.model", selectedModel.Name)
	if err := viper.WriteConfig(); err != nil {
		log.Error("Failed to save configuration", "error", err)
		mv.status = mv.status.WithStatus("Error: Failed to save configuration - " + err.Error())
	} else {
		log.Debug("Configuration saved successfully", "new_model", selectedModel.Name)
		mv.status = mv.status.WithStatus("Model changed to: " + selectedModel.Name)

		// Post event to notify other views (especially ChatView) that the model changed
		mv.screen.PostEvent(NewModelChangeEvent(selectedModel.Name))

		// Trigger screen refresh to update the current model highlighting
		mv.screen.Show()
	}
}

func (mv *ModelView) showPullModal() {
	mv.pullModal = mv.pullModal.Show("Pull Model", "Enter model name to pull:")
}

func (mv *ModelView) pullModel(modelName string) {
	log := logger.WithComponent("model_view")
	log.Debug("Starting model pull", "model_name", modelName)

	mv.status = mv.status.WithStatus("Pulling model: " + modelName + "...")

	go func() {
		err := mv.controller.Pull(modelName)
		if err != nil {
			log.Error("Failed to pull model", "model_name", modelName, "error", err)
			mv.screen.PostEvent(NewModelErrorEvent(err))
		} else {
			log.Debug("Model pull completed successfully", "model_name", modelName)
			mv.status = mv.status.WithStatus("Model pulled successfully: " + modelName)
			mv.refreshModels()
		}
	}()
}

func (mv *ModelView) showDeleteConfirmation() {
	if len(mv.modelList.Models) == 0 || mv.modelList.Selected < 0 || mv.modelList.Selected >= len(mv.modelList.Models) {
		return
	}

	selectedModel := mv.modelList.Models[mv.modelList.Selected]
	title := "Delete Model"
	message := selectedModel.Name
	mv.confirmationModal = mv.confirmationModal.Show(title, message)
}

func (mv *ModelView) deleteModel() {
	if len(mv.modelList.Models) == 0 || mv.modelList.Selected < 0 || mv.modelList.Selected >= len(mv.modelList.Models) {
		return
	}

	selectedModel := mv.modelList.Models[mv.modelList.Selected]
	log := logger.WithComponent("model_view")
	log.Debug("Starting model deletion", "model_name", selectedModel.Name)

	mv.loading = true
	mv.status = mv.status.WithStatus("Deleting model: " + selectedModel.Name + "...")

	go func() {
		err := mv.controller.Delete(selectedModel.Name)
		if err != nil {
			log.Error("Failed to delete model", "model_name", selectedModel.Name, "error", err)
			mv.screen.PostEvent(NewModelErrorEvent(err))
		} else {
			log.Debug("Model deletion completed successfully", "model_name", selectedModel.Name)
			mv.screen.PostEvent(NewModelDeletedEvent(selectedModel.Name))
		}
	}()
}

func (mv *ModelView) renderModelInfo(screen tcell.Screen, area Rect) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	// Clear the area
	for x := area.X; x < area.X+area.Width; x++ {
		for y := area.Y; y < area.Y+area.Height; y++ {
			screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
	}

	// Left side: Model count
	totalSizeGB := float64(mv.modelStats.Stats.TotalSize) / (1024 * 1024 * 1024)
	leftText := fmt.Sprintf("models: %d | size: %.1f GB", mv.modelStats.Stats.TotalModels, totalSizeGB)
	leftStyle := StyleDimText

	for i, r := range leftText {
		if area.X+i < area.X+area.Width {
			screen.SetContent(area.X+i, area.Y, r, nil, leftStyle)
		}
	}

	// Right side: Current model name (only the name, no extra text)
	currentModelName := mv.chatController.GetModel()
	if currentModelName != "" {
		rightStyle := StyleModelCurrent

		// Right-justify the current model name
		if len(currentModelName) <= area.Width && len(leftText)+len(currentModelName)+4 <= area.Width { // Ensure spacing
			startX := area.X + area.Width - len(currentModelName)
			for i, r := range currentModelName {
				screen.SetContent(startX+i, area.Y, r, nil, rightStyle)
			}
		}
	}
}

func (mv *ModelView) renderHelpText(screen tcell.Screen, area Rect) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	// Clear the area
	for x := area.X; x < area.X+area.Width; x++ {
		for y := area.Y; y < area.Y+area.Height; y++ {
			screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
	}

	helpText := "[n] new model | [ctrl-d] delete | [d] details | [enter] select/download | [r] refresh | [j/k] navigate"
	helpStyle := StyleDimText

	// Center the help text
	if len(helpText) <= area.Width {
		startX := area.X + (area.Width-len(helpText))/2
		for i, r := range helpText {
			screen.SetContent(startX+i, area.Y, r, nil, helpStyle)
		}
	}
}

func (mv *ModelView) startModelDownload(modelName string) {
	log := logger.WithComponent("model_view")
	log.Debug("Starting model download", "model_name", modelName)

	// Hide download modal and show progress modal
	mv.downloadModal = mv.downloadModal.Hide()

	// Create cancellable context
	mv.downloadCtx, mv.downloadCancel = context.WithCancel(context.Background())

	// Show progress modal
	mv.progressModal = mv.progressModal.Show("Downloading Model", modelName, "Preparing download...", true)

	// Start download in goroutine
	go func() {
		var lastProgress float64 = 0.0
		err := mv.controller.PullWithProgress(mv.downloadCtx, modelName, func(status string, completed, total int64) {
			// Calculate progress
			progress := 0.0
			if total > 0 {
				progress = float64(completed) / float64(total)
			}

			// Smooth out progress updates - only update if progress is actually advancing
			// This prevents the modal from jumping back to 0% during different download phases
			if progress > lastProgress || status == "success" {
				lastProgress = progress
				// Post progress event
				mv.screen.PostEvent(NewModelDownloadProgressEvent(modelName, status, progress))
			}
		})

		if err != nil {
			if err == context.Canceled {
				log.Debug("Model download cancelled", "model_name", modelName)
				mv.screen.PostEvent(NewModelDownloadErrorEvent(modelName, err))
			} else {
				log.Error("Model download failed", "model_name", modelName, "error", err)
				mv.screen.PostEvent(NewModelDownloadErrorEvent(modelName, err))
			}
		} else {
			log.Debug("Model download completed successfully", "model_name", modelName)
			mv.screen.PostEvent(NewModelDownloadCompleteEvent(modelName))
		}
	}()
}

func (mv *ModelView) HandleModelDownloadProgress(ev ModelDownloadProgressEvent) {
	log := logger.WithComponent("model_view")
	log.Debug("Handling ModelDownloadProgressEvent", "model", ev.ModelName, "status", ev.Status, "progress", ev.Progress)

	mv.progressModal = mv.progressModal.WithProgress(ev.Progress, ev.Status).NextSpinnerFrame()
}

func (mv *ModelView) HandleModelDownloadComplete(ev ModelDownloadCompleteEvent) {
	log := logger.WithComponent("model_view")
	log.Debug("Handling ModelDownloadCompleteEvent", "model", ev.ModelName)

	// Hide progress modal (download modal already hidden when download started)
	mv.progressModal = mv.progressModal.Hide()
	mv.downloadCtx = nil
	mv.downloadCancel = nil

	// Update status
	mv.status = mv.status.WithStatus("Model downloaded successfully: " + ev.ModelName)

	// Mark this model to be selected after the refresh
	mv.selectAfterRefresh = ev.ModelName

	// Refresh models list to show the new model
	mv.refreshModels()

	// Set as current model
	mv.chatController.SetModel(ev.ModelName)
	viper.Set("ollama.model", ev.ModelName)
	if err := viper.WriteConfig(); err != nil {
		log.Error("Failed to save configuration after download", "error", err)
	}

	// Post event to notify other views that the model changed
	mv.screen.PostEvent(NewModelChangeEvent(ev.ModelName))
}

func (mv *ModelView) HandleModelDownloadError(ev ModelDownloadErrorEvent) {
	log := logger.WithComponent("model_view")
	log.Error("Handling ModelDownloadErrorEvent", "model", ev.ModelName, "error", ev.Error)

	// Hide progress modal (download modal already hidden when download started)
	mv.progressModal = mv.progressModal.Hide()
	mv.downloadCtx = nil
	mv.downloadCancel = nil

	// Update status with error
	if ev.Error == context.Canceled {
		mv.status = mv.status.WithStatus("Model download cancelled: " + ev.ModelName)
	} else {
		mv.status = mv.status.WithStatus("Model download failed: " + ev.Error.Error())
	}
}

func (mv *ModelView) showModelDetails() {
	if len(mv.modelList.Models) == 0 || mv.modelList.Selected < 0 || mv.modelList.Selected >= len(mv.modelList.Models) {
		return
	}

	mv.detailsModel = mv.modelList.Models[mv.modelList.Selected]
	mv.showingDetails = true
	mv.screen.Show()
}

func (mv *ModelView) renderModelDetailsView(screen tcell.Screen, area Rect) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	// Clear the area
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
	}

	// Draw border
	borderStyle := StyleBorder
	drawBorder(screen, area, borderStyle)

	// Calculate content area with padding
	contentArea := Rect{
		X:      area.X + 2,
		Y:      area.Y + 1,
		Width:  area.Width - 4,
		Height: area.Height - 2,
	}

	titleStyle := StyleHeaderText
	labelStyle := StyleMenuNormal
	valueStyle := StyleModelCurrent
	currentModelStyle := StyleHighlight

	// Title
	title := fmt.Sprintf("Model Details: %s", mv.detailsModel.Name)
	renderText(screen, contentArea.X, contentArea.Y, title, titleStyle)

	// Check if this is the current model
	currentModel := mv.chatController.GetModel()
	isCurrentModel := mv.detailsModel.Name == currentModel

	y := contentArea.Y + 2

	// Basic information
	sizeGB := float64(mv.detailsModel.Size) / (1024 * 1024 * 1024)

	// Get rich model information
	modelInfo := models.GetModelInfo(mv.detailsModel.Name)

	details := []struct {
		label string
		value string
		style tcell.Style
	}{
		{"Name:", mv.detailsModel.Name, valueStyle},
		{"Size:", func() string {
			if mv.detailsModel.Size > 0 {
				return fmt.Sprintf("%.2f GB (%d bytes)", sizeGB, mv.detailsModel.Size)
			}
			return "Unknown (not downloaded)"
		}(), valueStyle},
		{"Parameters:", mv.detailsModel.ParameterSize, valueStyle},
		{"Quantization:", mv.detailsModel.QuantizationLevel, valueStyle},
		{"Tool Support:", func() string {
			if modelInfo.ToolCompatibility != models.ToolCompatibilityUnknown {
				return modelInfo.ToolCompatibility.String()
			}
			return "Unknown"
		}(), func() tcell.Style {
			switch modelInfo.ToolCompatibility {
			case models.ToolCompatibilityExcellent:
				return tcell.StyleDefault.Foreground(tcell.ColorGreen)
			case models.ToolCompatibilityGood:
				return tcell.StyleDefault.Foreground(tcell.ColorYellow)
			case models.ToolCompatibilityBasic:
				return tcell.StyleDefault.Foreground(tcell.ColorOrange)
			case models.ToolCompatibilityNone:
				return tcell.StyleDefault.Foreground(tcell.ColorRed)
			default:
				return valueStyle
			}
		}()},
		{"Recommended:", func() string {
			if modelInfo.RecommendedForTools {
				return "Yes"
			}
			return "No"
		}(), func() tcell.Style {
			if modelInfo.RecommendedForTools {
				return tcell.StyleDefault.Foreground(tcell.ColorGreen)
			}
			return valueStyle
		}()},
		{"Status:", func() string {
			if !mv.detailsModel.IsDownloaded {
				return "Available for download"
			} else if mv.detailsModel.IsRunning {
				return "Running"
			}
			return "Downloaded"
		}(), valueStyle},
		{"Current Model:", func() string {
			if isCurrentModel {
				return "Yes"
			}
			return "No"
		}(), func() tcell.Style {
			if isCurrentModel {
				return currentModelStyle
			}
			return valueStyle
		}()},
		{"Modified:", func() string {
			if mv.detailsModel.IsDownloaded && !mv.detailsModel.ModifiedAt.IsZero() {
				return mv.detailsModel.ModifiedAt.Format("2006-01-02 15:04:05")
			}
			return "Not downloaded"
		}(), valueStyle},
	}

	for _, detail := range details {
		if y >= contentArea.Y+contentArea.Height-2 {
			break
		}

		// Render label
		renderText(screen, contentArea.X, y, detail.label, labelStyle)

		// Render value (offset to align values)
		valueX := contentArea.X + 15
		if valueX < contentArea.X+contentArea.Width {
			renderText(screen, valueX, y, detail.value, detail.style)
		}

		y += 2
	}

	// Add model notes if available
	if modelInfo.Notes != "" && y < contentArea.Y+contentArea.Height-3 {
		y++ // Add spacing
		notesTitle := "Notes:"
		renderText(screen, contentArea.X, y, notesTitle, labelStyle)
		y++

		// Wrap notes text to fit content area
		notesStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)
		maxWidth := contentArea.Width - 4
		notes := modelInfo.Notes

		for len(notes) > 0 && y < contentArea.Y+contentArea.Height-2 {
			lineLength := maxWidth
			if len(notes) < lineLength {
				lineLength = len(notes)
			} else {
				// Find last space before max width to avoid breaking words
				for lineLength > 0 && notes[lineLength] != ' ' {
					lineLength--
				}
				if lineLength == 0 {
					lineLength = maxWidth // Fallback if no space found
				}
			}

			line := notes[:lineLength]
			notes = notes[lineLength:]
			if len(notes) > 0 && notes[0] == ' ' {
				notes = notes[1:] // Remove leading space
			}

			renderText(screen, contentArea.X+2, y, line, notesStyle)
			y++
		}
	}

	// Instructions at bottom
	instructionY := contentArea.Y + contentArea.Height - 1
	instruction := "[ESC] Back to list"
	instrStyle := StyleDimText

	// Center the instruction
	if len(instruction) <= contentArea.Width {
		startX := contentArea.X + (contentArea.Width-len(instruction))/2
		renderText(screen, startX, instructionY, instruction, instrStyle)
	}
}
