package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/killallgit/ryan/pkg/logger"
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

// NewModelsView creates a new models view
func NewModelsView() ModelsView {
	// Create table with only essential columns
	columns := []table.Column{
		{Title: "Name", Width: 35},
		{Title: "Size", Width: 12},
		{Title: "Parameters", Width: 15},
		{Title: "Modified", Width: 18},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Style the table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	// Create text input for download modal
	ti := textinput.New()
	ti.Placeholder = "Enter model name (e.g., llama3.2)"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 30

	// Create progress bar
	prog := progress.New(progress.WithDefaultGradient())

	return ModelsView{
		table:              t,
		apiClient:          ollama.NewAPIClient(),
		loading:            true,
		modalType:          ModalNone,
		textInput:          ti,
		progressBar:        prog,
		spinnerFrame:       0,
		spinnerChars:       []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		autoRefreshEnabled: true,
	}
}

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

// fetchModels returns a command to fetch models
func (v ModelsView) fetchModels() tea.Cmd {
	return func() tea.Msg {
		models, err := v.apiClient.ListModels()
		return fetchModelsMsg{models: models, err: err}
	}
}

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

// startDelete returns a command to start deleting a model
func (v ModelsView) startDelete(modelName string) tea.Cmd {
	return func() tea.Msg {
		err := v.apiClient.DeleteModel(modelName)
		return deleteCompleteMsg{success: err == nil, err: err}
	}
}

// updateTable rebuilds the table with current model state
func (v *ModelsView) updateTable() {
	// Convert models to table rows with simplified columns
	// Add "Pull model" row at the top
	rows := make([]table.Row, 0, len(v.models)+1)
	rows = append(rows, table.Row{
		"→ Pull model",
		"",
		"",
		"",
	})
	// Add downloading model if it's not in the list yet
	downloadingModelExists := false
	if v.downloadActive && v.downloadingModel != "" {
		for _, model := range v.models {
			if model.Name == v.downloadingModel {
				downloadingModelExists = true
				break
			}
		}

		// If downloading model doesn't exist in current models, add it as a temporary entry
		if !downloadingModelExists {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
			spinnerChar := v.spinnerChars[v.spinnerFrame%len(v.spinnerChars)]
			modelName := spinnerChar + " " + v.downloadingModel
			row := table.Row{
				dimStyle.Render(modelName),
				dimStyle.Render("Downloading..."),
				dimStyle.Render(""),
				dimStyle.Render(""),
			}
			rows = append(rows, row)
		}
	}

	for _, model := range v.models {
		modelName := model.Name
		size := ollama.FormatSize(model.Size)
		params := model.Details.ParameterSize
		modified := model.ModifiedAt.Format("2006-01-02 15:04")

		// Add spinner and dim if this model is being downloaded
		if v.downloadActive && v.downloadingModel == model.Name {
			spinnerChar := v.spinnerChars[v.spinnerFrame%len(v.spinnerChars)]
			modelName = spinnerChar + " " + model.Name
			// Apply dimmed styling to all fields for downloading model
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
			modelName = dimStyle.Render(modelName)
			size = dimStyle.Render(size)
			params = dimStyle.Render(params)
			modified = dimStyle.Render(modified)
		}

		row := table.Row{
			modelName,
			size,
			params,
			modified,
		}
		rows = append(rows, row)
	}
	v.table.SetRows(rows)
}

// getModelAtCursor returns information about what's at the current cursor position
func (v *ModelsView) getModelAtCursor(cursor int) (isDownloading bool, modelName string, modelIndex int) {
	if cursor == 0 {
		// "Pull model" row
		return false, "", -1
	}

	currentRow := 1 // Start after "Pull model" row

	// Check if there's a temporary downloading model
	if v.downloadActive && v.downloadingModel != "" {
		downloadingModelExists := false
		for _, model := range v.models {
			if model.Name == v.downloadingModel {
				downloadingModelExists = true
				break
			}
		}

		if !downloadingModelExists {
			// There's a temporary downloading model at row 1
			if cursor == currentRow {
				return true, v.downloadingModel, -1
			}
			currentRow++
		}
	}

	// Check regular models
	for i, model := range v.models {
		if cursor == currentRow {
			isDownloadingThis := v.downloadActive && v.downloadingModel == model.Name
			return isDownloadingThis, model.Name, i
		}
		currentRow++
	}

	return false, "", -1
}

// Init initializes the models view
func (v ModelsView) Init() tea.Cmd {
	// Start both initial model fetch and auto-refresh timer
	return tea.Batch(v.fetchModels(), autoRefreshTick())
}

// Update handles messages for the models view
func (v ModelsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.table.SetHeight(msg.Height - 8) // Leave room for header and footer

	case fetchModelsMsg:
		v.loading = false
		v.err = msg.err
		v.models = msg.models
		v.lastUpdate = time.Now()

		if msg.err == nil {
			v.updateTable()
		}

	case tea.KeyMsg:
		switch v.modalType {
		case ModalDetails:
			// Handle keys in details modal
			switch msg.String() {
			case "d", "D", "esc", "enter":
				v.modalType = ModalNone
				v.selectedModel = nil
			}
		case ModalDownload:
			// Handle keys in download modal
			switch msg.String() {
			case "esc":
				// Always allow ESC to close modal, but keep download running
				v.modalType = ModalNone
				v.textInput.SetValue("")
				// Don't reset downloadActive, progressChan, errorChan - keep download running
			case "ctrl+d":
				// Cancel download if active, otherwise just close modal
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
				}
			default:
				if !v.downloadActive {
					v.textInput, cmd = v.textInput.Update(msg)
					return v, cmd
				}
			}
		case ModalDelete:
			// Handle keys in delete confirmation modal
			switch msg.String() {
			case "esc":
				v.modalType = ModalNone
				v.modelToDelete = ""
			case "enter", "y", "Y":
				// Confirm delete
				if v.modelToDelete != "" {
					modelName := v.modelToDelete
					v.modalType = ModalNone
					v.modelToDelete = ""
					return v, v.startDelete(modelName)
				}
			case "n", "N":
				v.modalType = ModalNone
				v.modelToDelete = ""
			}
		default:
			// Handle keys in main view
			switch msg.String() {
			case "esc":
				// Return to previous view
				return v, func() tea.Msg {
					return SwitchToPreviousViewMsg{}
				}
			case "enter":
				// Show appropriate modal based on cursor position
				cursor := v.table.Cursor()
				if cursor == 0 {
					// "Pull model" row selected - show download modal
					v.modalType = ModalDownload
					v.textInput.Focus()
					v.textInput.SetValue("")
				} else {
					// Check what's at the cursor position
					isDownloading, modelName, modelIndex := v.getModelAtCursor(cursor)

					if isDownloading {
						// Show download progress modal for downloading model
						v.modalType = ModalDownload
						logger.Debug("Opening progress modal for downloading model: %s", modelName)
					} else if modelIndex >= 0 && modelIndex < len(v.models) {
						// Regular model selected - show details modal
						v.selectedModel = &v.models[modelIndex]
						v.modalType = ModalDetails
					}
				}
			case "ctrl+d":
				// Show delete confirmation for selected model or cancel download
				cursor := v.table.Cursor()
				if cursor > 0 {
					isDownloading, modelName, modelIndex := v.getModelAtCursor(cursor)

					if isDownloading {
						// Cancel the active download
						logger.Debug("Cancelling download of model: %s", modelName)
						v.downloadActive = false
						v.progressChan = nil
						v.errorChan = nil
						v.downloadingModel = ""
						v.progressPercent = 0
						v.progressStatus = ""
						// Update table to remove downloading model
						v.updateTable()
					} else if modelIndex >= 0 && modelIndex < len(v.models) {
						// Show delete confirmation for non-downloading models
						v.modelToDelete = modelName
						v.modalType = ModalDelete
					}
				}
			case "r", "R":
				// Refresh models
				v.loading = true
				v.err = nil
				return v, v.fetchModels()
			case "q", "ctrl+c":
				return v, tea.Quit
			}
		}

	case pullProgressMsg:
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

	case pullCompleteMsg:
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

	case deleteCompleteMsg:
		if msg.success {
			// Refresh models list
			v.loading = true
			return v, v.fetchModels()
		} else {
			v.err = msg.err
		}

	case startPullMsg:
		// Store channels and start monitoring
		v.progressChan = msg.progressChan
		v.errorChan = msg.errorChan
		return v, waitForProgress(msg.progressChan, msg.errorChan)

	case pollMsg:
		// Continue polling for progress updates
		if v.downloadActive && (v.progressChan != nil || v.errorChan != nil) {
			return v, waitForProgress(v.progressChan, v.errorChan)
		}

	case spinnerTickMsg:
		// Animate spinner only if download is active
		if v.downloadActive {
			v.spinnerFrame++
			v.updateTable()
			return v, spinnerTick() // Schedule next animation frame
		}

	case autoRefreshMsg:
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
	}

	// Update the table only if not showing modal
	if v.modalType == ModalNone {
		v.table, cmd = v.table.Update(msg)
	}
	return v, cmd
}

