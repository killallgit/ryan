package tui

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
)

// MessageNodeManager manages the state and lifecycle of message nodes
type MessageNodeManager struct {
	nodes         []MessageNode      // Ordered list of message nodes
	nodeIndex     map[string]int     // Map from node ID to index in nodes slice
	selectedNodes map[string]bool    // Set of currently selected node IDs
	focusedNode   string             // ID of the currently focused node
	registry      *NodeRegistry      // Factory for creating nodes
	nextNodeID    int                // Counter for generating unique node IDs
}

// NewMessageNodeManager creates a new message node manager
func NewMessageNodeManager() *MessageNodeManager {
	return &MessageNodeManager{
		nodes:         make([]MessageNode, 0),
		nodeIndex:     make(map[string]int),
		selectedNodes: make(map[string]bool),
		focusedNode:   "",
		registry:      NewNodeRegistry(),
		nextNodeID:    1,
	}
}

// SetMessages replaces all current nodes with nodes created from the given messages
func (mnm *MessageNodeManager) SetMessages(messages []chat.Message) {
	// Clear existing state
	mnm.nodes = make([]MessageNode, 0, len(messages))
	mnm.nodeIndex = make(map[string]int)
	mnm.selectedNodes = make(map[string]bool)
	mnm.focusedNode = ""
	
	// Create nodes for each message
	for _, msg := range messages {
		nodeID := mnm.generateNodeID()
		node := mnm.registry.CreateNode(msg, nodeID)
		mnm.addNode(node)
	}
}

// AddMessage adds a new message as a node at the end of the list
func (mnm *MessageNodeManager) AddMessage(msg chat.Message) string {
	nodeID := mnm.generateNodeID()
	node := mnm.registry.CreateNode(msg, nodeID)
	mnm.addNode(node)
	return nodeID
}

// UpdateMessage updates an existing message node
func (mnm *MessageNodeManager) UpdateMessage(nodeID string, newMsg chat.Message) bool {
	index, exists := mnm.nodeIndex[nodeID]
	if !exists {
		return false
	}
	
	// Preserve the current state
	currentState := mnm.nodes[index].State()
	currentBounds := mnm.nodes[index].Bounds()
	
	// Create new node with updated message
	newNode := mnm.registry.CreateNode(newMsg, nodeID)
	newNode = newNode.WithState(currentState)
	newNode = newNode.WithBounds(currentBounds)
	
	mnm.nodes[index] = newNode
	return true
}

// GetNodes returns all nodes in order
func (mnm *MessageNodeManager) GetNodes() []MessageNode {
	return mnm.nodes
}

// GetNode returns a specific node by ID
func (mnm *MessageNodeManager) GetNode(nodeID string) (MessageNode, bool) {
	index, exists := mnm.nodeIndex[nodeID]
	if !exists {
		return nil, false
	}
	return mnm.nodes[index], true
}

// SelectNode toggles the selection state of a node
func (mnm *MessageNodeManager) SelectNode(nodeID string) bool {
	index, exists := mnm.nodeIndex[nodeID]
	if !exists {
		return false
	}
	
	currentState := mnm.nodes[index].State()
	newState := currentState.ToggleSelected()
	mnm.nodes[index] = mnm.nodes[index].WithState(newState)
	
	// Update selection tracking
	if newState.Selected {
		mnm.selectedNodes[nodeID] = true
	} else {
		delete(mnm.selectedNodes, nodeID)
	}
	
	return true
}

// SetNodeSelected sets the selection state of a node
func (mnm *MessageNodeManager) SetNodeSelected(nodeID string, selected bool) bool {
	index, exists := mnm.nodeIndex[nodeID]
	if !exists {
		return false
	}
	
	currentState := mnm.nodes[index].State()
	newState := currentState.WithSelected(selected)
	mnm.nodes[index] = mnm.nodes[index].WithState(newState)
	
	// Update selection tracking
	if selected {
		mnm.selectedNodes[nodeID] = true
	} else {
		delete(mnm.selectedNodes, nodeID)
	}
	
	return true
}

// ClearSelection clears all selections
func (mnm *MessageNodeManager) ClearSelection() {
	for nodeID := range mnm.selectedNodes {
		mnm.SetNodeSelected(nodeID, false)
	}
}

// GetSelectedNodes returns the IDs of all selected nodes
func (mnm *MessageNodeManager) GetSelectedNodes() []string {
	selected := make([]string, 0, len(mnm.selectedNodes))
	for nodeID := range mnm.selectedNodes {
		selected = append(selected, nodeID)
	}
	sort.Strings(selected) // For consistent ordering
	return selected
}

// SetFocusedNode sets which node has keyboard focus
func (mnm *MessageNodeManager) SetFocusedNode(nodeID string) bool {
	// Clear previous focus
	if mnm.focusedNode != "" {
		if index, exists := mnm.nodeIndex[mnm.focusedNode]; exists {
			currentState := mnm.nodes[index].State()
			newState := currentState.WithFocused(false)
			mnm.nodes[index] = mnm.nodes[index].WithState(newState)
		}
	}
	
	// Set new focus
	if nodeID != "" {
		index, exists := mnm.nodeIndex[nodeID]
		if !exists {
			return false
		}
		
		currentState := mnm.nodes[index].State()
		newState := currentState.WithFocused(true)
		mnm.nodes[index] = mnm.nodes[index].WithState(newState)
	}
	
	mnm.focusedNode = nodeID
	return true
}

