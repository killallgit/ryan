package chat

import (
	"context"
	"testing"

	"github.com/killallgit/ryan/pkg/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLangChainVectorMemory_AddAndRetrieve(t *testing.T) {
	// Create mock embedder and vector store
	embedder := vectorstore.NewMockEmbedder(384)
	store, err := vectorstore.NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	// Create vector memory with lower threshold for mock embedder
	config := VectorMemoryConfig{
		CollectionName: "conversations",
		MaxRetrieved:   10,
		ScoreThreshold: 0.3, // Lower threshold for mock embedder
	}
	vm, err := NewLangChainVectorMemory(store, config)
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

	// Should retrieve messages about neural networks
	foundNeuralNetwork := false
	for _, msg := range relevant {
		if msg.Role == RoleAssistant && contains(msg.Content, "neural networks") {
			foundNeuralNetwork = true
			break
		}
	}
	assert.True(t, foundNeuralNetwork, "Should find message about neural networks")

	// Test retrieval - search for "machine learning"
	relevant, err = vm.GetRelevantMessages(ctx, "machine learning AI")
	require.NoError(t, err)
	assert.NotEmpty(t, relevant)

	// Should retrieve messages about machine learning
	foundML := false
	for _, msg := range relevant {
		if msg.Role == RoleAssistant && contains(msg.Content, "machine learning") {
			foundML = true
			break
		}
	}
	assert.True(t, foundML, "Should find message about machine learning")
}

func TestLangChainVectorMemory_WithExistingConversation(t *testing.T) {
	// Create mock embedder and vector store
	embedder := vectorstore.NewMockEmbedder(384)
	store, err := vectorstore.NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	// Create existing conversation
	conv := Conversation{
		Messages: []Message{
			NewUserMessage("What is Go programming?"),
			NewAssistantMessage("Go is a statically typed, compiled programming language designed at Google."),
			NewUserMessage("What makes Go special?"),
			NewAssistantMessage("Go is known for its simplicity, efficient concurrency with goroutines, and fast compilation."),
		},
	}

	// Create vector memory with existing conversation
	config := VectorMemoryConfig{
		CollectionName: "conversations",
		MaxRetrieved:   10,
		ScoreThreshold: 0.3, // Lower threshold for mock embedder
	}
	vm, err := NewLangChainVectorMemoryWithConversation(store, config, conv)
	require.NoError(t, err)

	ctx := context.Background()

	// Search for Go programming concepts
	relevant, err := vm.GetRelevantMessages(ctx, "goroutines concurrency")
	require.NoError(t, err)
	assert.NotEmpty(t, relevant)

	// Should find the message about Go's features
	foundGoroutines := false
	for _, msg := range relevant {
		if msg.Role == RoleAssistant && contains(msg.Content, "goroutines") {
			foundGoroutines = true
			break
		}
	}
	assert.True(t, foundGoroutines, "Should find message about goroutines")
}

