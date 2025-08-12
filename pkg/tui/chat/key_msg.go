package chat

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/process"
	"github.com/killallgit/ryan/pkg/stream/tui"
	"github.com/killallgit/ryan/pkg/tui/chat/status"
)

func handleKeyMsg(m chatModel, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check if the key should be handled by viewport for scrolling
	switch msg.Type {
	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd:
		// If textarea is not focused or is empty, let viewport handle scrolling
		if !m.textarea.Focused() || m.textarea.Value() == "" {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	case tea.KeyCtrlU:
		// Ctrl+U for half page up
		m.viewport.HalfPageUp()
		return m, nil
	case tea.KeyCtrlD:
		// Ctrl+D for half page down
		m.viewport.HalfPageDown()
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEscape:
		m.numEscPress++
		if m.numEscPress == 2 {
			m.textarea.Reset()
			m.numEscPress = 0
			return m, nil
		}
	case tea.KeyEnter:
		if msg.Alt {
			// Alt+Enter adds a newline
			break
		}
		if m.textarea.Value() != "" {
			userInput := m.textarea.Value()

			// Add message to chat history
			if m.chatManager != nil {
				m.chatManager.AddMessage(chat.RoleUser, userInput)
			}

			// Create user message node
			userNode := MessageNode{
				ID:        fmt.Sprintf("user-%d", time.Now().UnixNano()),
				Type:      "user",
				Content:   userInput,
				Timestamp: time.Now(),
			}
			m.nodes = append(m.nodes, userNode)

			// Clear input
			m.textarea.Reset()
			m.textarea.SetHeight(1)
			m.updateViewportHeight()
			m.updateViewportContent()

			// Update status bar to show sending
			statusModel, _ := m.statusBar.Update(status.StatusUpdateMsg{
				Status: "Sending",
				State:  process.StateSending,
			})
			m.statusBar = statusModel.(status.StatusModel)

			// Start streaming from registered provider
			return m, tui.StreamFromProvider(
				m.streamManager,
				"",          // Empty to use router
				userInput,   // Prompt
				"assistant", // Node type for response
			)
		}
	}

	// Let the textarea handle the key
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)

	// Recalculate and update height after any key input
	newHeight := m.calculateTextAreaHeight()
	if m.textarea.Height() != newHeight {
		m.textarea.SetHeight(newHeight)
		m.updateViewportHeight()
	}

	return m, cmd
}
