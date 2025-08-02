package tui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/spf13/viper"
)

type ModelView struct {
	controller        *controllers.ModelsController
	chatController    *controllers.ChatController
	modelList         ModelListDisplay
	modelStats        ModelStatsDisplay
	status            StatusBar
	layout            Layout
	loading           bool
	screen            tcell.Screen
	showStats         bool
	pullModal         TextInputModal
	confirmationModal ConfirmationModal
	downloadModal     DownloadPromptModal
	progressModal     ProgressModal
	downloadCtx       context.Context
	downloadCancel    context.CancelFunc
}

func NewModelView(controller *controllers.ModelsController, chatController *controllers.ChatController, screen tcell.Screen) *ModelView {
	width, height := screen.Size()

	log := logger.WithComponent("model_view")
	log.Debug("Creating new ModelView", "width", width, "height", height)

	view := &ModelView{
		controller:        controller,
		chatController:    chatController,
		modelList:         NewModelListDisplay(width, height-6),
		modelStats:        NewModelStatsDisplay(width, 4),
		status:            NewStatusBar(width).WithStatus("Ready").WithModelViewData(0, 0),
		layout:            NewLayout(width, height),
		loading:           false,
		screen:            screen,
		showStats:         true,
		pullModal:         NewTextInputModal(),
		confirmationModal: NewConfirmationModal(),
		downloadModal:     NewDownloadPromptModal(),
		progressModal:     NewProgressModal(),
		downloadCtx:       nil,
		downloadCancel:    nil,
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
	case tcell.KeyUp, tcell.KeyCtrlP:
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
		models := convertOllamaModelsToModelInfoWithStatus(response.Models, runningModelNames)
		log.Debug("Converted models to UI format", "total_models", len(models))

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
	mv.refreshModels()
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
	log.Debug("Changing model", "selected_model", selectedModel.Name)

	// Check if model exists locally, if not, show download prompt
	err := mv.chatController.ValidateModel(selectedModel.Name)
	if err != nil {
		log.Debug("Model not found locally, showing download prompt", "model", selectedModel.Name, "error", err)
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

	helpText := "[n] new model | [ctrl-d] delete | [enter] select | [r] refresh | [j/k] navigate"
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

	// Create cancellable context
	mv.downloadCtx, mv.downloadCancel = context.WithCancel(context.Background())

	// Show progress modal
	mv.progressModal = mv.progressModal.Show("Downloading Model", modelName, "Preparing download...", true)

	// Start download in goroutine
	go func() {
		err := mv.controller.PullWithProgress(mv.downloadCtx, modelName, func(status string, completed, total int64) {
			// Calculate progress
			progress := 0.0
			if total > 0 {
				progress = float64(completed) / float64(total)
			}

			// Post progress event
			mv.screen.PostEvent(NewModelDownloadProgressEvent(modelName, status, progress))
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

	// Hide progress modal
	mv.progressModal = mv.progressModal.Hide()
	mv.downloadCtx = nil
	mv.downloadCancel = nil

	// Update status
	mv.status = mv.status.WithStatus("Model downloaded successfully: " + ev.ModelName)

	// Refresh models list to show the new model
	mv.refreshModels()

	// Set as current model
	mv.chatController.SetModel(ev.ModelName)
	viper.Set("ollama.model", ev.ModelName)
	if err := viper.WriteConfig(); err != nil {
		log.Error("Failed to save configuration after download", "error", err)
	}
}

func (mv *ModelView) HandleModelDownloadError(ev ModelDownloadErrorEvent) {
	log := logger.WithComponent("model_view")
	log.Error("Handling ModelDownloadErrorEvent", "model", ev.ModelName, "error", ev.Error)

	// Hide progress modal
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
