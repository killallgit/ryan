package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/logger"
)

// FormatType represents different formatting types
type FormatType int

const (
	FormatTypeNone FormatType = iota
	FormatTypeThink
	FormatTypeBold
	FormatTypeItalic
	FormatTypeCode
	FormatTypeCodeBlock
	FormatTypeHeader
	FormatTypeList
)

// FormattedSegment represents a segment of text with formatting
type FormattedSegment struct {
	Content string
	Format  FormatType
	Style   tcell.Style
	Hidden  bool // For hidden think blocks after streaming
}

// FormatState tracks the state of a formatting region
type FormatState struct {
	Type       FormatType
	StartPos   int
	Tag        string // The opening tag (e.g., "<think>", "**", etc.)
	Attributes map[string]string
}

// StreamParser handles stateful parsing of streaming content with formatting
type StreamParser struct {
	buffer         string             // Buffer for incomplete tags
	formatStack    []FormatState      // Stack for nested formatting
	pendingContent strings.Builder    // Content being accumulated
	segments       []FormattedSegment // Parsed segments
	position       int                // Current position in the stream
}

// NewStreamParser creates a new streaming parser
func NewStreamParser() *StreamParser {
	return &StreamParser{
		formatStack: make([]FormatState, 0),
		segments:    make([]FormattedSegment, 0),
	}
}

// ParseChunk processes a chunk of streaming content and returns formatted segments
func (sp *StreamParser) ParseChunk(chunk string) []FormattedSegment {
	log := logger.WithComponent("stream_parser")
	log.Debug("ParseChunk called", "chunk_length", len(chunk), "buffer_length", len(sp.buffer))

	// Add chunk to buffer
	sp.buffer += chunk

	// Process the buffer
	newSegments := sp.processBuffer()

	log.Debug("ParseChunk result", "new_segments", len(newSegments), "remaining_buffer", len(sp.buffer))
	return newSegments
}

// processBuffer processes the current buffer and extracts complete segments
func (sp *StreamParser) processBuffer() []FormattedSegment {
	var newSegments []FormattedSegment
	processed := 0

	for processed < len(sp.buffer) {
		// Check if we're currently in a formatted block
		currentFormat := sp.getCurrentFormat()

		// Look for the next tag
		tagStart := sp.findNextTag(processed)
		
		if tagStart == -1 {
			// No more tags in buffer
			remaining := sp.buffer[processed:]
			if len(remaining) > 0 {
				// Check if this might be the start of an incomplete tag
				if strings.HasSuffix(remaining, "<") || 
				   strings.HasSuffix(remaining, "<t") ||
				   strings.HasSuffix(remaining, "<th") ||
				   strings.HasSuffix(remaining, "<thi") ||
				   strings.HasSuffix(remaining, "<thin") ||
				   strings.HasSuffix(remaining, "</") ||
				   strings.HasSuffix(remaining, "</t") ||
				   strings.HasSuffix(remaining, "</th") ||
				   strings.HasSuffix(remaining, "</thi") ||
				   strings.HasSuffix(remaining, "</thin") {
					// Keep potential incomplete tag in buffer
					sp.buffer = remaining
					return newSegments
				}
				
				// Add remaining content as a segment
				newSegments = append(newSegments, FormattedSegment{
					Content: remaining,
					Format:  currentFormat.Type,
					Style:   sp.getStyleForFormat(currentFormat.Type),
				})
			}
			sp.buffer = ""
			break
		}

		// Add content before the tag
		if tagStart > processed {
			content := sp.buffer[processed:tagStart]
			if content != "" {
				newSegments = append(newSegments, FormattedSegment{
					Content: content,
					Format:  currentFormat.Type,
					Style:   sp.getStyleForFormat(currentFormat.Type),
				})
			}
		}

		// Try to process the tag
		tag, tagEnd, tagProcessed := sp.processTag(tagStart)
		if tagProcessed {
			// Add the tag as a segment with FormatTypeNone
			newSegments = append(newSegments, FormattedSegment{
				Content: tag,
				Format:  FormatTypeNone,
				Style:   tcell.StyleDefault,
			})
			processed = tagEnd
		} else {
			// Tag is incomplete, keep remaining buffer
			sp.buffer = sp.buffer[tagStart:]
			return newSegments
		}
	}

	// Update buffer to remove processed content
	if processed > 0 && processed <= len(sp.buffer) {
		sp.buffer = sp.buffer[processed:]
	}

	return newSegments
}

// findNextTag finds the next potential tag start position
func (sp *StreamParser) findNextTag(startPos int) int {
	if startPos >= len(sp.buffer) {
		return -1
	}

	// Look for think tags
	if idx := strings.Index(sp.buffer[startPos:], "<"); idx != -1 {
		return startPos + idx
	}

	// Future: Look for markdown formatting (**, *, `, etc.)

	return -1
}

