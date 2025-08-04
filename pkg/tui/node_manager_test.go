package tui

import (
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
)

func TestMessageNodeManager_Basic(t *testing.T) {
	manager := NewMessageNodeManager()

	// Test empty manager
	if len(manager.GetNodes()) != 0 {
		t.Errorf("Expected empty manager to have 0 nodes, got %d", len(manager.GetNodes()))
	}

	// Create test messages
	userMsg := chat.Message{
		Role:      chat.RoleUser,
		Content:   "Hello, this is a test message",
		Timestamp: time.Now(),
	}

	assistantMsg := chat.Message{
		Role:      chat.RoleAssistant,
		Content:   "Hello! This is a response from the assistant.",
		Timestamp: time.Now(),
	}

	thinkingMsg := chat.Message{
		Role:      chat.RoleAssistant,
		Content:   "<think>I need to think about this carefully</think>This is my response after thinking.",
		Timestamp: time.Now(),
	}

	messages := []chat.Message{userMsg, assistantMsg, thinkingMsg}

	// Set messages
	manager.SetMessages(messages)

	// Verify nodes were created
	nodes := manager.GetNodes()
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(nodes))
	}

	// Test node types - all should be NodeTypeText since thinking blocks are removed
	expectedTypes := []MessageNodeType{NodeTypeText, NodeTypeText, NodeTypeText}
	for i, node := range nodes {
		if node.NodeType() != expectedTypes[i] {
			t.Errorf("Node %d: expected type %d, got %d", i, expectedTypes[i], node.NodeType())
		}
	}
}

func TestMessageNodeManager_Selection(t *testing.T) {
	manager := NewMessageNodeManager()

	// Create test message
	userMsg := chat.Message{
		Role:      chat.RoleUser,
		Content:   "Test message for selection",
		Timestamp: time.Now(),
	}

	manager.SetMessages([]chat.Message{userMsg})
	nodes := manager.GetNodes()

	if len(nodes) == 0 {
		t.Fatal("Expected at least 1 node")
	}

	nodeID := nodes[0].ID()

	// Test initial selection state
	if manager.IsNodeSelected(nodeID) {
		t.Error("Node should not be selected initially")
	}

	// Test selection
	if !manager.SelectNode(nodeID) {
		t.Error("SelectNode should return true for valid node")
	}

	if !manager.IsNodeSelected(nodeID) {
		t.Error("Node should be selected after SelectNode")
	}

	// Test deselection
	if !manager.SelectNode(nodeID) {
		t.Error("SelectNode should return true for valid node")
	}

	if manager.IsNodeSelected(nodeID) {
		t.Error("Node should not be selected after second SelectNode call")
	}

	// Test clear selection
	manager.SetNodeSelected(nodeID, true)
	if !manager.IsNodeSelected(nodeID) {
		t.Error("Node should be selected")
	}

	manager.ClearSelection()
	if manager.IsNodeSelected(nodeID) {
		t.Error("Node should not be selected after ClearSelection")
	}
}

func TestMessageNodeManager_Focus(t *testing.T) {
	manager := NewMessageNodeManager()

	// Create test messages
	msg1 := chat.Message{Role: chat.RoleUser, Content: "Message 1", Timestamp: time.Now()}
	msg2 := chat.Message{Role: chat.RoleAssistant, Content: "Message 2", Timestamp: time.Now()}

	manager.SetMessages([]chat.Message{msg1, msg2})
	nodes := manager.GetNodes()

	if len(nodes) != 2 {
		t.Fatal("Expected 2 nodes")
	}

	// Test initial focus state
	if manager.GetFocusedNode() != "" {
		t.Error("No node should be focused initially")
	}

	// Test setting focus
	nodeID := nodes[0].ID()
	if !manager.SetFocusedNode(nodeID) {
		t.Error("SetFocusedNode should return true for valid node")
	}

	if manager.GetFocusedNode() != nodeID {
		t.Error("Focused node should match set node")
	}

	// Test focus navigation
	if !manager.MoveFocusDown() {
		t.Error("MoveFocusDown should return true")
	}

	if manager.GetFocusedNode() == nodeID {
		t.Error("Focus should have moved to next node")
	}

	// Test focus wrap-around
	if !manager.MoveFocusDown() {
		t.Error("MoveFocusDown should return true")
	}

	if manager.GetFocusedNode() != nodes[0].ID() {
		t.Error("Focus should wrap around to first node")
	}

	// Test move up
	if !manager.MoveFocusUp() {
		t.Error("MoveFocusUp should return true")
	}

	if manager.GetFocusedNode() != nodes[1].ID() {
		t.Error("Focus should move to last node")
	}
}

