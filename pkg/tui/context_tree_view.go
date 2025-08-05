package tui

import (
	"fmt"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/rivo/tview"
)

// ContextTreeView represents the context tree interface using tview
type ContextTreeView struct {
	*tview.Flex
	
	// Components
	tree   *tview.TreeView
	info   *tview.TextView
	status *tview.TextView
	
	// State
	contextTree *chat.ContextTree
}

// NewContextTreeView creates a new context tree view
func NewContextTreeView() *ContextTreeView {
	ctv := &ContextTreeView{
		Flex: tview.NewFlex().SetDirection(tview.FlexRow),
	}
	
	// Create tree view
	root := tview.NewTreeNode("Conversations").
		SetColor(ColorYellow)
	
	ctv.tree = tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)
	
	ctv.tree.SetBorder(false).SetTitle("")
	ctv.tree.SetBackgroundColor(ColorBase00)
	
	// Create info panel
	ctv.info = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	ctv.info.SetBorder(false).SetTitle("")
	ctv.info.SetBackgroundColor(ColorBase01)
	
	// Create status bar
	ctv.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	ctv.status.SetBackgroundColor(ColorBase01)
	ctv.status.SetText("[#5c5044]Use arrow keys to navigate | Enter to select | Esc to go back[-]")
	
	// Layout with horizontal split for tree and info
	mainContent := tview.NewFlex().
		AddItem(ctv.tree, 0, 1, true).
		AddItem(ctv.info, 0, 1, false)
	
	ctv.AddItem(mainContent, 0, 1, true).
		AddItem(ctv.status, 1, 0, false)
	
	// Setup selection handler
	ctv.tree.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref != nil {
			ctv.showContextInfo(ref.(string))
		}
		
		// Toggle expansion
		node.SetExpanded(!node.IsExpanded())
	})
	
	// Add sample data
	ctv.addSampleData()
	
	return ctv
}

// UpdateTree updates the tree with new context data
func (ctv *ContextTreeView) UpdateTree(tree *chat.ContextTree) {
	ctv.contextTree = tree
	ctv.rebuildTree()
}

// rebuildTree rebuilds the tree view from the context tree
func (ctv *ContextTreeView) rebuildTree() {
	root := ctv.tree.GetRoot()
	root.ClearChildren()
	
	if ctv.contextTree == nil {
		return
	}
	
	// Add contexts to tree
	for _, context := range ctv.contextTree.Contexts {
		node := tview.NewTreeNode(context.Title).
			SetReference(context.ID).
			SetSelectable(true)
		
		if context.IsActive {
			node.SetColor(ColorGreen)
		}
		
		// Add message count
		msgCount := len(context.MessageIDs)
		node.SetText(fmt.Sprintf("%s (%d messages)", context.Title, msgCount))
		
		root.AddChild(node)
		
		// Add child branches
		if children, exists := ctv.contextTree.ParentIndex[context.ID]; exists {
			for _, childID := range children {
				if child, ok := ctv.contextTree.Contexts[childID]; ok {
					childNode := tview.NewTreeNode(child.Title).
						SetReference(child.ID).
						SetSelectable(true)
					node.AddChild(childNode)
				}
			}
		}
	}
}

// addSampleData adds sample conversation tree data
func (ctv *ContextTreeView) addSampleData() {
	root := ctv.tree.GetRoot()
	
	// Main conversation
	main := tview.NewTreeNode("Main Conversation").
		SetReference("main").
		SetColor(ColorGreen)
	
	// Add some messages as children
	msg1 := tview.NewTreeNode("User: Hello").SetSelectable(false)
	msg2 := tview.NewTreeNode("Assistant: Hi there!").SetSelectable(false)
	msg3 := tview.NewTreeNode("User: How are you?").SetSelectable(false)
	
	main.AddChild(msg1)
	main.AddChild(msg2)
	main.AddChild(msg3)
	
	// Alternative branch
	alt := tview.NewTreeNode("Alternative Response").
		SetReference("alt1")
	
	altMsg := tview.NewTreeNode("Assistant: Hello! How can I help?").SetSelectable(false)
	alt.AddChild(altMsg)
	
	root.AddChild(main)
	root.AddChild(alt)
}

// showContextInfo displays information about the selected context
func (ctv *ContextTreeView) showContextInfo(contextID string) {
	info := fmt.Sprintf("[#f5b761]Context ID:[-] %s\n\n", contextID)
	
	if ctv.contextTree != nil && ctv.contextTree.Contexts[contextID] != nil {
		ctx := ctv.contextTree.Contexts[contextID]
		info += fmt.Sprintf("[#61afaf]Title:[-] %s\n", ctx.Title)
		info += fmt.Sprintf("[#61afaf]Messages:[-] %d\n", len(ctx.MessageIDs))
		info += fmt.Sprintf("[#61afaf]Active:[-] %v\n", ctx.IsActive)
		
		if ctx.ParentID != nil {
			info += fmt.Sprintf("[#61afaf]Parent:[-] %s\n", *ctx.ParentID)
		}
	} else {
		// Sample info for demo
		info += "[#61afaf]Title:[-] Sample Context\n"
		info += "[#61afaf]Messages:[-] 3\n"
		info += "[#61afaf]Created:[-] Just now\n"
	}
	
	ctv.info.SetText(info)
}