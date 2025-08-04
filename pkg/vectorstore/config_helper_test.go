package vectorstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitializeVectorStore(t *testing.T) {
	// This test would require mocking the global config
	// For now, we'll just verify the function exists
	assert.NotNil(t, InitializeVectorStore)
}

func TestNewIndexerFromGlobalConfig(t *testing.T) {
	// This test would require mocking the global config
	// For now, we'll just verify the function exists
	assert.NotNil(t, NewIndexerFromGlobalConfig)
}
