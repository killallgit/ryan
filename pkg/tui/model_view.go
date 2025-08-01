package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/spf13/viper"
)

type ModelView struct {
	controller     *controllers.ModelsController
	chatController *controllers.ChatController
	modelList      ModelListDisplay
	modelStats     ModelStatsDisplay
	status         StatusBar
	layout         Layout
	loading        bool
	screen         tcell.Screen
	showStats      bool
	pullModal      TextInputModal
}

func NewModelView(controller *controllers.ModelsController, chatController *controllers.ChatController, screen tcell.Screen) *ModelView {
	width, height := screen.Size()

	log := logger.WithComponent("model_view")
	log.Debug("Creating new ModelView", "width", width, "height", height)

	view := &ModelView{
		controller:     controller,
		chatController: chatController,
		modelList:      NewModelListDisplay(width, height-6),
		modelStats:     NewModelStatsDisplay(width, 4),
		status:         NewStatusBar(width).WithStatus("Ready"),
		layout:         NewLayout(width, height),
		loading:        false,
		screen:         screen,
		showStats:      true,
		pullModal:      NewTextInputModal(),
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
	statsHeight := 6
	if !mv.showStats {
		statsHeight = 0
	}

	listArea := Rect{
		X:      area.X,
		Y:      area.Y,
		Width:  area.Width,
		Height: area.Height - statsHeight - 2, // -2 for status bar
	}

	statusArea := Rect{
		X:      area.X,
		Y:      area.Y + area.Height - 1,
		Width:  area.Width,
		Height: 1,
	}

	currentModel := mv.chatController.GetModel()
	RenderModelListWithCurrentModel(screen, mv.modelList, listArea, currentModel)

	if mv.showStats {
		statsArea := Rect{
			X:      area.X,
			Y:      listArea.Y + listArea.Height,
			Width:  area.Width,
			Height: statsHeight - 1,
		}
		RenderModelStats(screen, mv.modelStats, statsArea)
	}

	RenderStatus(screen, mv.status, statusArea)
	
	mv.pullModal.Render(screen, area)
}

func (mv *ModelView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool {
	if mv.loading {
		return true // Consume all events while loading
	}
	
	// Handle modal events first
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

			case 's', 'S':
				mv.toggleStats()
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

	statsHeight := 6
	if !mv.showStats {
		statsHeight = 0
	}

	mv.modelList = mv.modelList.WithSize(width, height-statsHeight-2)
	mv.modelStats = mv.modelStats.WithSize(width, statsHeight-1)
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
}

func (mv *ModelView) HandleModelError(ev ModelErrorEvent) {
	log := logger.WithComponent("model_view")
	log.Error("Handling ModelErrorEvent", "error", ev.Error)

	mv.loading = false
	mv.status = mv.status.WithStatus("Error: " + ev.Error.Error() + " - Press 'r' to retry")
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

func (mv *ModelView) toggleStats() {
	mv.showStats = !mv.showStats
	width, height := mv.screen.Size()
	mv.HandleResize(width, height)
}

func (mv *ModelView) changeModel() {
	if len(mv.modelList.Models) == 0 || mv.modelList.Selected < 0 || mv.modelList.Selected >= len(mv.modelList.Models) {
		return
	}

	selectedModel := mv.modelList.Models[mv.modelList.Selected]
	log := logger.WithComponent("model_view")
	log.Debug("Changing model", "selected_model", selectedModel.Name)

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
