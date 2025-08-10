package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// StreamFromProvider creates a command to stream from a registered provider
// If sourceID is empty, it will use the default provider
func StreamFromProvider(mgr *Manager, sourceID string, prompt string, nodeType string) tea.Cmd {
	return func() tea.Msg {
		// Use default provider if sourceID not specified
		if sourceID == "" {
			sourceID = "ollama-main"
		}

		source, exists := mgr.Registry.Get(sourceID)
		if !exists {
			return StreamEndMsg{
				StreamID: sourceID,
				Error:    fmt.Errorf("source %s not found", sourceID),
			}
		}

		// Create unique stream ID
		streamID := fmt.Sprintf("%s-%d", sourceID, time.Now().UnixNano())

		// Start the stream with prompt
		mgr.StartStream(streamID, source.Type, nodeType, prompt)

		// Return start message with prompt
		return StreamStartMsg{
			StreamID:   streamID,
			SourceType: nodeType,
			Prompt:     prompt,
		}
	}
}

// Message types for streaming
type StreamStartMsg struct {
	StreamID   string
	SourceType string
	Prompt     string
}

type StreamEndMsg struct {
	StreamID     string
	Error        error
	FinalContent string
}
