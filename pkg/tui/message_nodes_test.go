package tui

import (
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
)

func TestTextMessageNode_Basic(t *testing.T) {
	msg := chat.Message{
		Role:      chat.RoleUser,
		Content:   "This is a test message",
		Timestamp: time.Now(),
	}

	node := NewTextMessageNode(msg, "test-node-1")

	// Test basic properties
	if node.ID() != "test-node-1" {
		t.Errorf("Expected ID 'test-node-1', got '%s'", node.ID())
	}

	if node.NodeType() != NodeTypeText {
		t.Errorf("Expected NodeTypeText, got %d", node.NodeType())
	}

	if node.Message().Content != msg.Content {
		t.Error("Message content should match original")
	}

	// Test initial state
	state := node.State()
	if state.Selected || !state.Expanded || state.Focused || state.Hovered {
		t.Error("Initial state should be unselected, expanded, unfocused, unhovered")
	}
}

func TestTextMessageNode_LongContent(t *testing.T) {
	// Create a long message that should be collapsible
	longContent := ""
	for i := 0; i < 200; i++ {
		longContent += "This is a very long message that should trigger the collapsible behavior. "
	}

	msg := chat.Message{
		Role:      chat.RoleUser,
		Content:   longContent,
		Timestamp: time.Now(),
	}

	node := NewTextMessageNode(msg, "long-node")

	// Should be collapsible due to length
	if !node.IsCollapsible() {
		t.Error("Long message should be collapsible")
	}

	if !node.HasDetailView() {
		t.Error("Long message should have detail view")
	}

	// Test preview text
	preview := node.GetPreviewText()
	if len(preview) > 103 { // 100 chars + "..."
		t.Errorf("Preview text too long: %d chars", len(preview))
	}
}

func TestTextMessageNode_Rendering(t *testing.T) {
	msg := chat.Message{
		Role:      chat.RoleUser,
		Content:   "Short message",
		Timestamp: time.Now(),
	}

	node := NewTextMessageNode(msg, "render-test")

	// Test expanded rendering
	area := Rect{X: 0, Y: 0, Width: 50, Height: 10}
	expandedState := NewNodeState() // Default is expanded
	lines := node.Render(area, expandedState)

	if len(lines) == 0 {
		t.Error("Should render at least one line")
	}

	if lines[0].Text != "Short message" {
		t.Errorf("Expected 'Short message', got '%s'", lines[0].Text)
	}

	// For short messages, collapsed should be same as expanded
	collapsedState := expandedState.ToggleExpanded()
	collapsedLines := node.Render(area, collapsedState)

	if len(collapsedLines) != len(lines) {
		t.Error("Short message should render same when collapsed")
	}
}

func TestThinkingMessageNode_Basic(t *testing.T) {
	msg := chat.Message{
		Role:      chat.RoleAssistant,
		Content:   "<think>I need to think about this</think>Here is my response",
		Timestamp: time.Now(),
	}

	node := NewThinkingMessageNode(msg, "thinking-node")

	// Test basic properties
	if node.NodeType() != NodeTypeThinking {
		t.Errorf("Expected NodeTypeThinking, got %d", node.NodeType())
	}

	// Should be collapsible because it has thinking content
	if !node.IsCollapsible() {
		t.Error("Thinking message should be collapsible")
	}

	if !node.HasDetailView() {
		t.Error("Thinking message should have detail view")
	}

	// Test preview text (should be the response part)
	preview := node.GetPreviewText()
	if preview != "Here is my response" {
		t.Errorf("Expected response content in preview, got '%s'", preview)
	}
}

func TestThinkingMessageNode_Rendering(t *testing.T) {
	msg := chat.Message{
		Role:      chat.RoleAssistant,
		Content:   "<think>Complex thinking process here</think>Final response content",
		Timestamp: time.Now(),
	}

	node := NewThinkingMessageNode(msg, "thinking-render")
	area := Rect{X: 0, Y: 0, Width: 80, Height: 20}

	// Test expanded rendering
	expandedState := NewNodeState() // Default is expanded
	expandedLines := node.Render(area, expandedState)

	if len(expandedLines) == 0 {
		t.Error("Should render at least one line when expanded")
	}

	// Should contain thinking content when expanded
	hasThinking := false
	hasResponse := false
	for _, line := range expandedLines {
		if line.Text != "" {
			if line.Text == "Thinking: Complex thinking process here" {
				hasThinking = true
			}
			if line.Text == "Final response content" {
				hasResponse = true
			}
		}
	}

	if !hasThinking {
		t.Error("Expanded view should contain thinking content")
	}

	if !hasResponse {
		t.Error("Expanded view should contain response content")
	}

	// Test collapsed rendering
	collapsedState := expandedState.ToggleExpanded()
	// Create a new node with the collapsed state
	collapsedNode := node.WithState(collapsedState)
	collapsedLines := collapsedNode.Render(area, collapsedState)

	if len(collapsedLines) == 0 {
		t.Error("Should render at least one line when collapsed")
	}

	// Should show response content in collapsed mode (not thinking indicator)
	hasResponseContent := false
	for _, line := range collapsedLines {
		if line.Text != "" && line.Text == "Final response content" {
			hasResponseContent = true
			break
		}
	}

	if !hasResponseContent {
		t.Error("Collapsed view should show response content")
	}

	// Collapsed should generally be shorter than expanded (fewer lines)
	if len(collapsedLines) >= len(expandedLines) {
		t.Error("Collapsed view should typically be shorter than expanded view")
	}
}

