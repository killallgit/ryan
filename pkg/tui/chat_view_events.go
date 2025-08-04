package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/logger"
)

// Event handling methods for ChatView
// This file contains all key event processing and routing logic

func (cv *ChatView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool {
	// Handle modal events first
	if cv.helpModal.Visible {
		modal, handled := cv.helpModal.HandleKeyEvent(ev)
		cv.helpModal = modal
		return handled
	}

	if cv.progressModal.Visible {
		modal, cancel := cv.progressModal.HandleKeyEvent(ev)
		cv.progressModal = modal
		if cancel && cv.downloadCancel != nil {
			cv.downloadCancel()
		}
		return true
	}

	if cv.downloadModal.Visible {
		modal, confirmed, _ := cv.downloadModal.HandleKeyEvent(ev)
		cv.downloadModal = modal
		if confirmed {
			cv.startModelDownload(cv.downloadModal.ModelName)
		}
		return true
	}

	// Handle help key (?) in any mode
	if cv.handleHelpKey(ev) {
		return true
	}

	// Handle mode switching (Ctrl+N)
	if cv.handleModeSwitch(ev) {
		return true
	}

	// Handle keys based on current mode
	switch cv.mode {
	case ModeInput:
		return cv.handleInputModeKeys(ev, sending)
	case ModeNodes:
		return cv.handleNodeModeKeys(ev, sending)
	default:
		return cv.handleInputModeKeys(ev, sending) // Fallback to input mode
	}
}

func (cv *ChatView) handleHelpKey(ev *tcell.EventKey) bool {
	// F2 key to show help modal
	if ev.Key() == tcell.KeyF2 {
		cv.helpModal = cv.helpModal.Show()
		return true
	}
	return false
}

func (cv *ChatView) handleModeSwitch(ev *tcell.EventKey) bool {
	// Ctrl+N to switch modes - use tcell's built-in KeyCtrlN constant
	if ev.Key() == tcell.KeyCtrlN {
		cv.switchMode()
		return true
	}
	return false
}

func (cv *ChatView) switchMode() {
	log := logger.WithComponent("chat_view")

	switch cv.mode {
	case ModeInput:
		cv.mode = ModeNodes
		// Enable node-based rendering if not already enabled
		cv.messages = cv.messages.EnableNodes()
		// Auto-focus first node if none focused
		if cv.messages.NodeManager != nil && cv.messages.NodeManager.GetFocusedNode() == "" {
			cv.messages.MoveFocusDown() // This will focus the first node
		}
		log.Debug("Switched to node selection mode")
	case ModeNodes:
		cv.mode = ModeInput
		// Clear any node focus when switching back to input
		if cv.messages.NodeManager != nil {
			cv.messages.NodeManager.SetFocusedNode("")
		}
		log.Debug("Switched to input mode")
	}

	// Update status bar to show current mode
	cv.updateStatusForMode()
}

func (cv *ChatView) handleInputModeKeys(ev *tcell.EventKey, sending bool) bool {
	switch ev.Key() {
	case tcell.KeyEnter:
		if !sending {
			content := cv.sendMessage()
			if content != "" {
				cv.screen.PostEvent(NewChatMessageSendEvent(content))
				// Immediately update the UI to show the user message
				cv.updateMessages()
				cv.scrollToBottom()
			}
		}
		return true

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		cv.input = cv.input.DeleteBackward()
		return true

	case tcell.KeyLeft:
		cv.input = cv.input.WithCursor(cv.input.Cursor - 1)
		return true

	case tcell.KeyRight:
		cv.input = cv.input.WithCursor(cv.input.Cursor + 1)
		return true

	case tcell.KeyHome:
		cv.input = cv.input.WithCursor(0)
		return true

	case tcell.KeyEnd:
		cv.input = cv.input.WithCursor(len(cv.input.Content))
		return true

	case tcell.KeyUp:
		// Check for Ctrl+Up for node navigation
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			if cv.handleNodeNavigation(true) {
				return true
			}
		}
		cv.scrollUp()
		return true

	case tcell.KeyDown:
		// Check for Ctrl+Down for node navigation
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			if cv.handleNodeNavigation(false) {
				return true
			}
		}
		cv.scrollDown()
		return true

	case tcell.KeyPgUp:
		cv.pageUp()
		return true

	case tcell.KeyPgDn:
		cv.pageDown()
		return true

	case tcell.KeyTab:
		// Tab to expand/collapse focused node
		if cv.handleNodeToggleExpansion() {
			return true
		}

	default:
		if ev.Rune() != 0 {
			// Check for specific key combinations for node operations
			switch ev.Rune() {
			case ' ':
				// Space to select/deselect focused node
				if ev.Modifiers()&tcell.ModCtrl != 0 {
					if cv.handleNodeToggleSelection() {
						return true
					}
				}
			case 'a', 'A':
				// Ctrl+A to select all nodes
				if ev.Modifiers()&tcell.ModCtrl != 0 {
					if cv.handleSelectAllNodes() {
						return true
					}
				}
			case 'c', 'C':
				// Ctrl+C to clear selection (if not sending)
				if ev.Modifiers()&tcell.ModCtrl != 0 && !sending {
					if cv.handleClearNodeSelection() {
						return true
					}
				}
			}

			cv.input = cv.input.InsertRune(ev.Rune())
			return true
		}
	}

	return false
}

