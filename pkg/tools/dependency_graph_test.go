package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyGraph_Basic(t *testing.T) {
	graph := NewDependencyGraph()
	assert.NotNil(t, graph)
	
	// Add nodes
	err := graph.AddNode("A", "tool_a", map[string]any{"param": "value"})
	require.NoError(t, err)
	
	err = graph.AddNode("B", "tool_b", map[string]any{})
	require.NoError(t, err)
	
	// Verify nodes were added
	nodeA := graph.GetNode("A")
	require.NotNil(t, nodeA)
	assert.Equal(t, "A", nodeA.ID)
	assert.Equal(t, "tool_a", nodeA.ToolName)
	assert.Equal(t, StatusPending, nodeA.Status)
	
	nodeB := graph.GetNode("B")
	require.NotNil(t, nodeB)
	assert.Equal(t, "B", nodeB.ID)
}

func TestDependencyGraph_AddDependency(t *testing.T) {
	graph := NewDependencyGraph()
	
	// Add nodes
	graph.AddNode("A", "tool_a", map[string]any{})
	graph.AddNode("B", "tool_b", map[string]any{})
	graph.AddNode("C", "tool_c", map[string]any{})
	
	// Add dependencies: A depends on B, B depends on C
	err := graph.AddDependency("A", "B")
	require.NoError(t, err)
	
	err = graph.AddDependency("B", "C")
	require.NoError(t, err)
	
	// Verify dependencies
	nodeA := graph.GetNode("A")
	assert.Contains(t, nodeA.Dependencies, "B")
	
	nodeB := graph.GetNode("B")
	assert.Contains(t, nodeB.Dependencies, "C")
	assert.Contains(t, nodeB.Dependents, "A")
	
	nodeC := graph.GetNode("C")
	assert.Contains(t, nodeC.Dependents, "B")
}

func TestDependencyGraph_TopologicalSort(t *testing.T) {
	graph := NewDependencyGraph()
	
	// Create a simple dependency chain: A -> B -> C
	graph.AddNode("A", "tool_a", map[string]any{})
	graph.AddNode("B", "tool_b", map[string]any{})
	graph.AddNode("C", "tool_c", map[string]any{})
	
	graph.AddDependency("A", "B") // A depends on B
	graph.AddDependency("B", "C") // B depends on C
	
	order, err := graph.TopologicalSort()
	require.NoError(t, err)
	
	// Should be in order: C, B, A (dependencies first)
	assert.Equal(t, []string{"C", "B", "A"}, order)
}

func TestDependencyGraph_ComplexTopologicalSort(t *testing.T) {
	graph := NewDependencyGraph()
	
	// Create a more complex dependency graph
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	
	graph.AddNode("A", "tool_a", map[string]any{})
	graph.AddNode("B", "tool_b", map[string]any{})
	graph.AddNode("C", "tool_c", map[string]any{})
	graph.AddNode("D", "tool_d", map[string]any{})
	
	graph.AddDependency("A", "B") // A depends on B
	graph.AddDependency("A", "C") // A depends on C
	graph.AddDependency("B", "D") // B depends on D
	graph.AddDependency("C", "D") // C depends on D
	
	order, err := graph.TopologicalSort()
	require.NoError(t, err)
	
	// D should be first, A should be last
	assert.Equal(t, "D", order[0])
	assert.Equal(t, "A", order[len(order)-1])
	
	// B and C should come before A but after D
	assert.Contains(t, order, "B")
	assert.Contains(t, order, "C")
}

