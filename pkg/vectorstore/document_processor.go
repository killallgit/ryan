package vectorstore

import (
	"context"
	"fmt"
	"strings"
)

// DocumentProcessor handles common document processing operations
type DocumentProcessor struct {
	embedder     Embedder
	chunkSize    int
	chunkOverlap int
}

// NewDocumentProcessor creates a new document processor
func NewDocumentProcessor(embedder Embedder, chunkSize, chunkOverlap int) *DocumentProcessor {
	// Set defaults if not provided
	if chunkSize <= 0 {
		chunkSize = 1000
	}
	if chunkOverlap < 0 {
		chunkOverlap = 200
	}

	return &DocumentProcessor{
		embedder:     embedder,
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}
}

// ProcessDocument processes a single document, generating embedding if needed
func (dp *DocumentProcessor) ProcessDocument(ctx context.Context, doc *Document) error {
	// Validate document
	if err := validateDocumentID(doc.ID); err != nil {
		return fmt.Errorf("document validation failed: %w", err)
	}

	// Generate embedding if not provided
	if len(doc.Embedding) == 0 && doc.Content != "" {
		embedding, err := dp.embedder.EmbedText(ctx, doc.Content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
		doc.Embedding = embedding
	}

	return nil
}

// ProcessDocuments processes multiple documents in batch
func (dp *DocumentProcessor) ProcessDocuments(ctx context.Context, docs []Document) error {
	// Process in batches if needed
	if len(docs) > MaxBatchSize {
		return dp.processBatchedDocuments(ctx, docs)
	}

	// Validate all documents first
	for i, doc := range docs {
		if err := validateDocumentID(doc.ID); err != nil {
			return fmt.Errorf("document at index %d validation failed: %w", i, err)
		}
	}

	// Generate embeddings for documents without them
	for i := range docs {
		if len(docs[i].Embedding) == 0 && docs[i].Content != "" {
			embedding, err := dp.embedder.EmbedText(ctx, docs[i].Content)
			if err != nil {
				return fmt.Errorf("failed to generate embedding for document %s: %w", docs[i].ID, err)
			}
			docs[i].Embedding = embedding
		}
	}

	return nil
}

// processBatchedDocuments handles large document sets in batches
func (dp *DocumentProcessor) processBatchedDocuments(ctx context.Context, docs []Document) error {
	for i := 0; i < len(docs); i += MaxBatchSize {
		end := i + MaxBatchSize
		if end > len(docs) {
			end = len(docs)
		}

		batch := docs[i:end]
		if err := dp.ProcessDocuments(ctx, batch); err != nil {
			return fmt.Errorf("failed to process batch starting at index %d: %w", i, err)
		}
	}

	return nil
}

// ChunkDocument splits a document into smaller chunks with metadata
func (dp *DocumentProcessor) ChunkDocument(doc Document, baseMetadata map[string]interface{}) ([]Document, error) {
	if err := validateDocumentID(doc.ID); err != nil {
		return nil, fmt.Errorf("document validation failed: %w", err)
	}

	chunks := dp.ChunkText(doc.Content)
	docs := make([]Document, 0, len(chunks))

	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}

		// Create chunk ID
		chunkID := fmt.Sprintf("%s_chunk_%d", doc.ID, i)

		// Merge metadata
		metadata := make(map[string]interface{})

		// Copy base metadata
		for k, v := range baseMetadata {
			metadata[k] = v
		}

		// Copy document metadata
		for k, v := range doc.Metadata {
			metadata[k] = v
		}

		// Add chunk-specific metadata
		metadata["chunk_index"] = i
		metadata["chunk_total"] = len(chunks)
		metadata["parent_doc_id"] = doc.ID

		docs = append(docs, Document{
			ID:       chunkID,
			Content:  chunk,
			Metadata: metadata,
		})
	}

	return docs, nil
}

// ChunkText splits text into overlapping chunks
func (dp *DocumentProcessor) ChunkText(text string) []string {
	if len(text) <= dp.chunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + dp.chunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at a sentence or paragraph boundary
		if end < len(text) {
			// Look for sentence endings
			sentenceEnds := []string{". ", ".\n", "! ", "!\n", "? ", "?\n"}
			bestEnd := end

			for _, sep := range sentenceEnds {
				if idx := strings.LastIndex(text[start:end], sep); idx > 0 {
					bestEnd = start + idx + len(sep)
					break
				}
			}

			// If no sentence boundary, try paragraph
			if bestEnd == end {
				if idx := strings.LastIndex(text[start:end], "\n\n"); idx > 0 {
					bestEnd = start + idx + 2
				}
			}

			end = bestEnd
		}

		chunks = append(chunks, text[start:end])

		// Move start position with overlap
		if end < len(text) {
			start = end - dp.chunkOverlap
			if start < 0 {
				start = 0
			}
		} else {
			break
		}
	}

	return chunks
}

// ChunkCode splits code into logical chunks (currently line-based)
func (dp *DocumentProcessor) ChunkCode(code string) []string {
	lines := strings.Split(code, "\n")

	var chunks []string
	var currentChunk []string
	currentSize := 0

	for _, line := range lines {
		lineSize := len(line) + 1 // +1 for newline

		// Check if adding this line would exceed chunk size
		if currentSize+lineSize > dp.chunkSize && len(currentChunk) > 0 {
			chunks = append(chunks, strings.Join(currentChunk, "\n"))

			// Start new chunk with overlap
			overlapStart := len(currentChunk) - (dp.chunkOverlap / 50) // Rough line count for overlap
			if overlapStart < 0 {
				overlapStart = 0
			}
			currentChunk = currentChunk[overlapStart:]
			currentSize = 0
			for _, l := range currentChunk {
				currentSize += len(l) + 1
			}
		}

		currentChunk = append(currentChunk, line)
		currentSize += lineSize
	}

	// Add the last chunk
	if len(currentChunk) > 0 {
		chunks = append(chunks, strings.Join(currentChunk, "\n"))
	}

	return chunks
}