// View renders the models view
func (v ModelsView) View() string {
	// Show appropriate modal based on type
	switch v.modalType {
	case ModalDetails:
		if v.selectedModel != nil {
			return v.renderDetailsModal()
		}
	case ModalDownload:
		return v.renderDownloadModal()
	case ModalDelete:
		return v.renderDeleteModal()
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		MarginBottom(1)

	b.WriteString(headerStyle.Render("=== Ollama Models ==="))
	b.WriteString("\n\n")

	// Show loading, error, or table
	if v.loading {
		b.WriteString("Loading models...")
	} else if v.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", v.err)))
		b.WriteString("\n\nPress 'r' to retry")
	} else if len(v.models) == 0 {
		b.WriteString("No models found.\n")
		b.WriteString("Pull a model with: ollama pull <model-name>\n")
	} else {
		// Stats line
		statsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginBottom(1)
		stats := fmt.Sprintf("Total models: %d | Last updated: %s",
			len(v.models),
			v.lastUpdate.Format("15:04:05"))
		b.WriteString(statsStyle.Render(stats))
		b.WriteString("\n\n")

		// Table
		b.WriteString(v.table.View())
	}

	// Footer with controls
	b.WriteString("\n\n")
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	footer := "↑/↓: Navigate • Enter: Pull/Details • Ctrl+D: Delete • r: Refresh • Esc: Back • q: Quit"
	b.WriteString(footerStyle.Render(footer))

	return b.String()
}

