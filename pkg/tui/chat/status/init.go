package status

import tea "github.com/charmbracelet/bubbletea"

func (m StatusModel) Init() tea.Cmd {
	return m.spinner.Tick
}
