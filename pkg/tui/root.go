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
	previousView  int   // Remember previous view for navigation
	viewHistory   []int // Stack of view indices for navigation history
}

const maxViewHistorySize = 10 // Limit history stack size

// pushToHistory adds a view index to the history stack, maintaining size limits
func (m *rootModel) pushToHistory(viewIndex int) {
	m.viewHistory = append(m.viewHistory, viewIndex)
	// Keep only the last maxViewHistorySize items
	if len(m.viewHistory) > maxViewHistorySize {
		m.viewHistory = m.viewHistory[len(m.viewHistory)-maxViewHistorySize:]
	}
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
				newView := m.switcherModel.SelectedIndex()
				if newView != m.activeView {
					// Push current view to history stack
					m.pushToHistory(m.activeView)
					m.previousView = m.activeView
					m.activeView = newView
				}
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

	// Handle custom messages first
	switch msg := msg.(type) {
	case views.SwitchToPreviousViewMsg:
		// Pop from history stack to go back
		if len(m.viewHistory) > 0 {
			// Get the previous view from the stack
			previousViewIndex := m.viewHistory[len(m.viewHistory)-1]
			m.viewHistory = m.viewHistory[:len(m.viewHistory)-1]

			if previousViewIndex >= 0 && previousViewIndex < len(m.views) && previousViewIndex != m.activeView {
				m.activeView = previousViewIndex
				// Initialize the newly selected view
				return m, m.views[m.activeView].Init()
			}
		}
		return m, nil
	case views.SwitchToViewMsg:
		// Switch to specific view
		if msg.Index >= 0 && msg.Index < len(m.views) && msg.Index != m.activeView {
			// Push current view to history stack
			m.pushToHistory(m.activeView)
			m.previousView = m.activeView
			m.activeView = msg.Index
			// Initialize the newly selected view
			return m, m.views[m.activeView].Init()
		}
		return m, nil
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
			// Don't update previousView here since we're just opening the switcher
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
