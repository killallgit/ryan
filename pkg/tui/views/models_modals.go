package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/killallgit/ryan/pkg/ollama"
)

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

	return v.centerModal(modalStyle.Render(content.String()))
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

	return v.centerModal(modalStyle.Render(content.String()))
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

	return v.centerModal(modalStyle.Render(content.String()))
}

// centerModal centers a modal both vertically and horizontally
func (v ModelsView) centerModal(modal string) string {
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
