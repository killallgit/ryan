package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
)

type MessageDisplay struct {
	Messages    []chat.Message      // Legacy field for backward compatibility
	NodeManager *MessageNodeManager // New node-based message management
	Width       int
	Height      int
	Scroll      int
	UseNodes    bool // Flag to enable node-based rendering
}

func NewMessageDisplay(width, height int) MessageDisplay {
	return MessageDisplay{
		Messages:    []chat.Message{},
		NodeManager: NewMessageNodeManager(),
		Width:       width,
		Height:      height,
		Scroll:      0,
		UseNodes:    false, // Default to legacy mode for compatibility
	}
}

// NewNodeBasedMessageDisplay creates a new MessageDisplay that uses the node system
func NewNodeBasedMessageDisplay(width, height int) MessageDisplay {
	return MessageDisplay{
		Messages:    []chat.Message{},
		NodeManager: NewMessageNodeManager(),
		Width:       width,
		Height:      height,
		Scroll:      0,
		UseNodes:    true,
	}
}

func (md MessageDisplay) WithMessages(messages []chat.Message) MessageDisplay {
	updated := MessageDisplay{
		Messages:    messages,
		NodeManager: md.NodeManager,
		Width:       md.Width,
		Height:      md.Height,
		Scroll:      md.Scroll,
		UseNodes:    md.UseNodes,
	}

	// If using nodes, update the node manager
	if updated.UseNodes && updated.NodeManager != nil {
		updated.NodeManager.SetMessages(messages)
	}

	return updated
}

func (md MessageDisplay) WithSize(width, height int) MessageDisplay {
	return MessageDisplay{
		Messages:    md.Messages,
		NodeManager: md.NodeManager,
		Width:       width,
		Height:      height,
		Scroll:      md.Scroll,
		UseNodes:    md.UseNodes,
	}
}

func (md MessageDisplay) WithScroll(scroll int) MessageDisplay {
	return MessageDisplay{
		Messages:    md.Messages,
		NodeManager: md.NodeManager,
		Width:       md.Width,
		Height:      md.Height,
		Scroll:      scroll,
		UseNodes:    md.UseNodes,
	}
}

// EnableNodes enables node-based rendering
func (md MessageDisplay) EnableNodes() MessageDisplay {
	updated := md
	updated.UseNodes = true

	// Sync current messages to node manager
	if updated.NodeManager != nil && len(updated.Messages) > 0 {
		updated.NodeManager.SetMessages(updated.Messages)
	}

	return updated
}

// DisableNodes disables node-based rendering (fallback to legacy)
func (md MessageDisplay) DisableNodes() MessageDisplay {
	updated := md
	updated.UseNodes = false
	return updated
}

// HandleClick handles mouse clicks for node-based displays
func (md MessageDisplay) HandleClick(x, y int) (string, bool) {
	if !md.UseNodes || md.NodeManager == nil {
		return "", false
	}
	return md.NodeManager.HandleClick(x, y)
}

// HandleKeyEvent handles keyboard events for node-based displays
func (md MessageDisplay) HandleKeyEvent(ev *tcell.EventKey) bool {
	if !md.UseNodes || md.NodeManager == nil {
		return false
	}
	return md.NodeManager.HandleKeyEvent(ev)
}

// MoveFocusUp moves focus to the previous node
func (md MessageDisplay) MoveFocusUp() bool {
	if !md.UseNodes || md.NodeManager == nil {
		return false
	}
	return md.NodeManager.MoveFocusUp()
}

// MoveFocusDown moves focus to the next node
func (md MessageDisplay) MoveFocusDown() bool {
	if !md.UseNodes || md.NodeManager == nil {
		return false
	}
	return md.NodeManager.MoveFocusDown()
}

// GetSelectedNodes returns the IDs of selected nodes
func (md MessageDisplay) GetSelectedNodes() []string {
	if !md.UseNodes || md.NodeManager == nil {
		return []string{}
	}
	return md.NodeManager.GetSelectedNodes()
}

// ClearSelection clears all node selections
func (md MessageDisplay) ClearSelection() {
	if md.UseNodes && md.NodeManager != nil {
		md.NodeManager.ClearSelection()
	}
}

// Streaming support methods

// WithStreamingMessage creates a display with both regular and streaming messages
func (md MessageDisplay) WithStreamingMessage(messages []chat.Message, streamingMessage *chat.Message) MessageDisplay {
	updated := MessageDisplay{
		Messages:    messages,
		NodeManager: md.NodeManager,
		Width:       md.Width,
		Height:      md.Height,
		Scroll:      md.Scroll,
		UseNodes:    md.UseNodes,
	}

	// If using nodes, update with streaming support
	if updated.UseNodes && updated.NodeManager != nil {
		updated.NodeManager.SetMessagesWithStreaming(messages, streamingMessage)
	} else {
		// Legacy mode: append streaming message to regular messages
		if streamingMessage != nil && streamingMessage.Content != "" {
			messagesWithStreaming := make([]chat.Message, len(messages), len(messages)+1)
			copy(messagesWithStreaming, messages)
			messagesWithStreaming = append(messagesWithStreaming, *streamingMessage)
			updated.Messages = messagesWithStreaming
		}
	}

	return updated
}

// UpdateStreamingContent updates the content of the streaming message
func (md MessageDisplay) UpdateStreamingContent(content string) MessageDisplay {
	if !md.UseNodes || md.NodeManager == nil {
		// For legacy mode, this would need to be handled at the chat view level
		return md
	}

	streamingNodeID := md.NodeManager.GetStreamingNodeID()
	if streamingNodeID != "" {
		md.NodeManager.UpdateStreamingMessage(streamingNodeID, content)
	}

	return md
}

// ClearStreamingMessage removes the streaming message node
func (md MessageDisplay) ClearStreamingMessage() MessageDisplay {
	if !md.UseNodes || md.NodeManager == nil {
		return md
	}

	streamingNodeID := md.NodeManager.GetStreamingNodeID()
	if streamingNodeID != "" {
		md.NodeManager.RemoveStreamingMessage(streamingNodeID)
	}

	return md
}

// HasStreamingMessage checks if there's currently a streaming message
func (md MessageDisplay) HasStreamingMessage() bool {
	if !md.UseNodes || md.NodeManager == nil {
		return false
	}

	return md.NodeManager.HasStreamingMessage()
}
