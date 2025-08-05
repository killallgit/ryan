package chat

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultGraphAwareMemoryConfig(t *testing.T) {
	config := DefaultGraphAwareMemoryConfig()
	
	assert.Equal(t, 20, config.MaxRetrieved)
	assert.True(t, config.IncludeContext)
	assert.Equal(t, 3, config.ContextDepth)
	assert.NotZero(t, config.VectorConfig.MaxRetrieved)
}

func TestNewGraphAwareLangChainMemory(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{} // Mock would be better
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	require.NotNil(t, memory)
	
	assert.Equal(t, tree, memory.contextTree)
	assert.Equal(t, vectorManager, memory.vectorManager)
	assert.Equal(t, tree.ActiveContext, memory.currentContext)
	assert.Equal(t, config.MaxRetrieved, memory.maxRetrieved)
	assert.Equal(t, config.IncludeContext, memory.includeContext)
}

func TestGraphAwareLangChainMemory_SwitchContext(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{}
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	
	// Add message and create branch
	msg1 := NewUserMessage("Test message")
	tree.AddMessage(msg1, tree.RootContextID)
	
	branchContext, err := tree.BranchFromMessage(msg1.ID, "Test branch")
	require.NoError(t, err)
	
	t.Run("switch to existing context", func(t *testing.T) {
		err := memory.SwitchContext(branchContext.ID)
		require.NoError(t, err)
		
		assert.Equal(t, branchContext.ID, memory.currentContext)
		assert.Equal(t, branchContext.ID, memory.contextTree.ActiveContext)
	})
	
	t.Run("switch to non-existent context fails", func(t *testing.T) {
		err := memory.SwitchContext("nonexistent")
		assert.Error(t, err)
	})
}

func TestGraphAwareLangChainMemory_BranchFromMessage(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{}
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	
	// Add message
	msg1 := NewUserMessage("Test message")
	tree.AddMessage(msg1, tree.RootContextID)
	
	t.Run("create branch from existing message", func(t *testing.T) {
		branchContext, err := memory.BranchFromMessage(msg1.ID, "Test branch")
		require.NoError(t, err)
		require.NotNil(t, branchContext)
		
		assert.Equal(t, "Test branch", branchContext.Title)
		assert.Equal(t, msg1.ID, *branchContext.BranchPoint)
	})
	
	t.Run("branch from non-existent message fails", func(t *testing.T) {
		_, err := memory.BranchFromMessage("nonexistent", "Test branch")
		assert.Error(t, err)
	})
}

func TestGraphAwareLangChainMemory_GetConversationPath(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{}
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	
	// Add messages
	msg1 := NewUserMessage("Message 1")
	msg2 := NewAssistantMessage("Message 2")
	tree.AddMessage(msg1, tree.RootContextID)
	tree.AddMessage(msg2, tree.RootContextID)
	
	path, err := memory.GetConversationPath()
	require.NoError(t, err)
	assert.Len(t, path, 2)
	assert.Equal(t, msg1.ID, path[0].ID)
	assert.Equal(t, msg2.ID, path[1].ID)
}

func TestGraphAwareLangChainMemory_GetCurrentContextMessages(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{}
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	
	// Add messages to root context
	msg1 := NewUserMessage("Message 1")
	msg2 := NewAssistantMessage("Message 2")
	tree.AddMessage(msg1, tree.RootContextID)
	tree.AddMessage(msg2, tree.RootContextID)
	
	messages := memory.GetCurrentContextMessages()
	assert.Len(t, messages, 2)
	assert.Equal(t, msg1.ID, messages[0].ID)
	assert.Equal(t, msg2.ID, messages[1].ID)
}

func TestGraphAwareLangChainMemory_GetContextBranches(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{}
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	
	// Add message and create branches
	msg1 := NewUserMessage("Test message")
	tree.AddMessage(msg1, tree.RootContextID)
	
	branch1, err := tree.BranchFromMessage(msg1.ID, "Branch 1")
	require.NoError(t, err)
	branch2, err := tree.BranchFromMessage(msg1.ID, "Branch 2")
	require.NoError(t, err)
	
	branches := memory.GetContextBranches()
	assert.Len(t, branches, 2)
	
	branchIDs := []string{branches[0].ID, branches[1].ID}
	assert.Contains(t, branchIDs, branch1.ID)
	assert.Contains(t, branchIDs, branch2.ID)
}

func TestGraphAwareLangChainMemory_GetMessageBranches(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{}
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	
	// Add messages with parent-child relationship
	msg1 := NewUserMessage("Parent message")
	msg2 := NewAssistantMessage("Child message")
	tree.AddMessage(msg1, tree.RootContextID)
	tree.AddMessage(msg2, tree.RootContextID)
	
	branches := memory.GetMessageBranches(msg1.ID)
	assert.Len(t, branches, 1)
	assert.Equal(t, msg2.ID, branches[0].ID)
}

func TestGraphAwareLangChainMemory_mergeAndDeduplicateMessages(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{}
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	
	// Create test messages
	msg1 := NewUserMessage("Message 1")
	msg2 := NewAssistantMessage("Message 2")
	msg3 := NewUserMessage("Message 3")
	
	contextPath := []Message{msg1, msg2}
	relevantMessages := []Message{msg2, msg3} // msg2 is duplicate
	
	merged := memory.mergeAndDeduplicateMessages(contextPath, relevantMessages)
	
	assert.Len(t, merged, 3) // msg1, msg2, msg3 (no duplicates)
	assert.Equal(t, msg1.ID, merged[0].ID)
	assert.Equal(t, msg2.ID, merged[1].ID)
	assert.Equal(t, msg3.ID, merged[2].ID)
}

