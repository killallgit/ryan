package agents

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// ActivityStatus represents the status of an agent activity
type ActivityStatus string

const (
	ActivityStatusActive   ActivityStatus = "active"
	ActivityStatusPending  ActivityStatus = "pending"
	ActivityStatusComplete ActivityStatus = "complete"
	ActivityStatusError    ActivityStatus = "error"
	ActivityStatusIdle     ActivityStatus = "idle"
)

// OperationType represents the type of operation being performed
type OperationType string

const (
	OperationTypeTool      OperationType = "tool"
	OperationTypeAgent     OperationType = "agent_spawn"
	OperationTypeAnalysis  OperationType = "analysis"
	OperationTypePlanning  OperationType = "planning"
	OperationTypeExecution OperationType = "execution"
)

// ActivityNode represents a node in the activity tree
type ActivityNode struct {
	ID            string
	AgentName     string
	Operation     string
	OperationType OperationType
	Status        ActivityStatus
	Progress      float64
	Children      []*ActivityNode
	Parent        *ActivityNode
	StartTime     time.Time
	EndTime       *time.Time
	Error         error
	Mu            sync.RWMutex // Exported for external packages
}

// NewActivityNode creates a new activity node
func NewActivityNode(id, agentName, operation string, opType OperationType) *ActivityNode {
	return &ActivityNode{
		ID:            id,
		AgentName:     agentName,
		Operation:     operation,
		OperationType: opType,
		Status:        ActivityStatusPending,
		StartTime:     time.Now(),
		Children:      make([]*ActivityNode, 0),
	}
}

// AddChild adds a child node to this activity
func (n *ActivityNode) AddChild(child *ActivityNode) {
	n.Mu.Lock()
	defer n.Mu.Unlock()

	child.Parent = n
	n.Children = append(n.Children, child)
}

// RemoveChild removes a child node
func (n *ActivityNode) RemoveChild(childID string) {
	n.Mu.Lock()
	defer n.Mu.Unlock()

	for i, child := range n.Children {
		if child.ID == childID {
			n.Children = append(n.Children[:i], n.Children[i+1:]...)
			break
		}
	}
}

// UpdateStatus updates the status of the node
func (n *ActivityNode) UpdateStatus(status ActivityStatus) {
	n.Mu.Lock()
	defer n.Mu.Unlock()

	n.Status = status
	if status == ActivityStatusComplete || status == ActivityStatusError {
		now := time.Now()
		n.EndTime = &now
	}
}

// UpdateProgress updates the progress of the node
func (n *ActivityNode) UpdateProgress(progress float64) {
	n.Mu.Lock()
	defer n.Mu.Unlock()

	n.Progress = progress
}

// SetError sets an error on the node
func (n *ActivityNode) SetError(err error) {
	n.Mu.Lock()
	defer n.Mu.Unlock()

	n.Error = err
	n.Status = ActivityStatusError
}

// IsActive returns true if the node or any of its children are active
func (n *ActivityNode) IsActive() bool {
	n.Mu.RLock()
	defer n.Mu.RUnlock()

	if n.Status == ActivityStatusActive || n.Status == ActivityStatusPending {
		return true
	}

	for _, child := range n.Children {
		if child.IsActive() {
			return true
		}
	}

	return false
}

// GetDuration returns the duration of the activity
func (n *ActivityNode) GetDuration() time.Duration {
	n.Mu.RLock()
	defer n.Mu.RUnlock()

	if n.EndTime != nil {
		return n.EndTime.Sub(n.StartTime)
	}
	return time.Since(n.StartTime)
}

// ActivityTree manages the tree of agent activities
type ActivityTree struct {
	root     *ActivityNode
	nodes    map[string]*ActivityNode
	mu       sync.RWMutex
	log      *logger.Logger
	maxDepth int
	maxNodes int
}

// NewActivityTree creates a new activity tree
func NewActivityTree() *ActivityTree {
	root := &ActivityNode{
		ID:        "root",
		AgentName: "System",
		Operation: "idle",
		Status:    ActivityStatusIdle,
		StartTime: time.Now(),
		Children:  make([]*ActivityNode, 0),
	}

	return &ActivityTree{
		root:     root,
		nodes:    map[string]*ActivityNode{"root": root},
		log:      logger.WithComponent("activity_tree"),
		maxDepth: 10,
		maxNodes: 100,
	}
}