func (cv *ChatView) handleNodeModeKeys(ev *tcell.EventKey, sending bool) bool {
	// In node mode, most keys are for navigation and selection
	switch ev.Key() {
	case tcell.KeyEnter:
		// Enter toggles selection of focused node
		if cv.handleNodeToggleSelection() {
			return true
		}

	case tcell.KeyTab:
		// Tab toggles expansion of focused node
		if cv.handleNodeToggleExpansion() {
			return true
		}

	case tcell.KeyUp:
		// Up arrow moves focus up
		if cv.handleNodeNavigation(true) {
			return true
		}

	case tcell.KeyDown:
		// Down arrow moves focus down
		if cv.handleNodeNavigation(false) {
			return true
		}

	case tcell.KeyPgUp:
		cv.pageUp()
		return true

	case tcell.KeyPgDn:
		cv.pageDown()
		return true

	case tcell.KeyEscape:
		// Escape switches back to input mode
		cv.mode = ModeInput
		if cv.messages.NodeManager != nil {
			cv.messages.NodeManager.SetFocusedNode("")
		}
		cv.updateStatusForMode()
		return true

	default:
		if ev.Rune() != 0 {
			switch ev.Rune() {
			case 'j', 'J':
				// j key moves focus down (vim-style)
				if cv.handleNodeNavigation(false) {
					return true
				}

			case 'k', 'K':
				// k key moves focus up (vim-style)
				if cv.handleNodeNavigation(true) {
					return true
				}

			case ' ':
				// Space toggles selection of focused node
				if cv.handleNodeToggleSelection() {
					return true
				}

			case 'a', 'A':
				// a to select all nodes
				if cv.handleSelectAllNodes() {
					return true
				}

			case 'c', 'C':
				// c to clear selection
				if cv.handleClearNodeSelection() {
					return true
				}

			case 'i', 'I':
				// i to switch to input mode (vim-style)
				cv.mode = ModeInput
				if cv.messages.NodeManager != nil {
					cv.messages.NodeManager.SetFocusedNode("")
				}
				cv.updateStatusForMode()
				return true
			}
		}
	}

	return false
}

func (cv *ChatView) HandleMouseEvent(ev *tcell.EventMouse) bool {
	log := logger.WithComponent("chat_view")

	// Get mouse coordinates
	x, y := ev.Position()
	buttons := ev.Buttons()

	log.Debug("Chat view mouse event", "x", x, "y", y, "buttons", buttons)

	// Only handle left mouse button clicks
	if buttons&tcell.ButtonPrimary == 0 {
		return false
	}

	// Check if the click is in the message area
	messageArea, _, _, _ := cv.layout.CalculateAreas()

	if x >= messageArea.X && x < messageArea.X+messageArea.Width &&
		y >= messageArea.Y && y < messageArea.Y+messageArea.Height {

		// Handle click in message area
		if cv.messages.UseNodes && cv.messages.NodeManager != nil {
			// Use node-based click handling
			nodeID, handled := cv.messages.HandleClick(x, y)
			if handled {
				log.Debug("Node click handled", "node_id", nodeID, "x", x, "y", y)

				// Switch to node mode when a node is clicked
				if cv.mode == ModeInput {
					cv.mode = ModeNodes
					cv.updateStatusForMode()
					log.Debug("Switched to node mode due to mouse click on node")
				}

				// Focus the clicked node
				cv.messages.NodeManager.SetFocusedNode(nodeID)

				// Post node click event for further processing if needed
				cv.screen.PostEvent(NewMessageNodeClickEvent(nodeID, x-messageArea.X, y-messageArea.Y))
				return true
			}
		}

		// For legacy mode or if node handling didn't work,
		// we could implement basic click handling here
		log.Debug("Click in message area not handled by nodes", "x", x, "y", y)
		return true // Still consume the event even if not handled
	}

	// Click was not in message area
	return false
}
