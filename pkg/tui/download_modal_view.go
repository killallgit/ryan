package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
)

type DownloadModalView struct {
	modelsController *controllers.ModelsController
	chatController   *controllers.ChatController
	screen           tcell.Screen
	textInputModal   TextInputModal
	downloadModal    DownloadPromptModal
	progressModal    ProgressModal
	errorModal       ModalDialog
	width            int
	height           int
	instructions     []string
	isDownloading    bool
	currentModelName string
}

func NewDownloadModalView(modelsController *controllers.ModelsController, chatController *controllers.ChatController, screen tcell.Screen) *DownloadModalView {
	instructions := []string{
		"Download Models",
		"",
		"Press Enter to specify a model to download",
		"",
		"Popular models:",
		"  • llama3.2:3b    - Fast, lightweight model",
		"  • llama3.2:1b    - Ultra-fast, minimal model",
		"  • qwen2.5:7b     - High-quality general purpose",
		"  • phi3:mini      - Microsoft's efficient model",
		"  • gemma2:2b      - Google's compact model",
		"",
		"Enter model name in format: name:tag (e.g., llama3.2:3b)",
		"",
		"Controls:",
		"  Enter    - Open download dialog",
		"  Esc      - Return to previous view",
		"  Ctrl+P   - Toggle command palette",
	}

	return &DownloadModalView{
		modelsController: modelsController,
		chatController:   chatController,
		screen:           screen,
		textInputModal:   NewTextInputModal(),
		downloadModal:    NewDownloadPromptModal(),
		progressModal:    NewProgressModal(),
		errorModal:       NewModalDialog(),
		width:            80,
		height:           24,
		instructions:     instructions,
		isDownloading:    false,
		currentModelName: "",
	}
}

func (dmv *DownloadModalView) Name() string {
	return "download"
}

func (dmv *DownloadModalView) Description() string {
	return "Download Models - Download new AI models"
}

func (dmv *DownloadModalView) Render(screen tcell.Screen, area Rect) {
	dmv.width = area.Width
	dmv.height = area.Height

	// Clear the area
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
	}

	// Render main content
	dmv.renderInstructions(screen, area)

	// Render modals on top
	dmv.textInputModal.Render(screen, area)
	dmv.downloadModal.Render(screen, area)
	dmv.progressModal.Render(screen, area)
	dmv.errorModal.Render(screen, area)
}

func (dmv *DownloadModalView) renderInstructions(screen tcell.Screen, area Rect) {
	// Title style
	titleStyle := StyleHighlight.Bold(true)
	instructionStyle := tcell.StyleDefault.Foreground(ColorMenuNormal)
	exampleStyle := tcell.StyleDefault.Foreground(ColorModelName)
	controlStyle := StyleDimText

	y := area.Y + 2
	for i, line := range dmv.instructions {
		if y >= area.Y+area.Height-2 {
			break
		}

		style := instructionStyle
		if i == 0 {
			// Title
			style = titleStyle
			x := area.X + (area.Width-len(line))/2
			if x < area.X {
				x = area.X
			}
			renderTextWithLimit(screen, x, y, area.Width, line, style)
		} else if strings.HasPrefix(line, "  •") {
			// Example models
			style = exampleStyle
			renderTextWithLimit(screen, area.X+4, y, area.Width-4, line, style)
		} else if strings.HasPrefix(line, "Controls:") || strings.HasPrefix(line, "  Enter") || strings.HasPrefix(line, "  Esc") || strings.HasPrefix(line, "  Ctrl+P") {
			// Control instructions
			style = controlStyle
			renderTextWithLimit(screen, area.X+2, y, area.Width-2, line, style)
		} else if line != "" {
			// Regular instructions
			renderTextWithLimit(screen, area.X+2, y, area.Width-2, line, style)
		}

		y++
	}

	// Add status if downloading
	if dmv.isDownloading {
		statusLine := fmt.Sprintf("Currently downloading: %s", dmv.currentModelName)
		statusStyle := tcell.StyleDefault.Foreground(ColorProgressBar)
		renderTextWithLimit(screen, area.X+2, area.Y+area.Height-3, area.Width-4, statusLine, statusStyle)
	}
}

