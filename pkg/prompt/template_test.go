package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptTemplate(t *testing.T) {
	t.Run("basic template formatting", func(t *testing.T) {
		template := NewPromptTemplate(
			"Hello {{.name}}, welcome to {{.place}}!",
			[]string{"name", "place"},
		)

		result, err := template.Format(map[string]any{
			"name":  "Alice",
			"place": "Wonderland",
		})

		require.NoError(t, err)
		assert.Equal(t, "Hello Alice, welcome to Wonderland!", result)
	})

	t.Run("template with partial variables", func(t *testing.T) {
		template := NewPromptTemplate(
			"{{.greeting}} {{.name}}!",
			[]string{"greeting", "name"},
		)

		// Create a new template with partial variables
		partial := template.WithPartialVariables(map[string]any{
			"greeting": "Hello",
		})

		result, err := partial.Format(map[string]any{
			"name": "Bob",
		})

		require.NoError(t, err)
		assert.Equal(t, "Hello Bob!", result)
	})

	t.Run("template with variable metadata", func(t *testing.T) {
		template, err := NewPromptTemplateWithOptions(
			"Age: {{.age}}, Score: {{.score}}",
			[]string{"age", "score"},
			WithVariableMetadata(
				&Variable{
					Name:     "age",
					Required: true,
					Validator: func(v any) error {
						age, ok := v.(int)
						if !ok || age < 0 {
							return assert.AnError
						}
						return nil
					},
				},
				&Variable{
					Name:    "score",
					Default: 0,
				},
			),
		)

		require.NoError(t, err)

		// Test with valid values
		result, err := template.Format(map[string]any{
			"age": 25,
		})
		require.NoError(t, err)
		assert.Equal(t, "Age: 25, Score: 0", result)

		// Test with invalid age
		_, err = template.Format(map[string]any{
			"age": -5,
		})
		assert.Error(t, err)
	})

	t.Run("get input variables", func(t *testing.T) {
		template := NewPromptTemplate(
			"{{.var1}} and {{.var2}} and {{.var3}}",
			[]string{"var1", "var2", "var3"},
		)

		vars := template.GetInputVariables()
		assert.ElementsMatch(t, []string{"var1", "var2", "var3"}, vars)
	})
}

func TestChatPromptTemplate(t *testing.T) {
	t.Run("basic chat template", func(t *testing.T) {
		messages := []MessageDefinition{
			{Role: "system", Template: "You are a {{.role}}", Variables: []string{"role"}},
			{Role: "human", Template: "{{.question}}", Variables: []string{"question"}},
		}

		template, err := NewChatTemplateFromMessages(messages)
		require.NoError(t, err)

		result, err := template.Format(map[string]any{
			"role":     "helpful assistant",
			"question": "What is the capital of France?",
		})

		require.NoError(t, err)
		assert.Contains(t, result, "helpful assistant")
		assert.Contains(t, result, "What is the capital of France?")
	})

	t.Run("chat template with messages", func(t *testing.T) {
		messages := []MessageDefinition{
			{Role: "system", Template: "System: {{.sys}}", Variables: []string{"sys"}},
			{Role: "human", Template: "User: {{.msg}}", Variables: []string{"msg"}},
		}

		template, err := NewChatTemplateFromMessages(messages)
		require.NoError(t, err)

		msgs, err := template.FormatMessages(map[string]any{
			"sys": "Be helpful",
			"msg": "Hello",
		})

		require.NoError(t, err)
		assert.Len(t, msgs, 2)
	})

	t.Run("chat template input variables", func(t *testing.T) {
		messages := []MessageDefinition{
			{Role: "system", Template: "{{.sys1}} {{.sys2}}", Variables: []string{"sys1", "sys2"}},
			{Role: "human", Template: "{{.user}}", Variables: []string{"user"}},
		}

		template, err := NewChatTemplateFromMessages(messages)
		require.NoError(t, err)

		vars := template.GetInputVariables()
		assert.ElementsMatch(t, []string{"sys1", "sys2", "user"}, vars)
	})
}