// processTag attempts to process a tag starting at the given position
func (sp *StreamParser) processTag(pos int) (tag string, newPos int, processed bool) {
	remainingBuffer := sp.buffer[pos:]
	
	// Check if we have enough characters to identify a tag
	if len(remainingBuffer) < 2 {
		return "", pos, false // Need at least "<x"
	}
	
	// Check for opening think tags
	if sp.couldBeTag(remainingBuffer, "<think>") {
		if len(remainingBuffer) >= 7 { // Full "<think>"
			if sp.matchTag(pos, "<think>") {
				sp.pushFormat(FormatState{
					Type:     FormatTypeThink,
					StartPos: sp.position + pos,
					Tag:      "<think>",
				})
				return "<think>", pos + 7, true
			}
		} else {
			return "", pos, false // Incomplete tag
		}
	}
	
	if sp.couldBeTag(remainingBuffer, "<thinking>") {
		if len(remainingBuffer) >= 10 { // Full "<thinking>"
			if sp.matchTag(pos, "<thinking>") {
				sp.pushFormat(FormatState{
					Type:     FormatTypeThink,
					StartPos: sp.position + pos,
					Tag:      "<thinking>",
				})
				return "<thinking>", pos + 10, true
			}
		} else {
			return "", pos, false // Incomplete tag
		}
	}

	// Check for closing think tags
	if sp.couldBeTag(remainingBuffer, "</think>") {
		if len(remainingBuffer) >= 8 { // Full "</think>"
			if sp.matchTag(pos, "</think>") && sp.isInFormat(FormatTypeThink) {
				sp.popFormat(FormatTypeThink)
				return "</think>", pos + 8, true
			}
		} else {
			return "", pos, false // Incomplete tag
		}
	}
	
	if sp.couldBeTag(remainingBuffer, "</thinking>") {
		if len(remainingBuffer) >= 11 { // Full "</thinking>"
			if sp.matchTag(pos, "</thinking>") && sp.isInFormat(FormatTypeThink) {
				sp.popFormat(FormatTypeThink)
				return "</thinking>", pos + 11, true
			}
		} else {
			return "", pos, false // Incomplete tag
		}
	}

	// Not a think tag, treat the '<' as regular content
	return "", pos + 1, false
}

// couldBeTag checks if the buffer could be the start of a specific tag
func (sp *StreamParser) couldBeTag(buffer string, tag string) bool {
	// If buffer is longer than tag, check if it starts with the tag
	if len(buffer) >= len(tag) {
		return strings.HasPrefix(strings.ToLower(buffer), strings.ToLower(tag))
	}
	// If buffer is shorter, check if it's a prefix of the tag
	lowerBuffer := strings.ToLower(buffer)
	lowerTag := strings.ToLower(tag[:len(buffer)])
	return lowerBuffer == lowerTag
}

// matchTag checks if a tag matches at the current position
func (sp *StreamParser) matchTag(pos int, tag string) bool {
	if pos+len(tag) > len(sp.buffer) {
		return false
	}
	return strings.ToLower(sp.buffer[pos:pos+len(tag)]) == strings.ToLower(tag)
}

// getMatchedTag returns the matched tag at position
func (sp *StreamParser) getMatchedTag(pos int) string {
	tags := []string{"<think>", "<thinking>", "</think>", "</thinking>"}
	for _, tag := range tags {
		if sp.matchTag(pos, tag) {
			return tag
		}
	}
	return ""
}

// getCurrentFormat returns the current active format
func (sp *StreamParser) getCurrentFormat() FormatState {
	if len(sp.formatStack) > 0 {
		return sp.formatStack[len(sp.formatStack)-1]
	}
	return FormatState{Type: FormatTypeNone}
}

// pushFormat adds a new format to the stack
func (sp *StreamParser) pushFormat(format FormatState) {
	sp.formatStack = append(sp.formatStack, format)
}

// popFormat removes a format from the stack
func (sp *StreamParser) popFormat(formatType FormatType) {
	// Find and remove the matching format
	for i := len(sp.formatStack) - 1; i >= 0; i-- {
		if sp.formatStack[i].Type == formatType {
			sp.formatStack = append(sp.formatStack[:i], sp.formatStack[i+1:]...)
			break
		}
	}
}

// isInFormat checks if we're currently in a specific format
func (sp *StreamParser) isInFormat(formatType FormatType) bool {
	for _, format := range sp.formatStack {
		if format.Type == formatType {
			return true
		}
	}
	return false
}

// getStyleForFormat returns the appropriate style for a format type
func (sp *StreamParser) getStyleForFormat(formatType FormatType) tcell.Style {
	switch formatType {
	case FormatTypeThink:
		return StyleThinkingText // Dim + Italic
	case FormatTypeBold:
		return tcell.StyleDefault.Bold(true)
	case FormatTypeItalic:
		return tcell.StyleDefault.Italic(true)
	case FormatTypeCode:
		return tcell.StyleDefault.Foreground(tcell.ColorWhite)
	default:
		return tcell.StyleDefault
	}
}

// Finalize processes any remaining buffer content
func (sp *StreamParser) Finalize() []FormattedSegment {
	// Process any remaining buffer
	segments := sp.processBuffer()

	// Add any remaining content as plain text
	if sp.buffer != "" {
		segments = append(segments, FormattedSegment{
			Content: sp.buffer,
			Format:  FormatTypeNone,
			Style:   tcell.StyleDefault,
		})
		sp.buffer = ""
	}

	return segments
}

// Reset clears the parser state
func (sp *StreamParser) Reset() {
	sp.buffer = ""
	sp.formatStack = sp.formatStack[:0]
	sp.segments = sp.segments[:0]
	sp.pendingContent.Reset()
	sp.position = 0
}

// IsInThinkBlock returns true if currently parsing inside a think block
func (sp *StreamParser) IsInThinkBlock() bool {
	return sp.isInFormat(FormatTypeThink)
}