func (dmv *DownloadModalView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool {
	log := logger.WithComponent("download_modal_view")

	// Handle modal key events first
	if dmv.errorModal.Visible {
		dmv.errorModal = dmv.errorModal.Hide()
		return true
	}

	// Handle text input modal
	if dmv.textInputModal.Visible {
		var content string
		var submitted bool
		dmv.textInputModal, content, submitted = dmv.textInputModal.HandleKeyEvent(ev)

		if submitted && content != "" {
			// Show download confirmation modal
			dmv.downloadModal = dmv.downloadModal.Show(content)
			dmv.currentModelName = content
		}
		return true
	}

	// Handle download confirmation modal
	if dmv.downloadModal.Visible {
		var confirmed bool
		dmv.downloadModal, confirmed, _ = dmv.downloadModal.HandleKeyEvent(ev)

		if confirmed {
			dmv.startDownload(dmv.currentModelName)
		} else if !dmv.downloadModal.Visible {
			dmv.currentModelName = ""
		}
		return true
	}

	// Handle progress modal
	if dmv.progressModal.Visible {
		var cancelled bool
		dmv.progressModal, cancelled = dmv.progressModal.HandleKeyEvent(ev)

		if cancelled {
			dmv.cancelDownload()
		}
		return true
	}

	// Handle main view key events
	switch ev.Key() {
	case tcell.KeyEnter:
		// Open text input modal for model name
		dmv.textInputModal = dmv.textInputModal.Show("Download Model", "Enter model name (e.g., llama3.2:3b):")
		log.Debug("Opened text input modal for model download")
		return true

	case tcell.KeyEscape:
		// Let the app handle this to return to previous view
		return false

	default:
		// Handle character input for quick actions
		if ev.Rune() != 0 {
			switch ev.Rune() {
			case 'd', 'D':
				// Quick download shortcut
				dmv.textInputModal = dmv.textInputModal.Show("Download Model", "Enter model name (e.g., llama3.2:3b):")
				log.Debug("Opened text input modal via 'd' shortcut")
				return true
			}
		}
	}

	return false
}

func (dmv *DownloadModalView) HandleResize(width, height int) {
	dmv.width = width
	dmv.height = height
}

func (dmv *DownloadModalView) startDownload(modelName string) {
	log := logger.WithComponent("download_modal_view")
	log.Debug("Starting model download", "model_name", modelName)

	dmv.isDownloading = true
	dmv.currentModelName = modelName

	// Show progress modal
	dmv.progressModal = dmv.progressModal.Show("Downloading Model", modelName, "Initializing download...", true)

	// Start download in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("Panic in download goroutine", "panic", r)
				dmv.screen.PostEvent(NewModelDownloadErrorEvent(modelName, fmt.Errorf("download panic: %v", r)))
			}
		}()

		ctx := context.Background()
		err := dmv.modelsController.PullWithProgress(ctx, modelName, func(status string, completed, total int64) {
			// Calculate progress as a percentage
			progress := 0.0
			if total > 0 {
				progress = float64(completed) / float64(total)
			}
			// Post progress update event
			dmv.screen.PostEvent(NewModelDownloadProgressEvent(modelName, status, progress))
		})

		if err != nil {
			log.Error("Model download failed", "model_name", modelName, "error", err)
			dmv.screen.PostEvent(NewModelDownloadErrorEvent(modelName, err))
		} else {
			log.Debug("Model download completed", "model_name", modelName)
			dmv.screen.PostEvent(NewModelDownloadCompleteEvent(modelName))
		}
	}()
}

func (dmv *DownloadModalView) cancelDownload() {
	log := logger.WithComponent("download_modal_view")
	log.Debug("Cancelling model download", "model_name", dmv.currentModelName)

	dmv.isDownloading = false
	dmv.progressModal = dmv.progressModal.Hide()
	dmv.currentModelName = ""

	// TODO: Implement actual download cancellation in modelsController
	// For now, just hide the progress modal
}

// Event handlers for download events
func (dmv *DownloadModalView) HandleModelDownloadProgress(ev ModelDownloadProgressEvent) {
	if dmv.currentModelName == ev.ModelName && dmv.progressModal.Visible {
		dmv.progressModal = dmv.progressModal.WithProgress(ev.Progress, ev.Status)
		dmv.progressModal = dmv.progressModal.NextSpinnerFrame()
	}
}

func (dmv *DownloadModalView) HandleModelDownloadComplete(ev ModelDownloadCompleteEvent) {
	log := logger.WithComponent("download_modal_view")
	log.Debug("Download completed", "model_name", ev.ModelName)

	if dmv.currentModelName == ev.ModelName {
		dmv.isDownloading = false
		dmv.progressModal = dmv.progressModal.Hide()

		// Show success message
		successMsg := fmt.Sprintf("Successfully downloaded model: %s\n\nYou can now use this model in chat by selecting it in the models view.", ev.ModelName)
		dmv.errorModal = dmv.errorModal.WithError("Download Complete", successMsg)

		dmv.currentModelName = ""
	}
}

func (dmv *DownloadModalView) HandleModelDownloadError(ev ModelDownloadErrorEvent) {
	log := logger.WithComponent("download_modal_view")
	log.Error("Download failed", "model_name", ev.ModelName, "error", ev.Error)

	if dmv.currentModelName == ev.ModelName {
		dmv.isDownloading = false
		dmv.progressModal = dmv.progressModal.Hide()

		// Show error message
		errorMsg := fmt.Sprintf("Failed to download model: %s\n\nError: %v\n\nPlease check your internet connection and try again.", ev.ModelName, ev.Error)
		dmv.errorModal = dmv.errorModal.WithError("Download Failed", errorMsg)

		dmv.currentModelName = ""
	}
}
