package agents

import (
	"strings"
	"sync"

	"github.com/killallgit/ryan/pkg/logger"
)

// ContextManager manages shared state and context between agents
type ContextManager struct {
	sharedMemory *SharedMemory
	contextTree  *ContextTree
	propagator   *ContextPropagator
	log          *logger.Logger
}

// NewContextManager creates a new context manager
func NewContextManager() *ContextManager {
	return &ContextManager{
		sharedMemory: NewSharedMemory(),
		contextTree:  NewContextTree(),
		propagator:   NewContextPropagator(),
		log:          logger.WithComponent("context_manager"),
	}
}

// CreateContext creates a new execution context
func (cm *ContextManager) CreateContext(sessionID, requestID, userPrompt string) *ExecutionContext {
	return &ExecutionContext{
		SessionID:   sessionID,
		RequestID:   requestID,
		UserPrompt:  userPrompt,
		SharedData:  make(map[string]interface{}),
		FileContext: []FileInfo{},
		Artifacts:   make(map[string]interface{}),
		Options:     make(map[string]interface{}),
	}
}

// PropagateContext propagates relevant context from one execution to another
func (cm *ContextManager) PropagateContext(from, to *ExecutionContext, targetAgent string) {
	cm.propagator.Propagate(from, to, targetAgent)
}

// SharedMemory manages shared data between agents
type SharedMemory struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewSharedMemory creates a new shared memory
func NewSharedMemory() *SharedMemory {
	return &SharedMemory{
		data: make(map[string]interface{}),
	}
}

// Set stores a value in shared memory
func (sm *SharedMemory) Set(key string, value interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.data[key] = value
}

// Get retrieves a value from shared memory
func (sm *SharedMemory) Get(key string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	value, exists := sm.data[key]
	return value, exists
}

// GetAll returns all data in shared memory
func (sm *SharedMemory) GetAll() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	copy := make(map[string]interface{})
	for k, v := range sm.data {
		copy[k] = v
	}
	return copy
}

// ContextTree manages hierarchical context relationships
type ContextTree struct {
	root  *ContextNode
	nodes map[string]*ContextNode
	mu    sync.RWMutex
}

// NewContextTree creates a new context tree
func NewContextTree() *ContextTree {
	root := &ContextNode{
		ID:       "root",
		Children: make([]*ContextNode, 0),
		Data:     make(map[string]interface{}),
	}

	return &ContextTree{
		root:  root,
		nodes: map[string]*ContextNode{"root": root},
	}
}

// AddNode adds a node to the context tree
func (ct *ContextTree) AddNode(id, parentID string, data map[string]interface{}) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	parent, exists := ct.nodes[parentID]
	if !exists {
		parent = ct.root
	}

	node := &ContextNode{
		ID:       id,
		Parent:   parent,
		Children: make([]*ContextNode, 0),
		Data:     data,
	}

	parent.Children = append(parent.Children, node)
	ct.nodes[id] = node

	return nil
}

// GetNode retrieves a node from the tree
func (ct *ContextTree) GetNode(id string) (*ContextNode, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	node, exists := ct.nodes[id]
	return node, exists
}

// GetPath returns the path from root to a node
func (ct *ContextTree) GetPath(nodeID string) []*ContextNode {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	node, exists := ct.nodes[nodeID]
	if !exists {
		return nil
	}

	path := make([]*ContextNode, 0)
	current := node
	for current != nil {
		path = append([]*ContextNode{current}, path...)
		current = current.Parent
	}

	return path
}

// ContextNode represents a node in the context tree
type ContextNode struct {
	ID       string
	Parent   *ContextNode
	Children []*ContextNode
	Data     map[string]interface{}
}

// ContextPropagator handles intelligent context propagation
type ContextPropagator struct {
	rules []PropagationRule
	log   *logger.Logger
}

// NewContextPropagator creates a new context propagator
func NewContextPropagator() *ContextPropagator {
	return &ContextPropagator{
		rules: defaultPropagationRules(),
		log:   logger.WithComponent("context_propagator"),
	}
}

// Propagate propagates context based on rules
func (cp *ContextPropagator) Propagate(from, to *ExecutionContext, targetAgent string) {
	cp.log.Debug("Propagating context", "target_agent", targetAgent)

	// Apply propagation rules
	for _, rule := range cp.rules {
		if rule.ShouldApply(targetAgent) {
			rule.Apply(from, to, targetAgent)
		}
	}
}

// PropagationRule defines how context should be propagated
type PropagationRule interface {
	ShouldApply(targetAgent string) bool
	Apply(from, to *ExecutionContext, targetAgent string)
}

// FileContextRule propagates file context
type FileContextRule struct{}

func (r *FileContextRule) ShouldApply(targetAgent string) bool {
	// File context is relevant for most agents
	return true
}

func (r *FileContextRule) Apply(from, to *ExecutionContext, targetAgent string) {
	from.mu.RLock()
	defer from.mu.RUnlock()
	to.mu.Lock()
	defer to.mu.Unlock()

	// Copy file context
	for _, file := range from.FileContext {
		// Check if file already exists
		exists := false
		for _, existing := range to.FileContext {
			if existing.Path == file.Path {
				exists = true
				break
			}
		}
		if !exists {
			to.FileContext = append(to.FileContext, file)
		}
	}
}

// SharedDataRule propagates shared data selectively
type SharedDataRule struct{}

func (r *SharedDataRule) ShouldApply(targetAgent string) bool {
	return true
}

func (r *SharedDataRule) Apply(from, to *ExecutionContext, targetAgent string) {
	from.mu.RLock()
	defer from.mu.RUnlock()
	to.mu.Lock()
	defer to.mu.Unlock()

	// Copy relevant shared data
	for key, value := range from.SharedData {
		// Filter based on key patterns
		if shouldPropagateKey(key, targetAgent) {
			to.SharedData[key] = value
		}
	}
}

// shouldPropagateKey determines if a key should be propagated to an agent
func shouldPropagateKey(key, targetAgent string) bool {
	// Agent-specific filtering logic
	switch targetAgent {
	case "code_review":
		// Code review needs analysis results
		return strings.Contains(key, "analysis") || strings.Contains(key, "ast")
	case "file_operations":
		// File operations needs file-related data
		return strings.Contains(key, "file") || strings.Contains(key, "path")
	default:
		// Default: propagate most data
		return !strings.Contains(key, "internal")
	}
}

// ArtifactsRule propagates artifacts
type ArtifactsRule struct{}

func (r *ArtifactsRule) ShouldApply(targetAgent string) bool {
	// Artifacts are useful for most agents
	return targetAgent != "dispatcher"
}

func (r *ArtifactsRule) Apply(from, to *ExecutionContext, targetAgent string) {
	from.mu.RLock()
	defer from.mu.RUnlock()
	to.mu.Lock()
	defer to.mu.Unlock()

	if from.Artifacts != nil && len(from.Artifacts) > 0 {
		if to.Artifacts == nil {
			to.Artifacts = make(map[string]interface{})
		}
		for k, v := range from.Artifacts {
			to.Artifacts[k] = v
		}
	}
}

// defaultPropagationRules returns the default set of propagation rules
func defaultPropagationRules() []PropagationRule {
	return []PropagationRule{
		&FileContextRule{},
		&SharedDataRule{},
		&ArtifactsRule{},
	}
}
