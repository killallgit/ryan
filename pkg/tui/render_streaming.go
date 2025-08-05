package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
)

// StreamingRenderer handles incremental rendering of streaming content
type StreamingRenderer struct {
	lastRenderedLines int
	lastContent       string
}

// NewStreamingRenderer creates a new streaming renderer
func NewStreamingRenderer() *StreamingRenderer {
	return &StreamingRenderer{}
}

// RenderStreamingIncremental renders only the new content in a streaming message
// without clearing the entire area
func (sr *StreamingRenderer) RenderStreamingIncremental(
	screen tcell.Screen,
	display MessageDisplay,
	area Rect,
	streamingContent string,
	streamingThinking bool,
) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	// If this is new content or content has changed significantly, do a full render
	if sr.lastContent == "" || !strings.HasPrefix(streamingContent, sr.lastContent) {
		// Full render needed
		clearArea(screen, area)
		RenderMessagesWithStreamingState(screen, display, area, SpinnerComponent{}, streamingThinking)
		sr.lastContent = streamingContent
		sr.lastRenderedLines = sr.calculateRenderedLines(display, area.Width, streamingThinking)
		return
	}

	// Incremental render - only render the new content
	newContent := streamingContent[len(sr.lastContent):]
	if newContent == "" {
		return // Nothing new to render
	}

	// Calculate where to start rendering new content
	formatter := NewUnifiedFormatter()
	
	// Format only the new content
	newLines := formatter.Format(newContent, area.Width)
	
	// Calculate the Y position for new content, accounting for scroll
	startY := area.Y + sr.lastRenderedLines - display.Scroll
	
	// Render new lines
	for i, line := range newLines {
		y := startY + i
		if y < area.Y {
			continue // Above visible area due to scroll
		}
		if y >= area.Y+area.Height {
			break // Below visible area
		}
		
		// Apply appropriate styling
		style := StyleAssistantText
		if streamingThinking {
			style = StyleThinkingText
		}
		// Use the formatted line's style if it has been set
		if line.Style != tcell.StyleDefault {
			style = line.Style
		}
		
		// Clear the line first (in case of wrapped text)
		for x := area.X; x < area.X+area.Width; x++ {
			screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
		
		// Render the content
		content := line.Content
		if line.Indent > 0 {
			content = strings.Repeat(" ", line.Indent) + content
		}
		renderText(screen, area.X, y, content, style)
	}
	
	// Update state
	sr.lastContent = streamingContent
	sr.lastRenderedLines += len(newLines)
}

// Reset clears the streaming renderer state
func (sr *StreamingRenderer) Reset() {
	sr.lastContent = ""
	sr.lastRenderedLines = 0
}

// calculateRenderedLines calculates how many lines have been rendered
func (sr *StreamingRenderer) calculateRenderedLines(display MessageDisplay, width int, streamingThinking bool) int {
	totalLines := 0
	formatter := NewUnifiedFormatter()
	
	for i, msg := range display.Messages {
		if msg.Content != "" {
			formattedLines := formatter.Format(msg.Content, width)
			totalLines += len(formattedLines)
		}
		
		// Add empty line between messages (except after last)
		if i < len(display.Messages)-1 {
			totalLines++
		}
	}
	
	return totalLines
}

// RenderMessagesWithIncrementalStreaming provides incremental streaming updates
// This is a wrapper that manages when to use incremental vs full rendering
func RenderMessagesWithIncrementalStreaming(
	screen tcell.Screen,
	display MessageDisplay,
	area Rect,
	spinner SpinnerComponent,
	streamingThinking bool,
	streamingRenderer *StreamingRenderer,
	isStreamingUpdate bool,
) {
	if !isStreamingUpdate || streamingRenderer == nil {
		// Not a streaming update or no renderer, do full render
		RenderMessagesWithSpinnerAndStreaming(screen, display, area, spinner, streamingThinking)
		return
	}
	
	// Use incremental rendering for streaming updates
	// Extract streaming content from the last message
	if len(display.Messages) > 0 {
		lastMsg := display.Messages[len(display.Messages)-1]
		if lastMsg.Role == chat.RoleAssistant {
			streamingRenderer.RenderStreamingIncremental(
				screen,
				display,
				area,
				lastMsg.Content,
				streamingThinking,
			)
		}
	}
}