func TestGraphAwareLangChainMemory_GetMemoryVariables(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{}
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	
	// Add some messages
	msg1 := NewUserMessage("Hello")
	msg2 := NewAssistantMessage("Hi there")
	tree.AddMessage(msg1, tree.RootContextID)
	tree.AddMessage(msg2, tree.RootContextID)
	
	// Create a branch
	branch, err := tree.BranchFromMessage(msg1.ID, "Test branch")
	require.NoError(t, err)
	
	ctx := context.Background()
	vars, err := memory.GetMemoryVariables(ctx)
	require.NoError(t, err)
	
	// Check history is present
	assert.Contains(t, vars, "history")
	
	// Check context info
	assert.Contains(t, vars, "context_info")
	contextInfo := vars["context_info"].(map[string]any)
	assert.Equal(t, tree.RootContextID, contextInfo["context_id"])
	assert.Equal(t, "Main Conversation", contextInfo["context_title"])
	assert.Equal(t, 2, contextInfo["message_count"])
	
	// Check available branches
	assert.Contains(t, vars, "available_branches")
	branches := vars["available_branches"].([]map[string]any)
	assert.Len(t, branches, 1)
	assert.Equal(t, branch.ID, branches[0]["id"])
	assert.Equal(t, "Test branch", branches[0]["title"])
}

func TestGraphAwareLangChainMemory_formatMessagesForPrompt(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{}
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	
	// Create test messages and convert to LangChain format
	msg1 := NewUserMessage("Hello")
	msg2 := NewAssistantMessage("Hi there")
	msg3 := NewSystemMessage("System prompt")
	
	messages := []Message{msg1, msg2, msg3}
	langchainMessages := ConvertToLangChainMessages(messages)
	
	formatted := memory.formatMessagesForPrompt(langchainMessages)
	
	assert.Contains(t, formatted, "Human: Hello")
	assert.Contains(t, formatted, "AI: Hi there")
	assert.Contains(t, formatted, "System: System prompt")
}

func TestGraphAwareLangChainMemory_GetCurrentContext(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{}
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	
	currentContext := memory.GetCurrentContext()
	require.NotNil(t, currentContext)
	assert.Equal(t, tree.RootContextID, currentContext.ID)
	assert.Equal(t, "Main Conversation", currentContext.Title)
}

func TestGraphAwareLangChainMemory_GetContextTree(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{}
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	
	retrievedTree := memory.GetContextTree()
	assert.Equal(t, tree, retrievedTree)
}

func TestGraphAwareMemoryAdapter(t *testing.T) {
	tree := NewContextTree()
	vectorManager := &VectorContextManager{}
	config := DefaultGraphAwareMemoryConfig()
	
	memory, err := NewGraphAwareLangChainMemory(tree, vectorManager, config)
	require.NoError(t, err)
	
	adapter := &GraphAwareMemoryAdapter{memory}
	ctx := context.Background()
	
	t.Run("GetMemoryKey", func(t *testing.T) {
		key := adapter.GetMemoryKey(ctx)
		assert.Equal(t, "history", key)
	})
	
	t.Run("MemoryVariables", func(t *testing.T) {
		vars := adapter.MemoryVariables(ctx)
		expected := []string{"history", "context_info", "available_branches"}
		assert.Equal(t, expected, vars)
	})
	
	t.Run("LoadMemoryVariables", func(t *testing.T) {
		// Add some messages first
		msg1 := NewUserMessage("Hello")
		tree.AddMessage(msg1, tree.RootContextID)
		
		vars, err := adapter.LoadMemoryVariables(ctx, map[string]any{})
		require.NoError(t, err)
		assert.Contains(t, vars, "history")
		assert.Contains(t, vars, "context_info")
	})
}

func TestNewGraphAwareLangChainMemoryFromGlobalConfig(t *testing.T) {
	memory, err := NewGraphAwareLangChainMemoryFromGlobalConfig()
	assert.Nil(t, memory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "global config integration not yet implemented")
}

func TestNewUserMessageInContext(t *testing.T) {
	contextID := "test-context-123"
	content := "Hello world"
	
	msg := NewUserMessageInContext(content, contextID)
	
	assert.Equal(t, content, msg.Content)
	assert.Equal(t, contextID, msg.ContextID)
	assert.Equal(t, "user", msg.Role)
}

func TestNewAssistantMessageInContext(t *testing.T) {
	contextID := "test-context-123"
	content := "Hello back"
	
	msg := NewAssistantMessageInContext(content, contextID)
	
	assert.Equal(t, content, msg.Content)
	assert.Equal(t, contextID, msg.ContextID)
	assert.Equal(t, "assistant", msg.Role)
}

func TestNewBranchMessage(t *testing.T) {
	content := "Branch message"
	sourceMessageID := "source-123"
	newContextID := "new-context-456"
	role := "user"
	
	msg := NewBranchMessage(content, sourceMessageID, newContextID, role)
	
	assert.Equal(t, content, msg.Content)
	assert.Equal(t, newContextID, msg.ContextID)
	assert.Equal(t, role, msg.Role)
	assert.Equal(t, sourceMessageID, *msg.ParentID)
	assert.NotEmpty(t, msg.ID)
	assert.False(t, msg.Timestamp.IsZero())
	require.NotNil(t, msg.Metadata)
	assert.Equal(t, MessageSourceFinal, msg.Metadata.Source)
	assert.Equal(t, 0, msg.Metadata.Depth)
}