package stream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	t.Run("Multiple Sources", func(t *testing.T) {
		registry := NewRegistry()

		// Register multiple sources
		err := registry.Register("source1", "ollama", "provider1")
		require.NoError(t, err)

		err = registry.Register("source2", "openai", "provider2")
		require.NoError(t, err)

		err = registry.Register("source3", "anthropic", "provider3")
		require.NoError(t, err)

		// Check that all sources are registered
		source1, exists := registry.Get("source1")
		assert.True(t, exists)
		assert.Equal(t, "ollama", source1.Type)

		source2, exists := registry.Get("source2")
		assert.True(t, exists)
		assert.Equal(t, "openai", source2.Type)

		source3, exists := registry.Get("source3")
		assert.True(t, exists)
		assert.Equal(t, "anthropic", source3.Type)

		// Check list
		ids := registry.List()
		assert.Len(t, ids, 3)
		assert.Contains(t, ids, "source1")
		assert.Contains(t, ids, "source2")
		assert.Contains(t, ids, "source3")
	})

	t.Run("Default Source", func(t *testing.T) {
		registry := NewRegistry()

		// First registered becomes default
		err := registry.Register("first", "ollama", "provider1")
		require.NoError(t, err)

		defaultSource, err := registry.GetDefault()
		require.NoError(t, err)
		assert.Equal(t, "first", defaultSource.ID)

		// Can change default
		err = registry.Register("second", "openai", "provider2")
		require.NoError(t, err)

		err = registry.SetDefault("second")
		require.NoError(t, err)

		defaultSource, err = registry.GetDefault()
		require.NoError(t, err)
		assert.Equal(t, "second", defaultSource.ID)
	})

	t.Run("GetOrDefault", func(t *testing.T) {
		registry := NewRegistry()

		err := registry.Register("default", "ollama", "provider1")
		require.NoError(t, err)

		err = registry.Register("other", "openai", "provider2")
		require.NoError(t, err)

		// Get specific source
		source, err := registry.GetOrDefault("other")
		require.NoError(t, err)
		assert.Equal(t, "other", source.ID)

		// Fall back to default for non-existent
		source, err = registry.GetOrDefault("nonexistent")
		require.NoError(t, err)
		assert.Equal(t, "default", source.ID)

		// Empty ID returns default
		source, err = registry.GetOrDefault("")
		require.NoError(t, err)
		assert.Equal(t, "default", source.ID)
	})

	t.Run("Remove Source", func(t *testing.T) {
		registry := NewRegistry()

		err := registry.Register("source1", "ollama", "provider1")
		require.NoError(t, err)

		err = registry.Register("source2", "openai", "provider2")
		require.NoError(t, err)

		// Remove source
		err = registry.Remove("source1")
		require.NoError(t, err)

		_, exists := registry.Get("source1")
		assert.False(t, exists)

		// Default should switch to remaining source
		defaultSource, err := registry.GetDefault()
		require.NoError(t, err)
		assert.Equal(t, "source2", defaultSource.ID)
	})

	t.Run("Thread Safety", func(t *testing.T) {
		registry := NewRegistry()
		done := make(chan bool)

		// Concurrent writes
		go func() {
			for i := 0; i < 100; i++ {
				registry.Register("source", "type", "provider")
			}
			done <- true
		}()

		// Concurrent reads
		go func() {
			for i := 0; i < 100; i++ {
				registry.Get("source")
			}
			done <- true
		}()

		<-done
		<-done
	})
}
