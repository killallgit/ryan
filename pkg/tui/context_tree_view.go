package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
)

// ContextTreeView provides a visual representation of the conversation context tree
type ContextTreeView struct {
	tree         *chat.ContextTree
	width        int
	height       int
	scrollX      int
	scrollY      int
	selectedNode string // ID of selected context node
	expanded     map[string]bool
	visible      bool
	position     ContextTreePosition
}

// ContextTreePosition defines where the tree is displayed
type ContextTreePosition int

const (
	TreePositionRight ContextTreePosition = iota
	TreePositionLeft
	TreePositionBottom
	TreePositionFloat // Floating overlay
)

// TreeNode represents a node in the visual tree
type TreeNode struct {
	ContextID    string
	ParentID     string
	Label        string
	MessageCount int
	IsActive     bool
	IsExpanded   bool
	Children     []*TreeNode
	Depth        int
}

// NewContextTreeView creates a new context tree visualization
func NewContextTreeView(tree *chat.ContextTree, width, height int) *ContextTreeView {
	return &ContextTreeView{
		tree:         tree,
		width:        width,
		height:       height,
		scrollX:      0,
		scrollY:      0,
		selectedNode: tree.ActiveContext,
		expanded:     make(map[string]bool),
		visible:      false,
		position:     TreePositionRight,
	}
}

// WithTree updates the context tree
func (v *ContextTreeView) WithTree(tree *chat.ContextTree) *ContextTreeView {
	v.tree = tree
	return v
}

// WithSize updates the dimensions
func (v *ContextTreeView) WithSize(width, height int) *ContextTreeView {
	v.width = width
	v.height = height
	return v
}

// WithVisibility sets visibility
func (v *ContextTreeView) WithVisibility(visible bool) *ContextTreeView {
	v.visible = visible
	return v
}

// WithPosition sets the display position
func (v *ContextTreeView) WithPosition(pos ContextTreePosition) *ContextTreeView {
	v.position = pos
	return v
}

// Toggle visibility
func (v *ContextTreeView) Toggle() *ContextTreeView {
	v.visible = !v.visible
	return v
}

// SelectNode selects a context node
func (v *ContextTreeView) SelectNode(contextID string) *ContextTreeView {
	if _, exists := v.tree.Contexts[contextID]; exists {
		v.selectedNode = contextID
	}
	return v
}

// ToggleExpanded toggles expansion state of a node
func (v *ContextTreeView) ToggleExpanded(contextID string) *ContextTreeView {
	v.expanded[contextID] = !v.expanded[contextID]
	return v
}

// NavigateUp moves selection up in the tree
func (v *ContextTreeView) NavigateUp() *ContextTreeView {
	nodes := v.buildTreeNodes()
	flatNodes := v.flattenNodes(nodes)

	for i, node := range flatNodes {
		if node.ContextID == v.selectedNode && i > 0 {
			v.selectedNode = flatNodes[i-1].ContextID
			v.ensureNodeVisible(i - 1)
			break
		}
	}
	return v
}

// NavigateDown moves selection down in the tree
func (v *ContextTreeView) NavigateDown() *ContextTreeView {
	nodes := v.buildTreeNodes()
	flatNodes := v.flattenNodes(nodes)

	for i, node := range flatNodes {
		if node.ContextID == v.selectedNode && i < len(flatNodes)-1 {
			v.selectedNode = flatNodes[i+1].ContextID
			v.ensureNodeVisible(i + 1)
			break
		}
	}
	return v
}

// NavigateToParent moves to parent context
func (v *ContextTreeView) NavigateToParent() *ContextTreeView {
	if ctx, exists := v.tree.Contexts[v.selectedNode]; exists && ctx.ParentID != nil {
		v.selectedNode = *ctx.ParentID
	}
	return v
}

// NavigateToChild moves to first child
func (v *ContextTreeView) NavigateToChild() *ContextTreeView {
	if children, exists := v.tree.ParentIndex[v.selectedNode]; exists && len(children) > 0 {
		v.selectedNode = children[0]
	}
	return v
}

// buildTreeNodes constructs the visual tree structure
func (v *ContextTreeView) buildTreeNodes() []*TreeNode {
	rootNodes := []*TreeNode{}

	// Find root contexts (no parent)
	for id, ctx := range v.tree.Contexts {
		if ctx.ParentID == nil {
			node := v.buildTreeNode(id, 0)
			rootNodes = append(rootNodes, node)
		}
	}

	return rootNodes
}

