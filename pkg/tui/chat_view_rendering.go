package tui

import (
	"github.com/gdamore/tcell/v2"
)

// UI rendering methods for ChatView
// This file contains rendering coordination and UI layout logic

func (cv *ChatView) Render(screen tcell.Screen, area Rect) {
	messageArea, alertArea, inputArea, statusArea := cv.layout.CalculateAreas()

	// Use streaming-aware render function that can apply thinking styles during streaming
	spinner := SpinnerComponent{
		IsVisible: cv.alert.IsSpinnerVisible,
		Frame:     cv.alert.SpinnerFrame,
		Text:      cv.alert.SpinnerText,
	}
	RenderMessagesWithStreamingState(screen, cv.messages, messageArea, spinner, cv.isStreamingThinking)

	// Update status row with current token count and render it
	// Get the most current token count from both status and controller
	statusTokens := cv.status.PromptTokens + cv.status.ResponseTokens
	promptTokens, responseTokens := cv.controller.GetTokenUsage()
	controllerTokens := promptTokens + responseTokens

	// Use whichever is higher (more recent)
	totalTokens := statusTokens
	if controllerTokens > statusTokens {
		totalTokens = controllerTokens
	}

	cv.statusRow = cv.statusRow.WithTokens(totalTokens).UpdateDuration()
	RenderStatusRow(screen, alertArea, cv.statusRow)

	RenderInput(screen, cv.input, inputArea)
	RenderStatus(screen, cv.status, statusArea)

	// Render modals on top
	cv.downloadModal.Render(screen, area)
	cv.progressModal.Render(screen, area)
	cv.helpModal.Render(screen, area)
}
