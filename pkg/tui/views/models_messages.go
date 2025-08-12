package views

import "github.com/killallgit/ryan/pkg/ollama"

// fetchModelsMsg is sent when models are fetched
type fetchModelsMsg struct {
	models []ollama.Model
	err    error
}

// pullProgressMsg is sent during model download
type pullProgressMsg struct {
	progress ollama.PullProgress
}

// pullCompleteMsg is sent when download completes
type pullCompleteMsg struct {
	success bool
	err     error
}

// deleteCompleteMsg is sent when delete completes
type deleteCompleteMsg struct {
	success bool
	err     error
}

// pollMsg triggers the next polling cycle
type pollMsg struct{}

// spinnerTickMsg is sent to animate the spinner
type spinnerTickMsg struct{}

// autoRefreshMsg is sent to trigger automatic model list refresh
type autoRefreshMsg struct{}

// startPullMsg is sent when a pull operation starts
type startPullMsg struct {
	progressChan <-chan ollama.PullProgress
	errorChan    <-chan error
}
