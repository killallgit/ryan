package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/vectorstore"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	// Create vector store with embedder
	embedderConfig := vectorstore.EmbedderConfig{
		Provider: "mock", // Use "ollama" or "openai" for real embeddings
		Model:    "nomic-embed-text",
		BaseURL:  "http://localhost:11434",
	}

	embedder, err := vectorstore.CreateEmbedder(embedderConfig)
	if err != nil {
		log.Fatal("Failed to create embedder:", err)
	}

	// Create persistent vector store
	persistDir := "/tmp/ryan-langchain-memory"
	os.MkdirAll(persistDir, 0755)

	store, err := vectorstore.NewChromemStore(embedder, persistDir, true)
	if err != nil {
		log.Fatal("Failed to create vector store:", err)
	}
	defer store.Close()

	// Create vector memory configuration
	memoryConfig := chat.VectorMemoryConfig{
		CollectionName: "chat_sessions",
		MaxRetrieved:   5,
		ScoreThreshold: 0.7,
	}

	// Create vector memory
	vectorMemory, err := chat.NewLangChainVectorMemory(store, memoryConfig)
	if err != nil {
		log.Fatal("Failed to create vector memory:", err)
	}

	// Create memory adapter for LangChain
	memoryAdapter := &chat.VectorMemoryAdapter{LangChainVectorMemory: vectorMemory}

	// Initialize LLM (using OpenAI in this example)
	llm, err := openai.New()
	if err != nil {
		log.Fatal("Failed to create LLM:", err)
	}

	// Create conversation chain with vector memory
	chain := chains.NewConversationChain(llm, memoryAdapter)

	ctx := context.Background()

	// Simulate a conversation
	fmt.Println("=== LangChain with Vector Memory Demo ===\n")

	// First conversation
	queries := []string{
		"Hi! I'm learning about machine learning. Can you help?",
		"What are the main types of machine learning?",
		"Can you explain supervised learning in more detail?",
		"What about unsupervised learning?",
	}

	for _, query := range queries {
		fmt.Printf("User: %s\n", query)

		// Run the chain
		result, err := chains.Run(ctx, chain, query)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		fmt.Printf("Assistant: %s\n\n", result)
	}

	// Now ask a question that requires context from earlier in the conversation
	fmt.Println("\n=== Testing Semantic Retrieval ===\n")

	contextQuery := "Can you summarize what we discussed about the different types of ML?"
	fmt.Printf("User: %s\n", contextQuery)

	// The vector memory should retrieve relevant context
	result, err := chains.Run(ctx, chain, contextQuery)
	if err != nil {
		log.Fatal("Error:", err)
	}

	fmt.Printf("Assistant: %s\n\n", result)

	// Demonstrate searching conversation history
	fmt.Println("\n=== Direct Memory Search ===\n")

	searchQuery := "supervised learning algorithms"
	fmt.Printf("Searching for: %s\n", searchQuery)

	relevantMessages, err := vectorMemory.GetRelevantMessages(ctx, searchQuery)
	if err != nil {
		log.Fatal("Failed to search memory:", err)
	}

	fmt.Printf("Found %d relevant messages:\n", len(relevantMessages))
	for i, msg := range relevantMessages {
		fmt.Printf("%d. [%s] %s\n", i+1, msg.Role, truncate(msg.Content, 80))
	}

	// Show persistence
	fmt.Println("\n=== Testing Persistence ===")
	fmt.Println("Vector memory is automatically persisted to:", persistDir)
	fmt.Println("Restart the program to see conversations are retained!")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}