// buildTreeNode recursively builds a tree node
func (v *ContextTreeView) buildTreeNode(contextID string, depth int) *TreeNode {
	ctx := v.tree.Contexts[contextID]

	// Count messages in this context
	messageCount := 0
	for _, msg := range v.tree.Messages {
		if msg.ContextID == contextID {
			messageCount++
		}
	}

	// Create node
	node := &TreeNode{
		ContextID:    contextID,
		Label:        v.getContextLabel(ctx),
		MessageCount: messageCount,
		IsActive:     contextID == v.tree.ActiveContext,
		IsExpanded:   v.expanded[contextID],
		Children:     []*TreeNode{},
		Depth:        depth,
	}

	if ctx.ParentID != nil {
		node.ParentID = *ctx.ParentID
	}

	// Add children if expanded
	if v.expanded[contextID] || depth == 0 {
		if children, exists := v.tree.ParentIndex[contextID]; exists {
			for _, childID := range children {
				childNode := v.buildTreeNode(childID, depth+1)
				node.Children = append(node.Children, childNode)
			}
		}
	}

	return node
}

// getContextLabel generates a label for a context
func (v *ContextTreeView) getContextLabel(ctx *chat.Context) string {
	// Get branch message if this is a branch point
	if ctx.BranchPoint != nil {
		if msg, exists := v.tree.Messages[*ctx.BranchPoint]; exists {
			// Truncate message content for display
			content := msg.Content
			if len(content) > 30 {
				content = content[:27] + "..."
			}
			return fmt.Sprintf("Branch: %s", content)
		}
	}

	// Default label with timestamp
	return fmt.Sprintf("Context %s", ctx.Created.Format("15:04:05"))
}

// flattenNodes flattens the tree for navigation
func (v *ContextTreeView) flattenNodes(nodes []*TreeNode) []*TreeNode {
	flat := []*TreeNode{}
	for _, node := range nodes {
		flat = append(flat, node)
		if node.IsExpanded || node.Depth == 0 {
			flat = append(flat, v.flattenNodes(node.Children)...)
		}
	}
	return flat
}

// ensureNodeVisible adjusts scroll to make node visible
func (v *ContextTreeView) ensureNodeVisible(index int) {
	if index < v.scrollY {
		v.scrollY = index
	} else if index >= v.scrollY+v.height-2 {
		v.scrollY = index - v.height + 3
	}
}

// Render draws the context tree view
func (v *ContextTreeView) Render(screen tcell.Screen, area Rect) {
	if !v.visible || v.tree == nil {
		return
	}

	// Calculate actual render area based on position
	renderArea := v.calculateRenderArea(area)

	// Draw border
	v.drawBorder(screen, renderArea)

	// Draw title
	title := " Context Tree "
	titleX := renderArea.X + (renderArea.Width-len(title))/2
	titleStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	for i, ch := range title {
		screen.SetContent(titleX+i, renderArea.Y, ch, nil, titleStyle)
	}

	// Draw tree content
	nodes := v.buildTreeNodes()
	flatNodes := v.flattenNodes(nodes)

	// If no nodes, show empty message
	if len(flatNodes) == 0 {
		emptyMsg := "No contexts"
		msgStyle := tcell.StyleDefault.Foreground(tcell.ColorGray).Background(tcell.ColorBlack)
		msgX := renderArea.X + (renderArea.Width-len(emptyMsg))/2
		msgY := renderArea.Y + renderArea.Height/2
		for i, ch := range emptyMsg {
			screen.SetContent(msgX+i, msgY, ch, nil, msgStyle)
		}
		return
	}

	y := renderArea.Y + 1
	for i := v.scrollY; i < len(flatNodes) && y < renderArea.Y+renderArea.Height-1; i++ {
		node := flatNodes[i]
		v.renderNode(screen, node, renderArea.X+1, y, renderArea.Width-2)
		y++
	}

	// Draw scroll indicators
	scrollStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	if v.scrollY > 0 {
		screen.SetContent(renderArea.X+renderArea.Width-2, renderArea.Y+1, '▲', nil, scrollStyle)
	}
	if v.scrollY+v.height-2 < len(flatNodes) {
		screen.SetContent(renderArea.X+renderArea.Width-2, renderArea.Y+renderArea.Height-2, '▼', nil, scrollStyle)
	}
}

// calculateRenderArea determines where to render based on position
func (v *ContextTreeView) calculateRenderArea(area Rect) Rect {
	switch v.position {
	case TreePositionRight:
		width := v.width
		if width > area.Width/3 {
			width = area.Width / 3
		}
		return Rect{
			X:      area.X + area.Width - width,
			Y:      area.Y,
			Width:  width,
			Height: area.Height,
		}
	case TreePositionLeft:
		width := v.width
		if width > area.Width/3 {
			width = area.Width / 3
		}
		return Rect{
			X:      area.X,
			Y:      area.Y,
			Width:  width,
			Height: area.Height,
		}
	case TreePositionBottom:
		height := v.height
		if height > area.Height/3 {
			height = area.Height / 3
		}
		return Rect{
			X:      area.X,
			Y:      area.Y + area.Height - height,
			Width:  area.Width,
			Height: height,
		}
	case TreePositionFloat:
		// Center the floating window
		width := v.width
		height := v.height
		if width > area.Width*2/3 {
			width = area.Width * 2 / 3
		}
		if height > area.Height*2/3 {
			height = area.Height * 2 / 3
		}
		return Rect{
			X:      area.X + (area.Width-width)/2,
			Y:      area.Y + (area.Height-height)/2,
			Width:  width,
			Height: height,
		}
	}
	return area
}

