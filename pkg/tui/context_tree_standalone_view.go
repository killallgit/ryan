package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
)

// ContextTreeStandaloneView is a full-screen view for the context tree
type ContextTreeStandaloneView struct {
	contextTreeView *ContextTreeView
	tree            *chat.ContextTree
	width           int
	height          int
}

// NewContextTreeStandaloneView creates a new standalone context tree view
func NewContextTreeStandaloneView() *ContextTreeStandaloneView {
	return &ContextTreeStandaloneView{
		contextTreeView: nil,
		tree:            nil,
		width:           80,
		height:          24,
	}
}

// Name returns the view name for registration
func (v *ContextTreeStandaloneView) Name() string {
	return "context-tree"
}

// Description returns a description for the command palette
func (v *ContextTreeStandaloneView) Description() string {
	return "Context Tree - View conversation branching and context hierarchy"
}

// UpdateTree updates the context tree data
func (v *ContextTreeStandaloneView) UpdateTree(tree *chat.ContextTree) {
	v.tree = tree
	if v.contextTreeView != nil {
		v.contextTreeView = v.contextTreeView.WithTree(tree)
	}
}

// Render renders the standalone context tree view
func (v *ContextTreeStandaloneView) Render(screen tcell.Screen, area Rect) {
	// Initialize context tree view if needed
	if v.contextTreeView == nil {
		if v.tree != nil {
			v.contextTreeView = NewContextTreeView(v.tree, area.Width, area.Height)
			v.contextTreeView = v.contextTreeView.WithVisibility(true).WithPosition(TreePositionFloat)
		} else {
			// Show message that no context tree is available
			v.renderNoContextMessage(screen, area)
			return
		}
	}

	// Update size if changed
	if v.width != area.Width || v.height != area.Height {
		v.width = area.Width
		v.height = area.Height
		v.contextTreeView = v.contextTreeView.WithSize(area.Width, area.Height)
	}

	// Render the context tree taking up the full area
	v.contextTreeView.Render(screen, area)
}

// renderNoContextMessage shows a message when no context tree is available
func (v *ContextTreeStandaloneView) renderNoContextMessage(screen tcell.Screen, area Rect) {
	// Clear the area with background
	bgStyle := tcell.StyleDefault.Background(tcell.ColorBlack)
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			screen.SetContent(x, y, ' ', nil, bgStyle)
		}
	}

	// Show centered message
	message := "No Context Tree Available"
	subMessage := "Start a conversation to see context branching"

	messageStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack).Bold(true)
	subMessageStyle := tcell.StyleDefault.Foreground(tcell.ColorGray).Background(tcell.ColorBlack)

	// Center the messages
	messageX := area.X + (area.Width-len(message))/2
	messageY := area.Y + area.Height/2 - 1
	subMessageX := area.X + (area.Width-len(subMessage))/2
	subMessageY := messageY + 2

	// Draw main message
	for i, ch := range message {
		screen.SetContent(messageX+i, messageY, ch, nil, messageStyle)
	}

	// Draw sub message
	for i, ch := range subMessage {
		screen.SetContent(subMessageX+i, subMessageY, ch, nil, subMessageStyle)
	}

	// Show instructions
	instructions := "Press Escape to return to chat"
	instructionsStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	instructionsX := area.X + (area.Width-len(instructions))/2
	instructionsY := subMessageY + 3

	for i, ch := range instructions {
		screen.SetContent(instructionsX+i, instructionsY, ch, nil, instructionsStyle)
	}
}

// HandleKeyEvent processes keyboard input
func (v *ContextTreeStandaloneView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool {
	// Handle escape to close view
	if ev.Key() == tcell.KeyEscape {
		return false // Let the view manager handle switching back
	}

	// If we have a context tree view, let it handle the event
	if v.contextTreeView != nil {
		return v.contextTreeView.HandleKeyEvent(ev)
	}

	return false
}

// HandleResize updates the view size
func (v *ContextTreeStandaloneView) HandleResize(width, height int) {
	v.width = width
	v.height = height
	if v.contextTreeView != nil {
		v.contextTreeView = v.contextTreeView.WithSize(width, height)
	}
}

// GetSelectedContext returns the currently selected context ID
func (v *ContextTreeStandaloneView) GetSelectedContext() string {
	if v.contextTreeView != nil {
		return v.contextTreeView.GetSelectedContext()
	}
	return ""
}

// SetSelectedContext sets the currently selected context
func (v *ContextTreeStandaloneView) SetSelectedContext(contextID string) {
	if v.contextTreeView != nil {
		v.contextTreeView = v.contextTreeView.SelectNode(contextID)
	}
}
