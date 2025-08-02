package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
)

// NodeBounds represents the screen coordinates and dimensions of a rendered node
type NodeBounds struct {
	X      int // Left coordinate
	Y      int // Top coordinate  
	Width  int // Width in characters
	Height int // Height in lines
}

// NodeState represents the interactive state of a message node
type NodeState struct {
	Selected  bool // Whether this node is currently selected
	Expanded  bool // Whether this node is expanded (for collapsible content)
	Focused   bool // Whether this node has keyboard focus
	Hovered   bool // Whether mouse is hovering over this node
}

// NodeRenderCache stores pre-computed rendering data for performance
type NodeRenderCache struct {
	Lines      []string     // Pre-wrapped text lines
	Styles     []tcell.Style // Style for each line
	Valid      bool         // Whether cache is valid
	LastWidth  int         // Width used for last cache computation
}

// MessageNode represents a renderable, interactive message with its own state
type MessageNode interface {
	// Core properties
	ID() string                    // Unique identifier for this node
	Message() chat.Message         // The underlying chat message
	NodeType() MessageNodeType     // Type of node (text, thinking, tool, etc.)
	
	// State management
	State() NodeState              // Current interactive state
	WithState(state NodeState) MessageNode // Return new node with updated state
	
	// Rendering
	Render(area Rect, state NodeState) []RenderedLine // Render node content within area
	CalculateHeight(width int) int     // Calculate required height for given width
	Bounds() NodeBounds               // Current screen bounds (set by renderer)
	WithBounds(bounds NodeBounds) MessageNode // Update bounds after rendering
	
	// Event handling
	HandleClick(x, y int) (handled bool, newState NodeState) // Handle mouse click
	HandleKeyEvent(ev *tcell.EventKey) (handled bool, newState NodeState) // Handle keyboard
	
	// Content queries
	IsCollapsible() bool          // Whether this node can be collapsed
	HasDetailView() bool          // Whether this node has expandable details
	GetPreviewText() string       // Short preview text for collapsed state
}

// MessageNodeType represents different types of message nodes
type MessageNodeType int

const (
	NodeTypeText MessageNodeType = iota
	NodeTypeThinking
	NodeTypeToolCall
	NodeTypeToolResult
	NodeTypeSystem
	NodeTypeError
)

// RenderedLine represents a single line of rendered content with its style
type RenderedLine struct {
	Text   string      // The text content
	Style  tcell.Style // Styling for this line
	Indent int         // Indentation level
}

// NodeRegistry manages creation of different node types
type NodeRegistry struct {
	factories map[MessageNodeType]NodeFactory
}

// NodeFactory creates MessageNode instances for specific message types
type NodeFactory interface {
	CreateNode(msg chat.Message, id string) MessageNode
	CanHandle(msg chat.Message) bool
}

// NewNodeRegistry creates a new node registry with default factories
func NewNodeRegistry() *NodeRegistry {
	registry := &NodeRegistry{
		factories: make(map[MessageNodeType]NodeFactory),
	}
	
	// Register default node factories
	registry.RegisterFactory(NodeTypeText, &TextNodeFactory{})
	registry.RegisterFactory(NodeTypeThinking, &ThinkingNodeFactory{})
	registry.RegisterFactory(NodeTypeToolCall, &ToolCallNodeFactory{})
	registry.RegisterFactory(NodeTypeToolResult, &ToolResultNodeFactory{})
	registry.RegisterFactory(NodeTypeSystem, &SystemNodeFactory{})
	registry.RegisterFactory(NodeTypeError, &ErrorNodeFactory{})
	
	return registry
}

// RegisterFactory registers a factory for a specific node type
func (nr *NodeRegistry) RegisterFactory(nodeType MessageNodeType, factory NodeFactory) {
	nr.factories[nodeType] = factory
}

// CreateNode creates a MessageNode for the given message
func (nr *NodeRegistry) CreateNode(msg chat.Message, id string) MessageNode {
	// Find the appropriate factory based on message content and role
	for _, factory := range nr.factories {
		if factory.CanHandle(msg) {
			return factory.CreateNode(msg, id)
		}
	}
	
	// Fallback to text node
	return nr.factories[NodeTypeText].CreateNode(msg, id)
}

// Helper functions for node state management

// NewNodeState creates a new node state with default values
func NewNodeState() NodeState {
	return NodeState{
		Selected: false,
		Expanded: true, // Default to expanded
		Focused:  false,
		Hovered:  false,
	}
}

// ToggleSelected toggles the selected state
func (ns NodeState) ToggleSelected() NodeState {
	return NodeState{
		Selected: !ns.Selected,
		Expanded: ns.Expanded,
		Focused:  ns.Focused,
		Hovered:  ns.Hovered,
	}
}

// ToggleExpanded toggles the expanded state
func (ns NodeState) ToggleExpanded() NodeState {
	return NodeState{
		Selected: ns.Selected,
		Expanded: !ns.Expanded,
		Focused:  ns.Focused,
		Hovered:  ns.Hovered,
	}
}

// WithSelected sets the selected state
func (ns NodeState) WithSelected(selected bool) NodeState {
	return NodeState{
		Selected: selected,
		Expanded: ns.Expanded,
		Focused:  ns.Focused,
		Hovered:  ns.Hovered,
	}
}

// WithFocused sets the focused state
func (ns NodeState) WithFocused(focused bool) NodeState {
	return NodeState{
		Selected: ns.Selected,
		Expanded: ns.Expanded,
		Focused:  focused,
		Hovered:  ns.Hovered,
	}
}

// WithHovered sets the hovered state
func (ns NodeState) WithHovered(hovered bool) NodeState {
	return NodeState{
		Selected: ns.Selected,
		Expanded: ns.Expanded,
		Focused:  ns.Focused,
		Hovered:  hovered,
	}
}