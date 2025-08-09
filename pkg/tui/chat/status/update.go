package status

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/process"
)

func (m StatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case StatusUpdateMsg:
		m.status = msg.Status
		if msg.State != "" {
			m.processState = msg.State
			m.icon = msg.State.GetIcon()
		}
		return m, nil

	case StartStreamingMsg:
		m.isActive = true
		m.startTime = time.Now()
		m.processState = msg.State
		if msg.State != "" {
			m.icon = msg.State.GetIcon()
		} else if msg.Icon != "" {
			m.icon = msg.Icon
		} else {
			m.icon = process.StateReceiving.GetIcon()
		}
		m.status = "Streaming"
		return m, tea.Batch(
			m.spinner.Tick,
			tickEvery(),
		)

	case SetProcessStateMsg:
		m.processState = msg.State
		m.icon = msg.State.GetIcon()
		m.status = msg.State.GetDisplayName()
		return m, nil

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
