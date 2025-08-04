package vectorstore

import (
	"errors"
	"fmt"
	"strings"
)

// EmbeddingError provides context for embedding operation failures
type EmbeddingError struct {
	Operation string
	Provider  string
	Cause     error
}

func (e *EmbeddingError) Error() string {
	return fmt.Sprintf("%s failed for %s: %v", e.Operation, e.Provider, e.Cause)
}

func (e *EmbeddingError) Unwrap() error {
	return e.Cause
}

// wrapEmbeddingError wraps an error with embedding operation context
func wrapEmbeddingError(err error, operation, provider string) error {
	if err == nil {
		return nil
	}

	return &EmbeddingError{
		Operation: operation,
		Provider:  provider,
		Cause:     err,
	}
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error patterns that indicate transient failures
	errStr := strings.ToLower(err.Error())

	// Network and timeout errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "no such host") {
		return true
	}

	// Rate limiting errors
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "429") {
		return true
	}

	// Server errors
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") {
		return true
	}

	return false
}

// ValidationError represents an input validation failure
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

// Common validation errors
var (
	ErrEmptyText       = errors.New("empty text")
	ErrTextTooLong     = errors.New("text exceeds maximum length")
	ErrBatchTooLarge   = errors.New("batch size exceeds maximum")
	ErrEmptyCollection = errors.New("collection name cannot be empty")
	ErrEmptyDocumentID = errors.New("document ID cannot be empty")
)

// validateCollectionName checks if a collection name is valid
func validateCollectionName(name string) error {
	if name == "" {
		return ErrEmptyCollection
	}

	// Collection names should be alphanumeric with underscores
	for i, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return &ValidationError{
				Field:   "collection_name",
				Message: fmt.Sprintf("invalid character '%c' at position %d", r, i),
			}
		}
	}

	// Reasonable length limits
	if len(name) > 128 {
		return &ValidationError{
			Field:   "collection_name",
			Message: "name exceeds maximum length of 128 characters",
		}
	}

	return nil
}

// validateDocumentID checks if a document ID is valid
func validateDocumentID(id string) error {
	if id == "" {
		return ErrEmptyDocumentID
	}

	// Document IDs should not contain newlines or control characters
	for _, r := range id {
		if r < 32 || r == 127 {
			return &ValidationError{
				Field:   "document_id",
				Message: "ID contains invalid control characters",
			}
		}
	}

	// Reasonable length limit
	if len(id) > 256 {
		return &ValidationError{
			Field:   "document_id",
			Message: "ID exceeds maximum length of 256 characters",
		}
	}

	return nil
}
