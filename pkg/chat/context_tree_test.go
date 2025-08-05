package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContextTree(t *testing.T) {
	tree := NewContextTree()
	
	require.NotNil(t, tree)
	assert.NotEmpty(t, tree.RootContextID)
	assert.Len(t, tree.Contexts, 1)
	assert.NotNil(t, tree.Messages)
	assert.NotNil(t, tree.ParentIndex)
	assert.NotNil(t, tree.ChildIndex)
	assert.Equal(t, tree.RootContextID, tree.ActiveContext)
	
	// Verify root context properties
	rootContext := tree.Contexts[tree.RootContextID]
	require.NotNil(t, rootContext)
	assert.Equal(t, tree.RootContextID, rootContext.ID)
	assert.Nil(t, rootContext.ParentID)
	assert.Nil(t, rootContext.BranchPoint)
	assert.Equal(t, "Main Conversation", rootContext.Title)
	assert.True(t, rootContext.IsActive)
	assert.Empty(t, rootContext.MessageIDs)
}

func TestContextTree_AddMessage(t *testing.T) {
	tree := NewContextTree()
	
	// Create test message
	msg1 := NewUserMessage("Hello world")
	
	t.Run("add message to root context", func(t *testing.T) {
		err := tree.AddMessage(msg1, tree.RootContextID)
		require.NoError(t, err)
		
		// Verify message was added
		assert.Len(t, tree.Messages, 1)
		storedMsg := tree.Messages[msg1.ID]
		require.NotNil(t, storedMsg)
		assert.Equal(t, tree.RootContextID, storedMsg.ContextID)
		assert.Nil(t, storedMsg.ParentID)
		
		// Verify context was updated
		rootContext := tree.Contexts[tree.RootContextID]
		assert.Len(t, rootContext.MessageIDs, 1)
		assert.Contains(t, rootContext.MessageIDs, msg1.ID)
	})
	
	t.Run("add second message creates parent relationship", func(t *testing.T) {
		msg2 := NewAssistantMessage("Hello back!")
		
		err := tree.AddMessage(msg2, tree.RootContextID)
		require.NoError(t, err)
		
		// Verify parent relationship
		storedMsg2 := tree.Messages[msg2.ID]
		require.NotNil(t, storedMsg2)
		assert.Equal(t, msg1.ID, *storedMsg2.ParentID)
		
		// Verify indices
		assert.Equal(t, msg1.ID, tree.ChildIndex[msg2.ID])
		assert.Contains(t, tree.ParentIndex[msg1.ID], msg2.ID)
	})
	
	t.Run("add message to non-existent context fails", func(t *testing.T) {
		msg3 := NewUserMessage("Test")
		
		err := tree.AddMessage(msg3, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context not found")
	})
}

func TestContextTree_BranchFromMessage(t *testing.T) {
	tree := NewContextTree()
	
	// Add initial message
	msg1 := NewUserMessage("Original message")
	err := tree.AddMessage(msg1, tree.RootContextID)
	require.NoError(t, err)
	
	t.Run("create branch from existing message", func(t *testing.T) {
		branchContext, err := tree.BranchFromMessage(msg1.ID, "Alternative branch")
		require.NoError(t, err)
		require.NotNil(t, branchContext)
		
		// Verify branch context properties
		assert.NotEmpty(t, branchContext.ID)
		assert.Equal(t, tree.RootContextID, *branchContext.ParentID)
		assert.Equal(t, msg1.ID, *branchContext.BranchPoint)
		assert.Equal(t, "Alternative branch", branchContext.Title)
		assert.False(t, branchContext.IsActive)
		assert.Empty(t, branchContext.MessageIDs)
		
		// Verify context was stored
		assert.Contains(t, tree.Contexts, branchContext.ID)
		
		// Verify source message was marked as branch point
		sourceMsg := tree.Messages[msg1.ID]
		require.NotNil(t, sourceMsg.Metadata)
		assert.True(t, sourceMsg.Metadata.BranchPoint)
		assert.Equal(t, 1, sourceMsg.Metadata.ChildCount)
		
		// Verify relationship indices
		assert.Contains(t, tree.ParentIndex[tree.RootContextID], branchContext.ID)
		assert.Equal(t, tree.RootContextID, tree.ChildIndex[branchContext.ID])
	})
	
	t.Run("branch from non-existent message fails", func(t *testing.T) {
		_, err := tree.BranchFromMessage("nonexistent", "Test branch")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message not found")
	})
}

func TestContextTree_GetConversationPath(t *testing.T) {
	tree := NewContextTree()
	
	// Build test conversation tree
	msg1 := NewUserMessage("Root message 1")
	msg2 := NewAssistantMessage("Root response 1")
	msg3 := NewUserMessage("Root message 2")
	
	err := tree.AddMessage(msg1, tree.RootContextID)
	require.NoError(t, err)
	err = tree.AddMessage(msg2, tree.RootContextID)
	require.NoError(t, err)
	err = tree.AddMessage(msg3, tree.RootContextID)
	require.NoError(t, err)
	
	// Create branch from msg2
	branchContext, err := tree.BranchFromMessage(msg2.ID, "Branch")
	require.NoError(t, err)
	
	// Add messages to branch
	branchMsg1 := NewUserMessage("Branch message 1")
	branchMsg2 := NewAssistantMessage("Branch response 1")
	
	err = tree.AddMessage(branchMsg1, branchContext.ID)
	require.NoError(t, err)
	err = tree.AddMessage(branchMsg2, branchContext.ID)
	require.NoError(t, err)
	
	t.Run("get root context path", func(t *testing.T) {
		path, err := tree.GetConversationPath(tree.RootContextID)
		require.NoError(t, err)
		assert.Len(t, path, 3)
		assert.Equal(t, msg1.ID, path[0].ID)
		assert.Equal(t, msg2.ID, path[1].ID)
		assert.Equal(t, msg3.ID, path[2].ID)
	})
	
	t.Run("get branch context path", func(t *testing.T) {
		path, err := tree.GetConversationPath(branchContext.ID)
		require.NoError(t, err)
		// The path should include all root messages, then the branch messages
		assert.True(t, len(path) >= 4) // At least msg1, msg2, branchMsg1, branchMsg2
		// Check that the branch messages are included
		foundBranchMsg1 := false
		foundBranchMsg2 := false
		for _, msg := range path {
			if msg.ID == branchMsg1.ID {
				foundBranchMsg1 = true
			}
			if msg.ID == branchMsg2.ID {
				foundBranchMsg2 = true
			}
		}
		assert.True(t, foundBranchMsg1, "Branch message 1 should be in path")
		assert.True(t, foundBranchMsg2, "Branch message 2 should be in path")
	})
	
	t.Run("get path for non-existent context fails", func(t *testing.T) {
		_, err := tree.GetConversationPath("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context not found")
	})
}

func TestContextTree_SwitchContext(t *testing.T) {
	tree := NewContextTree()
	
	// Add message and create branch
	msg1 := NewUserMessage("Test message")
	tree.AddMessage(msg1, tree.RootContextID)
	
	branchContext, err := tree.BranchFromMessage(msg1.ID, "Test branch")
	require.NoError(t, err)
	
	t.Run("switch to existing context", func(t *testing.T) {
		err := tree.SwitchContext(branchContext.ID)
		require.NoError(t, err)
		
		// Verify active context changed
		assert.Equal(t, branchContext.ID, tree.ActiveContext)
		assert.True(t, tree.Contexts[branchContext.ID].IsActive)
		assert.False(t, tree.Contexts[tree.RootContextID].IsActive)
		
		// Verify GetActiveContext returns correct context
		activeContext := tree.GetActiveContext()
		assert.Equal(t, branchContext.ID, activeContext.ID)
	})
	
	t.Run("switch to non-existent context fails", func(t *testing.T) {
		err := tree.SwitchContext("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context not found")
	})
}

func TestContextTree_GetContextBranches(t *testing.T) {
	tree := NewContextTree()
	
	// Add message
	msg1 := NewUserMessage("Test message")
	tree.AddMessage(msg1, tree.RootContextID)
	
	// Create multiple branches
	branch1, err := tree.BranchFromMessage(msg1.ID, "Branch 1")
	require.NoError(t, err)
	branch2, err := tree.BranchFromMessage(msg1.ID, "Branch 2")
	require.NoError(t, err)
	
	t.Run("get branches for root context", func(t *testing.T) {
		branches := tree.GetContextBranches(tree.RootContextID)
		assert.Len(t, branches, 2)
		
		branchIDs := []string{branches[0].ID, branches[1].ID}
		assert.Contains(t, branchIDs, branch1.ID)
		assert.Contains(t, branchIDs, branch2.ID)
	})
	
	t.Run("get branches for context with no branches", func(t *testing.T) {
		branches := tree.GetContextBranches(branch1.ID)
		assert.Empty(t, branches)
	})
	
	t.Run("get branches for non-existent context", func(t *testing.T) {
		branches := tree.GetContextBranches("nonexistent")
		assert.Empty(t, branches)
	})
}

func TestContextTree_GetMessage(t *testing.T) {
	tree := NewContextTree()
	
	msg1 := NewUserMessage("Test message")
	tree.AddMessage(msg1, tree.RootContextID)
	
	t.Run("get existing message", func(t *testing.T) {
		retrieved := tree.GetMessage(msg1.ID)
		require.NotNil(t, retrieved)
		assert.Equal(t, msg1.ID, retrieved.ID)
		assert.Equal(t, msg1.Content, retrieved.Content)
	})
	
	t.Run("get non-existent message", func(t *testing.T) {
		retrieved := tree.GetMessage("nonexistent")
		assert.Nil(t, retrieved)
	})
}

func TestContextTree_GetContext(t *testing.T) {
	tree := NewContextTree()
	
	t.Run("get existing context", func(t *testing.T) {
		context := tree.GetContext(tree.RootContextID)
		require.NotNil(t, context)
		assert.Equal(t, tree.RootContextID, context.ID)
	})
	
	t.Run("get non-existent context", func(t *testing.T) {
		context := tree.GetContext("nonexistent")
		assert.Nil(t, context)
	})
}

func TestContextTree_GetContextMessages(t *testing.T) {
	tree := NewContextTree()
	
	// Add messages to root context
	msg1 := NewUserMessage("Message 1")
	msg2 := NewAssistantMessage("Message 2")
	tree.AddMessage(msg1, tree.RootContextID)
	tree.AddMessage(msg2, tree.RootContextID)
	
	t.Run("get messages from existing context", func(t *testing.T) {
		messages := tree.GetContextMessages(tree.RootContextID)
		assert.Len(t, messages, 2)
		assert.Equal(t, msg1.ID, messages[0].ID)
		assert.Equal(t, msg2.ID, messages[1].ID)
	})
	
	t.Run("get messages from non-existent context", func(t *testing.T) {
		messages := tree.GetContextMessages("nonexistent")
		assert.Empty(t, messages)
	})
}