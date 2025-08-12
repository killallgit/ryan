package switcher

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/killallgit/ryan/pkg/tui/views"
)

// viewItem implements list.Item for views
type viewItem struct {
	view views.View
}

func (i viewItem) FilterValue() string { return i.view.Name() }
func (i viewItem) Title() string       { return i.view.Name() }
func (i viewItem) Description() string { return i.view.Description() }

// Model represents the view switcher modal
type Model struct {
	list          list.Model
	views         []views.View
	selectedIndex int
	width         int
	height        int
}

// New creates a new switcher model
func New(views []views.View) Model {
	// Convert views to list items
	items := make([]list.Item, len(views))
	for i, v := range views {
		items[i] = viewItem{view: v}
	}

	// Create delegate with minimal styling
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetHeight(1)
	delegate.SetSpacing(0)

	// Style the selected item
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Background(lipgloss.Color("235"))

	delegate.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("250"))

	// Create list
	l := list.New(items, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	// Calculate size based on content
	listHeight := len(views) // Just the items, no extra padding

	// Find the longest name for width calculation
	maxWidth := 0
	for _, v := range views {
		if len(v.Name()) > maxWidth {
			maxWidth = len(v.Name())
		}
	}

	// Add some padding for the list item styling
	listWidth := maxWidth + 6
	if listWidth < 20 {
		listWidth = 20 // Minimum width
	}

	l.SetSize(listWidth, listHeight)

	return Model{
		list:  l,
		views: views,
	}
}

// Init initializes the switcher
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the switcher
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Let the list handle all updates including navigation
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.selectedIndex = m.list.Index()

	return m, cmd
}

// View renders the switcher
func (m Model) View() string {
	return m.list.View()
}

// SelectedIndex returns the currently selected index
func (m Model) SelectedIndex() int {
	return m.list.Index()
}

// SetSelectedIndex sets the selected index
func (m *Model) SetSelectedIndex(index int) {
	if index >= 0 && index < len(m.views) {
		m.list.Select(index)
		m.selectedIndex = index
	}
}

// Width returns the width of the list
func (m Model) Width() int {
	return m.list.Width()
}

// Height returns the height of the list
func (m Model) Height() int {
	return m.list.Height()
}
