package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamParser_BasicThinkBlock(t *testing.T) {
	parser := NewStreamParser()

	// Test complete think block in one chunk
	segments := parser.ParseChunk("<think>This is thinking content</think>This is response")

	assert.Len(t, segments, 4)
	assert.Equal(t, "<think>", segments[0].Content)
	assert.Equal(t, FormatTypeNone, segments[0].Format)

	assert.Equal(t, "This is thinking content", segments[1].Content)
	assert.Equal(t, FormatTypeThink, segments[1].Format)
	assert.Equal(t, StyleThinkingText, segments[1].Style)

	assert.Equal(t, "</think>", segments[2].Content)
	assert.Equal(t, FormatTypeNone, segments[2].Format)

	assert.Equal(t, "This is response", segments[3].Content)
	assert.Equal(t, FormatTypeNone, segments[3].Format)
}

func TestStreamParser_PartialTags(t *testing.T) {
	tests := []struct {
		name     string
		chunks   []string
		expected []string
		formats  []FormatType
	}{
		{
			name:   "think tag split across chunks",
			chunks: []string{"<thi", "nk>Inside thinking</think>Outside"},
			expected: []string{
				"<think>",
				"Inside thinking",
				"</think>",
				"Outside",
			},
			formats: []FormatType{
				FormatTypeNone,
				FormatTypeThink,
				FormatTypeNone,
				FormatTypeNone,
			},
		},
		{
			name:   "closing tag split",
			chunks: []string{"<think>Content</thi", "nk>After"},
			expected: []string{
				"<think>",
				"Content",
				"</think>",
				"After",
			},
			formats: []FormatType{
				FormatTypeNone,
				FormatTypeThink,
				FormatTypeNone,
				FormatTypeNone,
			},
		},
		{
			name:   "multiple chunks in think block",
			chunks: []string{"<think>", "Part 1 ", "Part 2", "</think>", "Done"},
			expected: []string{
				"<think>",
				"Part 1 ",
				"Part 2",
				"</think>",
				"Done",
			},
			formats: []FormatType{
				FormatTypeNone,
				FormatTypeThink,
				FormatTypeThink,
				FormatTypeNone,
				FormatTypeNone,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewStreamParser()
			var allSegments []FormattedSegment

			// Parse all chunks
			for _, chunk := range tt.chunks {
				segments := parser.ParseChunk(chunk)
				allSegments = append(allSegments, segments...)
			}

			// Finalize to get any remaining content
			finalSegments := parser.Finalize()
			allSegments = append(allSegments, finalSegments...)

			// Verify segments
			assert.Len(t, allSegments, len(tt.expected))
			for i, segment := range allSegments {
				assert.Equal(t, tt.expected[i], segment.Content, "Segment %d content mismatch", i)
				assert.Equal(t, tt.formats[i], segment.Format, "Segment %d format mismatch", i)
			}
		})
	}
}

func TestStreamParser_NestedContent(t *testing.T) {
	parser := NewStreamParser()

	// Test thinking block with newlines and complex content
	content := `<think>
This is a multi-line
thinking block with various content
</think>
Now the response begins`

	segments := parser.ParseChunk(content)

	// Find thinking content segment
	var thinkSegment *FormattedSegment
	for _, seg := range segments {
		if seg.Format == FormatTypeThink {
			thinkSegment = &seg
			break
		}
	}

	assert.NotNil(t, thinkSegment)
	assert.Contains(t, thinkSegment.Content, "multi-line")
	assert.Equal(t, StyleThinkingText, thinkSegment.Style)
}

func TestStreamParser_CaseInsensitive(t *testing.T) {
	parser := NewStreamParser()

	// Test case variations
	segments := parser.ParseChunk("<THINK>Upper case</THINK><thinking>Mixed case</thinking>")

	// Count thinking segments
	thinkCount := 0
	for _, seg := range segments {
		if seg.Format == FormatTypeThink {
			thinkCount++
		}
	}

	assert.Equal(t, 2, thinkCount, "Should detect both think blocks regardless of case")
}

func TestStreamParser_IncompleteTagAtEnd(t *testing.T) {
	parser := NewStreamParser()

	// First chunk ends with incomplete tag
	segments1 := parser.ParseChunk("Some content <thi")
	assert.Len(t, segments1, 1)
	assert.Equal(t, "Some content ", segments1[0].Content)

	// Complete the tag in next chunk
	segments2 := parser.ParseChunk("nk>Thinking here</think>")

	// Should now get the thinking content
	hasThinkContent := false
	for _, seg := range segments2 {
		if seg.Format == FormatTypeThink {
			hasThinkContent = true
			assert.Equal(t, "Thinking here", seg.Content)
		}
	}
	assert.True(t, hasThinkContent, "Should have thinking content after completing tag")
}

func TestStreamParser_Reset(t *testing.T) {
	parser := NewStreamParser()

	// Parse some content
	parser.ParseChunk("<think>First")
	assert.True(t, parser.IsInThinkBlock())

	// Reset
	parser.Reset()
	assert.False(t, parser.IsInThinkBlock())
	assert.Empty(t, parser.buffer)

	// Parse new content
	segments := parser.ParseChunk("New content")
	assert.Len(t, segments, 1)
	assert.Equal(t, "New content", segments[0].Content)
	assert.Equal(t, FormatTypeNone, segments[0].Format)
}

func TestStreamParser_IsInThinkBlock(t *testing.T) {
	parser := NewStreamParser()

	assert.False(t, parser.IsInThinkBlock())

	parser.ParseChunk("<think>")
	assert.True(t, parser.IsInThinkBlock())

	parser.ParseChunk("content</think>")
	assert.False(t, parser.IsInThinkBlock())
}

func TestStreamParser_StyleApplication(t *testing.T) {
	parser := NewStreamParser()

	segments := parser.ParseChunk("<think>Styled content</think>")

	// Find the thinking content segment
	for _, seg := range segments {
		if seg.Format == FormatTypeThink {
			// Verify the style matches StyleThinkingText
			assert.Equal(t, StyleThinkingText, seg.Style, "Should have thinking text style")
		}
	}
}
