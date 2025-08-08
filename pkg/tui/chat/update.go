package chat

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/streaming"
	"github.com/killallgit/ryan/pkg/tui/chat/status"
)

type (
	errMsg error
)

func (m chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowResize(msg.Width, msg.Height)
		// Update status bar width
		statusModel, _ := m.statusBar.Update(msg)
		m.statusBar = statusModel.(status.StatusModel)

	case tea.KeyMsg:
		// All key handling happens in handleKeyMsg
		return handleKeyMsg(m, msg)

	case errMsg:
		m.err = msg
		return m, nil

	case streaming.StreamStartMsg:
		// Create new node for this stream
		node := MessageNode{
			ID:          msg.StreamID,
			Type:        msg.SourceType,
			Content:     "",
			Timestamp:   time.Now(),
			StreamID:    msg.StreamID,
			IsStreaming: true,
		}
		m.nodes = append(m.nodes, node)
		m.isStreaming = true
		m.currentStream = msg.StreamID

		// Update status bar
		statusModel, _ := m.statusBar.Update(status.StartStreamingMsg{Icon: "↓"})
		m.statusBar = statusModel.(status.StatusModel)

		// Begin streaming content with the prompt from the message
		return m, streaming.StreamContent(m.streamManager, msg.StreamID, "ollama-main", msg.Prompt)

	case streaming.StreamChunkMsg:
		// Find the node for this stream and append content
		for i := range m.nodes {
			if m.nodes[i].StreamID == msg.StreamID {
				m.nodes[i].Content += msg.Content
				break
			}
		}

		// Update viewport with all nodes
		m.updateViewportContent()
		return m, nil

	case streaming.StreamEndMsg:
		// Mark stream as complete
		for i := range m.nodes {
			if m.nodes[i].StreamID == msg.StreamID {
				m.nodes[i].IsStreaming = false
				if msg.Error != nil {
					m.nodes[i].Type = "error"
					m.nodes[i].Content = fmt.Sprintf("Error: %v", msg.Error)
				} else if msg.FinalContent != "" {
					m.nodes[i].Content = msg.FinalContent
				}
				break
			}
		}

		m.isStreaming = false
		m.currentStream = ""
		m.updateViewportContent()

		// Update status bar
		statusModel, _ := m.statusBar.Update(status.StopStreamingMsg{})
		m.statusBar = statusModel.(status.StatusModel)
		return m, nil

	default:
		// Update status bar
		statusModel, statusCmd := m.statusBar.Update(msg)
		m.statusBar = statusModel.(status.StatusModel)
		cmds = append(cmds, statusCmd)

		// Update textarea for other messages (like blink cursor)
		var tiCmd tea.Cmd
		m.textarea, tiCmd = m.textarea.Update(msg)
		cmds = append(cmds, tiCmd)

		// Update viewport for other messages
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}
