package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type rootModel struct {
	*AppState
	ctx        context.Context
	activeView int
	views      []tea.Model
	width      int
	height     int
}

func (m rootModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, view := range m.views {
		cmds = append(cmds, view.Init())
	}
	return tea.Batch(cmds...)
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Help):
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	if m.activeView < len(m.views) {
		var cmd tea.Cmd
		m.views[m.activeView], cmd = m.views[m.activeView].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m rootModel) View() string {
	return m.views[m.activeView].View()
}

func NewRootModel(ctx context.Context, appState *AppState, views ...tea.Model) *rootModel {
	return &rootModel{
		AppState:   appState,
		ctx:        ctx,
		activeView: 0,
		views:      views,
	}
}