// drawBorder draws the border around the tree view
func (v *ContextTreeView) drawBorder(screen tcell.Screen, area Rect) {
	// Use a background color to make it opaque
	bgStyle := tcell.StyleDefault.Background(tcell.ColorBlack)
	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)

	// First clear the entire area with background
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			screen.SetContent(x, y, ' ', nil, bgStyle)
		}
	}

	// Draw corners
	screen.SetContent(area.X, area.Y, '┌', nil, borderStyle)
	screen.SetContent(area.X+area.Width-1, area.Y, '┐', nil, borderStyle)
	screen.SetContent(area.X, area.Y+area.Height-1, '└', nil, borderStyle)
	screen.SetContent(area.X+area.Width-1, area.Y+area.Height-1, '┘', nil, borderStyle)

	// Draw horizontal lines
	for x := area.X + 1; x < area.X+area.Width-1; x++ {
		screen.SetContent(x, area.Y, '─', nil, borderStyle)
		screen.SetContent(x, area.Y+area.Height-1, '─', nil, borderStyle)
	}

	// Draw vertical lines
	for y := area.Y + 1; y < area.Y+area.Height-1; y++ {
		screen.SetContent(area.X, y, '│', nil, borderStyle)
		screen.SetContent(area.X+area.Width-1, y, '│', nil, borderStyle)
	}
}

// renderNode renders a single tree node
func (v *ContextTreeView) renderNode(screen tcell.Screen, node *TreeNode, x, y, maxWidth int) {
	// Prepare style with black background
	style := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	if node.ContextID == v.selectedNode {
		style = style.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite)
	}
	if node.IsActive {
		style = style.Bold(true).Foreground(tcell.ColorGreen).Background(tcell.ColorBlack)
	}

	// Build node text
	indent := strings.Repeat("  ", node.Depth)
	prefix := "├─"
	if len(v.tree.ParentIndex[node.ParentID]) > 0 && node.ContextID == v.tree.ParentIndex[node.ParentID][len(v.tree.ParentIndex[node.ParentID])-1] {
		prefix = "└─"
	}

	// Expansion indicator
	expansionIndicator := ""
	if children, hasChildren := v.tree.ParentIndex[node.ContextID]; hasChildren && len(children) > 0 {
		if node.IsExpanded {
			expansionIndicator = "▼ "
		} else {
			expansionIndicator = "▶ "
		}
	} else {
		expansionIndicator = "  "
	}

	// Format label with message count
	label := fmt.Sprintf("%s (%d msgs)", node.Label, node.MessageCount)
	if node.IsActive {
		label = "● " + label
	}

	// Combine all parts
	text := fmt.Sprintf("%s%s%s%s", indent, prefix, expansionIndicator, label)

	// Truncate if too long
	if len(text) > maxWidth {
		text = text[:maxWidth-3] + "..."
	}

	// Draw the text
	col := 0
	for _, ch := range text {
		if col < maxWidth {
			screen.SetContent(x+col, y, ch, nil, style)
			col++
		}
	}
	// Fill the rest of the line with background
	for col < maxWidth {
		screen.SetContent(x+col, y, ' ', nil, style)
		col++
	}
}

// HandleKeyEvent processes keyboard input for the tree view
func (v *ContextTreeView) HandleKeyEvent(ev *tcell.EventKey) bool {
	if !v.visible {
		return false
	}

	switch ev.Key() {
	case tcell.KeyUp:
		v.NavigateUp()
		return true
	case tcell.KeyDown:
		v.NavigateDown()
		return true
	case tcell.KeyLeft:
		if v.expanded[v.selectedNode] {
			v.ToggleExpanded(v.selectedNode)
		} else {
			v.NavigateToParent()
		}
		return true
	case tcell.KeyRight:
		if children, hasChildren := v.tree.ParentIndex[v.selectedNode]; hasChildren && len(children) > 0 {
			if !v.expanded[v.selectedNode] {
				v.ToggleExpanded(v.selectedNode)
			} else {
				v.NavigateToChild()
			}
		}
		return true
	case tcell.KeyEnter:
		// Switch to selected context
		if v.tree != nil && v.selectedNode != v.tree.ActiveContext {
			// This would trigger a context switch event
			return true
		}
		return false
	case tcell.KeyEscape:
		v.visible = false
		return true
	}

	switch ev.Rune() {
	case ' ':
		// Toggle expansion
		v.ToggleExpanded(v.selectedNode)
		return true
	case 't', 'T':
		// Toggle visibility
		v.Toggle()
		return true
	}

	return false
}

// GetSelectedContext returns the currently selected context ID
func (v *ContextTreeView) GetSelectedContext() string {
	return v.selectedNode
}

// IsVisible returns whether the tree view is visible
func (v *ContextTreeView) IsVisible() bool {
	return v.visible
}