// GetFocusedNode returns the ID of the currently focused node
func (mnm *MessageNodeManager) GetFocusedNode() string {
	return mnm.focusedNode
}

// MoveFocusUp moves focus to the previous node
func (mnm *MessageNodeManager) MoveFocusUp() bool {
	if len(mnm.nodes) == 0 {
		return false
	}
	
	currentIndex := -1
	if mnm.focusedNode != "" {
		if index, exists := mnm.nodeIndex[mnm.focusedNode]; exists {
			currentIndex = index
		}
	}
	
	// Move to previous node, or last node if none focused
	newIndex := currentIndex - 1
	if newIndex < 0 {
		newIndex = len(mnm.nodes) - 1
	}
	
	if newIndex >= 0 && newIndex < len(mnm.nodes) {
		return mnm.SetFocusedNode(mnm.nodes[newIndex].ID())
	}
	
	return false
}

// MoveFocusDown moves focus to the next node
func (mnm *MessageNodeManager) MoveFocusDown() bool {
	if len(mnm.nodes) == 0 {
		return false
	}
	
	currentIndex := -1
	if mnm.focusedNode != "" {
		if index, exists := mnm.nodeIndex[mnm.focusedNode]; exists {
			currentIndex = index
		}
	}
	
	// Move to next node, or first node if none focused
	newIndex := currentIndex + 1
	if newIndex >= len(mnm.nodes) {
		newIndex = 0
	}
	
	if newIndex >= 0 && newIndex < len(mnm.nodes) {
		return mnm.SetFocusedNode(mnm.nodes[newIndex].ID())
	}
	
	return false
}

// ToggleNodeExpansion toggles the expanded state of a node
func (mnm *MessageNodeManager) ToggleNodeExpansion(nodeID string) bool {
	index, exists := mnm.nodeIndex[nodeID]
	if !exists {
		return false
	}
	
	node := mnm.nodes[index]
	if !node.IsCollapsible() {
		return false
	}
	
	currentState := node.State()
	newState := currentState.ToggleExpanded()
	mnm.nodes[index] = node.WithState(newState)
	
	return true
}

// HandleClick handles a mouse click at the given coordinates
// Returns the node ID that was clicked, if any
func (mnm *MessageNodeManager) HandleClick(x, y int) (string, bool) {
	// Find which node contains the click coordinates
	for _, node := range mnm.nodes {
		bounds := node.Bounds()
		if x >= bounds.X && x < bounds.X+bounds.Width &&
		   y >= bounds.Y && y < bounds.Y+bounds.Height {
			
			// Let the node handle the click
			relativeX := x - bounds.X
			relativeY := y - bounds.Y
			
			if handled, newState := node.HandleClick(relativeX, relativeY); handled {
				// Update the node with new state
				index := mnm.nodeIndex[node.ID()]
				mnm.nodes[index] = node.WithState(newState)
				
				// Update our tracking
				if newState.Selected {
					mnm.selectedNodes[node.ID()] = true
				} else {
					delete(mnm.selectedNodes, node.ID())
				}
				
				return node.ID(), true
			}
		}
	}
	
	return "", false
}

// HandleKeyEvent handles a keyboard event for the focused node
func (mnm *MessageNodeManager) HandleKeyEvent(ev *tcell.EventKey) bool {
	if mnm.focusedNode == "" {
		return false
	}
	
	index, exists := mnm.nodeIndex[mnm.focusedNode]
	if !exists {
		return false
	}
	
	node := mnm.nodes[index]
	if handled, newState := node.HandleKeyEvent(ev); handled {
		// Update the node with new state
		mnm.nodes[index] = node.WithState(newState)
		
		// Update our tracking
		if newState.Selected {
			mnm.selectedNodes[node.ID()] = true
		} else {
			delete(mnm.selectedNodes, node.ID())
		}
		
		return true
	}
	
	return false
}

// UpdateNodeBounds updates the screen bounds for a node
func (mnm *MessageNodeManager) UpdateNodeBounds(nodeID string, bounds NodeBounds) bool {
	index, exists := mnm.nodeIndex[nodeID]
	if !exists {
		return false
	}
	
	mnm.nodes[index] = mnm.nodes[index].WithBounds(bounds)
	return true
}

// SetNodeHovered sets the hover state for a node
func (mnm *MessageNodeManager) SetNodeHovered(nodeID string, hovered bool) bool {
	index, exists := mnm.nodeIndex[nodeID]
	if !exists {
		return false
	}
	
	currentState := mnm.nodes[index].State()
	newState := currentState.WithHovered(hovered)
	mnm.nodes[index] = mnm.nodes[index].WithState(newState)
	
	return true
}

