package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaultTemplate(t *testing.T) {
	// Clear registry to ensure clean test
	DefaultRegistry.Clear()

	t.Run("load YAML template from pkg/templates", func(t *testing.T) {
		template, err := LoadDefaultTemplate("qa.yaml")
		require.NoError(t, err)
		assert.NotNil(t, template)

		// Test formatting
		result, err := template.Format(map[string]any{
			"question": "What is Go?",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "What is Go?")
	})

	t.Run("load JSON chat template from pkg/templates", func(t *testing.T) {
		template, err := LoadDefaultChatTemplate("chat_assistant.json")
		require.NoError(t, err)
		assert.NotNil(t, template)

		// Test formatting
		result, err := template.Format(map[string]any{
			"instructions": "Be helpful and concise",
			"query":        "Hello!",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "helpful AI assistant")
		assert.Contains(t, result, "Be helpful and concise")
		assert.Contains(t, result, "Hello!")
	})
}

func TestLoadAllTemplates(t *testing.T) {
	// Clear registry first
	DefaultRegistry.Clear()

	// Load all templates
	err := LoadAllTemplates()
	require.NoError(t, err)

	// Check that templates were loaded
	templates := DefaultRegistry.List()
	assert.NotEmpty(t, templates)

	// Verify specific templates are available
	_, err = DefaultRegistry.Get("qa")
	assert.NoError(t, err)

	_, err = DefaultRegistry.Get("chat_assistant")
	assert.NoError(t, err)
}

func TestGetTemplatesPath(t *testing.T) {
	path := getTemplatesPath()
	assert.Contains(t, path, "templates")
	assert.NotContains(t, path, "prompt/templates") // Should be pkg/templates not pkg/prompt/templates
}
