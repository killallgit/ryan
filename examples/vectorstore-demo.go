package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/killallgit/ryan/pkg/vectorstore"
)

func main() {
	// Create a temporary directory for persistence
	tempDir := "/tmp/ryan-vectorstore-demo"
	os.MkdirAll(tempDir, 0755)

	// Configure the vector store
	config := vectorstore.Config{
		Provider:          "chromem",
		PersistenceDir:    tempDir,
		EnablePersistence: true,
		Collections: []vectorstore.CollectionConfig{
			{
				Name: "knowledge-base",
				Metadata: map[string]interface{}{
					"description": "Demo knowledge base",
				},
			},
		},
		EmbedderConfig: vectorstore.EmbedderConfig{
			Provider: "mock", // Use mock for demo, switch to "ollama" for real use
		},
	}

	// Create the vector store manager
	manager, err := vectorstore.NewManager(config)
	if err != nil {
		log.Fatal("Failed to create vector store manager:", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Add some documents
	fmt.Println("Adding documents to knowledge base...")
	documents := []vectorstore.Document{
		{
			ID:      "doc1",
			Content: "The Go programming language is an open source project to make programmers more productive.",
			Metadata: map[string]interface{}{
				"type":   "programming",
				"source": "golang.org",
			},
		},
		{
			ID:      "doc2",
			Content: "Go is expressive, concise, clean, and efficient. Its concurrency mechanisms make it easy to write programs.",
			Metadata: map[string]interface{}{
				"type":   "programming",
				"source": "golang.org",
			},
		},
		{
			ID:      "doc3",
			Content: "Vector databases enable semantic search by converting text into high-dimensional vectors.",
			Metadata: map[string]interface{}{
				"type":   "database",
				"source": "technical",
			},
		},
		{
			ID:      "doc4",
			Content: "LangChain is a framework for developing applications powered by language models.",
			Metadata: map[string]interface{}{
				"type":   "ai",
				"source": "langchain",
			},
		},
	}

	err = manager.IndexDocuments(ctx, "knowledge-base", documents)
	if err != nil {
		log.Fatal("Failed to index documents:", err)
	}

	// Perform searches
	queries := []string{
		"Go programming concurrency",
		"vector search and embeddings",
		"language model applications",
	}

	for _, query := range queries {
		fmt.Printf("\nSearching for: %q\n", query)
		results, err := manager.Search(ctx, "knowledge-base", query, 2)
		if err != nil {
			log.Printf("Search failed: %v", err)
			continue
		}

		for i, result := range results {
			fmt.Printf("%d. Score: %.3f\n", i+1, result.Score)
			fmt.Printf("   Content: %s\n", result.Document.Content)
			fmt.Printf("   Type: %v\n", result.Document.Metadata["type"])
		}
	}

	// Show collection info
	info, err := manager.GetCollectionInfo("knowledge-base")
	if err != nil {
		log.Printf("Failed to get collection info: %v", err)
	} else {
		fmt.Printf("\nCollection '%s' contains %d documents\n", info.Name, info.DocumentCount)
	}

	// Demonstrate persistence
	fmt.Println("\nClosing and reopening to test persistence...")
	manager.Close()

	// Create new manager with same config
	manager2, err := vectorstore.NewManager(config)
	if err != nil {
		log.Fatal("Failed to recreate vector store manager:", err)
	}
	defer manager2.Close()

	// Search again to verify persistence
	fmt.Println("\nSearching after reload...")
	results, err := manager2.Search(ctx, "knowledge-base", "Go programming", 1)
	if err != nil {
		log.Fatal("Search after reload failed:", err)
	}

	if len(results) > 0 {
		fmt.Printf("Found: %s\n", results[0].Document.Content)
		fmt.Println("✓ Persistence working!")
	} else {
		fmt.Println("✗ No results found after reload")
	}
}