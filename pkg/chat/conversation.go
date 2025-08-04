package chat

import (
	"strings"
	"time"
)

// Conversation now wraps a ContextTree and provides backwards-compatible interface
type Conversation struct {
	Tree  *ContextTree
	Model string
}

// NewConversation creates a new conversation with a context tree
func NewConversation(model string) Conversation {
	return Conversation{
		Tree:  NewContextTree(),
		Model: model,
	}
}

// NewConversationWithSystem creates a new conversation with a system message
func NewConversationWithSystem(model, systemPrompt string) Conversation {
	conv := NewConversation(model)
	if systemPrompt != "" {
		systemMsg := NewSystemMessage(systemPrompt)
		conv.Tree.AddMessage(systemMsg, conv.Tree.ActiveContext)
	}
	return conv
}

// NewConversationFromTree creates a conversation from an existing context tree
func NewConversationFromTree(tree *ContextTree, model string) Conversation {
	return Conversation{
		Tree:  tree,
		Model: model,
	}
}

// AddMessage adds a message to the active context (backwards compatibility)
func AddMessage(conv Conversation, msg Message) Conversation {
	// Clone the tree to maintain immutability
	newTree := cloneContextTree(conv.Tree)

	// Add message to active context
	newTree.AddMessage(msg, newTree.ActiveContext)

	return Conversation{
		Tree:  newTree,
		Model: conv.Model,
	}
}

// AddMessageToContext adds a message to a specific context
func AddMessageToContext(conv Conversation, msg Message, contextID string) (Conversation, error) {
	// Clone the tree to maintain immutability
	newTree := cloneContextTree(conv.Tree)

	// Add message to specified context
	if err := newTree.AddMessage(msg, contextID); err != nil {
		return conv, err
	}

	return Conversation{
		Tree:  newTree,
		Model: conv.Model,
	}, nil
}

// GetMessages returns all messages in the active context path (backwards compatibility)
func GetMessages(conv Conversation) []Message {
	messages, err := conv.Tree.GetConversationPath(conv.Tree.ActiveContext)
	if err != nil {
		// Fallback to active context messages only
		return conv.Tree.GetContextMessages(conv.Tree.ActiveContext)
	}
	return messages
}

// GetContextMessages returns messages in a specific context
func GetContextMessages(conv Conversation, contextID string) []Message {
	return conv.Tree.GetContextMessages(contextID)
}

// GetActiveContextMessages returns messages in the currently active context
func GetActiveContextMessages(conv Conversation) []Message {
	return conv.Tree.GetContextMessages(conv.Tree.ActiveContext)
}

// GetMessageCount returns the number of messages in the active context path
func GetMessageCount(conv Conversation) int {
	return len(GetMessages(conv))
}

// GetLastMessage returns the last message in the active context
func GetLastMessage(conv Conversation) (Message, bool) {
	activeMessages := conv.Tree.GetContextMessages(conv.Tree.ActiveContext)
	if len(activeMessages) == 0 {
		return Message{}, false
	}
	return activeMessages[len(activeMessages)-1], true
}

// GetLastAssistantMessage returns the last assistant message in the active context path
func GetLastAssistantMessage(conv Conversation) (Message, bool) {
	messages := GetMessages(conv)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.IsAssistant() {
			return msg, true
		}
	}
	return Message{}, false
}

// GetLastUserMessage returns the last user message in the active context path
func GetLastUserMessage(conv Conversation) (Message, bool) {
	messages := GetMessages(conv)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.IsUser() {
			return msg, true
		}
	}
	return Message{}, false
}

// GetMessagesByRole returns all messages with the specified role in the active context path
func GetMessagesByRole(conv Conversation, role string) []Message {
	var result []Message
	messages := GetMessages(conv)
	for _, msg := range messages {
		if msg.Role == role {
			result = append(result, msg)
		}
	}
	return result
}

// GetMessagesAfter returns messages after a specific timestamp in the active context path
func GetMessagesAfter(conv Conversation, timestamp time.Time) []Message {
	var result []Message
	messages := GetMessages(conv)
	for _, msg := range messages {
		if msg.Timestamp.After(timestamp) {
			result = append(result, msg)
		}
	}
	return result
}

