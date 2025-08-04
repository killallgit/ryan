package tools

import (
	"fmt"
	"sync"
)

// DependencyGraph represents a directed acyclic graph of tool dependencies
type DependencyGraph struct {
	nodes map[string]*DependencyNode
	edges map[string][]string // node_id -> [dependent_node_ids]
	mu    sync.RWMutex
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	ID           string                 `json:"id"`
	ToolName     string                 `json:"tool_name"`
	Parameters   map[string]any         `json:"parameters"`
	Dependencies []string               `json:"dependencies"` // IDs of nodes this depends on
	Dependents   []string               `json:"dependents"`   // IDs of nodes that depend on this
	Status       DependencyNodeStatus   `json:"status"`
}

// DependencyNodeStatus represents the execution status of a node
type DependencyNodeStatus string

const (
	StatusPending   DependencyNodeStatus = "pending"
	StatusReady     DependencyNodeStatus = "ready"
	StatusExecuting DependencyNodeStatus = "executing" 
	StatusCompleted DependencyNodeStatus = "completed"
	StatusFailed    DependencyNodeStatus = "failed"
)

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*DependencyNode),
		edges: make(map[string][]string),
	}
}

// AddNode adds a node to the dependency graph
func (dg *DependencyGraph) AddNode(id, toolName string, parameters map[string]any) error {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	
	if _, exists := dg.nodes[id]; exists {
		return fmt.Errorf("node with ID %s already exists", id)
	}
	
	dg.nodes[id] = &DependencyNode{
		ID:           id,
		ToolName:     toolName,
		Parameters:   parameters,
		Dependencies: make([]string, 0),
		Dependents:   make([]string, 0),
		Status:       StatusPending,
	}
	
	dg.edges[id] = make([]string, 0)
	
	return nil
}

// AddDependency adds a dependency relationship (from depends on to)
func (dg *DependencyGraph) AddDependency(from, to string) error {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	
	// Validate nodes exist
	fromNode, fromExists := dg.nodes[from]
	toNode, toExists := dg.nodes[to]
	
	if !fromExists {
		return fmt.Errorf("from node %s does not exist", from)
	}
	if !toExists {
		return fmt.Errorf("to node %s does not exist", to) 
	}
	
	// Check for cycles - if there's already a path from 'to' to 'from', 
	// then adding 'from -> to' would create a cycle
	if dg.hasPath(to, from) {
		return fmt.Errorf("adding dependency %s -> %s would create a cycle", from, to)
	}
	
	// Add dependency
	fromNode.Dependencies = append(fromNode.Dependencies, to)
	toNode.Dependents = append(toNode.Dependents, from)
	
	// Update edges (to -> from means from depends on to)
	dg.edges[to] = append(dg.edges[to], from)
	
	return nil
}

// GetNode returns a node by ID
func (dg *DependencyGraph) GetNode(id string) *DependencyNode {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	return dg.nodes[id]
}

// GetNodes returns all nodes
func (dg *DependencyGraph) GetNodes() map[string]*DependencyNode {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	// Return a copy to prevent external modification
	nodes := make(map[string]*DependencyNode, len(dg.nodes))
	for id, node := range dg.nodes {
		// Create a copy of the node
		nodeCopy := *node
		nodeCopy.Dependencies = make([]string, len(node.Dependencies))
		copy(nodeCopy.Dependencies, node.Dependencies)
		nodeCopy.Dependents = make([]string, len(node.Dependents))
		copy(nodeCopy.Dependents, node.Dependents)
		
		nodes[id] = &nodeCopy
	}
	
	return nodes
}

// TopologicalSort returns nodes in dependency order (dependencies first)
func (dg *DependencyGraph) TopologicalSort() ([]string, error) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	// Kahn's algorithm for topological sorting
	
	// Count incoming edges for each node
	inDegree := make(map[string]int)
	for id, node := range dg.nodes {
		inDegree[id] = len(node.Dependencies)
	}
	
	// Queue of nodes with no incoming edges
	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	
	// Result list
	result := make([]string, 0, len(dg.nodes))
	
	// Process queue
	for len(queue) > 0 {
		// Remove node from queue
		current := queue[0]
		queue = queue[1:]
		
		// Add to result
		result = append(result, current)
		
		// Process all dependents
		for _, dependent := range dg.nodes[current].Dependents {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}
	
	// Check for cycles
	if len(result) != len(dg.nodes) {
		return nil, fmt.Errorf("dependency graph contains cycles")
	}
	
	return result, nil
}

// GetExecutableNodes returns nodes that are ready to execute (all dependencies completed)
func (dg *DependencyGraph) GetExecutableNodes() []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	executable := make([]string, 0)
	
	for id, node := range dg.nodes {
		if node.Status == StatusPending {
			// Check if all dependencies are completed
			allCompleted := true
			for _, depID := range node.Dependencies {
				depNode := dg.nodes[depID]
				if depNode.Status != StatusCompleted {
					allCompleted = false
					break
				}
			}
			
			if allCompleted {
				executable = append(executable, id)
			}
		}
	}
	
	return executable
}