func TestLangChainVectorMemory_Clear(t *testing.T) {
	// Create mock embedder and vector store
	embedder := vectorstore.NewMockEmbedder(384)
	store, err := vectorstore.NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	// Create vector memory
	config := VectorMemoryConfig{
		CollectionName: "conversations",
		MaxRetrieved:   10,
		ScoreThreshold: 0.3, // Lower threshold for mock embedder
	}
	vm, err := NewLangChainVectorMemory(store, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Add messages
	err = vm.AddMessage(ctx, NewUserMessage("Hello"))
	require.NoError(t, err)
	err = vm.AddMessage(ctx, NewAssistantMessage("Hi there!"))
	require.NoError(t, err)

	// Verify messages exist
	assert.Len(t, vm.GetConversation().Messages, 2)

	// Clear memory
	err = vm.Clear(ctx)
	require.NoError(t, err)

	// Verify messages are cleared
	assert.Empty(t, vm.GetConversation().Messages)

	// Verify vector store is cleared
	relevant, err := vm.GetRelevantMessages(ctx, "Hello")
	require.NoError(t, err)
	assert.Empty(t, relevant)
}

func TestLangChainVectorMemory_GetMemoryVariables(t *testing.T) {
	// Create mock embedder and vector store
	embedder := vectorstore.NewMockEmbedder(384)
	store, err := vectorstore.NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	// Create vector memory
	config := VectorMemoryConfig{
		CollectionName: "conversations",
		MaxRetrieved:   10,
		ScoreThreshold: 0.3, // Lower threshold for mock embedder
	}
	vm, err := NewLangChainVectorMemory(store, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Add conversation about programming
	messages := []Message{
		NewUserMessage("Tell me about Python"),
		NewAssistantMessage("Python is a high-level, interpreted programming language known for its simplicity."),
		NewUserMessage("What about Java?"),
		NewAssistantMessage("Java is a class-based, object-oriented programming language designed for portability."),
		NewUserMessage("Which language is better for machine learning?"),
	}

	for _, msg := range messages {
		err := vm.AddMessage(ctx, msg)
		require.NoError(t, err)
	}

	// Get memory variables
	vars, err := vm.GetMemoryVariables(ctx)
	require.NoError(t, err)

	// Should have history
	assert.Contains(t, vars, "history")

	// Should have relevant context for the last user query about ML
	assert.Contains(t, vars, "relevant_context")
	relevantContext, ok := vars["relevant_context"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, relevantContext)

	// Context should include Python info (more relevant to ML)
	assert.Contains(t, relevantContext, "Python")
}

func TestVectorMemoryAdapter(t *testing.T) {
	// Create mock embedder and vector store
	embedder := vectorstore.NewMockEmbedder(384)
	store, err := vectorstore.NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	// Create vector memory
	config := VectorMemoryConfig{
		CollectionName: "conversations",
		MaxRetrieved:   10,
		ScoreThreshold: 0.3, // Lower threshold for mock embedder
	}
	vm, err := NewLangChainVectorMemory(store, config)
	require.NoError(t, err)

	// Create adapter
	adapter := &VectorMemoryAdapter{LangChainVectorMemory: vm}

	ctx := context.Background()

	// Test memory key
	assert.Equal(t, "history", adapter.GetMemoryKey(ctx))

	// Test memory variables
	vars := adapter.MemoryVariables(ctx)
	assert.Contains(t, vars, "history")
	assert.Contains(t, vars, "relevant_context")

	// Test save context
	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}
	err = adapter.SaveContext(ctx, inputs, outputs)
	require.NoError(t, err)

	// Test load memory variables
	loadedVars, err := adapter.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err)
	assert.Contains(t, loadedVars, "history")
}

func TestLangChainVectorMemory_ScoreThreshold(t *testing.T) {
	// Create mock embedder and vector store
	embedder := vectorstore.NewMockEmbedder(384)
	store, err := vectorstore.NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	// Create vector memory with high score threshold
	config := VectorMemoryConfig{
		CollectionName: "test_threshold",
		MaxRetrieved:   5,
		ScoreThreshold: 0.9, // High threshold
	}
	vm, err := NewLangChainVectorMemory(store, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Add diverse messages
	messages := []Message{
		NewUserMessage("I love cats"),
		NewAssistantMessage("Cats are wonderful pets!"),
		NewUserMessage("Tell me about quantum physics"),
		NewAssistantMessage("Quantum physics studies the behavior of matter and energy at the smallest scales."),
	}

	for _, msg := range messages {
		err := vm.AddMessage(ctx, msg)
		require.NoError(t, err)
	}

	// Search for something unrelated with high threshold
	relevant, err := vm.GetRelevantMessages(ctx, "dogs and puppies")
	require.NoError(t, err)

	// With high threshold, should get few or no results for unrelated query
	assert.LessOrEqual(t, len(relevant), 1, "High threshold should filter out low-similarity results")
}

// Helper function
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
