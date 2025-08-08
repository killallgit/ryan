package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type rootModel struct {
	ctx        context.Context
	activeView int
	views      []tea.Model
	width      int
	height     int
	sub        chan struct{}
	quitting   bool
	spinner    spinner.Model
}
type responseMsg struct{}

// // Simulate a process that sends events at an irregular interval in real time.
// // In this case, we'll send events on the channel at a random interval between
// // 100 to 1000 milliseconds. As a command, Bubble Tea will run this
// // asynchronously.
// func listenForActivity(sub chan struct{}) tea.Cmd {
// 	return func() tea.Msg {
// 		for {
// 			sub <- struct{}{}
// 		}
// 	}
// }

// // A command that waits for the activity on a channel.
// func waitForActivity(sub chan struct{}) tea.Cmd {
// 	return func() tea.Msg {
// 		return responseMsg(<-sub)
// 	}
// }

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

func NewRootModel(ctx context.Context, views ...tea.Model) *rootModel {
	return &rootModel{
		ctx:        ctx,
		activeView: 0,
		views:      views,
		sub:        make(chan struct{}),
	}
}