// CalculateTotalHeight calculates the total height needed to render all nodes
func (mnm *MessageNodeManager) CalculateTotalHeight(width int) int {
	totalHeight := 0
	for i, node := range mnm.nodes {
		nodeHeight := node.CalculateHeight(width)
		totalHeight += nodeHeight
		
		// Add spacing between nodes (except after the last one)
		if i < len(mnm.nodes)-1 {
			totalHeight += 1
		}
	}
	return totalHeight
}

// Helper methods

func (mnm *MessageNodeManager) generateNodeID() string {
	id := fmt.Sprintf("node_%d", mnm.nextNodeID)
	mnm.nextNodeID++
	return id
}

func (mnm *MessageNodeManager) addNode(node MessageNode) {
	index := len(mnm.nodes)
	mnm.nodes = append(mnm.nodes, node)
	mnm.nodeIndex[node.ID()] = index
}

// GetNodeCount returns the total number of nodes
func (mnm *MessageNodeManager) GetNodeCount() int {
	return len(mnm.nodes)
}

// IsNodeSelected returns whether a specific node is selected
func (mnm *MessageNodeManager) IsNodeSelected(nodeID string) bool {
	return mnm.selectedNodes[nodeID]
}

// GetSelectionCount returns the number of selected nodes
func (mnm *MessageNodeManager) GetSelectionCount() int {
	return len(mnm.selectedNodes)
}

// Clear removes all nodes and resets state
func (mnm *MessageNodeManager) Clear() {
	mnm.nodes = make([]MessageNode, 0)
	mnm.nodeIndex = make(map[string]int)
	mnm.selectedNodes = make(map[string]bool)
	mnm.focusedNode = ""
}

// Streaming support

// SetMessagesWithStreaming replaces all nodes and handles a streaming message
func (mnm *MessageNodeManager) SetMessagesWithStreaming(messages []chat.Message, streamingMessage *chat.Message) {
	// Set regular messages first
	mnm.SetMessages(messages)
	
	// Add streaming message as a temporary node if provided
	if streamingMessage != nil && streamingMessage.Content != "" {
		streamingNodeID := mnm.AddStreamingMessage(*streamingMessage)
		// The streaming node is automatically added to the end
		_ = streamingNodeID
	}
}

// AddStreamingMessage adds a temporary streaming message node
func (mnm *MessageNodeManager) AddStreamingMessage(streamingMsg chat.Message) string {
	nodeID := "streaming_" + mnm.generateNodeID()
	node := mnm.registry.CreateNode(streamingMsg, nodeID)
	
	// Mark this as a streaming node (we could add metadata for this)
	mnm.addNode(node)
	return nodeID
}

// UpdateStreamingMessage updates the content of a streaming message node
func (mnm *MessageNodeManager) UpdateStreamingMessage(nodeID string, newContent string) bool {
	index, exists := mnm.nodeIndex[nodeID]
	if !exists {
		return false
	}
	
	// Preserve the current state and bounds
	currentNode := mnm.nodes[index]
	currentState := currentNode.State()
	currentBounds := currentNode.Bounds()
	
	// Create updated message
	updatedMessage := currentNode.Message()
	updatedMessage.Content = newContent
	
	// Create new node with updated content
	newNode := mnm.registry.CreateNode(updatedMessage, nodeID)
	newNode = newNode.WithState(currentState)
	newNode = newNode.WithBounds(currentBounds)
	
	mnm.nodes[index] = newNode
	return true
}

// RemoveStreamingMessage removes a streaming message node
func (mnm *MessageNodeManager) RemoveStreamingMessage(nodeID string) bool {
	index, exists := mnm.nodeIndex[nodeID]
	if !exists {
		return false
	}
	
	// Remove from selections if selected
	delete(mnm.selectedNodes, nodeID)
	
	// Clear focus if this node was focused
	if mnm.focusedNode == nodeID {
		mnm.focusedNode = ""
	}
	
	// Remove from nodes slice
	mnm.nodes = append(mnm.nodes[:index], mnm.nodes[index+1:]...)
	
	// Rebuild index map
	mnm.rebuildIndex()
	
	return true
}

// GetStreamingNodeID returns the ID of the streaming node (if any)
func (mnm *MessageNodeManager) GetStreamingNodeID() string {
	// Check if the last node is a streaming node (starts with "streaming_")
	if len(mnm.nodes) > 0 {
		lastNode := mnm.nodes[len(mnm.nodes)-1]
		if len(lastNode.ID()) > 10 && lastNode.ID()[:10] == "streaming_" {
			return lastNode.ID()
		}
	}
	return ""
}

// HasStreamingMessage checks if there's currently a streaming message
func (mnm *MessageNodeManager) HasStreamingMessage() bool {
	return mnm.GetStreamingNodeID() != ""
}

// Helper method to rebuild the index after removal
func (mnm *MessageNodeManager) rebuildIndex() {
	mnm.nodeIndex = make(map[string]int)
	for i, node := range mnm.nodes {
		mnm.nodeIndex[node.ID()] = i
	}
}