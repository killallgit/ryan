package views

import tea "github.com/charmbracelet/bubbletea"

// View represents a switchable view in the application
type View interface {
	tea.Model
	Name() string        // Display name for the view
	Description() string // Optional description
}
