package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/ollama"
)

// startPull returns a command to start pulling a model
func (v ModelsView) startPull(modelName string) tea.Cmd {
	return func() tea.Msg {
		progressChan, errorChan, err := v.apiClient.PullModel(modelName)
		if err != nil {
			return pullCompleteMsg{success: false, err: err}
		}

		// Return initial message and start progress monitoring
		return startPullMsg{
			progressChan: progressChan,
			errorChan:    errorChan,
		}
	}
}

// spinnerTick returns a command that sends a spinner tick after a delay
func spinnerTick() tea.Cmd {
	return tea.Tick(time.Millisecond*150, func(t time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

// autoRefreshTick returns a command that triggers periodic model list refresh
func autoRefreshTick() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return autoRefreshMsg{}
	})
}

// waitForProgress waits for the next progress update
func waitForProgress(progressChan <-chan ollama.PullProgress, errorChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		// Return immediately if both channels are nil
		if progressChan == nil && errorChan == nil {
			return pullCompleteMsg{success: true}
		}

		select {
		case progress, ok := <-progressChan:
			if !ok {
				// Progress channel closed - download completed
				return pullCompleteMsg{success: true}
			}
			if progress.Status == "success" {
				return pullCompleteMsg{success: true}
			}
			return pullProgressMsg{progress: progress}
		case err, ok := <-errorChan:
			if !ok {
				// Error channel closed without error - this is normal completion
				return pullCompleteMsg{success: true}
			}
			if err != nil {
				// Actual error occurred
				return pullCompleteMsg{success: false, err: err}
			}
			// err is nil but channel is still open - this shouldn't happen in normal flow
			// but if it does, treat it as an error condition
			return pullCompleteMsg{success: false, err: fmt.Errorf("unexpected nil error from error channel")}
		}
	}
}

// handleDownloadStart initiates a new download
func (v *ModelsView) handleDownloadStart(modelName string) (tea.Model, tea.Cmd) {
	logger.Debug("Starting download for model: %s", modelName)
	v.downloadActive = true
	v.downloadingModel = modelName
	v.progressPercent = 0
	v.progressStatus = "Starting download..."
	v.errorMessage = ""
	v.spinnerFrame = 0 // Reset spinner animation
	// Start both download and spinner animation
	return v, tea.Batch(v.startPull(modelName), spinnerTick())
}

// handleDownloadCancel cancels an active download
func (v *ModelsView) handleDownloadCancel() {
	if v.downloadActive {
		logger.Debug("Cancelling download of model: %s", v.downloadingModel)
		// Cancel the download by resetting all download state
		v.downloadActive = false
		v.progressChan = nil
		v.errorChan = nil
		v.downloadingModel = ""
		v.progressPercent = 0
		v.progressStatus = ""
		v.errorMessage = "Download cancelled by user"
	}
}

// handleDownloadCancelFromList cancels download and updates table
func (v *ModelsView) handleDownloadCancelFromList() {
	if v.downloadActive {
		logger.Debug("Cancelling download of model: %s", v.downloadingModel)
		v.downloadActive = false
		v.progressChan = nil
		v.errorChan = nil
		v.downloadingModel = ""
		v.progressPercent = 0
		v.progressStatus = ""
		// Update table to remove downloading model
		v.updateTable()
	}
}

// handlePullProgress processes download progress updates
func (v *ModelsView) handlePullProgress(msg pullProgressMsg) (tea.Model, tea.Cmd) {
	if v.downloadActive {
		// Update progress
		progress := msg.progress
		logger.Debug("TUI received progress: status=%s, total=%d, completed=%d", progress.Status, progress.Total, progress.Completed)
		v.progressStatus = progress.Status
		if progress.Total > 0 {
			v.progressPercent = float64(progress.Completed) / float64(progress.Total)
		}
		// Update table to show spinner
		v.updateTable()
		// Continue waiting for more updates
		if v.progressChan != nil || v.errorChan != nil {
			return v, waitForProgress(v.progressChan, v.errorChan)
		}
	}
	return v, nil
}

// handlePullComplete processes download completion
func (v *ModelsView) handlePullComplete(msg pullCompleteMsg) (tea.Model, tea.Cmd) {
	v.downloadActive = false
	v.progressChan = nil
	v.errorChan = nil
	v.downloadingModel = ""
	if msg.success {
		// Don't automatically close modal on success, let user see completion
		v.progressStatus = "Download completed successfully! Press ESC to close."
		v.errorMessage = ""
		// Refresh models list in background
		v.loading = true
		return v, v.fetchModels()
	} else {
		v.errorMessage = fmt.Sprintf("Download failed: %v", msg.err)
		v.progressStatus = ""
	}
	return v, nil
}

// handleStartPull processes initial download setup
func (v *ModelsView) handleStartPull(msg startPullMsg) (tea.Model, tea.Cmd) {
	// Store channels and start monitoring
	v.progressChan = msg.progressChan
	v.errorChan = msg.errorChan
	return v, waitForProgress(msg.progressChan, msg.errorChan)
}

// handleSpinnerTick processes spinner animation
func (v *ModelsView) handleSpinnerTick() (tea.Model, tea.Cmd) {
	// Animate spinner only if download is active
	if v.downloadActive {
		v.spinnerFrame++
		v.updateTable()
		return v, spinnerTick() // Schedule next animation frame
	}
	return v, nil
}

// handleAutoRefresh processes automatic refresh
func (v *ModelsView) handleAutoRefresh() (tea.Model, tea.Cmd) {
	// Periodically refresh models list (especially useful when downloads complete)
	if v.autoRefreshEnabled {
		// Only refresh if not actively downloading to avoid interfering with progress
		if !v.downloadActive {
			return v, tea.Batch(v.fetchModels(), autoRefreshTick())
		} else {
			// If downloading, just schedule next refresh
			return v, autoRefreshTick()
		}
	}
	return v, nil
}

// handleDownloadModalKeys processes download modal key inputs
func (v *ModelsView) handleDownloadModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Always allow ESC to close modal, but keep download running
		v.modalType = ModalNone
		v.textInput.SetValue("")
		// Don't reset downloadActive, progressChan, errorChan - keep download running
	case "ctrl+d":
		// Cancel download if active, otherwise just close modal
		if v.downloadActive {
			v.handleDownloadCancel()
			// Keep modal open to show cancellation message
		} else {
			// No active download, just close modal
			v.modalType = ModalNone
			v.textInput.SetValue("")
		}
	case "enter":
		if !v.downloadActive {
			modelName := strings.TrimSpace(v.textInput.Value())
			if modelName != "" {
				return v.handleDownloadStart(modelName)
			}
		}
	default:
		if !v.downloadActive {
			var cmd tea.Cmd
			v.textInput, cmd = v.textInput.Update(msg)
			return v, cmd
		}
	}
	return v, nil
}
