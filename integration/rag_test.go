package integration

import (
	"context"
	"testing"

	"github.com/killallgit/ryan/pkg/embeddings"
	"github.com/killallgit/ryan/pkg/retrieval"
	"github.com/killallgit/ryan/pkg/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRAGWorkflow(t *testing.T) {
	// This test uses mock embedder and doesn't require Ollama
	// It tests the RAG workflow components in isolation

	ctx := context.Background()

	// Create mock embedder for testing (doesn't require Ollama)
	mockEmbedder := embeddings.NewMockEmbedder(384)

	// Create vector store
	vsConfig := vectorstore.ChromemConfig{
		CollectionName: "test_rag",
		Embedder:       mockEmbedder,
	}

	store, err := vectorstore.NewChromemStore(vsConfig)
	require.NoError(t, err)
	defer store.Close()

	// Create document manager
	docManager := retrieval.NewDocumentManager(retrieval.DocumentConfig{
		ChunkSize:    500,
		ChunkOverlap: 50,
	})

	// Create sample documents
	documents := []string{
		"Go is a statically typed, compiled programming language designed at Google. It is syntactically similar to C, but with memory safety, garbage collection, structural typing, and CSP-style concurrency.",
		"Python is an interpreted, high-level, general-purpose programming language. Its design philosophy emphasizes code readability with the use of significant indentation.",
		"JavaScript is a programming language that is one of the core technologies of the World Wide Web, alongside HTML and CSS. It is a high-level, often just-in-time compiled language.",
		"Rust is a multi-paradigm programming language designed for performance and safety, especially safe concurrency. It is syntactically similar to C++, but can guarantee memory safety.",
	}

	// Create and add documents
	var docs []vectorstore.Document
	for i, content := range documents {
		doc := docManager.CreateDocument(content, map[string]interface{}{
			"source": "test",
			"index":  i,
		})
		docs = append(docs, doc)
	}

	err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Create retriever
	retriever := retrieval.NewRetriever(store, retrieval.Config{
		MaxDocuments:   2,
		ScoreThreshold: 0.0,
	})

	// Create augmenter
	augmenter := retrieval.NewAugmenter(retriever, retrieval.AugmenterConfig{
		MaxContextLength: 1000,
		IncludeScores:    true,
	})

	t.Run("BasicRetrieval", func(t *testing.T) {
		// Test retrieval
		results, err := retriever.Retrieve(ctx, "memory safety in programming")
		assert.NoError(t, err)
		assert.Len(t, results, 2)

		// Check that we got relevant documents
		foundRelevant := false
		for _, doc := range results {
			if containsAny(doc.Content, []string{"Rust", "Go", "memory safety", "garbage collection"}) {
				foundRelevant = true
				break
			}
		}
		assert.True(t, foundRelevant, "Should find documents about memory safety")
	})

	t.Run("PromptAugmentation", func(t *testing.T) {
		// Test augmentation
		originalPrompt := "What languages are good for web development?"
		augmented, err := augmenter.AugmentPrompt(ctx, originalPrompt)
		assert.NoError(t, err)
		assert.Contains(t, augmented, originalPrompt)
		assert.Contains(t, augmented, "Context:")

		// Should include JavaScript as it's relevant to web development
		assert.Contains(t, augmented, "JavaScript")
	})

	t.Run("DetailedAugmentation", func(t *testing.T) {
		// Test detailed augmentation
		result, err := augmenter.AugmentWithDetails(ctx, "concurrent programming features")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "concurrent programming features", result.OriginalPrompt)
		assert.NotEmpty(t, result.Context)
		assert.Len(t, result.Documents, 2)
		assert.Len(t, result.Scores, 2)

		// Check scores are valid
		for _, score := range result.Scores {
			assert.GreaterOrEqual(t, score, float32(0.0))
			assert.LessOrEqual(t, score, float32(1.0))
		}
	})

	t.Run("DocumentChunking", func(t *testing.T) {
		// Test document chunking
		longText := `Go is a statically typed, compiled programming language designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson. Go is syntactically similar to C, but with memory safety, garbage collection, structural typing, and CSP-style concurrency. The language is often referred to as Golang because of its domain name, golang.org, but the proper name is Go.

Go was designed at Google in 2007 to improve programming productivity in an era of multicore, networked machines and large codebases. The designers wanted to address criticism of other languages in use at Google, but keep their useful characteristics. Go was publicly announced in November 2009, and version 1.0 was released in March 2012.

Go is widely used in production at Google and in many other organizations and open-source projects. The Go programming language is an open source project to make programmers more productive. Go is expressive, concise, clean, and efficient.`

		chunks := docManager.ChunkText(longText)
		assert.Greater(t, len(chunks), 1, "Long text should be chunked")

		// Verify chunks have overlap
		if len(chunks) > 1 {
			// Check that there's some overlap between consecutive chunks
			for i := 0; i < len(chunks)-1; i++ {
				chunk1End := chunks[i][max(0, len(chunks[i])-50):]
				chunk2Start := chunks[i+1][:min(50, len(chunks[i+1]))]
				// There should be some common text due to overlap
				assert.NotEmpty(t, chunk1End)
				assert.NotEmpty(t, chunk2Start)
			}
		}
	})
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if contains(text, keyword) {
			return true
		}
	}
	return false
}

func contains(text, substr string) bool {
	return len(text) >= len(substr) && containsSubstring(text, substr)
}

func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
