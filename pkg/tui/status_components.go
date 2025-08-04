package tui

import (
	"time"
)

type StatusBar struct {
	Model          string
	Status         string
	Width          int
	PromptTokens   int
	ResponseTokens int
	ModelAvailable bool
	// Model management view specific fields
	IsModelView bool
	TotalModels int
	TotalSize   int64
}

func NewStatusBar(width int) StatusBar {
	return StatusBar{
		Model:          "",
		Status:         "Ready",
		Width:          width,
		PromptTokens:   0,
		ResponseTokens: 0,
		ModelAvailable: true,
		IsModelView:    false,
		TotalModels:    0,
		TotalSize:      0,
	}
}

func (sb StatusBar) WithModel(model string) StatusBar {
	return StatusBar{
		Model:          model,
		Status:         sb.Status,
		Width:          sb.Width,
		PromptTokens:   sb.PromptTokens,
		ResponseTokens: sb.ResponseTokens,
		ModelAvailable: sb.ModelAvailable,
		IsModelView:    sb.IsModelView,
		TotalModels:    sb.TotalModels,
		TotalSize:      sb.TotalSize,
	}
}

func (sb StatusBar) WithStatus(status string) StatusBar {
	return StatusBar{
		Model:          sb.Model,
		Status:         status,
		Width:          sb.Width,
		PromptTokens:   sb.PromptTokens,
		ResponseTokens: sb.ResponseTokens,
		ModelAvailable: sb.ModelAvailable,
		IsModelView:    sb.IsModelView,
		TotalModels:    sb.TotalModels,
		TotalSize:      sb.TotalSize,
	}
}

func (sb StatusBar) WithWidth(width int) StatusBar {
	return StatusBar{
		Model:          sb.Model,
		Status:         sb.Status,
		Width:          width,
		PromptTokens:   sb.PromptTokens,
		ResponseTokens: sb.ResponseTokens,
		ModelAvailable: sb.ModelAvailable,
		IsModelView:    sb.IsModelView,
		TotalModels:    sb.TotalModels,
		TotalSize:      sb.TotalSize,
	}
}

func (sb StatusBar) WithTokens(promptTokens, responseTokens int) StatusBar {
	return StatusBar{
		Model:          sb.Model,
		Status:         sb.Status,
		Width:          sb.Width,
		PromptTokens:   promptTokens,
		ResponseTokens: responseTokens,
		ModelAvailable: sb.ModelAvailable,
		IsModelView:    sb.IsModelView,
		TotalModels:    sb.TotalModels,
		TotalSize:      sb.TotalSize,
	}
}

func (sb StatusBar) WithModelAvailability(available bool) StatusBar {
	return StatusBar{
		Model:          sb.Model,
		Status:         sb.Status,
		Width:          sb.Width,
		PromptTokens:   sb.PromptTokens,
		ResponseTokens: sb.ResponseTokens,
		ModelAvailable: available,
		IsModelView:    sb.IsModelView,
		TotalModels:    sb.TotalModels,
		TotalSize:      sb.TotalSize,
	}
}

func (sb StatusBar) WithModelViewData(totalModels int, totalSize int64) StatusBar {
	return StatusBar{
		Model:          sb.Model,
		Status:         sb.Status,
		Width:          sb.Width,
		PromptTokens:   sb.PromptTokens,
		ResponseTokens: sb.ResponseTokens,
		ModelAvailable: sb.ModelAvailable,
		IsModelView:    true,
		TotalModels:    totalModels,
		TotalSize:      totalSize,
	}
}

type AlertDisplay struct {
	IsSpinnerVisible bool
	SpinnerFrame     int
	SpinnerText      string
	ErrorMessage     string
	Width            int
	StartTime        time.Time     // Track when operation started
	CurrentDuration  time.Duration // Current operation duration
}

// Enhanced status row component that replaces alert area
type StatusRowDisplay struct {
	IsSpinnerVisible bool
	SpinnerFrame     int
	FeedbackText     string
	StartTime        time.Time
	CurrentDuration  time.Duration
	TokenCount       int
	Width            int
}

func NewAlertDisplay(width int) AlertDisplay {
	return AlertDisplay{
		IsSpinnerVisible: false,
		SpinnerFrame:     0,
		SpinnerText:      "",
		ErrorMessage:     "",
		Width:            width,
	}
}

func (ad AlertDisplay) WithSpinner(visible bool, text string) AlertDisplay {
	return AlertDisplay{
		IsSpinnerVisible: visible,
		SpinnerFrame:     ad.SpinnerFrame,
		SpinnerText:      text,
		ErrorMessage:     "", // Clear error when showing spinner
		Width:            ad.Width,
	}
}

func (ad AlertDisplay) WithError(errorMessage string) AlertDisplay {
	return AlertDisplay{
		IsSpinnerVisible: false, // Hide spinner when showing error
		SpinnerFrame:     ad.SpinnerFrame,
		SpinnerText:      ad.SpinnerText,
		ErrorMessage:     errorMessage,
		Width:            ad.Width,
	}
}

func (ad AlertDisplay) Clear() AlertDisplay {
	return AlertDisplay{
		IsSpinnerVisible: false,
		SpinnerFrame:     ad.SpinnerFrame,
		SpinnerText:      ad.SpinnerText,
		ErrorMessage:     "",
		Width:            ad.Width,
	}
}

func (ad AlertDisplay) WithWidth(width int) AlertDisplay {
	return AlertDisplay{
		IsSpinnerVisible: ad.IsSpinnerVisible,
		SpinnerFrame:     ad.SpinnerFrame,
		SpinnerText:      ad.SpinnerText,
		ErrorMessage:     ad.ErrorMessage,
		Width:            width,
	}
}