// GetMessagesBefore returns messages before a specific timestamp in the active context path
func GetMessagesBefore(conv Conversation, timestamp time.Time) []Message {
	var result []Message
	messages := GetMessages(conv)
	for _, msg := range messages {
		if msg.Timestamp.Before(timestamp) {
			result = append(result, msg)
		}
	}
	return result
}

// IsEmpty returns true if the conversation has no messages
func IsEmpty(conv Conversation) bool {
	return len(GetMessages(conv)) == 0
}

// HasSystemMessage returns true if there's a system message in the active context path
func HasSystemMessage(conv Conversation) bool {
	messages := GetMessages(conv)
	for _, msg := range messages {
		if msg.IsSystem() {
			return true
		}
	}
	return false
}

// WithModel returns a conversation with a different model
func WithModel(conv Conversation, model string) Conversation {
	return Conversation{
		Tree:  conv.Tree,
		Model: model,
	}
}

// Context-aware enhanced functions

// BranchFromMessage creates a new conversation branch from any message
func BranchFromMessage(conv Conversation, messageID, title string) (Conversation, *Context, error) {
	// Clone the tree to maintain immutability
	newTree := cloneContextTree(conv.Tree)

	// Create the branch
	context, err := newTree.BranchFromMessage(messageID, title)
	if err != nil {
		return conv, nil, err
	}

	return Conversation{
		Tree:  newTree,
		Model: conv.Model,
	}, context, nil
}

// SwitchToContext switches the active context
func SwitchToContext(conv Conversation, contextID string) (Conversation, error) {
	// Clone the tree to maintain immutability
	newTree := cloneContextTree(conv.Tree)

	// Switch context
	if err := newTree.SwitchContext(contextID); err != nil {
		return conv, err
	}

	return Conversation{
		Tree:  newTree,
		Model: conv.Model,
	}, nil
}

// GetActiveContext returns the currently active context
func GetActiveContext(conv Conversation) *Context {
	return conv.Tree.GetActiveContext()
}

// GetContextBranches returns all child contexts of the current active context
func GetContextBranches(conv Conversation) []*Context {
	return conv.Tree.GetContextBranches(conv.Tree.ActiveContext)
}

// GetMessageBranches returns all child messages of a specific message
func GetMessageBranches(conv Conversation, messageID string) []*Message {
	return conv.Tree.GetMessageBranches(messageID)
}

// Enhanced conversation management functions with context awareness

// AddMessageWithDeduplication adds a message while preventing duplicates based on source and content
func AddMessageWithDeduplication(conv Conversation, msg Message) Conversation {
	// If this is a final message, remove any optimistic messages with similar content
	if msg.GetSource() == MessageSourceFinal && msg.IsUser() {
		conv = RemoveOptimisticMessages(conv, msg.Content)
	}

	// Add the new message
	return AddMessage(conv, msg)
}

// RemoveOptimisticMessages removes optimistic messages that match the given content
func RemoveOptimisticMessages(conv Conversation, content string) Conversation {
	// Clone the tree
	newTree := cloneContextTree(conv.Tree)

	// Get active context messages
	activeContext := newTree.GetActiveContext()
	if activeContext == nil {
		return conv
	}

	var filteredMessageIDs []string

	for _, msgID := range activeContext.MessageIDs {
		msg := newTree.Messages[msgID]
		if msg == nil {
			continue
		}

		// Keep message if it's not optimistic or doesn't match content
		if !msg.IsOptimistic() ||
			strings.TrimSpace(msg.Content) != strings.TrimSpace(content) {
			filteredMessageIDs = append(filteredMessageIDs, msgID)
		} else {
			// Remove from tree structures
			delete(newTree.Messages, msgID)
			delete(newTree.ParentIndex, msgID)
			delete(newTree.ChildIndex, msgID)
		}
	}

	activeContext.MessageIDs = filteredMessageIDs

	return Conversation{
		Tree:  newTree,
		Model: conv.Model,
	}
}