func TestMessageNodeManager_Expansion(t *testing.T) {
	manager := NewMessageNodeManager()

	// Create a tool call message that's collapsible
	toolCallMsg := chat.Message{
		Role:      chat.RoleAssistant,
		Content:   "I'll help you with that",
		Timestamp: time.Now(),
		ToolCalls: []chat.ToolCall{
			{
				Function: chat.ToolFunction{
					Name: "execute_bash",
					Arguments: map[string]any{
						"command": "ls -la",
					},
				},
			},
		},
	}

	manager.SetMessages([]chat.Message{toolCallMsg})
	nodes := manager.GetNodes()

	if len(nodes) != 1 {
		t.Fatal("Expected 1 node")
	}

	node := nodes[0]
	nodeID := node.ID()

	// Test that tool call nodes are collapsible
	if !node.IsCollapsible() {
		t.Error("Tool call node should be collapsible")
	}

	// Test initial expanded state (should be expanded by default)
	if !node.State().Expanded {
		t.Error("Node should be expanded by default")
	}

	// Test toggle expansion
	if !manager.ToggleNodeExpansion(nodeID) {
		t.Error("ToggleNodeExpansion should return true for collapsible node")
	}

	// Get updated node
	updatedNode, exists := manager.GetNode(nodeID)
	if !exists {
		t.Fatal("Node should exist")
	}

	if updatedNode.State().Expanded {
		t.Error("Node should be collapsed after toggle")
	}

	// Test toggle back to expanded
	if !manager.ToggleNodeExpansion(nodeID) {
		t.Error("ToggleNodeExpansion should return true")
	}
}

func TestMessageNodeManager_Streaming(t *testing.T) {
	manager := NewMessageNodeManager()

	// Create base messages
	userMsg := chat.Message{Role: chat.RoleUser, Content: "Test question", Timestamp: time.Now()}
	messages := []chat.Message{userMsg}

	// Create streaming message
	streamingMsg := chat.Message{
		Role:      chat.RoleAssistant,
		Content:   "Streaming response...",
		Timestamp: time.Now(),
	}

	// Test setting messages with streaming
	manager.SetMessagesWithStreaming(messages, &streamingMsg)
	nodes := manager.GetNodes()

	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes (1 regular + 1 streaming), got %d", len(nodes))
	}

	// Test that streaming message was added
	if !manager.HasStreamingMessage() {
		t.Error("Manager should have streaming message")
	}

	streamingNodeID := manager.GetStreamingNodeID()
	if streamingNodeID == "" {
		t.Error("Should have streaming node ID")
	}

	// Test updating streaming content
	if !manager.UpdateStreamingMessage(streamingNodeID, "Updated streaming content") {
		t.Error("UpdateStreamingMessage should return true")
	}

	streamingNode, exists := manager.GetNode(streamingNodeID)
	if !exists {
		t.Fatal("Streaming node should exist")
	}

	if streamingNode.Message().Content != "Updated streaming content" {
		t.Error("Streaming message content should be updated")
	}

	// Test removing streaming message
	if !manager.RemoveStreamingMessage(streamingNodeID) {
		t.Error("RemoveStreamingMessage should return true")
	}

	if manager.HasStreamingMessage() {
		t.Error("Manager should not have streaming message after removal")
	}

	// Should only have the original message now
	nodes = manager.GetNodes()
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node after removing streaming message, got %d", len(nodes))
	}
}