func (ad AlertDisplay) NextSpinnerFrame() AlertDisplay {
	if !ad.IsSpinnerVisible {
		return ad
	}

	return AlertDisplay{
		IsSpinnerVisible: ad.IsSpinnerVisible,
		SpinnerFrame:     (ad.SpinnerFrame + 1) % GetSpinnerFrameCount(),
		SpinnerText:      ad.SpinnerText,
		ErrorMessage:     ad.ErrorMessage,
		Width:            ad.Width,
	}
}

func (ad AlertDisplay) GetSpinnerFrame() string {
	if !ad.IsSpinnerVisible {
		return ""
	}
	return GetSpinnerFrame(ad.SpinnerFrame)
}

func (ad AlertDisplay) GetDisplayText() string {
	if ad.ErrorMessage != "" {
		return ad.ErrorMessage
	}
	if ad.IsSpinnerVisible {
		// Only return the spinner character, no text
		return ad.GetSpinnerFrame()
	}
	return ""
}

// StatusRowDisplay methods
func NewStatusRowDisplay(width int) StatusRowDisplay {
	return StatusRowDisplay{
		IsSpinnerVisible: false,
		SpinnerFrame:     0,
		FeedbackText:     "",
		StartTime:        time.Time{},
		CurrentDuration:  0,
		TokenCount:       0,
		Width:            width,
	}
}

func (srd StatusRowDisplay) WithSpinner(visible bool, feedbackText string) StatusRowDisplay {
	startTime := srd.StartTime
	if visible && srd.StartTime.IsZero() {
		startTime = time.Now()
	} else if !visible {
		startTime = time.Time{}
	}

	return StatusRowDisplay{
		IsSpinnerVisible: visible,
		SpinnerFrame:     srd.SpinnerFrame,
		FeedbackText:     feedbackText,
		StartTime:        startTime,
		CurrentDuration:  srd.CurrentDuration,
		TokenCount:       srd.TokenCount,
		Width:            srd.Width,
	}
}

func (srd StatusRowDisplay) WithTokens(tokenCount int) StatusRowDisplay {
	return StatusRowDisplay{
		IsSpinnerVisible: srd.IsSpinnerVisible,
		SpinnerFrame:     srd.SpinnerFrame,
		FeedbackText:     srd.FeedbackText,
		StartTime:        srd.StartTime,
		CurrentDuration:  srd.CurrentDuration,
		TokenCount:       tokenCount,
		Width:            srd.Width,
	}
}

func (srd StatusRowDisplay) WithDuration(duration time.Duration) StatusRowDisplay {
	return StatusRowDisplay{
		IsSpinnerVisible: srd.IsSpinnerVisible,
		SpinnerFrame:     srd.SpinnerFrame,
		FeedbackText:     srd.FeedbackText,
		StartTime:        srd.StartTime,
		CurrentDuration:  duration,
		TokenCount:       srd.TokenCount,
		Width:            srd.Width,
	}
}

func (srd StatusRowDisplay) WithWidth(width int) StatusRowDisplay {
	return StatusRowDisplay{
		IsSpinnerVisible: srd.IsSpinnerVisible,
		SpinnerFrame:     srd.SpinnerFrame,
		FeedbackText:     srd.FeedbackText,
		StartTime:        srd.StartTime,
		CurrentDuration:  srd.CurrentDuration,
		TokenCount:       srd.TokenCount,
		Width:            width,
	}
}

func (srd StatusRowDisplay) NextSpinnerFrame() StatusRowDisplay {
	return StatusRowDisplay{
		IsSpinnerVisible: srd.IsSpinnerVisible,
		SpinnerFrame:     (srd.SpinnerFrame + 1) % GetSpinnerFrameCount(),
		FeedbackText:     srd.FeedbackText,
		StartTime:        srd.StartTime,
		CurrentDuration:  srd.CurrentDuration,
		TokenCount:       srd.TokenCount,
		Width:            srd.Width,
	}
}

func (srd StatusRowDisplay) UpdateDuration() StatusRowDisplay {
	duration := srd.CurrentDuration
	if srd.IsSpinnerVisible && !srd.StartTime.IsZero() {
		duration = time.Since(srd.StartTime)
	}

	return StatusRowDisplay{
		IsSpinnerVisible: srd.IsSpinnerVisible,
		SpinnerFrame:     srd.SpinnerFrame,
		FeedbackText:     srd.FeedbackText,
		StartTime:        srd.StartTime,
		CurrentDuration:  duration,
		TokenCount:       srd.TokenCount,
		Width:            srd.Width,
	}
}

func (srd StatusRowDisplay) Clear() StatusRowDisplay {
	return StatusRowDisplay{
		IsSpinnerVisible: false,
		SpinnerFrame:     srd.SpinnerFrame,
		FeedbackText:     "",
		StartTime:        time.Time{},
		CurrentDuration:  0,
		TokenCount:       srd.TokenCount,
		Width:            srd.Width,
	}
}

// ClearSpinnerOnly clears only the spinner and feedback text, preserving token count
func (srd StatusRowDisplay) ClearSpinnerOnly() StatusRowDisplay {
	return StatusRowDisplay{
		IsSpinnerVisible: false,
		SpinnerFrame:     srd.SpinnerFrame,
		FeedbackText:     "",
		StartTime:        time.Time{},
		CurrentDuration:  0,
		TokenCount:       srd.TokenCount, // Preserve token count
		Width:            srd.Width,
	}
}
