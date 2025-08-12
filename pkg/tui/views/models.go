package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/killallgit/ryan/pkg/ollama"
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
	showDetails   bool
	selectedModel *ollama.Model
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

	return ModelsView{
		table:     t,
		apiClient: ollama.NewAPIClient(),
		loading:   true,
	}
}

// fetchModelsMsg is sent when models are fetched
type fetchModelsMsg struct {
	models []ollama.Model
	err    error
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
	return v.fetchModels()
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
			// Convert models to table rows with simplified columns
			rows := make([]table.Row, 0, len(msg.models))
			for _, model := range msg.models {
				row := table.Row{
					model.Name,
					ollama.FormatSize(model.Size),
					model.Details.ParameterSize,
					model.ModifiedAt.Format("2006-01-02 15:04"),
				}
				rows = append(rows, row)
			}
			v.table.SetRows(rows)
		}

	case tea.KeyMsg:
		if v.showDetails {
			// Handle keys in details modal
			switch msg.String() {
			case "d", "D", "esc", "enter":
				v.showDetails = false
				v.selectedModel = nil
			}
		} else {
			// Handle keys in main view
			switch msg.String() {
			case "esc":
				// Return to previous view
				return v, func() tea.Msg {
					return SwitchToPreviousViewMsg{}
				}
			case "d", "D":
				// Show details for selected model
				if len(v.models) > 0 && v.table.Cursor() < len(v.models) {
					v.selectedModel = &v.models[v.table.Cursor()]
					v.showDetails = true
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
	}

	// Update the table only if not showing details
	if !v.showDetails {
		v.table, cmd = v.table.Update(msg)
	}
	return v, cmd
}

// View renders the models view
func (v ModelsView) View() string {
	// Show details modal if active
	if v.showDetails && v.selectedModel != nil {
		return v.renderDetailsModal()
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
	footer := "↑/↓: Navigate • d: Details • r: Refresh • Esc: Back • Ctrl+P: Switch view • q: Quit"
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

// Name returns the display name for this view
func (v ModelsView) Name() string {
	return "Models"
}

// Description returns the description for this view
func (v ModelsView) Description() string {
	return "Manage Ollama models"
}
