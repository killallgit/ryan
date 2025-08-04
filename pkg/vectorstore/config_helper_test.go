package vectorstore

import (
	"testing"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.VectorStoreConfig
		wantErr bool
		check   func(t *testing.T, m *Manager)
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "disabled vector store",
			config: &config.VectorStoreConfig{
				Enabled: false,
			},
			wantErr: true,
		},
		{
			name: "valid config with mock embedder",
			config: &config.VectorStoreConfig{
				Enabled:           true,
				Provider:          "chromem",
				PersistenceDir:    "",
				EnablePersistence: false,
				Embedder: config.VectorStoreEmbedderConfig{
					Provider: "mock",
				},
			},
			wantErr: false,
			check: func(t *testing.T, m *Manager) {
				assert.NotNil(t, m)
				assert.NotNil(t, m.GetStore())
				assert.NotNil(t, m.GetEmbedder())
			},
		},
		{
			name: "config with collections",
			config: &config.VectorStoreConfig{
				Enabled:           true,
				Provider:          "chromem",
				PersistenceDir:    "",
				EnablePersistence: false,
				Embedder: config.VectorStoreEmbedderConfig{
					Provider: "mock",
				},
				Collections: []config.VectorStoreCollectionConfig{
					{
						Name: "test_collection",
						Metadata: map[string]interface{}{
							"type": "test",
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, m *Manager) {
				assert.NotNil(t, m)
				// Verify collection was created
				col, err := m.GetStore().GetCollection("test_collection")
				assert.NoError(t, err)
				assert.NotNil(t, col)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewFromConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			defer manager.Close()

			if tt.check != nil {
				tt.check(t, manager)
			}
		})
	}
}

func TestNewIndexerFromConfig(t *testing.T) {
	// Create a mock store
	embedder := NewMockEmbedder(384)
	store, err := NewChromemStore(embedder, "", false)
	require.NoError(t, err)
	defer store.Close()

	tests := []struct {
		name           string
		config         *config.VectorStoreIndexerConfig
		collectionName string
		wantErr        bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &config.VectorStoreIndexerConfig{
				ChunkSize:    1000,
				ChunkOverlap: 200,
			},
			collectionName: "test",
			wantErr:        false,
		},
		{
			name: "invalid chunk overlap",
			config: &config.VectorStoreIndexerConfig{
				ChunkSize:    100,
				ChunkOverlap: 200, // Overlap > size
			},
			collectionName: "test",
			wantErr:        true,
		},
		{
			name: "zero chunk size uses default",
			config: &config.VectorStoreIndexerConfig{
				ChunkSize:    0,
				ChunkOverlap: 100,
			},
			collectionName: "test",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexer, err := NewIndexerFromConfig(store, tt.config, tt.collectionName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, indexer)
		})
	}
}