// ReplaceOptimisticMessage replaces an optimistic message with a final one
func ReplaceOptimisticMessage(conv Conversation, optimisticContent string, finalMsg Message) Conversation {
	// Clone the tree
	newTree := cloneContextTree(conv.Tree)

	// Find and replace the optimistic message
	activeContext := newTree.GetActiveContext()
	if activeContext == nil {
		return AddMessage(conv, finalMsg)
	}

	for _, msgID := range activeContext.MessageIDs {
		msg := newTree.Messages[msgID]
		if msg != nil && msg.IsOptimistic() &&
			strings.TrimSpace(msg.Content) == strings.TrimSpace(optimisticContent) {

			// Replace the message
			finalMsg.ContextID = newTree.ActiveContext
			newTree.Messages[msgID] = &finalMsg
			return Conversation{
				Tree:  newTree,
				Model: conv.Model,
			}
		}
	}

	// If no optimistic message found, just add the final message
	return AddMessage(Conversation{Tree: newTree, Model: conv.Model}, finalMsg)
}

// GetOptimisticMessages returns all optimistic messages in the active context
func GetOptimisticMessages(conv Conversation) []Message {
	var optimistic []Message
	messages := GetMessages(conv)
	for _, msg := range messages {
		if msg.IsOptimistic() {
			optimistic = append(optimistic, msg)
		}
	}
	return optimistic
}

// GetStreamingMessages returns all streaming messages in the active context
func GetStreamingMessages(conv Conversation) []Message {
	var streaming []Message
	messages := GetMessages(conv)
	for _, msg := range messages {
		if msg.IsStreaming() {
			streaming = append(streaming, msg)
		}
	}
	return streaming
}

// RemoveStreamingMessages removes all streaming messages with the given stream ID
func RemoveStreamingMessages(conv Conversation, streamID string) Conversation {
	// Clone the tree
	newTree := cloneContextTree(conv.Tree)

	// Remove streaming messages from all contexts
	for _, context := range newTree.Contexts {
		var filteredMessageIDs []string

		for _, msgID := range context.MessageIDs {
			msg := newTree.Messages[msgID]
			if msg == nil {
				continue
			}

			// Keep message if it's not streaming or has different stream ID
			if !msg.IsStreaming() || msg.GetStreamID() != streamID {
				filteredMessageIDs = append(filteredMessageIDs, msgID)
			} else {
				// Remove from tree structures
				delete(newTree.Messages, msgID)
				delete(newTree.ParentIndex, msgID)
				delete(newTree.ChildIndex, msgID)
			}
		}

		context.MessageIDs = filteredMessageIDs
	}

	return Conversation{
		Tree:  newTree,
		Model: conv.Model,
	}
}

// Helper function to clone a context tree (deep copy)
func cloneContextTree(original *ContextTree) *ContextTree {
	clone := &ContextTree{
		RootContextID: original.RootContextID,
		Contexts:      make(map[string]*Context),
		Messages:      make(map[string]*Message),
		ParentIndex:   make(map[string][]string),
		ChildIndex:    make(map[string]string),
		ActiveContext: original.ActiveContext,
	}

	// Clone contexts
	for id, context := range original.Contexts {
		clonedContext := &Context{
			ID:          context.ID,
			ParentID:    context.ParentID,
			BranchPoint: context.BranchPoint,
			Title:       context.Title,
			Created:     context.Created,
			MessageIDs:  make([]string, len(context.MessageIDs)),
			IsActive:    context.IsActive,
		}
		copy(clonedContext.MessageIDs, context.MessageIDs)
		clone.Contexts[id] = clonedContext
	}

	// Clone messages
	for id, msg := range original.Messages {
		clonedMsg := *msg // Shallow copy is fine for Message
		clone.Messages[id] = &clonedMsg
	}

	// Clone indices
	for parent, children := range original.ParentIndex {
		clone.ParentIndex[parent] = make([]string, len(children))
		copy(clone.ParentIndex[parent], children)
	}

	for child, parent := range original.ChildIndex {
		clone.ChildIndex[child] = parent
	}

	return clone
}
