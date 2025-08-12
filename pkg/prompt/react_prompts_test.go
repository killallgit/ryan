package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadReactPrompt(t *testing.T) {
	// Create a temporary directory for test prompts
	tmpDir := t.TempDir()
	testPromptDir := filepath.Join(tmpDir, "prompts", "unified")
	require.NoError(t, os.MkdirAll(testPromptDir, 0755))

	// Create a test prompt file
	promptContent := `# Test Execute Mode Prompt

You are an AI assistant in execute mode.

Available tools:
{{.tool_descriptions}}

Previous conversation:
{{.history}}

User input: {{.input}}`

	promptFile := filepath.Join(testPromptDir, "SYSTEM_PROMPT.md")
	require.NoError(t, os.WriteFile(promptFile, []byte(promptContent), 0644))

	// Change to the temporary directory for the test
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	// Test loading the prompt
	template, err := LoadReactPrompt("unified")
	require.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "unified", template.GetMode())

	// Test formatting the prompt
	result, err := template.Format("tool1, tool2", "previous chat", "hello world")
	require.NoError(t, err)
	assert.Contains(t, result, "tool1, tool2")
	assert.Contains(t, result, "previous chat")
	assert.Contains(t, result, "hello world")
}

func TestLoadReactPromptFileNotFound(t *testing.T) {
	// Test with non-existent directory
	_, err := LoadReactPrompt("non-existent-mode")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read prompt file")
}

func TestNewReactPromptTemplate(t *testing.T) {
	defaultTemplate := DefaultExecuteModePrompt()
	template := NewReactPromptTemplate(defaultTemplate, "test-mode")

	assert.NotNil(t, template)
	assert.Equal(t, "test-mode", template.GetMode())
	assert.Equal(t, defaultTemplate, template.GetTemplate())
}

func TestDefaultPrompts(t *testing.T) {
	t.Run("DefaultUnifiedPrompt", func(t *testing.T) {
		template := DefaultUnifiedPrompt()
		result, err := template.Format(map[string]any{
			"tool_descriptions": "test tools",
			"history":           "test history",
			"input":             "test input",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "test tools")
		assert.Contains(t, result, "test history")
		assert.Contains(t, result, "test input")
		assert.Contains(t, strings.ToLower(result), "react framework")
		assert.Contains(t, strings.ToLower(result), "task complexity")
	})

	t.Run("DefaultExecuteModePrompt_BackwardCompatibility", func(t *testing.T) {
		template := DefaultExecuteModePrompt()
		result, err := template.Format(map[string]any{
			"tool_descriptions": "test tools",
			"history":           "test history",
			"input":             "test input",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "test tools")
		assert.Contains(t, result, "test history")
		assert.Contains(t, result, "test input")
		// Should be same as unified prompt now
		assert.Contains(t, strings.ToLower(result), "react framework")
	})

	t.Run("DefaultPlanModePrompt_BackwardCompatibility", func(t *testing.T) {
		template := DefaultPlanModePrompt()
		result, err := template.Format(map[string]any{
			"tool_descriptions": "test tools",
			"history":           "test history",
			"input":             "test input",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "test tools")
		assert.Contains(t, result, "test history")
		assert.Contains(t, result, "test input")
		// Should be same as unified prompt now
		assert.Contains(t, strings.ToLower(result), "react framework")
	})
}

func TestReactPromptTemplateFormat(t *testing.T) {
	// Test with actual template content
	template := NewReactPromptTemplate(DefaultUnifiedPrompt(), "unified")

	result, err := template.Format(
		"- FileRead: Read files from disk\n- Git: Execute git commands",
		"Human: Hello\nAI: Hi there!",
		"What files are in this directory?",
	)

	require.NoError(t, err)
	assert.Contains(t, result, "FileRead: Read files")
	assert.Contains(t, result, "Git: Execute git")
	assert.Contains(t, result, "Human: Hello")
	assert.Contains(t, result, "What files are in this directory?")
}