func TestToolCallMessageNode_Basic(t *testing.T) {
	msg := chat.Message{
		Role:      chat.RoleAssistant,
		Content:   "",
		Timestamp: time.Now(),
		ToolCalls: []chat.ToolCall{
			{
				Function: chat.ToolFunction{
					Name: "test_tool",
					Arguments: map[string]any{
						"param1": "value1",
						"param2": 42,
					},
				},
			},
		},
	}

	node := NewToolCallMessageNode(msg, "tool-node")

	// Test basic properties
	if node.NodeType() != NodeTypeToolCall {
		t.Errorf("Expected NodeTypeToolCall, got %d", node.NodeType())
	}

	// Should be collapsible to hide/show arguments
	if !node.IsCollapsible() {
		t.Error("Tool call message should be collapsible")
	}

	if !node.HasDetailView() {
		t.Error("Tool call message should have detail view")
	}

	// Test preview text
	preview := node.GetPreviewText()
	expectedPrefix := "Tool: test_tool"
	if preview != expectedPrefix {
		t.Errorf("Expected preview to start with '%s', got '%s'", expectedPrefix, preview)
	}
}

func TestToolCallMessageNode_Rendering(t *testing.T) {
	msg := chat.Message{
		Role:      chat.RoleAssistant,
		Content:   "",
		Timestamp: time.Now(),
		ToolCalls: []chat.ToolCall{
			{
				Function: chat.ToolFunction{
					Name:      "example_tool",
					Arguments: map[string]any{"arg": "value"},
				},
			},
		},
	}

	node := NewToolCallMessageNode(msg, "tool-render")
	area := Rect{X: 0, Y: 0, Width: 80, Height: 10}

	// Test expanded rendering
	expandedState := NewNodeState() // Default is expanded
	expandedLines := node.Render(area, expandedState)

	if len(expandedLines) == 0 {
		t.Error("Should render at least one line")
	}

	// Should show tool name and arguments when expanded
	foundToolWithArgs := false
	for _, line := range expandedLines {
		if line.Text != "" && len(line.Text) > 20 { // Tool line with args should be longer
			foundToolWithArgs = true
			break
		}
	}

	if !foundToolWithArgs {
		t.Error("Expanded view should show tool arguments")
	}

	// Test collapsed rendering
	collapsedState := expandedState.ToggleExpanded()
	collapsedLines := node.Render(area, collapsedState)

	if len(collapsedLines) == 0 {
		t.Error("Should render at least one line when collapsed")
	}

	// Collapsed view should typically be shorter
	if len(collapsedLines) > len(expandedLines) {
		t.Error("Collapsed view should not be longer than expanded view")
	}
}

func TestNodeStateHelpers(t *testing.T) {
	state := NewNodeState()

	// Test initial state
	if state.Selected || !state.Expanded || state.Focused || state.Hovered {
		t.Error("Initial state should be unselected, expanded, unfocused, unhovered")
	}

	// Test toggles
	toggled := state.ToggleSelected()
	if !toggled.Selected {
		t.Error("ToggleSelected should set selected to true")
	}

	toggled = state.ToggleExpanded()
	if toggled.Expanded {
		t.Error("ToggleExpanded should set expanded to false")
	}

	// Test setters
	withSelected := state.WithSelected(true)
	if !withSelected.Selected {
		t.Error("WithSelected(true) should set selected to true")
	}

	withFocused := state.WithFocused(true)
	if !withFocused.Focused {
		t.Error("WithFocused(true) should set focused to true")
	}

	withHovered := state.WithHovered(true)
	if !withHovered.Hovered {
		t.Error("WithHovered(true) should set hovered to true")
	}

	// Test immutability - original state should be unchanged
	if state.Selected || !state.Expanded || state.Focused || state.Hovered {
		t.Error("Original state should remain unchanged after operations")
	}
}
