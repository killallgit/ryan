package vectorstore

import (
	"errors"
	"testing"
)

func TestValidateCollectionName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid name", "test_collection", false},
		{"Valid with dash", "test-collection", false},
		{"Valid alphanumeric", "test123", false},
		{"Empty name", "", true},
		{"Special characters", "test@collection", true},
		{"Spaces", "test collection", true},
		{"Too long", string(make([]byte, 129)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCollectionName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCollectionName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDocumentID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid ID", "doc123", false},
		{"Valid UUID", "550e8400-e29b-41d4-a716-446655440000", false},
		{"Valid with special", "doc_123-test", false},
		{"Empty ID", "", true},
		{"Control character", "doc\x00123", true},
		{"Newline", "doc\n123", true},
		{"Too long", string(make([]byte, 257)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDocumentID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDocumentID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"Timeout error", errors.New("request timeout"), true},
		{"Connection refused", errors.New("connection refused"), true},
		{"Rate limit", errors.New("rate limit exceeded"), true},
		{"429 status", errors.New("status code 429"), true},
		{"500 error", errors.New("internal server error 500"), true},
		{"502 bad gateway", errors.New("502 bad gateway"), true},
		{"503 unavailable", errors.New("503 service unavailable"), true},
		{"Normal error", errors.New("invalid input"), false},
		{"Nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableError(tt.err); got != tt.want {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmbeddingError(t *testing.T) {
	cause := errors.New("connection failed")
	err := &EmbeddingError{
		Operation: "embed_text",
		Provider:  "ollama",
		Cause:     cause,
	}

	expected := "embed_text failed for ollama: connection failed"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}

	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}
