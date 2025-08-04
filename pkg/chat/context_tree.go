package chat

import (
	"fmt"
	"time"
)

// Context represents a conversation branch with its own message sequence
type Context struct {
	ID          string    `json:"id"`
	ParentID    *string   `json:"parent_id"`    // Context this branched from
	BranchPoint *string   `json:"branch_point"` // Message ID where branch occurred
	Title       string    `json:"title"`        // User-defined or auto-generated
	Created     time.Time `json:"created"`
	MessageIDs  []string  `json:"message_ids"` // Linear message sequence in this context
	IsActive    bool      `json:"is_active"`   // Current active context
}

// ContextTree manages all conversation branches and their relationships
type ContextTree struct {
	RootContextID string              `json:"root_context_id"`
	Contexts      map[string]*Context `json:"contexts"`
	Messages      map[string]*Message `json:"messages"`       // All messages by ID
	ParentIndex   map[string][]string `json:"parent_index"`   // parentID -> childIDs
	ChildIndex    map[string]string   `json:"child_index"`    // childID -> parentID
	ActiveContext string              `json:"active_context"` // Currently active context
}

// NewContextTree creates a new context tree with a root context
func NewContextTree() *ContextTree {
	rootContextID := generateContextID()

	rootContext := &Context{
		ID:          rootContextID,
		ParentID:    nil,
		BranchPoint: nil,
		Title:       "Main Conversation",
		Created:     time.Now(),
		MessageIDs:  []string{},
		IsActive:    true,
	}

	return &ContextTree{
		RootContextID: rootContextID,
		Contexts:      map[string]*Context{rootContextID: rootContext},
		Messages:      make(map[string]*Message),
		ParentIndex:   make(map[string][]string),
		ChildIndex:    make(map[string]string),
		ActiveContext: rootContextID,
	}
}

// AddMessage adds a message to the tree and the specified context
func (ct *ContextTree) AddMessage(message Message, contextID string) error {
	// Ensure context exists
	context, exists := ct.Contexts[contextID]
	if !exists {
		return fmt.Errorf("context not found: %s", contextID)
	}

	// Set the message's context
	message.ContextID = contextID

	// Set parent relationship if this isn't the first message in context
	if len(context.MessageIDs) > 0 {
		lastMessageID := context.MessageIDs[len(context.MessageIDs)-1]
		message.ParentID = &lastMessageID

		// Update child index
		ct.ChildIndex[message.ID] = lastMessageID

		// Update parent index
		if ct.ParentIndex[lastMessageID] == nil {
			ct.ParentIndex[lastMessageID] = []string{}
		}
		ct.ParentIndex[lastMessageID] = append(ct.ParentIndex[lastMessageID], message.ID)
	}

	// Calculate depth
	if message.Metadata != nil {
		message.Metadata.Depth = len(context.MessageIDs)
	}

	// Add to tree structures
	ct.Messages[message.ID] = &message
	context.MessageIDs = append(context.MessageIDs, message.ID)

	return nil
}

// BranchFromMessage creates a new conversation branch from any message
func (ct *ContextTree) BranchFromMessage(messageID string, title string) (*Context, error) {
	sourceMsg, exists := ct.Messages[messageID]
	if !exists {
		return nil, fmt.Errorf("message not found: %s", messageID)
	}

	// Create new context
	newContext := &Context{
		ID:          generateContextID(),
		ParentID:    &sourceMsg.ContextID,
		BranchPoint: &messageID,
		Title:       title,
		Created:     time.Now(),
		MessageIDs:  []string{},
		IsActive:    false,
	}

	// Add to tree
	ct.Contexts[newContext.ID] = newContext

	// Mark source message as branch point
	if sourceMsg.Metadata != nil {
		sourceMsg.Metadata.BranchPoint = true
		sourceMsg.Metadata.ChildCount++
	}

	// Update context relationship indexes
	if ct.ParentIndex[sourceMsg.ContextID] == nil {
		ct.ParentIndex[sourceMsg.ContextID] = []string{}
	}
	ct.ParentIndex[sourceMsg.ContextID] = append(ct.ParentIndex[sourceMsg.ContextID], newContext.ID)
	ct.ChildIndex[newContext.ID] = sourceMsg.ContextID

	return newContext, nil
}

