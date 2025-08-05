package agents

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestActivityTree(t *testing.T) {
	// Create a new activity tree
	tree := NewActivityTree()

	// Add some activities
	node1, err := tree.AddActivity("task1", "", "FileAgent", "reading main.go", OperationTypeTool)
	if err != nil {
		t.Fatalf("Failed to add activity: %v", err)
	}

	node2, err := tree.AddActivity("task2", "", "SearchAgent", "searching for patterns", OperationTypeAnalysis)
	if err != nil {
		t.Fatalf("Failed to add activity: %v", err)
	}

	// Update statuses
	tree.UpdateActivity("task1", ActivityStatusActive, 50)
	tree.UpdateActivity("task2", ActivityStatusPending, 0)

	// Test String() output
	output := tree.String()
	if output == "" {
		t.Error("Expected non-empty tree output")
	}

	// Verify the output contains our agents
	if !containsStr(output, "FileAgent") {
		t.Error("Expected output to contain FileAgent")
	}
	if !containsStr(output, "SearchAgent") {
		t.Error("Expected output to contain SearchAgent")
	}

	// Test AddNode method
	node3 := &ActivityNode{
		ID:            "task3",
		AgentName:     "ChatController",
		Operation:     "bash(ls -la)",
		OperationType: OperationTypeTool,
		Status:        ActivityStatusActive,
		StartTime:     time.Now(),
	}
	err = tree.AddNode(node3)
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	// Verify node was added
	if node, exists := tree.GetNode("task3"); !exists {
		t.Error("Expected node3 to be in tree")
	} else if node.AgentName != "ChatController" {
		t.Errorf("Expected agent name ChatController, got %s", node.AgentName)
	}

	// Test activity completion
	tree.CompleteActivity("task1")
	if node1.Status != ActivityStatusComplete {
		t.Error("Expected task1 to be complete")
	}

	// Test error handling
	testErr := errors.New("test error")
	tree.ErrorActivity("task2", testErr)
	if node2.Status != ActivityStatusError {
		t.Error("Expected task2 to have error status")
	}

	// Test statistics
	stats := tree.GetStatistics()
	if stats["total_nodes"].(int) != 3 {
		t.Errorf("Expected 3 total nodes, got %d", stats["total_nodes"].(int))
	}
}

func TestActivityTreeFormatting(t *testing.T) {
	tree := NewActivityTree()

	// Create a nested structure
	_, err := tree.AddActivity("parent", "", "ScrumMaster", "planning sprint", OperationTypePlanning)
	if err != nil {
		t.Fatalf("Failed to add parent: %v", err)
	}
	tree.UpdateActivity("parent", ActivityStatusActive, 30)

	_, err = tree.AddActivity("child1", "parent", "FileAgent", "reading files", OperationTypeTool)
	if err != nil {
		t.Fatalf("Failed to add child1: %v", err)
	}
	tree.UpdateActivity("child1", ActivityStatusActive, 50)

	_, err = tree.AddActivity("child2", "parent", "SearchAgent", "searching", OperationTypeAnalysis)
	if err != nil {
		t.Fatalf("Failed to add child2: %v", err)
	}
	tree.UpdateActivity("child2", ActivityStatusPending, 0)

	// Get formatted output
	output := tree.FormatTree()
	t.Logf("Tree output:\n%s", output)

	// Verify tree structure
	if !containsStr(output, "├──") || !containsStr(output, "└──") {
		t.Error("Expected tree to contain branch characters")
	}

	// Verify hierarchy
	if !containsStr(output, "ScrumMaster") {
		t.Error("Expected parent node in output")
	}
}

func TestActivityNodeOperations(t *testing.T) {
	node := NewActivityNode("test", "TestAgent", "testing", OperationTypeTool)

	// Test initial state
	if node.Status != ActivityStatusPending {
		t.Error("Expected initial status to be pending")
	}

	// Test status update
	node.UpdateStatus(ActivityStatusActive)
	if node.Status != ActivityStatusActive {
		t.Error("Expected status to be active")
	}

	// Test progress update
	node.UpdateProgress(75.5)
	if node.Progress != 75.5 {
		t.Errorf("Expected progress to be 75.5, got %f", node.Progress)
	}

	// Test error setting
	testErr := errors.New("test error")
	node.SetError(testErr)
	if node.Status != ActivityStatusError {
		t.Error("Expected status to be error after SetError")
	}

	// Test duration calculation
	time.Sleep(100 * time.Millisecond)
	duration := node.GetDuration()
	if duration < 100*time.Millisecond {
		t.Error("Expected duration to be at least 100ms")
	}

	// Test child operations
	child := NewActivityNode("child", "ChildAgent", "child op", OperationTypeAgent)
	node.AddChild(child)
	if len(node.Children) != 1 {
		t.Error("Expected node to have 1 child")
	}

	// Test IsActive
	child.UpdateStatus(ActivityStatusActive)
	if !node.IsActive() {
		t.Error("Expected node to be active when child is active")
	}
}

func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
