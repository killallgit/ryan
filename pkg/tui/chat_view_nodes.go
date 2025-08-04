package tui

import (
	"github.com/killallgit/ryan/pkg/logger"
)

// Node interaction and management methods for ChatView
// This file contains all node navigation, selection, and management logic

func (cv *ChatView) handleNodeNavigation(up bool) bool {
	// Only handle if using nodes
	if !cv.messages.UseNodes || cv.messages.NodeManager == nil {
		return false
	}

	log := logger.WithComponent("chat_view")

	var moved bool
	if up {
		moved = cv.messages.MoveFocusUp()
		log.Debug("Node navigation up", "moved", moved)
	} else {
		moved = cv.messages.MoveFocusDown()
		log.Debug("Node navigation down", "moved", moved)
	}

	if moved {
		// Post focus change event
		focusedNodeID := cv.messages.NodeManager.GetFocusedNode()
		cv.screen.PostEvent(NewMessageNodeFocusEvent(focusedNodeID))

		// Update status bar to show focused node
		cv.updateStatusForMode()

		// Auto-scroll to keep focused node visible
		cv.autoScrollToFocusedNode()
	}

	return moved
}

func (cv *ChatView) handleNodeToggleSelection() bool {
	// Only handle if using nodes
	if !cv.messages.UseNodes || cv.messages.NodeManager == nil {
		return false
	}

	focusedNodeID := cv.messages.NodeManager.GetFocusedNode()
	if focusedNodeID == "" {
		return false
	}

	log := logger.WithComponent("chat_view")
	log.Debug("Toggling selection for focused node", "node_id", focusedNodeID)

	if cv.messages.NodeManager.SelectNode(focusedNodeID) {
		// Get the new selection state
		isSelected := cv.messages.NodeManager.IsNodeSelected(focusedNodeID)
		cv.screen.PostEvent(NewMessageNodeSelectEvent(focusedNodeID, isSelected))
		return true
	}

	return false
}

func (cv *ChatView) handleNodeToggleExpansion() bool {
	// Only handle if using nodes
	if !cv.messages.UseNodes || cv.messages.NodeManager == nil {
		return false
	}

	focusedNodeID := cv.messages.NodeManager.GetFocusedNode()
	if focusedNodeID == "" {
		return false
	}

	log := logger.WithComponent("chat_view")
	log.Debug("Toggling expansion for focused node", "node_id", focusedNodeID)

	if cv.messages.NodeManager.ToggleNodeExpansion(focusedNodeID) {
		// Get the node to check its new state
		if node, exists := cv.messages.NodeManager.GetNode(focusedNodeID); exists {
			cv.screen.PostEvent(NewMessageNodeExpandEvent(focusedNodeID, node.State().Expanded))
		}
		return true
	}

	return false
}

func (cv *ChatView) handleSelectAllNodes() bool {
	// Only handle if using nodes
	if !cv.messages.UseNodes || cv.messages.NodeManager == nil {
		return false
	}

	log := logger.WithComponent("chat_view")
	log.Debug("Selecting all nodes")

	// Get all nodes and select them
	nodes := cv.messages.NodeManager.GetNodes()
	for _, node := range nodes {
		cv.messages.NodeManager.SetNodeSelected(node.ID(), true)
	}

	// Post selection events for all nodes
	for _, node := range nodes {
		cv.screen.PostEvent(NewMessageNodeSelectEvent(node.ID(), true))
	}

	return len(nodes) > 0
}

func (cv *ChatView) handleClearNodeSelection() bool {
	// Only handle if using nodes
	if !cv.messages.UseNodes || cv.messages.NodeManager == nil {
		return false
	}

	log := logger.WithComponent("chat_view")
	log.Debug("Clearing all node selections")

	selectedNodes := cv.messages.GetSelectedNodes()
	cv.messages.ClearSelection()

	// Post deselection events
	for _, nodeID := range selectedNodes {
		cv.screen.PostEvent(NewMessageNodeSelectEvent(nodeID, false))
	}

	return len(selectedNodes) > 0
}

func (cv *ChatView) autoScrollToFocusedNode() {
	if !cv.messages.UseNodes || cv.messages.NodeManager == nil {
		return
	}

	focusedNodeID := cv.messages.NodeManager.GetFocusedNode()
	if focusedNodeID == "" {
		return
	}

	// Get the focused node to check its bounds
	node, exists := cv.messages.NodeManager.GetNode(focusedNodeID)
	if !exists {
		return
	}

	bounds := node.Bounds()
	log := logger.WithComponent("chat_view")
	log.Debug("Auto-scrolling to focused node",
		"node_id", focusedNodeID,
		"node_y", bounds.Y,
		"node_height", bounds.Height,
		"current_scroll", cv.messages.Scroll,
		"display_height", cv.messages.Height)

	// Calculate visible area bounds
	visibleTop := cv.messages.Scroll
	visibleBottom := cv.messages.Scroll + cv.messages.Height

	// Check if node is fully visible
	nodeTop := bounds.Y
	nodeBottom := bounds.Y + bounds.Height

	// Calculate new scroll position if needed
	newScroll := cv.messages.Scroll

	if nodeTop < visibleTop {
		// Node is above visible area - scroll up to show it at the top
		newScroll = nodeTop
		log.Debug("Scrolling up to show focused node", "new_scroll", newScroll)
	} else if nodeBottom > visibleBottom {
		// Node is below visible area - scroll down to show it at the bottom
		newScroll = nodeBottom - cv.messages.Height
		log.Debug("Scrolling down to show focused node", "new_scroll", newScroll)
	}

	// Ensure scroll doesn't go negative
	if newScroll < 0 {
		newScroll = 0
	}

	// Update scroll if it changed
	if newScroll != cv.messages.Scroll {
		cv.messages = cv.messages.WithScroll(newScroll)
		log.Debug("Updated scroll position", "old_scroll", cv.messages.Scroll, "new_scroll", newScroll)
	}
}
