package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/logger"
)

// State management methods for ChatView
// This file contains view state management, synchronization, and status updates

func (cv *ChatView) updateStatusForMode() {
	// Update the status to show the current mode with enhanced visual indicators
	var modeText string
	if cv.mode == ModeNodes {
		focusedNode := ""
		if cv.messages.NodeManager != nil {
			focusedNode = cv.messages.NodeManager.GetFocusedNode()
		}

		if focusedNode != "" {
			displayNode := focusedNode
			if len(focusedNode) > 8 {
				displayNode = focusedNode[:8]
			}
			modeText = fmt.Sprintf("🎯 Node Select | Focused: %s | j/k=nav, Tab=expand, Space=select, Esc/i=input", displayNode)
		} else {
			modeText = "🎯 Node Select | j/k=navigate, Tab=expand, Space=select, Esc/i=input"
		}
	} else {
		modeText = "✏️ Input | Ctrl+N=node mode, F2=help"
	}
	cv.status = cv.status.WithStatus(modeText)
}

func (cv *ChatView) sendMessage() string {
	log := logger.WithComponent("chat_view")
	content := strings.TrimSpace(cv.input.Content)
	log.Debug("sendMessage called", "content", content, "length", len(content))

	if content == "" {
		log.Debug("Empty message, skipping send")
		return ""
	}

	// Check if current model is available
	currentModel := cv.controller.GetModel()
	if err := cv.controller.ValidateModel(currentModel); err != nil {
		log.Debug("Model not available, showing download prompt", "model", currentModel, "error", err)
		// Store the message to send after download
		cv.pendingMessage = content
		cv.input = cv.input.Clear()
		cv.downloadModal = cv.downloadModal.Show(currentModel)
		return ""
	}

	cv.input = cv.input.Clear()
	log.Debug("Message content prepared for send", "content", content)

	return content
}

func (cv *ChatView) HandleMessageResponse(response MessageResponseEvent) {
	cv.status = cv.status.WithStatus("Ready")
	cv.alert = cv.alert.Clear()

	// Update token information
	promptTokens, responseTokens := cv.controller.GetTokenUsage()
	cv.status = cv.status.WithTokens(promptTokens, responseTokens)

	cv.updateMessages()
	cv.scrollToBottom()
}

func (cv *ChatView) HandleMessageError(error MessageErrorEvent) {
	cv.status = cv.status.WithStatus("Ready") // Keep status simple
	// Don't set alert error - only show error in chat messages
	cv.alert = cv.alert.Clear()

	// Update messages to show the error message that was added to conversation
	cv.updateMessages()
	cv.scrollToBottom()
}

func (cv *ChatView) SyncWithAppState(sending bool) {
	log := logger.WithComponent("chat_view")
	log.Debug("Syncing ChatView state", "app_sending", sending)

	if sending {
		cv.alert = cv.alert.WithSpinner(true, "")
		cv.statusRow = cv.statusRow.WithSpinner(true, "Sending...")
	} else {
		// Always clear alert since errors only show in chat messages now
		cv.alert = cv.alert.Clear()
		cv.statusRow = cv.statusRow.ClearSpinnerOnly() // Preserve token count
	}
}

func (cv *ChatView) UpdateSpinnerFrame() {
	cv.alert = cv.alert.NextSpinnerFrame()
	cv.statusRow = cv.statusRow.NextSpinnerFrame()
}

func (cv *ChatView) updateMessages() {
	history := cv.controller.GetHistory()
	// Filter out system messages - they should not be displayed to the user
	var filteredHistory []chat.Message
	for _, msg := range history {
		if msg.Role != chat.RoleSystem {
			filteredHistory = append(filteredHistory, msg)
		}
	}
	cv.messages = cv.messages.WithMessages(filteredHistory)
}

