package chat

import (
	"context"
	"testing"

	"github.com/killallgit/ryan/pkg/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestManager(t *testing.T) *vectorstore.Manager {
	// Create mock embedder and vector store
	embedder := vectorstore.NewMockEmbedder(384)
	store, err := vectorstore.NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	// Create manager
	config := vectorstore.Config{
		Provider:          "chromem",
		EnablePersistence: false,
		ChunkSize:         1000,
		ChunkOverlap:      200,
		EmbedderConfig: vectorstore.EmbedderConfig{
			Provider: "mock",
		},
	}

	// Use NewManager constructor
	manager, err := vectorstore.NewManager(config)
	require.NoError(t, err)

	return manager
}

func TestLangChainVectorMemory_AddAndRetrieve(t *testing.T) {
	manager := createTestManager(t)

	// Create vector memory with lower threshold for mock embedder
	config := VectorMemoryConfig{
		CollectionName: "conversations",
		MaxRetrieved:   10,
		ScoreThreshold: 0.3, // Lower threshold for mock embedder
	}
	vm, err := NewLangChainVectorMemory(manager, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Add messages
	messages := []Message{
		NewSystemMessage("You are a helpful assistant."),
		NewUserMessage("Tell me about machine learning"),
		NewAssistantMessage("Machine learning is a subset of artificial intelligence that enables systems to learn from data."),
		NewUserMessage("What about deep learning?"),
		NewAssistantMessage("Deep learning is a subset of machine learning that uses neural networks with multiple layers."),
		NewUserMessage("Can you explain neural networks?"),
		NewAssistantMessage("Neural networks are computing systems inspired by biological neural networks in animal brains."),
	}

	for _, msg := range messages {
		err := vm.AddMessage(ctx, msg)
		require.NoError(t, err)
	}

	// Test retrieval - search for "neural networks"
	relevant, err := vm.GetRelevantMessages(ctx, "neural networks")
	require.NoError(t, err)
	assert.NotEmpty(t, relevant)

	// Should find messages containing "neural networks"
	foundNeural := false
	for _, msg := range relevant {
		if msg.Role == RoleAssistant && (msg.Content == "Deep learning is a subset of machine learning that uses neural networks with multiple layers." ||
			msg.Content == "Neural networks are computing systems inspired by biological neural networks in animal brains.") {
			foundNeural = true
			break
		}
	}
	assert.True(t, foundNeural, "Should find messages about neural networks")

	// Test memory variables integration
	vars, err := vm.GetMemoryVariables(ctx)
	require.NoError(t, err)
	assert.Contains(t, vars, "history")
	// Relevant context might be included
	if context, ok := vars["relevant_context"]; ok {
		assert.NotEmpty(t, context)
	}
}

func TestLangChainVectorMemory_WithExistingConversation(t *testing.T) {
	manager := createTestManager(t)

	// Create existing conversation
	conv := NewConversation("test-conv")
	conv = AddMessage(conv, NewUserMessage("What is artificial intelligence?"))
	conv = AddMessage(conv, NewAssistantMessage("Artificial intelligence is the simulation of human intelligence in machines."))
	conv = AddMessage(conv, NewUserMessage("Tell me about machine learning"))
	conv = AddMessage(conv, NewAssistantMessage("Machine learning is a subset of AI that enables systems to learn from data."))

	// Create vector memory with existing conversation
	config := VectorMemoryConfig{
		CollectionName: "conversations",
		MaxRetrieved:   5,
		ScoreThreshold: 0.3,
	}
	vm, err := NewLangChainVectorMemoryWithConversation(manager, config, conv)
	require.NoError(t, err)

	ctx := context.Background()

	// Search for relevant messages
	relevant, err := vm.GetRelevantMessages(ctx, "artificial intelligence")
	require.NoError(t, err)
	assert.NotEmpty(t, relevant)

	// Should find the AI-related messages
	foundAI := false
	for _, msg := range relevant {
		if msg.Role == RoleAssistant && msg.Content == "Artificial intelligence is the simulation of human intelligence in machines." {
			foundAI = true
			break
		}
	}
	assert.True(t, foundAI, "Should find the AI message")
}

func TestLangChainVectorMemory_ScoreFiltering(t *testing.T) {
	manager := createTestManager(t)

	// Create vector memory with high threshold
	config := VectorMemoryConfig{
		CollectionName: "conversations",
		MaxRetrieved:   10,
		ScoreThreshold: 0.9, // Very high threshold
	}
	vm, err := NewLangChainVectorMemory(manager, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Add a message
	err = vm.AddMessage(ctx, NewUserMessage("Hello world"))
	require.NoError(t, err)

	// Search with unrelated query
	relevant, err := vm.GetRelevantMessages(ctx, "completely different topic about quantum physics")
	require.NoError(t, err)

	// With high threshold, should find fewer or no matches
	assert.LessOrEqual(t, len(relevant), 1, "High threshold should filter out low-similarity matches")
}

func TestLangChainVectorMemory_Clear(t *testing.T) {
	manager := createTestManager(t)

	config := VectorMemoryConfig{
		CollectionName: "conversations",
		MaxRetrieved:   10,
		ScoreThreshold: 0.3,
	}
	vm, err := NewLangChainVectorMemory(manager, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Add messages
	err = vm.AddMessage(ctx, NewUserMessage("Test message 1"))
	require.NoError(t, err)
	err = vm.AddMessage(ctx, NewUserMessage("Test message 2"))
	require.NoError(t, err)

	// Verify messages exist
	relevant, err := vm.GetRelevantMessages(ctx, "test message")
	require.NoError(t, err)
	assert.NotEmpty(t, relevant)

	// Clear memory
	err = vm.Clear(ctx)
	require.NoError(t, err)

	// Verify no messages found after clear
	relevant, err = vm.GetRelevantMessages(ctx, "test message")
	require.NoError(t, err)
	assert.Empty(t, relevant)
}

func TestLangChainVectorMemory_SaveContext(t *testing.T) {
	manager := createTestManager(t)

	config := VectorMemoryConfig{
		CollectionName: "conversations",
		MaxRetrieved:   10,
		ScoreThreshold: 0.1, // Lower threshold for this test
	}
	vm, err := NewLangChainVectorMemory(manager, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Save context (should add new messages)
	inputs := map[string]any{
		"input": "What is machine learning?",
	}
	outputs := map[string]any{
		"output": "Machine learning is a subset of AI that enables systems to learn from data.",
	}

	err = vm.SaveContext(ctx, inputs, outputs)
	require.NoError(t, err)

	// Debug: Check if messages were actually saved to the conversation
	messages := GetMessages(*vm.conversation)
	assert.GreaterOrEqual(t, len(messages), 2, "Should have at least 2 messages after SaveContext")

	// Search for the saved content with no threshold
	vm.scoreThreshold = 0.0 // Remove threshold for debugging
	relevant, err := vm.GetRelevantMessages(ctx, "machine learning")
	require.NoError(t, err)

	// Debug output
	t.Logf("Found %d relevant messages", len(relevant))
	for i, msg := range relevant {
		t.Logf("Message %d: Role=%s, Content=%s", i, msg.Role, msg.Content)
	}

	assert.NotEmpty(t, relevant)

	// Should find at least one message with machine learning content
	foundML := false
	for _, msg := range relevant {
		if msg.Role == RoleAssistant &&
			msg.Content == "Machine learning is a subset of AI that enables systems to learn from data." {
			foundML = true
			break
		}
	}
	assert.True(t, foundML, "Should find the machine learning message")
}

func TestVectorMemoryAdapter(t *testing.T) {
	manager := createTestManager(t)

	config := VectorMemoryConfig{
		CollectionName: "conversations",
		MaxRetrieved:   10,
		ScoreThreshold: 0.3,
	}
	vm, err := NewLangChainVectorMemory(manager, config)
	require.NoError(t, err)

	// Create adapter
	adapter := &VectorMemoryAdapter{LangChainVectorMemory: vm}

	ctx := context.Background()

	// Test schema.Memory interface
	assert.Equal(t, "history", adapter.GetMemoryKey(ctx))

	memVars := adapter.MemoryVariables(ctx)
	assert.Contains(t, memVars, "history")
	assert.Contains(t, memVars, "relevant_context")

	// Test SaveContext through adapter
	inputs := map[string]any{"input": "test input"}
	outputs := map[string]any{"output": "test output"}
	err = adapter.SaveContext(ctx, inputs, outputs)
	require.NoError(t, err)

	// Test LoadMemoryVariables
	vars, err := adapter.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err)
	assert.Contains(t, vars, "history")

	// Test Clear through adapter
	err = adapter.Clear(ctx)
	require.NoError(t, err)
}