// renderDetailsModal renders the details modal for a selected model
func (v ModelsView) renderDetailsModal() string {
	// Modal styles
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("170")).
		Padding(1, 2).
		Width(70).
		MaxWidth(v.width - 4)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render(fmt.Sprintf("Model Details: %s", v.selectedModel.Name)))
	content.WriteString("\n\n")

	// Model details
	digestValue := v.selectedModel.Digest
	if len(digestValue) > 12 {
		digestValue = digestValue[:12] + "..."
	}

	details := []struct {
		label string
		value string
	}{
		{"Name", v.selectedModel.Name},
		{"Size", ollama.FormatSize(v.selectedModel.Size)},
		{"Digest", digestValue},
		{"Modified", v.selectedModel.ModifiedAt.Format("2006-01-02 15:04:05")},
		{"Format", v.selectedModel.Details.Format},
		{"Family", v.selectedModel.Details.Family},
		{"Parameter Size", v.selectedModel.Details.ParameterSize},
		{"Quantization", v.selectedModel.Details.QuantizationLevel},
	}

	for _, d := range details {
		content.WriteString(labelStyle.Render(d.label + ": "))
		content.WriteString(valueStyle.Render(d.value))
		content.WriteString("\n")
	}

	// Additional families if present
	if len(v.selectedModel.Details.Families) > 0 {
		content.WriteString("\n")
		content.WriteString(labelStyle.Render("Families: "))
		content.WriteString(valueStyle.Render(strings.Join(v.selectedModel.Details.Families, ", ")))
		content.WriteString("\n")
	}

	// Footer
	content.WriteString("\n")
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)
	content.WriteString(footerStyle.Render("Press 'd', 'esc', or 'enter' to close"))

	// Center the modal
	modal := modalStyle.Render(content.String())

	// Calculate vertical padding to center
	lines := strings.Count(modal, "\n") + 1
	topPadding := (v.height - lines) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// Build final view with padding
	var final strings.Builder
	for i := 0; i < topPadding; i++ {
		final.WriteString("\n")
	}

	// Center horizontally
	modalWidth := lipgloss.Width(modal)
	leftPadding := (v.width - modalWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	paddedModal := lipgloss.NewStyle().
		MarginLeft(leftPadding).
		Render(modal)

	final.WriteString(paddedModal)

	return final.String()
}