// AddActivity adds a new activity to the tree
func (at *ActivityTree) AddActivity(id, parentID, agentName, operation string, opType OperationType) (*ActivityNode, error) {
	at.mu.Lock()
	defer at.mu.Unlock()

	// Check limits
	if len(at.nodes) >= at.maxNodes {
		return nil, fmt.Errorf("maximum number of nodes (%d) reached", at.maxNodes)
	}

	// Check if node already exists
	if _, exists := at.nodes[id]; exists {
		return nil, fmt.Errorf("node with ID %s already exists", id)
	}

	// Create new node
	node := NewActivityNode(id, agentName, operation, opType)

	// Find parent
	parent := at.root
	if parentID != "" {
		if p, exists := at.nodes[parentID]; exists {
			parent = p
		} else {
			at.log.Warn("Parent node not found, using root", "parent_id", parentID)
		}
	}

	// Check depth
	if at.getDepth(parent) >= at.maxDepth {
		return nil, fmt.Errorf("maximum depth (%d) reached", at.maxDepth)
	}

	// Add to tree
	parent.AddChild(node)
	at.nodes[id] = node

	at.log.Debug("Added activity",
		"id", id,
		"agent", agentName,
		"operation", operation,
		"parent", parentID)

	return node, nil
}

// UpdateActivity updates an existing activity
func (at *ActivityTree) UpdateActivity(id string, status ActivityStatus, progress float64) error {
	at.mu.RLock()
	defer at.mu.RUnlock()

	node, exists := at.nodes[id]
	if !exists {
		return fmt.Errorf("node with ID %s not found", id)
	}

	node.UpdateStatus(status)
	if progress >= 0 {
		node.UpdateProgress(progress)
	}

	return nil
}

// CompleteActivity marks an activity as complete
func (at *ActivityTree) CompleteActivity(id string) error {
	return at.UpdateActivity(id, ActivityStatusComplete, 100)
}

// ErrorActivity marks an activity as errored
func (at *ActivityTree) ErrorActivity(id string, err error) error {
	at.mu.RLock()
	defer at.mu.RUnlock()

	node, exists := at.nodes[id]
	if !exists {
		return fmt.Errorf("node with ID %s not found", id)
	}

	node.SetError(err)
	return nil
}

// RemoveActivity removes an activity and its children from the tree
func (at *ActivityTree) RemoveActivity(id string) error {
	at.mu.Lock()
	defer at.mu.Unlock()

	node, exists := at.nodes[id]
	if !exists {
		return fmt.Errorf("node with ID %s not found", id)
	}

	// Remove from parent
	if node.Parent != nil {
		node.Parent.RemoveChild(id)
	}

	// Remove node and all children from map
	at.removeNodeRecursive(node)

	return nil
}

// removeNodeRecursive removes a node and all its children from the nodes map
func (at *ActivityTree) removeNodeRecursive(node *ActivityNode) {
	// Remove children first
	for _, child := range node.Children {
		at.removeNodeRecursive(child)
	}

	// Remove this node
	delete(at.nodes, node.ID)
}

// GetActiveActivities returns all currently active activities
func (at *ActivityTree) GetActiveActivities() []*ActivityNode {
	at.mu.RLock()
	defer at.mu.RUnlock()

	active := make([]*ActivityNode, 0)
	for _, node := range at.nodes {
		if node.Status == ActivityStatusActive || node.Status == ActivityStatusPending {
			active = append(active, node)
		}
	}

	return active
}

// GetRootChildren returns the top-level activities
func (at *ActivityTree) GetRootChildren() []*ActivityNode {
	at.mu.RLock()
	defer at.mu.RUnlock()

	return at.root.Children
}

// GetNode returns a specific node by ID
func (at *ActivityTree) GetNode(id string) (*ActivityNode, bool) {
	at.mu.RLock()
	defer at.mu.RUnlock()

	node, exists := at.nodes[id]
	return node, exists
}

// IsEmpty returns true if there are no active activities
func (at *ActivityTree) IsEmpty() bool {
	at.mu.RLock()
	defer at.mu.RUnlock()

	return len(at.root.Children) == 0
}

// Clear removes all activities except the root
func (at *ActivityTree) Clear() {
	at.mu.Lock()
	defer at.mu.Unlock()

	at.root.Children = make([]*ActivityNode, 0)
	at.nodes = map[string]*ActivityNode{"root": at.root}
}

// getDepth calculates the depth of a node in the tree
func (at *ActivityTree) getDepth(node *ActivityNode) int {
	depth := 0
	current := node
	for current.Parent != nil {
		depth++
		current = current.Parent
	}
	return depth
}

