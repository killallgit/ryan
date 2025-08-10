package retrieval

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/vectorstore"
)

// DocumentManager handles document processing and management
type DocumentManager struct {
	config DocumentConfig
}

// DocumentConfig contains configuration for document management
type DocumentConfig struct {
	// ChunkSize is the maximum size of document chunks
	ChunkSize int

	// ChunkOverlap is the overlap between chunks
	ChunkOverlap int

	// IDPrefix for generated document IDs
	IDPrefix string
}

// NewDocumentManager creates a new document manager
func NewDocumentManager(config DocumentConfig) *DocumentManager {
	if config.ChunkSize == 0 {
		config.ChunkSize = 1000
	}
	if config.ChunkOverlap == 0 {
		config.ChunkOverlap = 200
	}
	return &DocumentManager{
		config: config,
	}
}

// CreateDocument creates a document from content
func (m *DocumentManager) CreateDocument(content string, metadata map[string]interface{}) vectorstore.Document {
	id := m.generateID(content)
	if m.config.IDPrefix != "" {
		id = m.config.IDPrefix + "_" + id
	}

	return vectorstore.Document{
		ID:       id,
		Content:  content,
		Metadata: metadata,
	}
}

// ChunkDocument splits a document into smaller chunks
func (m *DocumentManager) ChunkDocument(doc vectorstore.Document) []vectorstore.Document {
	chunks := m.splitText(doc.Content, m.config.ChunkSize, m.config.ChunkOverlap)

	result := make([]vectorstore.Document, len(chunks))
	for i, chunk := range chunks {
		// Create metadata for chunk
		chunkMetadata := make(map[string]interface{})
		for k, v := range doc.Metadata {
			chunkMetadata[k] = v
		}
		chunkMetadata["chunk_index"] = i
		chunkMetadata["total_chunks"] = len(chunks)
		chunkMetadata["parent_id"] = doc.ID

		result[i] = vectorstore.Document{
			ID:       fmt.Sprintf("%s_chunk_%d", doc.ID, i),
			Content:  chunk,
			Metadata: chunkMetadata,
		}
	}

	return result
}

// ChunkText splits text into chunks
func (m *DocumentManager) ChunkText(text string) []string {
	return m.splitText(text, m.config.ChunkSize, m.config.ChunkOverlap)
}

// splitText splits text into overlapping chunks
func (m *DocumentManager) splitText(text string, chunkSize, overlap int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	runes := []rune(text)

	for start := 0; start < len(runes); {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		// Try to break at sentence boundary
		chunk := string(runes[start:end])
		if end < len(runes) {
			// Look for sentence ending
			lastPeriod := strings.LastIndex(chunk, ". ")
			lastQuestion := strings.LastIndex(chunk, "? ")
			lastExclaim := strings.LastIndex(chunk, "! ")

			// Find the last sentence boundary
			lastBoundary := max(lastPeriod, lastQuestion, lastExclaim)
			if lastBoundary > 0 && lastBoundary > chunkSize/2 {
				chunk = chunk[:lastBoundary+2]
				end = start + len([]rune(chunk))
			}
		}

		chunks = append(chunks, strings.TrimSpace(chunk))

		// Move forward with overlap
		if end >= len(runes) {
			break
		}
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

// generateID generates a unique ID for content
func (m *DocumentManager) generateID(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// max returns the maximum of three integers
func max(a, b, c int) int {
	if a >= b && a >= c {
		return a
	}
	if b >= a && b >= c {
		return b
	}
	return c
}

// DocumentLoader provides utilities for loading documents
type DocumentLoader struct {
	manager *DocumentManager
}

// NewDocumentLoader creates a new document loader
func NewDocumentLoader(manager *DocumentManager) *DocumentLoader {
	return &DocumentLoader{
		manager: manager,
	}
}

// LoadFromStrings creates documents from strings
func (l *DocumentLoader) LoadFromStrings(contents []string, metadataList []map[string]interface{}) []vectorstore.Document {
	documents := make([]vectorstore.Document, len(contents))

	for i, content := range contents {
		var metadata map[string]interface{}
		if i < len(metadataList) {
			metadata = metadataList[i]
		}
		documents[i] = l.manager.CreateDocument(content, metadata)
	}

	return documents
}

// LoadAndChunk loads and chunks documents
func (l *DocumentLoader) LoadAndChunk(contents []string, metadataList []map[string]interface{}) []vectorstore.Document {
	var allChunks []vectorstore.Document

	for i, content := range contents {
		var metadata map[string]interface{}
		if i < len(metadataList) {
			metadata = metadataList[i]
		}

		doc := l.manager.CreateDocument(content, metadata)
		chunks := l.manager.ChunkDocument(doc)
		allChunks = append(allChunks, chunks...)
	}

	return allChunks
}