func (cv *ChatView) HandleModelChange(ev ModelChangeEvent) {
	log := logger.WithComponent("chat_view")
	log.Debug("Handling ModelChangeEvent in chat view", "model_name", ev.ModelName)

	// Update the status bar to show the new model
	cv.status = cv.status.WithModel(ev.ModelName)
	log.Debug("Updated chat view status bar with new model", "model_name", ev.ModelName)
}

func (cv *ChatView) startModelDownload(modelName string) {
	log := logger.WithComponent("chat_view")
	log.Debug("Starting model download from chat view", "model_name", modelName)

	// Create cancellable context
	cv.downloadCtx, cv.downloadCancel = context.WithCancel(context.Background())

	// Show progress modal
	cv.progressModal = cv.progressModal.Show("Downloading Model", modelName, "Preparing download...", true)

	// Start download in goroutine
	go func() {
		var lastProgress float64 = 0.0
		err := cv.modelsController.PullWithProgress(cv.downloadCtx, modelName, func(status string, completed, total int64) {
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
				cv.screen.PostEvent(NewModelDownloadProgressEvent(modelName, status, progress))
			}
		})

		if err != nil {
			if err == context.Canceled {
				log.Debug("Model download cancelled in chat view", "model_name", modelName)
				cv.screen.PostEvent(NewModelDownloadErrorEvent(modelName, err))
			} else {
				log.Error("Model download failed in chat view", "model_name", modelName, "error", err)
				cv.screen.PostEvent(NewModelDownloadErrorEvent(modelName, err))
			}
		} else {
			log.Debug("Model download completed successfully in chat view", "model_name", modelName)
			cv.screen.PostEvent(NewModelDownloadCompleteEvent(modelName))
		}
	}()
}

func (cv *ChatView) HandleModelDownloadProgress(ev ModelDownloadProgressEvent) {
	log := logger.WithComponent("chat_view")
	log.Debug("Handling ModelDownloadProgressEvent in chat view", "model", ev.ModelName, "status", ev.Status, "progress", ev.Progress)

	cv.progressModal = cv.progressModal.WithProgress(ev.Progress, ev.Status).NextSpinnerFrame()
}

func (cv *ChatView) HandleModelDownloadComplete(ev ModelDownloadCompleteEvent) {
	log := logger.WithComponent("chat_view")
	log.Debug("Handling ModelDownloadCompleteEvent in chat view", "model", ev.ModelName)

	// Hide progress modal
	cv.progressModal = cv.progressModal.Hide()
	cv.downloadCtx = nil
	cv.downloadCancel = nil

	// Update status
	cv.status = cv.status.WithStatus("Model downloaded successfully: " + ev.ModelName)

	// Set as current model
	cv.controller.SetModel(ev.ModelName)
	cv.status = cv.status.WithModel(ev.ModelName)

	// Force screen refresh to update UI immediately (hide modal)
	cv.screen.Show()

	// Send the pending message if we have one
	if cv.pendingMessage != "" {
		log.Debug("Sending pending message after download", "message", cv.pendingMessage)
		cv.screen.PostEvent(NewChatMessageSendEvent(cv.pendingMessage))
		cv.pendingMessage = ""
	}
}

func (cv *ChatView) HandleModelDownloadError(ev ModelDownloadErrorEvent) {
	log := logger.WithComponent("chat_view")
	log.Error("Handling ModelDownloadErrorEvent in chat view", "model", ev.ModelName, "error", ev.Error)

	// Hide progress modal
	cv.progressModal = cv.progressModal.Hide()
	cv.downloadCtx = nil
	cv.downloadCancel = nil

	// Clear pending message since download failed
	cv.pendingMessage = ""

	// Update status with error
	if ev.Error == context.Canceled {
		cv.status = cv.status.WithStatus("Model download cancelled: " + ev.ModelName)
	} else {
		cv.status = cv.status.WithStatus("Model download failed: " + ev.Error.Error())
	}

	// Force screen refresh to update UI immediately (hide modal)
	cv.screen.Show()
}
