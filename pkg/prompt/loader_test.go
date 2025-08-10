package prompt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileLoader(t *testing.T) {
	// Create a temporary directory for test templates
	tmpDir := t.TempDir()

	t.Run("load raw template file", func(t *testing.T) {
		// Create a simple template file
		content := "Hello {{.name}}, welcome to {{.place}}!"
		templatePath := filepath.Join(tmpDir, "greeting.txt")
		err := os.WriteFile(templatePath, []byte(content), 0644)
		require.NoError(t, err)

		loader := NewFileLoader(tmpDir)
		template, err := loader.Load("greeting.txt")
		require.NoError(t, err)

		result, err := template.Format(map[string]any{
			"name":  "Alice",
			"place": "Wonderland",
		})
		require.NoError(t, err)
		assert.Equal(t, "Hello Alice, welcome to Wonderland!", result)
	})

	t.Run("load JSON template file", func(t *testing.T) {
		jsonContent := `{
			"name": "test",
			"template": "Name: {{.name}}, Age: {{.age}}",
			"variables": ["name", "age"],
			"metadata": [
				{
					"Name": "age",
					"Type": "int",
					"Required": true,
					"Default": 0
				}
			]
		}`

		templatePath := filepath.Join(tmpDir, "person.json")
		err := os.WriteFile(templatePath, []byte(jsonContent), 0644)
		require.NoError(t, err)

		loader := NewFileLoader(tmpDir)
		template, err := loader.Load("person.json")
		require.NoError(t, err)

		result, err := template.Format(map[string]any{
			"name": "Bob",
			"age":  30,
		})
		require.NoError(t, err)
		assert.Equal(t, "Name: Bob, Age: 30", result)
	})

	t.Run("load YAML template file", func(t *testing.T) {
		yamlContent := `
name: test
template: "Hello {{.name}}"
variables:
  - name
partials:
  greeting: "Hi"
`
		templatePath := filepath.Join(tmpDir, "test.yaml")
		err := os.WriteFile(templatePath, []byte(yamlContent), 0644)
		require.NoError(t, err)

		loader := NewFileLoader(tmpDir)
		template, err := loader.Load("test.yaml")
		require.NoError(t, err)
		assert.NotNil(t, template)
	})

	t.Run("load chat template", func(t *testing.T) {
		chatContent := `{
			"name": "assistant",
			"messages": [
				{
					"role": "system",
					"template": "You are a {{.role}}",
					"variables": ["role"]
				},
				{
					"role": "human",
					"template": "{{.message}}",
					"variables": ["message"]
				}
			]
		}`

		templatePath := filepath.Join(tmpDir, "chat.json")
		err := os.WriteFile(templatePath, []byte(chatContent), 0644)
		require.NoError(t, err)

		loader := NewFileLoader(tmpDir)
		chat, err := loader.LoadChat("chat.json")
		require.NoError(t, err)

		result, err := chat.Format(map[string]any{
			"role":    "helpful assistant",
			"message": "Hello!",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "helpful assistant")
		assert.Contains(t, result, "Hello!")
	})
}

func TestStringLoader(t *testing.T) {
	t.Run("load string template", func(t *testing.T) {
		loader := NewStringLoader()
		loader.AddTemplate("test", "Hello {{.name}}!", []string{"name"})

		template, err := loader.Load("test")
		require.NoError(t, err)

		result, err := template.Format(map[string]any{
			"name": "World",
		})
		require.NoError(t, err)
		assert.Equal(t, "Hello World!", result)
	})

	t.Run("load chat template", func(t *testing.T) {
		loader := NewStringLoader()
		messages := []MessageDefinition{
			{Role: "system", Template: "System prompt", Variables: []string{}},
			{Role: "human", Template: "{{.query}}", Variables: []string{"query"}},
		}
		loader.AddChatTemplate("chat", messages)

		template, err := loader.LoadChat("chat")
		require.NoError(t, err)

		result, err := template.Format(map[string]any{
			"query": "Test query",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Test query")
	})

	t.Run("template not found", func(t *testing.T) {
		loader := NewStringLoader()
		_, err := loader.Load("nonexistent")
		assert.Error(t, err)
	})
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected []string
	}{
		{
			name:     "simple variables",
			template: "Hello {{.name}}, you are {{.age}} years old",
			expected: []string{"name", "age"},
		},
		{
			name:     "no variables",
			template: "Hello world!",
			expected: []string{},
		},
		{
			name:     "repeated variables",
			template: "{{.name}} is {{.name}} and {{.name}}",
			expected: []string{"name"},
		},
		{
			name:     "complex template",
			template: "User {{.user}} has role {{.role}} in {{.org}}",
			expected: []string{"user", "role", "org"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := extractVariables(tt.template)
			assert.ElementsMatch(t, tt.expected, vars)
		})
	}
}

func TestQuickTemplates(t *testing.T) {
	t.Run("quick template", func(t *testing.T) {
		template := QuickTemplate("Hello {{.name}}!")

		result, err := template.Format(map[string]any{
			"name": "Quick",
		})
		require.NoError(t, err)
		assert.Equal(t, "Hello Quick!", result)
	})

	t.Run("quick chat template", func(t *testing.T) {
		template := QuickChatTemplate(
			"You are a helpful assistant",
			"Answer this: {{.question}}",
		)

		result, err := template.Format(map[string]any{
			"question": "What is 2+2?",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "helpful assistant")
		assert.Contains(t, result, "What is 2+2?")
	})
}
