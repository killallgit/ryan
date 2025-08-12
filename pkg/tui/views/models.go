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
	"github.com/killallgit/ryan/pkg/ollama"
)

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

// fetchModels returns a command to fetch models
func (v ModelsView) fetchModels() tea.Cmd {
	return func() tea.Msg {
		models, err := v.apiClient.ListModels()
		return fetchModelsMsg{models: models, err: err}
	}
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
			return v.handleDownloadModalKeys(msg)
		case ModalDelete:
			return v.handleDeleteConfirmation(msg)
		default:
			// Handle keys in main view
			switch msg.String() {
			case "esc":
				// Return to previous view
				return v, func() tea.Msg {
					return SwitchToPreviousViewMsg{}
				}
			case "enter":
				return v.handleEnterKey()
			case "ctrl+d":
				return v.handleDeleteFromList()
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
		return v.handlePullProgress(msg)

	case pullCompleteMsg:
		return v.handlePullComplete(msg)

	case deleteCompleteMsg:
		return v.handleDeleteComplete(msg)

	case startPullMsg:
		return v.handleStartPull(msg)

	case pollMsg:
		// Continue polling for progress updates
		if v.downloadActive && (v.progressChan != nil || v.errorChan != nil) {
			return v, waitForProgress(v.progressChan, v.errorChan)
		}

	case spinnerTickMsg:
		return v.handleSpinnerTick()

	case autoRefreshMsg:
		return v.handleAutoRefresh()
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

// Name returns the display name for this view
func (v ModelsView) Name() string {
	return "Models"
}

// Description returns the description for this view
func (v ModelsView) Description() string {
	return "Manage Ollama models"
}
