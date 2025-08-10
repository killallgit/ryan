package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	t.Run("register and get template", func(t *testing.T) {
		registry := NewRegistry()

		template := NewPromptTemplate("Hello {{.name}}", []string{"name"})
		err := registry.Register("greeting", template)
		require.NoError(t, err)

		retrieved, err := registry.Get("greeting")
		require.NoError(t, err)
		assert.Equal(t, template, retrieved)
	})

	t.Run("register duplicate template", func(t *testing.T) {
		registry := NewRegistry()

		template1 := NewPromptTemplate("Template 1", []string{})
		template2 := NewPromptTemplate("Template 2", []string{})

		err := registry.Register("test", template1)
		require.NoError(t, err)

		err = registry.Register("test", template2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("get non-existent template", func(t *testing.T) {
		registry := NewRegistry()

		_, err := registry.Get("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("list templates", func(t *testing.T) {
		registry := NewRegistry()

		template1 := NewPromptTemplate("T1", []string{})
		template2 := NewPromptTemplate("T2", []string{})
		template3 := NewPromptTemplate("T3", []string{})

		registry.Register("template1", template1)
		registry.Register("template2", template2)
		registry.Register("template3", template3)

		names := registry.List()
		assert.Len(t, names, 3)
		assert.ElementsMatch(t, []string{"template1", "template2", "template3"}, names)
	})

	t.Run("clear registry", func(t *testing.T) {
		registry := NewRegistry()

		template := NewPromptTemplate("Test", []string{})
		registry.Register("test", template)

		assert.Len(t, registry.List(), 1)

		registry.Clear()
		assert.Len(t, registry.List(), 0)
	})

	t.Run("concurrent access", func(t *testing.T) {
		registry := NewRegistry()
		template := NewPromptTemplate("Concurrent {{.test}}", []string{"test"})

		// Register template
		err := registry.Register("concurrent", template)
		require.NoError(t, err)

		// Concurrent reads
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				retrieved, err := registry.Get("concurrent")
				assert.NoError(t, err)
				assert.NotNil(t, retrieved)
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestGlobalRegistry(t *testing.T) {
	// Clear the default registry first
	DefaultRegistry.Clear()

	t.Run("must register and get", func(t *testing.T) {
		template := NewPromptTemplate("Global {{.test}}", []string{"test"})

		// Should not panic
		assert.NotPanics(t, func() {
			MustRegister("global_test", template)
		})

		// Should not panic and return template
		var retrieved Template
		assert.NotPanics(t, func() {
			retrieved = MustGet("global_test")
		})
		assert.Equal(t, template, retrieved)
	})

	t.Run("must get panics on missing", func(t *testing.T) {
		assert.Panics(t, func() {
			MustGet("nonexistent_template")
		})
	})

	t.Run("must register panics on duplicate", func(t *testing.T) {
		template := NewPromptTemplate("Test", []string{})
		MustRegister("duplicate_test", template)

		assert.Panics(t, func() {
			MustRegister("duplicate_test", template)
		})
	})
}

func TestCommonTemplates(t *testing.T) {
	// Re-register common templates since they may have been cleared
	registerBuiltInTemplates()

	// Test that common templates are registered
	t.Run("simple qa template", func(t *testing.T) {
		template := GetSimpleQATemplate()
		assert.NotNil(t, template)

		result, err := template.Format(map[string]any{
			"question": "What is the meaning of life?",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "What is the meaning of life?")
	})

	t.Run("summarization template", func(t *testing.T) {
		template := GetSummarizationTemplate()
		assert.NotNil(t, template)

		result, err := template.Format(map[string]any{
			"text":  "Long text here...",
			"style": "concise",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Long text here...")
		assert.Contains(t, result, "concise")
	})

	t.Run("code generation template", func(t *testing.T) {
		template := GetCodeGenTemplate()
		assert.NotNil(t, template)

		result, err := template.Format(map[string]any{
			"language":     "Go",
			"description":  "sorts a list",
			"requirements": "Must be efficient",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Go")
		assert.Contains(t, result, "sorts a list")
		assert.Contains(t, result, "Must be efficient")
	})

	t.Run("simple assistant chat template", func(t *testing.T) {
		template := GetSimpleAssistantTemplate()
		assert.NotNil(t, template)

		result, err := template.Format(map[string]any{
			"instructions": "Be concise",
			"query":        "Hello!",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Be concise")
		assert.Contains(t, result, "Hello!")
	})

	t.Run("simple RAG chat template", func(t *testing.T) {
		template := GetSimpleRAGTemplate()
		assert.NotNil(t, template)

		result, err := template.Format(map[string]any{
			"context":  "Paris is the capital of France.",
			"question": "What is the capital of France?",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Paris is the capital of France.")
		assert.Contains(t, result, "What is the capital of France?")
	})
}
