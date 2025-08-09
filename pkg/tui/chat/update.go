package chat

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/chat"
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

		// Update status bar - start in thinking state
		statusModel, _ := m.statusBar.Update(status.StartStreamingMsg{
			State: status.StateThinking,
		})
		m.statusBar = statusModel.(status.StatusModel)

		// Start channel-based streaming
		go m.startLLMStream(msg.StreamID, msg.Prompt)

		// Start listening for chunks on the channel
		return m, waitForChunk(m.chunkChan)

	case chunkMsg:
		// Channel-based chunk message

		// Handle stream end
		if msg.IsEnd {
			// Mark stream as complete
			for i := range m.nodes {
				if m.nodes[i].StreamID == msg.StreamID {
					m.nodes[i].IsStreaming = false
					if msg.Error != nil {
						m.nodes[i].Type = "error"
						m.nodes[i].Content = fmt.Sprintf("Error: %v", msg.Error)
					} else {
						// Save assistant response to chat history
						if m.chatManager != nil {
							m.chatManager.AddMessage(chat.RoleAssistant, m.nodes[i].Content)
						}
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
		}

		// Handle regular chunk
		for i := range m.nodes {
			if m.nodes[i].StreamID == msg.StreamID {
				// If this is the first content, transition from thinking to receiving
				if m.nodes[i].Content == "" && msg.Content != "" {
					statusModel, _ := m.statusBar.Update(status.SetProcessStateMsg{
						State: status.StateReceiving,
					})
					m.statusBar = statusModel.(status.StatusModel)
				}
				m.nodes[i].Content += msg.Content
				break
			}
		}

		// Update viewport with all nodes
		m.updateViewportContent()

		// Continue listening for more chunks
		return m, waitForChunk(m.chunkChan)

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

// startLLMStream starts streaming from agent and sends chunks to channel
func (m chatModel) startLLMStream(streamID, prompt string) {
	if m.agent == nil {
		m.chunkChan <- StreamChunk{
			StreamID: streamID,
			IsEnd:    true,
			Error:    fmt.Errorf("agent not initialized"),
		}
		return
	}

	// Create a stream handler that sends chunks to the channel
	streamHandler := &channelStreamHandler{
		streamID:  streamID,
		chunkChan: m.chunkChan,
	}

	// Use agent to generate streaming response
	ctx := context.Background()
	err := m.agent.ExecuteStream(ctx, prompt, streamHandler)
	if err != nil {
		m.chunkChan <- StreamChunk{
			StreamID: streamID,
			IsEnd:    true,
			Error:    err,
		}
	}
}

// channelStreamHandler implements agent.StreamHandler to send chunks to a channel
type channelStreamHandler struct {
	streamID  string
	chunkChan chan<- StreamChunk
}

func (h *channelStreamHandler) OnChunk(chunk string) error {
	h.chunkChan <- StreamChunk{
		StreamID: h.streamID,
		Content:  chunk,
		IsEnd:    false,
	}
	return nil
}

func (h *channelStreamHandler) OnComplete(finalContent string) error {
	h.chunkChan <- StreamChunk{
		StreamID: h.streamID,
		IsEnd:    true,
	}
	return nil
}

func (h *channelStreamHandler) OnError(err error) {
	h.chunkChan <- StreamChunk{
		StreamID: h.streamID,
		IsEnd:    true,
		Error:    err,
	}
}
