package status

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m StatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case spinner.TickMsg:
		if m.isActive {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case StatusUpdateMsg:
		m.status = msg.Status
		return m, nil

	case StartStreamingMsg:
		m.isActive = true
		m.startTime = time.Now()
		m.icon = msg.Icon
		m.status = "Streaming"
		return m, tea.Batch(
			m.spinner.Tick,
			tickEvery(),
		)

	case StopStreamingMsg:
		m.isActive = false
		m.status = ""
		m.icon = ""
		m.timer = 0
		return m, nil

	case UpdateTokensMsg:
		m.tokensSent += msg.Sent
		m.tokensRecv += msg.Recv
		return m, nil

	case TickMsg:
		if m.isActive {
			m.timer = time.Since(m.startTime)
			return m, tickEvery()
		}
		return m, nil
	}

	return m, nil
}

// tickEvery returns a command that sends a tick message every second
func tickEvery() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
