package retrieval

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/vectorstore"
	"github.com/tmc/langchaingo/schema"
)

// Retriever provides document retrieval capabilities
type Retriever struct {
	vectorStore vectorstore.VectorStore
	config      Config
}

// Config contains configuration for the retriever
type Config struct {
	// MaxDocuments is the maximum number of documents to retrieve
	MaxDocuments int

	// ScoreThreshold is the minimum similarity score required
	ScoreThreshold float32

	// MaxContextLength is the maximum context length for augmentation
	MaxContextLength int

	// IncludeMetadata determines if metadata should be included
	IncludeMetadata bool
}

// NewRetriever creates a new retriever
func NewRetriever(store vectorstore.VectorStore, config Config) *Retriever {
	if config.MaxDocuments == 0 {
		config.MaxDocuments = 4
	}
	return &Retriever{
		vectorStore: store,
		config:      config,
	}
}

// Retrieve retrieves relevant documents for a query
func (r *Retriever) Retrieve(ctx context.Context, query string) ([]vectorstore.Document, error) {
	if r.vectorStore == nil {
		return nil, fmt.Errorf("vector store not initialized")
	}

	results, err := r.vectorStore.SimilaritySearchWithScore(
		ctx,
		query,
		r.config.MaxDocuments,
		r.config.ScoreThreshold,
	)
	if err != nil {
		return nil, fmt.Errorf("similarity search failed: %w", err)
	}

	documents := make([]vectorstore.Document, len(results))
	for i, result := range results {
		documents[i] = result.Document
	}

	return documents, nil
}

// RetrieveWithScores retrieves documents with their similarity scores
func (r *Retriever) RetrieveWithScores(ctx context.Context, query string) ([]vectorstore.SearchResult, error) {
	if r.vectorStore == nil {
		return nil, fmt.Errorf("vector store not initialized")
	}

	return r.vectorStore.SimilaritySearchWithScore(
		ctx,
		query,
		r.config.MaxDocuments,
		r.config.ScoreThreshold,
	)
}

// FormatDocuments formats retrieved documents into a context string
func (r *Retriever) FormatDocuments(documents []vectorstore.Document) string {
	if len(documents) == 0 {
		return ""
	}

	var parts []string
	for i, doc := range documents {
		content := doc.Content
		if r.config.IncludeMetadata && len(doc.Metadata) > 0 {
			content = fmt.Sprintf("[Source: %v]\n%s", doc.Metadata, content)
		}
		parts = append(parts, fmt.Sprintf("Document %d:\n%s", i+1, content))
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// LangChainRetriever adapts our Retriever to LangChain's retriever interface
type LangChainRetriever struct {
	retriever *Retriever
}

// NewLangChainRetriever creates a new LangChain-compatible retriever
func NewLangChainRetriever(retriever *Retriever) *LangChainRetriever {
	return &LangChainRetriever{
		retriever: retriever,
	}
}

// GetRelevantDocuments implements the LangChain retriever interface
func (l *LangChainRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	docs, err := l.retriever.Retrieve(ctx, query)
	if err != nil {
		return nil, err
	}

	// Convert to LangChain documents
	lcDocs := make([]schema.Document, len(docs))
	for i, doc := range docs {
		lcDocs[i] = schema.Document{
			PageContent: doc.Content,
			Metadata:    doc.Metadata,
		}
	}

	return lcDocs, nil
}