// GetConversationPath returns the full conversation path from root to specified context
func (ct *ContextTree) GetConversationPath(contextID string) ([]Message, error) {
	context, exists := ct.Contexts[contextID]
	if !exists {
		return nil, fmt.Errorf("context not found: %s", contextID)
	}

	var path []Message

	// Traverse up to root, collecting branch point messages and their lineage
	currentContext := context
	var contextChain []*Context

	// Build chain from target to root
	for currentContext != nil {
		contextChain = append([]*Context{currentContext}, contextChain...)

		if currentContext.ParentID != nil {
			currentContext = ct.Contexts[*currentContext.ParentID]
		} else {
			break
		}
	}

	// Build path by following the chain
	for i, ctx := range contextChain {
		if i == 0 {
			// Root context - add all messages up to branch point (if any)
			if ctx.BranchPoint != nil {
				branchMessages := ct.getMessagesUpToPoint(ct.RootContextID, *ctx.BranchPoint)
				path = append(path, branchMessages...)
			} else {
				// Add all messages in root context
				for _, msgID := range ctx.MessageIDs {
					if msg := ct.Messages[msgID]; msg != nil {
						path = append(path, *msg)
					}
				}
			}
		} else {
			// Non-root context - add all messages
			for _, msgID := range ctx.MessageIDs {
				if msg := ct.Messages[msgID]; msg != nil {
					path = append(path, *msg)
				}
			}
		}
	}

	return path, nil
}

// getMessagesUpToPoint gets all messages in a context up to and including a specific message
func (ct *ContextTree) getMessagesUpToPoint(contextID, messageID string) []Message {
	context := ct.Contexts[contextID]
	if context == nil {
		return []Message{}
	}

	var messages []Message
	for _, msgID := range context.MessageIDs {
		if msg := ct.Messages[msgID]; msg != nil {
			messages = append(messages, *msg)
		}
		if msgID == messageID {
			break
		}
	}

	return messages
}

// SwitchContext changes the active context
func (ct *ContextTree) SwitchContext(contextID string) error {
	if _, exists := ct.Contexts[contextID]; !exists {
		return fmt.Errorf("context not found: %s", contextID)
	}

	// Deactivate current context
	if currentContext := ct.Contexts[ct.ActiveContext]; currentContext != nil {
		currentContext.IsActive = false
	}

	// Activate new context
	ct.ActiveContext = contextID
	ct.Contexts[contextID].IsActive = true

	return nil
}

// GetActiveContext returns the currently active context
func (ct *ContextTree) GetActiveContext() *Context {
	return ct.Contexts[ct.ActiveContext]
}

// GetContextBranches returns all child contexts of the specified context
func (ct *ContextTree) GetContextBranches(contextID string) []*Context {
	var branches []*Context

	if childIDs, exists := ct.ParentIndex[contextID]; exists {
		for _, childID := range childIDs {
			if context := ct.Contexts[childID]; context != nil {
				branches = append(branches, context)
			}
		}
	}

	return branches
}

// GetMessageBranches returns all child messages of the specified message
func (ct *ContextTree) GetMessageBranches(messageID string) []*Message {
	var branches []*Message

	if childIDs, exists := ct.ParentIndex[messageID]; exists {
		for _, childID := range childIDs {
			if message := ct.Messages[childID]; message != nil {
				branches = append(branches, message)
			}
		}
	}

	return branches
}

// GetMessage returns a message by ID
func (ct *ContextTree) GetMessage(id string) *Message {
	return ct.Messages[id]
}

// GetContext returns a context by ID
func (ct *ContextTree) GetContext(id string) *Context {
	return ct.Contexts[id]
}

// GetContextMessages returns all messages in a specific context
func (ct *ContextTree) GetContextMessages(contextID string) []Message {
	context := ct.Contexts[contextID]
	if context == nil {
		return []Message{}
	}

	var messages []Message
	for _, msgID := range context.MessageIDs {
		if msg := ct.Messages[msgID]; msg != nil {
			messages = append(messages, *msg)
		}
	}

	return messages
}

// DeleteContext removes a context and all its messages (recursive for child contexts)
func (ct *ContextTree) DeleteContext(contextID string) error {
	if contextID == ct.RootContextID {
		return fmt.Errorf("cannot delete root context")
	}

	context := ct.Contexts[contextID]
	if context == nil {
		return fmt.Errorf("context not found: %s", contextID)
	}

	// Recursively delete child contexts
	if childIDs, exists := ct.ParentIndex[contextID]; exists {
		for _, childID := range childIDs {
			ct.DeleteContext(childID)
		}
		delete(ct.ParentIndex, contextID)
	}

	// Remove messages
	for _, msgID := range context.MessageIDs {
		delete(ct.Messages, msgID)
		delete(ct.ParentIndex, msgID)
		delete(ct.ChildIndex, msgID)
	}

	// Remove context
	delete(ct.Contexts, contextID)
	delete(ct.ChildIndex, contextID)

	// Remove from parent's child list
	if context.ParentID != nil {
		if siblings := ct.ParentIndex[*context.ParentID]; siblings != nil {
			for i, siblingID := range siblings {
				if siblingID == contextID {
					ct.ParentIndex[*context.ParentID] = append(siblings[:i], siblings[i+1:]...)
					break
				}
			}
		}
	}

	// If deleting active context, switch to root
	if ct.ActiveContext == contextID {
		ct.SwitchContext(ct.RootContextID)
	}

	return nil
}
