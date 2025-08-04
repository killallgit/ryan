package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleFormatter_FormatContentSegments(t *testing.T) {
	formatter := NewSimpleFormatter(80)

	tests := []struct {
		name     string
		segments []ContentSegment
		expected int // Expected number of formatted lines
	}{
		{
			name: "code block formatting",
			segments: []ContentSegment{
				{
					Type:     ContentTypeCodeBlock,
					Content:  "function hello() {\n  console.log('Hello');\n}",
					Language: "javascript",
				},
			},
			expected: 6, // Empty line + language label + 3 content lines + empty line
		},
		{
			name: "header formatting",
			segments: []ContentSegment{
				{
					Type:    ContentTypeHeader,
					Content: "Main Header",
					Level:   1,
				},
			},
			expected: 2, // Empty line + header (no underline in clean format)
		},
		{
			name: "list formatting",
			segments: []ContentSegment{
				{
					Type:    ContentTypeList,
					Content: "First item",
					Level:   1,
				},
				{
					Type:    ContentTypeList,
					Content: "Second item",
					Level:   1,
				},
			},
			expected: 3, // Two list items + spacing
		},
		{
			name: "mixed content",
			segments: []ContentSegment{
				{
					Type:    ContentTypeText,
					Content: "Regular text",
				},
				{
					Type:     ContentTypeCodeBlock,
					Content:  "console.log('test');",
					Language: "js",
				},
			},
			expected: 6, // Text line + empty line + language label + code line + empty line
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatContentSegments(tt.segments)
			assert.Len(t, result, tt.expected, "Unexpected number of formatted lines")

			// Verify that all lines have content or are intentionally empty
			for i, line := range result {
				assert.NotNil(t, line.Style, "Line %d should have a style", i)
				// Content can be empty for spacing/borders, but style should always be set
			}
		})
	}
}

func TestDetectContentTypes(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[ContentType]bool
	}{
		{
			name:    "code block detection",
			content: "Here's some code:\n```go\nfunc main() {}\n```",
			expected: map[ContentType]bool{
				ContentTypeText:      true,
				ContentTypeCodeBlock: true,
			},
		},
		{
			name:    "inline code detection",
			content: "Use the `fmt.Println` function",
			expected: map[ContentType]bool{
				ContentTypeText:       true,
				ContentTypeInlineCode: true,
			},
		},
		{
			name:    "header detection",
			content: "# Main Title\n## Subtitle",
			expected: map[ContentType]bool{
				ContentTypeText:   true,
				ContentTypeHeader: true,
			},
		},
		{
			name:    "list detection",
			content: "Items:\n- First\n- Second",
			expected: map[ContentType]bool{
				ContentTypeText: true,
				ContentTypeList: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectContentTypes(tt.content)

			for expectedType, shouldBePresent := range tt.expected {
				if shouldBePresent {
					assert.True(t, result[expectedType], "Content type %v should be detected", expectedType)
				} else {
					assert.False(t, result[expectedType], "Content type %v should not be detected", expectedType)
				}
			}
		})
	}
}

func TestShouldUseSimpleFormatting(t *testing.T) {
	tests := []struct {
		name         string
		contentTypes map[ContentType]bool
		expected     bool
	}{
		{
			name: "should use simple formatting for code blocks",
			contentTypes: map[ContentType]bool{
				ContentTypeCodeBlock: true,
			},
			expected: true,
		},
		{
			name: "should use simple formatting for headers",
			contentTypes: map[ContentType]bool{
				ContentTypeHeader: true,
			},
			expected: true,
		},
		{
			name: "should not use simple formatting for plain text",
			contentTypes: map[ContentType]bool{
				ContentTypeText: true,
			},
			expected: false,
		},
		{
			name: "should use simple formatting for inline code with other content",
			contentTypes: map[ContentType]bool{
				ContentTypeText:       true,
				ContentTypeInlineCode: true,
			},
			expected: true,
		},
		{
			name: "should not use simple formatting for inline code alone",
			contentTypes: map[ContentType]bool{
				ContentTypeInlineCode: true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldUseSimpleFormatting(tt.contentTypes)
			assert.Equal(t, tt.expected, result)
		})
	}
}
