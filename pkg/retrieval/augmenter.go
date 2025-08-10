package retrieval

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/vectorstore"
)

// Augmenter handles prompt augmentation with retrieved context
type Augmenter struct {
	retriever *Retriever
	config    AugmenterConfig
}

// AugmenterConfig contains configuration for the augmenter
type AugmenterConfig struct {
	// Template for augmented prompts
	Template string

	// MaxContextLength limits the context size
	MaxContextLength int

	// IncludeScores determines if scores should be included
	IncludeScores bool

	// MinRelevanceScore filters out low-relevance documents
	MinRelevanceScore float32
}

// DefaultTemplate is the default prompt template
const DefaultTemplate = `Answer the following question based on the provided context. If the context doesn't contain relevant information, say so.

Context:
%s

Question: %s

Answer:`

// NewAugmenter creates a new augmenter
func NewAugmenter(retriever *Retriever, config AugmenterConfig) *Augmenter {
	if config.Template == "" {
		config.Template = DefaultTemplate
	}
	if config.MaxContextLength == 0 {
		config.MaxContextLength = 4000 // Default max context
	}
	return &Augmenter{
		retriever: retriever,
		config:    config,
	}
}

// AugmentPrompt augments a prompt with retrieved context
func (a *Augmenter) AugmentPrompt(ctx context.Context, prompt string) (string, error) {
	// Retrieve relevant documents
	results, err := a.retriever.RetrieveWithScores(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve context: %w", err)
	}

	// Filter by relevance score if configured
	if a.config.MinRelevanceScore > 0 {
		filtered := make([]vectorstore.SearchResult, 0, len(results))
		for _, result := range results {
			if result.Score >= a.config.MinRelevanceScore {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	// Format context
	context := a.formatContext(results)

	// Truncate if necessary
	if len(context) > a.config.MaxContextLength {
		context = context[:a.config.MaxContextLength] + "..."
	}

	// Apply template
	augmented := fmt.Sprintf(a.config.Template, context, prompt)
	return augmented, nil
}

// formatContext formats retrieved documents as context
func (a *Augmenter) formatContext(results []vectorstore.SearchResult) string {
	if len(results) == 0 {
		return "No relevant context found."
	}

	var parts []string
	for i, result := range results {
		content := result.Document.Content

		// Add score if configured
		if a.config.IncludeScores {
			content = fmt.Sprintf("[Relevance: %.2f]\n%s", result.Score, content)
		}

		// Add source metadata if available
		if source, ok := result.Document.Metadata["source"].(string); ok {
			content = fmt.Sprintf("[Source: %s]\n%s", source, content)
		}

		parts = append(parts, fmt.Sprintf("%d. %s", i+1, content))
	}

	return strings.Join(parts, "\n\n")
}

// GetContext retrieves and formats context without augmenting the prompt
func (a *Augmenter) GetContext(ctx context.Context, query string) (string, error) {
	results, err := a.retriever.RetrieveWithScores(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve context: %w", err)
	}

	return a.formatContext(results), nil
}

// AugmentResult contains the result of prompt augmentation
type AugmentResult struct {
	// OriginalPrompt is the original user prompt
	OriginalPrompt string

	// AugmentedPrompt is the prompt with added context
	AugmentedPrompt string

	// Context is the retrieved context
	Context string

	// Documents are the retrieved documents
	Documents []vectorstore.Document

	// Scores are the relevance scores
	Scores []float32
}

// AugmentWithDetails provides detailed augmentation results
func (a *Augmenter) AugmentWithDetails(ctx context.Context, prompt string) (*AugmentResult, error) {
	// Retrieve relevant documents
	results, err := a.retriever.RetrieveWithScores(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve context: %w", err)
	}

	// Extract documents and scores
	documents := make([]vectorstore.Document, len(results))
	scores := make([]float32, len(results))
	for i, result := range results {
		documents[i] = result.Document
		scores[i] = result.Score
	}

	// Format context
	context := a.formatContext(results)

	// Create augmented prompt
	augmented := fmt.Sprintf(a.config.Template, context, prompt)

	return &AugmentResult{
		OriginalPrompt:  prompt,
		AugmentedPrompt: augmented,
		Context:         context,
		Documents:       documents,
		Scores:          scores,
	}, nil
}