func TestDependencyGraph_CycleDetection(t *testing.T) {
	graph := NewDependencyGraph()
	
	graph.AddNode("A", "tool_a", map[string]any{})
	graph.AddNode("B", "tool_b", map[string]any{})
	graph.AddNode("C", "tool_c", map[string]any{})
	
	// Create a cycle: A -> B -> C -> A
	graph.AddDependency("A", "B")
	graph.AddDependency("B", "C")
	
	// This should fail due to cycle
	err := graph.AddDependency("C", "A")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestDependencyGraph_CycleDetectionInSort(t *testing.T) {
	graph := NewDependencyGraph()
	
	// Manually create a cycle by bypassing the cycle check
	graph.AddNode("A", "tool_a", map[string]any{})
	graph.AddNode("B", "tool_b", map[string]any{})
	
	// Manually add cyclic dependencies
	graph.nodes["A"].Dependencies = []string{"B"}
	graph.nodes["B"].Dependencies = []string{"A"}
	graph.nodes["A"].Dependents = []string{"B"}
	graph.nodes["B"].Dependents = []string{"A"}
	
	_, err := graph.TopologicalSort()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycles")
}

func TestDependencyGraph_GetExecutableNodes(t *testing.T) {
	graph := NewDependencyGraph()
	
	graph.AddNode("A", "tool_a", map[string]any{})
	graph.AddNode("B", "tool_b", map[string]any{})
	graph.AddNode("C", "tool_c", map[string]any{})
	
	graph.AddDependency("A", "B") // A depends on B
	graph.AddDependency("B", "C") // B depends on C
	
	// Initially, only C should be executable (no dependencies)
	executable := graph.GetExecutableNodes()
	assert.Equal(t, []string{"C"}, executable)
	
	// Mark C as completed
	graph.MarkStatus("C", StatusCompleted)
	
	// Now B should be executable
	executable = graph.GetExecutableNodes()
	assert.Equal(t, []string{"B"}, executable)
	
	// Mark B as completed
	graph.MarkStatus("B", StatusCompleted)
	
	// Now A should be executable
	executable = graph.GetExecutableNodes()
	assert.Equal(t, []string{"A"}, executable)
}

func TestDependencyGraph_StatusTracking(t *testing.T) {
	graph := NewDependencyGraph()
	
	graph.AddNode("A", "tool_a", map[string]any{})
	
	// Initially pending
	node := graph.GetNode("A")
	assert.Equal(t, StatusPending, node.Status)
	
	// Change status
	err := graph.MarkStatus("A", StatusExecuting)
	require.NoError(t, err)
	
	node = graph.GetNode("A")
	assert.Equal(t, StatusExecuting, node.Status)
	
	// Change to completed
	err = graph.MarkStatus("A", StatusCompleted)
	require.NoError(t, err)
	
	node = graph.GetNode("A")
	assert.Equal(t, StatusCompleted, node.Status)
}

func TestDependencyGraph_Stats(t *testing.T) {
	graph := NewDependencyGraph()
	
	graph.AddNode("A", "tool_a", map[string]any{})
	graph.AddNode("B", "tool_b", map[string]any{})
	graph.AddNode("C", "tool_c", map[string]any{})
	
	graph.AddDependency("A", "B")
	graph.AddDependency("A", "C")
	
	graph.MarkStatus("A", StatusExecuting)
	graph.MarkStatus("B", StatusCompleted)
	// C remains pending
	
	stats := graph.GetStats()
	
	assert.Equal(t, 3, stats.TotalNodes)
	assert.Equal(t, 2, stats.MaxDependencies) // A has 2 dependencies
	assert.Equal(t, 1, stats.MaxDependents)   // B and C each have 1 dependent
	
	// Check status counts
	assert.Equal(t, 1, stats.StatusCounts[StatusPending])   // C
	assert.Equal(t, 1, stats.StatusCounts[StatusExecuting]) // A
	assert.Equal(t, 1, stats.StatusCounts[StatusCompleted]) // B
}

func TestDependencyGraph_Validation(t *testing.T) {
	graph := NewDependencyGraph()
	
	graph.AddNode("A", "tool_a", map[string]any{})
	graph.AddNode("B", "tool_b", map[string]any{})
	graph.AddDependency("A", "B")
	
	// Should validate successfully
	err := graph.Validate()
	assert.NoError(t, err)
	
	// Manually corrupt the graph to test validation
	graph.nodes["A"].Dependencies = append(graph.nodes["A"].Dependencies, "NonExistent")
	
	err = graph.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-existent dependency")
}

func TestDependencyGraph_Clone(t *testing.T) {
	graph := NewDependencyGraph()
	
	graph.AddNode("A", "tool_a", map[string]any{"param": "value"})
	graph.AddNode("B", "tool_b", map[string]any{})
	graph.AddDependency("A", "B")
	graph.MarkStatus("A", StatusExecuting)
	
	// Clone the graph
	clone := graph.Clone()
	
	// Verify clone is independent
	assert.NotSame(t, graph, clone)
	
	// Verify clone has same structure
	cloneNodeA := clone.GetNode("A")
	require.NotNil(t, cloneNodeA)
	assert.Equal(t, "A", cloneNodeA.ID)
	assert.Equal(t, "tool_a", cloneNodeA.ToolName)
	assert.Equal(t, StatusExecuting, cloneNodeA.Status)
	assert.Contains(t, cloneNodeA.Dependencies, "B")
	
	// Verify modifications to original don't affect clone
	graph.MarkStatus("A", StatusCompleted)
	
	originalNode := graph.GetNode("A")
	cloneNode := clone.GetNode("A")
	
	assert.Equal(t, StatusCompleted, originalNode.Status)
	assert.Equal(t, StatusExecuting, cloneNode.Status) // Should be unchanged
}

func TestDependencyGraph_ErrorCases(t *testing.T) {
	graph := NewDependencyGraph()
	
	// Test adding duplicate node
	graph.AddNode("A", "tool_a", map[string]any{})
	err := graph.AddNode("A", "tool_a2", map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	
	// Test adding dependency with non-existent nodes
	err = graph.AddDependency("NonExistent", "A")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
	
	err = graph.AddDependency("A", "NonExistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
	
	// Test marking status of non-existent node
	err = graph.MarkStatus("NonExistent", StatusCompleted)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestDependencyGraph_GetNodes(t *testing.T) {
	graph := NewDependencyGraph()
	
	graph.AddNode("A", "tool_a", map[string]any{"param": "value"})
	graph.AddNode("B", "tool_b", map[string]any{})
	
	nodes := graph.GetNodes()
	
	// Should return a copy
	assert.Len(t, nodes, 2)
	assert.Contains(t, nodes, "A")
	assert.Contains(t, nodes, "B")
	
	// Verify it's a copy by modifying the returned map
	delete(nodes, "A")
	
	// Original should still have both nodes
	originalNodes := graph.GetNodes()
	assert.Len(t, originalNodes, 2)
}

func TestDependencyGraph_String(t *testing.T) {
	graph := NewDependencyGraph()
	
	graph.AddNode("A", "tool_a", map[string]any{})
	graph.AddNode("B", "tool_b", map[string]any{})
	graph.AddDependency("A", "B")
	
	str := graph.String()
	
	assert.Contains(t, str, "DependencyGraph with 2 nodes")
	assert.Contains(t, str, "A [pending]")
	assert.Contains(t, str, "B [pending]")
}