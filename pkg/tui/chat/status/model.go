package status

import tea "github.com/charmbracelet/bubbletea"

type statusModel struct {
}

func NewStatusModel() statusModel {
	return statusModel{}
}

func (m statusModel) Init() tea.Cmd {
	return nil
}

func (m statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m statusModel) View() string {
	return ""
}
