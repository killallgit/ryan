package tui

import (
	"regexp"
	"strings"
	"testing"
)

// stripANSI removes ANSI escape codes from a string
func stripANSI(str string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m|\x1b\[[0-9;]*[A-Za-z]`)
	return ansiRegex.ReplaceAllString(str, "")
}

func TestRenderManager(t *testing.T) {
	theme := DefaultTheme()
	rm, err := NewRenderManager(theme, 80)
	if err != nil {
		t.Fatalf("Failed to create render manager: %v", err)
	}

	tests := []struct {
		name          string
		content       string
		contentType   ContentType
		role          string
		shouldContain []string
	}{
		{
			name:          "Plain text",
			content:       "This is plain text",
			contentType:   ContentTypePlain,
			role:          "user",
			shouldContain: []string{"plain text"},
		},
		{
			name: "Markdown with headers",
			content: `# Header 1
## Header 2
### Header 3

This is a paragraph with **bold** and *italic* text.`,
			contentType:   ContentTypeMarkdown,
			role:          "assistant",
			shouldContain: []string{"Header 1", "Header 2", "Header 3", "bold", "italic"},
		},
		{
			name: "Code block",
			content: `func main() {
	fmt.Println("Hello, World!")
}`,
			contentType:   ContentTypeCode,
			role:          "assistant",
			shouldContain: []string{"func", "main", "Println"},
		},
		{
			name: "JSON content",
			content: `{
	"name": "test",
	"value": 123
}`,
			contentType:   ContentTypeJSON,
			role:          "assistant",
			shouldContain: []string{"name", "test", "value", "123"},
		},
		{
			name: "Mixed content with thinking",
			content: `<thinking>
This is my thought process.
</thinking>

Here's the answer with **markdown**.`,
			contentType:   ContentTypeMixed,
			role:          "assistant",
			shouldContain: []string{"answer", "markdown"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := rm.Render(tt.content, tt.contentType, tt.role)

			// Check that output is not empty
			if rendered == "" {
				t.Errorf("Rendered output is empty")
			}

			// Strip ANSI codes for content checking
			cleanRendered := stripANSI(rendered)

			// Check that expected content is present
			for _, expected := range tt.shouldContain {
				if !strings.Contains(cleanRendered, expected) {
					t.Errorf("Expected output to contain '%s', but it doesn't.\nClean Output: %s", expected, cleanRendered)
				}
			}
		})
	}
}

func TestContentTypeDetection(t *testing.T) {
	theme := DefaultTheme()
	rm, err := NewRenderManager(theme, 80)
	if err != nil {
		t.Fatalf("Failed to create render manager: %v", err)
	}

	tests := []struct {
		name     string
		content  string
		expected ContentType
	}{
		{
			name:     "Plain text",
			content:  "This is just plain text without any formatting",
			expected: ContentTypePlain,
		},
		{
			name:     "Markdown headers",
			content:  "# This is a header\n\nSome content",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "Markdown list",
			content:  "- Item 1\n- Item 2\n- Item 3",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "Code block in markdown",
			content:  "Here's some code:\n\n```python\nprint('hello')\n```",
			expected: ContentTypeMixed,
		},
		{
			name:     "JSON object",
			content:  `{"key": "value", "number": 42}`,
			expected: ContentTypeJSON,
		},
		{
			name:     "JSON array",
			content:  `["item1", "item2", "item3"]`,
			expected: ContentTypeJSON,
		},
		{
			name:     "Content with thinking blocks",
			content:  "<thinking>Some thoughts</thinking>\n\nThe answer is 42.",
			expected: ContentTypeMixed,
		},
		{
			name:     "Code patterns",
			content:  "func main() {\n\tfmt.Println(\"Hello\")\n}",
			expected: ContentTypeCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := rm.DetectContentType(tt.content)
			if detected != tt.expected {
				t.Errorf("Expected content type %v, got %v for content: %s", tt.expected, detected, tt.content)
			}
		})
	}
}

func TestStreamingContent(t *testing.T) {
	theme := DefaultTheme()
	rm, err := NewRenderManager(theme, 80)
	if err != nil {
		t.Fatalf("Failed to create render manager: %v", err)
	}

	tests := []struct {
		name    string
		content string
		role    string
	}{
		{
			name:    "Partial content",
			content: "This is streaming content that's not yet compl",
			role:    "assistant",
		},
		{
			name:    "Partial thinking block",
			content: "<thinking>This is an incomplete thought",
			role:    "assistant",
		},
		{
			name:    "Complete thinking block in stream",
			content: "<thinking>Complete thought</thinking>\nAnd some content",
			role:    "assistant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := rm.RenderStreamingContent(tt.content, tt.role)

			// Should always have cursor at the end (with tview formatting)
			if !strings.HasSuffix(rendered, "█[-]") {
				t.Errorf("Streaming content should end with cursor, got: %s", rendered)
			}
		})
	}
}

func TestToolOutput(t *testing.T) {
	theme := DefaultTheme()
	rm, err := NewRenderManager(theme, 80)
	if err != nil {
		t.Fatalf("Failed to create render manager: %v", err)
	}

	tests := []struct {
		name     string
		toolName string
		output   string
	}{
		{
			name:     "Plain tool output",
			toolName: "bash",
			output:   "Command executed successfully",
		},
		{
			name:     "JSON tool output",
			toolName: "api_call",
			output:   `{"status": "success", "data": {"id": 123}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := rm.RenderToolOutput(tt.toolName, tt.output)

			// Should contain tool emoji and name
			if !strings.Contains(rendered, "🔧") {
				t.Errorf("Tool output should contain tool emoji")
			}
			if !strings.Contains(rendered, tt.toolName) {
				t.Errorf("Tool output should contain tool name '%s'", tt.toolName)
			}
		})
	}
}
