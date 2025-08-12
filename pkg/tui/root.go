package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/killallgit/ryan/pkg/tui/switcher"
	"github.com/killallgit/ryan/pkg/tui/views"
)

type rootModel struct {
	ctx           context.Context
	activeView    int
	views         []views.View
	width         int
	height        int
	showSwitcher  bool
	switcherModel switcher.Model
	previousView  int // Remember previous view when opening switcher
}

func (m rootModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, view := range m.views {
		cmds = append(cmds, view.Init())
	}
	return tea.Batch(cmds...)
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle window resize for all components
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update switcher dimensions
		var cmd tea.Cmd
		switcherModel, cmd := m.switcherModel.Update(msg)
		m.switcherModel = switcherModel.(switcher.Model)
		cmds = append(cmds, cmd)

		// Pass to active view
		if m.activeView < len(m.views) {
			viewModel, cmd := m.views[m.activeView].Update(msg)
			m.views[m.activeView] = viewModel.(views.View)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	// If switcher is showing, handle its input
	if m.showSwitcher {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc", "ctrl+c":
				// Close switcher without changing view
				m.showSwitcher = false
				return m, nil
			case "enter":
				// Switch to selected view
				m.activeView = m.switcherModel.SelectedIndex()
				m.showSwitcher = false
				// Initialize the newly selected view
				return m, m.views[m.activeView].Init()
			}
		}

		// Pass all other messages to the switcher (including navigation)
		var cmd tea.Cmd
		switcherModel, cmd := m.switcherModel.Update(msg)
		m.switcherModel = switcherModel.(switcher.Model)
		return m, cmd
	}

	// Handle global keys
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case msg.String() == "ctrl+p":
			// Toggle switcher
			m.showSwitcher = true
			m.previousView = m.activeView
			// Set the current view as selected in switcher
			m.switcherModel.SetSelectedIndex(m.activeView)
			return m, nil
		case key.Matches(msg, keys.Help):
			return m, nil
		}
	}

	// Pass messages to active view when switcher is not showing
	if m.activeView < len(m.views) {
		var cmd tea.Cmd
		viewModel, cmd := m.views[m.activeView].Update(msg)
		m.views[m.activeView] = viewModel.(views.View)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m rootModel) View() string {
	baseView := m.views[m.activeView].View()

	if m.showSwitcher {
		// Get switcher content
		switcherContent := m.switcherModel.View()

		// Create a modal style with border and padding
		modalStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			BorderForeground(lipgloss.Color("240"))

		// Render the modal with the switcher content
		modal := modalStyle.Render(switcherContent)

		// Center the modal over the base view using lipgloss.Place
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			modal,
			lipgloss.WithWhitespaceBackground(lipgloss.NoColor{}),
		)
	}

	return baseView
}

func NewRootModel(ctx context.Context, viewList []views.View) *rootModel {
	return &rootModel{
		ctx:           ctx,
		activeView:    0,
		views:         viewList,
		switcherModel: switcher.New(viewList),
	}
}