// FormatTree returns a formatted string representation of the tree
func (at *ActivityTree) FormatTree() string {
	at.mu.RLock()
	defer at.mu.RUnlock()

	if len(at.root.Children) == 0 {
		return ""
	}

	var output strings.Builder
	for i, child := range at.root.Children {
		isLast := i == len(at.root.Children)-1
		at.formatNode(&output, child, "", isLast)
	}

	return output.String()
}

// formatNode recursively formats a node and its children
func (at *ActivityTree) formatNode(output *strings.Builder, node *ActivityNode, prefix string, isLast bool) {
	node.Mu.RLock()
	defer node.Mu.RUnlock()

	// Choose connector
	connector := "├── "
	childPrefix := prefix + "│   "
	if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	}

	// Format node line
	line := fmt.Sprintf("%s%s", connector, node.AgentName)

	// Add operation if present
	if node.Operation != "" && node.Operation != "idle" {
		line += fmt.Sprintf(" › %s", node.Operation)
	}

	// Add status indicator
	switch node.Status {
	case ActivityStatusActive:
		line += " ●"
	case ActivityStatusPending:
		line += " ○"
	case ActivityStatusError:
		line += " ✗"
	case ActivityStatusComplete:
		line += " ✓"
	}

	// Add progress if applicable
	if node.Progress > 0 && node.Progress < 100 {
		line += fmt.Sprintf(" [%.0f%%]", node.Progress)
	}

	output.WriteString(prefix + line + "\n")

	// Format children
	for i, child := range node.Children {
		childIsLast := i == len(node.Children)-1
		at.formatNode(output, child, childPrefix, childIsLast)
	}
}

// PruneCompleted removes completed activities older than the specified duration
func (at *ActivityTree) PruneCompleted(olderThan time.Duration) {
	at.mu.Lock()
	defer at.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)

	// Collect nodes to remove
	toRemove := make([]string, 0)
	for id, node := range at.nodes {
		if id == "root" {
			continue
		}

		if node.Status == ActivityStatusComplete && node.EndTime != nil {
			if node.EndTime.Before(cutoff) {
				toRemove = append(toRemove, id)
			}
		}
	}

	// Remove collected nodes
	for _, id := range toRemove {
		if node, exists := at.nodes[id]; exists {
			if node.Parent != nil {
				node.Parent.RemoveChild(id)
			}
			at.removeNodeRecursive(node)
		}
	}
}

// GetStatistics returns statistics about the activity tree
func (at *ActivityTree) GetStatistics() map[string]interface{} {
	at.mu.RLock()
	defer at.mu.RUnlock()

	stats := map[string]interface{}{
		"total_nodes":     len(at.nodes) - 1, // Exclude root
		"active_nodes":    0,
		"pending_nodes":   0,
		"completed_nodes": 0,
		"error_nodes":     0,
		"max_depth":       0,
	}

	for id, node := range at.nodes {
		if id == "root" {
			continue
		}

		switch node.Status {
		case ActivityStatusActive:
			stats["active_nodes"] = stats["active_nodes"].(int) + 1
		case ActivityStatusPending:
			stats["pending_nodes"] = stats["pending_nodes"].(int) + 1
		case ActivityStatusComplete:
			stats["completed_nodes"] = stats["completed_nodes"].(int) + 1
		case ActivityStatusError:
			stats["error_nodes"] = stats["error_nodes"].(int) + 1
		}

		depth := at.getDepth(node)
		if depth > stats["max_depth"].(int) {
			stats["max_depth"] = depth
		}
	}

	return stats
}

// AddNode adds a node directly to the tree (simplified API)
func (at *ActivityTree) AddNode(node *ActivityNode) error {
	at.mu.Lock()
	defer at.mu.Unlock()

	// Check limits
	if len(at.nodes) >= at.maxNodes {
		return fmt.Errorf("maximum number of nodes (%d) reached", at.maxNodes)
	}

	// Check if node already exists
	if _, exists := at.nodes[node.ID]; exists {
		return fmt.Errorf("node with ID %s already exists", node.ID)
	}

	// Add to root if no parent specified
	if node.Parent == nil {
		at.root.AddChild(node)
	}

	// Add to nodes map
	at.nodes[node.ID] = node

	return nil
}

// String returns a string representation using FormatTree
func (at *ActivityTree) String() string {
	return at.FormatTree()
}