// MarkStatus marks a node with the given status
func (dg *DependencyGraph) MarkStatus(id string, status DependencyNodeStatus) error {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	
	node, exists := dg.nodes[id]
	if !exists {
		return fmt.Errorf("node %s does not exist", id)
	}
	
	node.Status = status
	return nil
}

// GetStats returns statistics about the dependency graph
func (dg *DependencyGraph) GetStats() DependencyGraphStats {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	stats := DependencyGraphStats{
		TotalNodes: len(dg.nodes),
		StatusCounts: make(map[DependencyNodeStatus]int),
	}
	
	var maxDeps, maxDependents int
	var totalDeps, totalDependents int
	
	for _, node := range dg.nodes {
		stats.StatusCounts[node.Status]++
		
		depCount := len(node.Dependencies)
		dependentCount := len(node.Dependents)
		
		totalDeps += depCount
		totalDependents += dependentCount
		
		if depCount > maxDeps {
			maxDeps = depCount
		}
		if dependentCount > maxDependents {
			maxDependents = dependentCount
		}
	}
	
	stats.MaxDependencies = maxDeps
	stats.MaxDependents = maxDependents
	stats.AvgDependencies = float64(totalDeps) / float64(len(dg.nodes))
	stats.AvgDependents = float64(totalDependents) / float64(len(dg.nodes))
	
	return stats
}

// DependencyGraphStats contains statistics about the dependency graph
type DependencyGraphStats struct {
	TotalNodes        int                              `json:"total_nodes"`
	MaxDependencies   int                              `json:"max_dependencies"`
	MaxDependents     int                              `json:"max_dependents"`
	AvgDependencies   float64                          `json:"avg_dependencies"`
	AvgDependents     float64                          `json:"avg_dependents"`
	StatusCounts      map[DependencyNodeStatus]int     `json:"status_counts"`
}

// hasPath checks if there's a path from 'from' to 'to' using DFS
func (dg *DependencyGraph) hasPath(from, to string) bool {
	if from == to {
		return true
	}
	
	visited := make(map[string]bool)
	return dg.dfsPath(from, to, visited)
}

// dfsPath performs depth-first search to find a path
func (dg *DependencyGraph) dfsPath(current, target string, visited map[string]bool) bool {
	if current == target {
		return true
	}
	
	if visited[current] {
		return false
	}
	
	// Check if current node exists
	node, exists := dg.nodes[current]
	if !exists {
		return false
	}
	
	visited[current] = true
	
	// Check all dependencies (nodes that current depends on)
	for _, dependency := range node.Dependencies {
		if dg.dfsPath(dependency, target, visited) {
			return true
		}
	}
	
	return false
}

// Validate performs comprehensive validation of the dependency graph
func (dg *DependencyGraph) Validate() error {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	// First validate node consistency (do this before topological sort)
	for id, node := range dg.nodes {
		// Check that all dependencies exist
		for _, depID := range node.Dependencies {
			if _, exists := dg.nodes[depID]; !exists {
				return fmt.Errorf("node %s has non-existent dependency %s", id, depID)
			}
		}
		
		// Check that all dependents exist
		for _, depID := range node.Dependents {
			if _, exists := dg.nodes[depID]; !exists {
				return fmt.Errorf("node %s has non-existent dependent %s", id, depID)
			}
		}
		
		// Check bidirectional consistency
		for _, depID := range node.Dependencies {
			depNode := dg.nodes[depID]
			found := false
			for _, dependentID := range depNode.Dependents {
				if dependentID == id {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("dependency relationship %s -> %s is not bidirectional", id, depID)
			}
		}
	}
	
	// Check for cycles using topological sort (after node validation)
	_, err := dg.TopologicalSort()
	if err != nil {
		return fmt.Errorf("graph validation failed: %w", err)
	}
	
	return nil
}

// Clone creates a deep copy of the dependency graph
func (dg *DependencyGraph) Clone() *DependencyGraph {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	clone := NewDependencyGraph()
	
	// Copy nodes
	for id, node := range dg.nodes {
		// Deep copy parameters
		params := make(map[string]any)
		for k, v := range node.Parameters {
			params[k] = v
		}
		
		err := clone.AddNode(id, node.ToolName, params)
		if err != nil {
			// This shouldn't happen in a clone operation
			continue
		}
		
		// Copy status
		clone.nodes[id].Status = node.Status
	}
	
	// Copy dependencies
	for id, node := range dg.nodes {
		for _, depID := range node.Dependencies {
			clone.AddDependency(id, depID)
		}
	}
	
	return clone
}

// String returns a string representation of the dependency graph
func (dg *DependencyGraph) String() string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	result := fmt.Sprintf("DependencyGraph with %d nodes:\n", len(dg.nodes))
	
	for id, node := range dg.nodes {
		result += fmt.Sprintf("  %s [%s] -> deps: %v, dependents: %v\n",
			id, node.Status, node.Dependencies, node.Dependents)
	}
	
	return result
}