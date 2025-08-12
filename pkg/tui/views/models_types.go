package views

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/killallgit/ryan/pkg/ollama"
)

// ModalType represents the type of modal currently shown
type ModalType int

const (
	ModalNone ModalType = iota
	ModalDetails
	ModalDownload
	ModalDelete
)

// ModelsView displays available Ollama models
type ModelsView struct {
	width         int
	height        int
	table         table.Model
	models        []ollama.Model
	apiClient     *ollama.APIClient
	loading       bool
	err           error
	lastUpdate    time.Time
	modalType     ModalType
	selectedModel *ollama.Model

	// Download modal components
	textInput        textinput.Model
	progressBar      progress.Model
	downloadActive   bool
	progressPercent  float64
	progressStatus   string
	progressChan     <-chan ollama.PullProgress
	errorChan        <-chan error
	downloadingModel string
	errorMessage     string

	// Delete confirmation
	modelToDelete string

	// Spinner animation
	spinnerFrame int
	spinnerChars []string

	// Automatic refresh
	autoRefreshEnabled bool
}
