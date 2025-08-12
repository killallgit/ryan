package views

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SettingsView displays application settings
type SettingsView struct {
	width  int
	height int
}

// NewSettingsView creates a new settings view
func NewSettingsView() SettingsView {
	return SettingsView{}
}

// Init initializes the settings view
func (v SettingsView) Init() tea.Cmd {
	return nil
}

// Update handles messages for the settings view
func (v SettingsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	}
	return v, nil
}

// View renders the settings view
func (v SettingsView) View() string {
	var b strings.Builder

	b.WriteString("=== Settings ===\n\n")

	// Display Ollama configuration
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "Not configured"
	}
	b.WriteString(fmt.Sprintf("Ollama Host: %s\n", ollamaHost))

	// Display default model
	defaultModel := os.Getenv("OLLAMA_DEFAULT_MODEL")
	if defaultModel == "" {
		defaultModel = "Not configured"
	}
	b.WriteString(fmt.Sprintf("Default Model: %s\n", defaultModel))

	// Display embedding model
	embeddingModel := os.Getenv("OLLAMA_EMBEDDING_MODEL")
	if embeddingModel == "" {
		embeddingModel = "Not configured"
	}
	b.WriteString(fmt.Sprintf("Embedding Model: %s\n", embeddingModel))

	b.WriteString("\n")
	b.WriteString("Press Ctrl+P to switch views\n")
	b.WriteString("Press q to quit\n")

	return b.String()
}

// Name returns the display name for this view
func (v SettingsView) Name() string {
	return "Settings"
}

// Description returns the description for this view
func (v SettingsView) Description() string {
	return "Application configuration"
}