// renderDownloadModal renders the download modal
func (v ModelsView) renderDownloadModal() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("170")).
		Padding(1, 2).
		Width(50).
		MaxWidth(v.width - 4)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		MarginBottom(1)

	var content strings.Builder
	content.WriteString(titleStyle.Render("Pull Model"))
	content.WriteString("\n\n")

	if v.downloadActive {
		content.WriteString(fmt.Sprintf("Downloading %s...\n\n", v.downloadingModel))
		content.WriteString(v.progressStatus)
		content.WriteString("\n\n")
		if v.progressPercent > 0 {
			content.WriteString(v.progressBar.ViewAs(v.progressPercent))
		} else {
			content.WriteString(v.progressBar.View())
		}
		content.WriteString("\n\n")
		content.WriteString("Press [Esc] to close  [Ctrl+D] to cancel download")
	} else if v.errorMessage != "" {
		// Show error state
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		content.WriteString("Enter model name:\n")
		content.WriteString(v.textInput.View())
		content.WriteString("\n\n")
		content.WriteString(errorStyle.Render(v.errorMessage))
		content.WriteString("\n\n")
		content.WriteString("[Enter] Try Again  [Esc] Close")
	} else {
		content.WriteString("Enter model name:\n")
		content.WriteString(v.textInput.View())
		content.WriteString("\n\n")
		content.WriteString("Examples: llama3.2, mixtral:8x7b, codellama")
		content.WriteString("\n\n")
		content.WriteString("[Enter] OK  [Esc] Close")
	}

	// Center the modal
	modal := modalStyle.Render(content.String())
	lines := strings.Count(modal, "\n") + 1
	topPadding := (v.height - lines) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	var final strings.Builder
	for i := 0; i < topPadding; i++ {
		final.WriteString("\n")
	}

	modalWidth := lipgloss.Width(modal)
	leftPadding := (v.width - modalWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	paddedModal := lipgloss.NewStyle().
		MarginLeft(leftPadding).
		Render(modal)

	final.WriteString(paddedModal)
	return final.String()
}

// renderDeleteModal renders the delete confirmation modal
func (v ModelsView) renderDeleteModal() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Width(40).
		MaxWidth(v.width - 4)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		MarginBottom(1)

	var content strings.Builder
	content.WriteString(titleStyle.Render("Delete Model"))
	content.WriteString("\n\n")
	content.WriteString(fmt.Sprintf("Delete %s?", v.modelToDelete))
	content.WriteString("\n\n")
	content.WriteString("This action cannot be undone.")
	content.WriteString("\n\n")
	content.WriteString("[Enter/Y] OK  [N/Esc] Cancel")

	// Center the modal
	modal := modalStyle.Render(content.String())
	lines := strings.Count(modal, "\n") + 1
	topPadding := (v.height - lines) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	var final strings.Builder
	for i := 0; i < topPadding; i++ {
		final.WriteString("\n")
	}

	modalWidth := lipgloss.Width(modal)
	leftPadding := (v.width - modalWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	paddedModal := lipgloss.NewStyle().
		MarginLeft(leftPadding).
		Render(modal)

	final.WriteString(paddedModal)
	return final.String()
}

// Name returns the display name for this view
func (v ModelsView) Name() string {
	return "Models"
}

// Description returns the description for this view
func (v ModelsView) Description() string {
	return "Manage Ollama models"
}
