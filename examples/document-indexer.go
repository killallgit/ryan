package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/killallgit/ryan/pkg/vectorstore"
)

func main() {
	// Command line flags
	var (
		indexCmd    = flag.NewFlagSet("index", flag.ExitOnError)
		searchCmd   = flag.NewFlagSet("search", flag.ExitOnError)
		
		// Index flags
		indexPath     = indexCmd.String("path", "", "Path to file or directory to index")
		indexPatterns = indexCmd.String("patterns", "", "Comma-separated file patterns (e.g., *.go,*.md)")
		persistDir    = indexCmd.String("persist", "/tmp/ryan-doc-index", "Persistence directory")
		chunkSize     = indexCmd.Int("chunk-size", 1000, "Chunk size in characters")
		chunkOverlap  = indexCmd.Int("chunk-overlap", 200, "Chunk overlap in characters")
		
		// Search flags
		searchQuery   = searchCmd.String("query", "", "Search query")
		searchK       = searchCmd.Int("k", 5, "Number of results to return")
		searchPersist = searchCmd.String("persist", "/tmp/ryan-doc-index", "Persistence directory")
	)

	// Parse command
	if len(os.Args) < 2 {
		fmt.Println("Usage: document-indexer <command> [arguments]")
		fmt.Println("\nCommands:")
		fmt.Println("  index   - Index documents")
		fmt.Println("  search  - Search indexed documents")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "index":
		indexCmd.Parse(os.Args[2:])
		if *indexPath == "" {
			fmt.Println("Error: -path is required")
			indexCmd.PrintDefaults()
			os.Exit(1)
		}
		runIndex(*indexPath, *indexPatterns, *persistDir, *chunkSize, *chunkOverlap)
		
	case "search":
		searchCmd.Parse(os.Args[2:])
		if *searchQuery == "" {
			fmt.Println("Error: -query is required")
			searchCmd.PrintDefaults()
			os.Exit(1)
		}
		runSearch(*searchQuery, *searchK, *searchPersist)
		
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runIndex(path, patterns, persistDir string, chunkSize, chunkOverlap int) {
	fmt.Printf("Indexing documents from: %s\n", path)
	
	// Create embedder
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
	os.MkdirAll(persistDir, 0755)
	store, err := vectorstore.NewChromemStore(embedder, persistDir, true)
	if err != nil {
		log.Fatal("Failed to create vector store:", err)
	}
	defer store.Close()

	// Create indexer
	indexerConfig := vectorstore.IndexerConfig{
		CollectionName: "documents",
		ChunkSize:      chunkSize,
		ChunkOverlap:   chunkOverlap,
	}

	indexer, err := vectorstore.NewDocumentIndexer(store, indexerConfig)
	if err != nil {
		log.Fatal("Failed to create indexer:", err)
	}

	ctx := context.Background()

	// Check if path is file or directory
	info, err := os.Stat(path)
	if err != nil {
		log.Fatal("Failed to stat path:", err)
	}

	if info.IsDir() {
		// Index directory
		var patternList []string
		if patterns != "" {
			patternList = strings.Split(patterns, ",")
			fmt.Printf("Using patterns: %v\n", patternList)
		}
		
		err = indexer.IndexDirectory(ctx, path, patternList)
		if err != nil {
			log.Fatal("Failed to index directory:", err)
		}
		fmt.Println("Directory indexed successfully!")
		
	} else {
		// Index single file
		err = indexer.IndexFile(ctx, path)
		if err != nil {
			log.Fatal("Failed to index file:", err)
		}
		fmt.Printf("File %s indexed successfully!\n", path)
	}

	fmt.Printf("\nDocuments have been indexed to: %s\n", persistDir)
}

func runSearch(query string, k int, persistDir string) {
	fmt.Printf("Searching for: %s\n\n", query)

	// Create embedder
	embedderConfig := vectorstore.EmbedderConfig{
		Provider: "mock", // Use same provider as indexing
		Model:    "nomic-embed-text",
		BaseURL:  "http://localhost:11434",
	}

	embedder, err := vectorstore.CreateEmbedder(embedderConfig)
	if err != nil {
		log.Fatal("Failed to create embedder:", err)
	}

	// Open existing vector store
	store, err := vectorstore.NewChromemStore(embedder, persistDir, true)
	if err != nil {
		log.Fatal("Failed to open vector store:", err)
	}
	defer store.Close()

	// Create indexer
	indexerConfig := vectorstore.DefaultIndexerConfig()
	indexer, err := vectorstore.NewDocumentIndexer(store, indexerConfig)
	if err != nil {
		log.Fatal("Failed to create indexer:", err)
	}

	ctx := context.Background()

	// Search documents
	docs, err := indexer.SearchDocuments(ctx, query, k)
	if err != nil {
		log.Fatal("Failed to search documents:", err)
	}

	if len(docs) == 0 {
		fmt.Println("No documents found.")
		return
	}

	// Display results
	fmt.Printf("Found %d documents:\n\n", len(docs))
	for i, doc := range docs {
		fmt.Printf("--- Result %d ---\n", i+1)
		fmt.Printf("Source: %s\n", doc.Metadata["source"])
		
		if chunkIdx, ok := doc.Metadata["chunk_index"]; ok {
			fmt.Printf("Chunk: %v of %v\n", chunkIdx, doc.Metadata["chunk_total"])
		}
		
		// Truncate content for display
		content := doc.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		fmt.Printf("Content:\n%s\n\n", content)
	}
}

// Example usage:
// go run document-indexer.go index -path ./pkg/vectorstore -patterns "*.go" -chunk-size 500
// go run document-indexer.go search -query "vector store embedding"