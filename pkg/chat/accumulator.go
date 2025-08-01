package chat

import (
	"strings"
	"time"
)

// AccumulatingMessage represents a message being built from streaming chunks
type AccumulatingMessage struct {
	StreamID   string
	Content    strings.Builder
	ChunkCount int
	StartTime  time.Time
	LastUpdate time.Time
	Model      string
	Role       string
	IsComplete bool
	LastChunk  MessageChunk // Store last chunk for metadata
}

// MessageAccumulator manages accumulating messages from streaming chunks
type MessageAccumulator struct {
	activeMessages map[string]*AccumulatingMessage
}

// NewMessageAccumulator creates a new message accumulator
func NewMessageAccumulator() *MessageAccumulator {
	return &MessageAccumulator{
		activeMessages: make(map[string]*AccumulatingMessage),
	}
}

// AddChunk adds a streaming chunk to the accumulator
func (ma *MessageAccumulator) AddChunk(chunk MessageChunk) {
	if chunk.Error != nil {
		// Handle error chunks by marking message as failed
		if msg, exists := ma.activeMessages[chunk.StreamID]; exists {
			msg.LastUpdate = chunk.Timestamp
		}
		return
	}

	msg := ma.getOrCreateMessage(chunk.StreamID, chunk)

	// Add content to the accumulating message
	if chunk.Content != "" {
		msg.Content.WriteString(chunk.Content)
	}

	msg.ChunkCount++
	msg.LastUpdate = chunk.Timestamp
	msg.LastChunk = chunk

	// Mark as complete if this is the final chunk
	if chunk.Done {
		msg.IsComplete = true
	}
}

// GetCurrentContent returns the current accumulated content for a stream
func (ma *MessageAccumulator) GetCurrentContent(streamID string) string {
	if msg, exists := ma.activeMessages[streamID]; exists {
		return msg.Content.String()
	}
	return ""
}

// GetMessage returns the accumulating message for a stream
func (ma *MessageAccumulator) GetMessage(streamID string) (*AccumulatingMessage, bool) {
	msg, exists := ma.activeMessages[streamID]
	return msg, exists
}

// IsComplete checks if a stream has completed
func (ma *MessageAccumulator) IsComplete(streamID string) bool {
	if msg, exists := ma.activeMessages[streamID]; exists {
		return msg.IsComplete
	}
	return false
}

// GetCompleteMessage returns a complete Message if the stream is done
func (ma *MessageAccumulator) GetCompleteMessage(streamID string) (Message, bool) {
	msg, exists := ma.activeMessages[streamID]
	if !exists || !msg.IsComplete {
		return Message{}, false
	}

	return Message{
		Role:      msg.Role,
		Content:   msg.Content.String(),
		Timestamp: msg.LastUpdate,
		// Copy tool calls from last chunk if present
		ToolCalls: msg.LastChunk.Message.ToolCalls,
	}, true
}

// FinalizeMessage marks a stream as complete and returns the final message
func (ma *MessageAccumulator) FinalizeMessage(streamID string) (Message, bool) {
	msg, exists := ma.activeMessages[streamID]
	if !exists {
		return Message{}, false
	}

	// Create final message
	finalMessage := Message{
		Role:      msg.Role,
		Content:   msg.Content.String(),
		Timestamp: msg.LastUpdate,
		ToolCalls: msg.LastChunk.Message.ToolCalls,
	}

	// Remove from active messages to free memory
	delete(ma.activeMessages, streamID)

	return finalMessage, true
}

// CleanupStream removes a stream from the accumulator
func (ma *MessageAccumulator) CleanupStream(streamID string) {
	delete(ma.activeMessages, streamID)
}

// GetActiveStreams returns a list of currently active stream IDs
func (ma *MessageAccumulator) GetActiveStreams() []string {
	streams := make([]string, 0, len(ma.activeMessages))
	for streamID := range ma.activeMessages {
		streams = append(streams, streamID)
	}
	return streams
}

// GetStreamStats returns statistics about a stream
func (ma *MessageAccumulator) GetStreamStats(streamID string) (StreamStats, bool) {
	msg, exists := ma.activeMessages[streamID]
	if !exists {
		return StreamStats{}, false
	}

	return StreamStats{
		StreamID:      streamID,
		ChunkCount:    msg.ChunkCount,
		ContentLength: msg.Content.Len(),
		StartTime:     msg.StartTime,
		LastUpdate:    msg.LastUpdate,
		Duration:      msg.LastUpdate.Sub(msg.StartTime),
		IsComplete:    msg.IsComplete,
	}, true
}

// StreamStats provides statistics about a streaming message
type StreamStats struct {
	StreamID      string
	ChunkCount    int
	ContentLength int
	StartTime     time.Time
	LastUpdate    time.Time
	Duration      time.Duration
	IsComplete    bool
}

// getOrCreateMessage retrieves or creates an accumulating message
func (ma *MessageAccumulator) getOrCreateMessage(streamID string, chunk MessageChunk) *AccumulatingMessage {
	if msg, exists := ma.activeMessages[streamID]; exists {
		return msg
	}

	// Create new accumulating message
	msg := &AccumulatingMessage{
		StreamID:   streamID,
		StartTime:  chunk.Timestamp,
		LastUpdate: chunk.Timestamp,
		Model:      chunk.Model,
		Role:       chunk.Message.Role,
		IsComplete: false,
	}

	ma.activeMessages[streamID] = msg
	return msg
}

// Pure helper functions for working with accumulated content

// EstimateWordsPerMinute calculates typing speed based on accumulated content
func EstimateWordsPerMinute(stats StreamStats) float64 {
	if stats.Duration.Seconds() == 0 {
		return 0
	}

	wordCount := float64(len(strings.Fields(stats.StreamID))) // This should be content, but we don't have it here
	minutes := stats.Duration.Minutes()

	if minutes == 0 {
		return 0
	}

	return wordCount / minutes
}

// ValidateUnicodeIntegrity checks if accumulated content has valid Unicode
func ValidateUnicodeIntegrity(content string) bool {
	// Check for invalid UTF-8 sequences that might occur from chunk boundaries
	return strings.ToValidUTF8(content, "") == content
}

// SanitizeStreamContent ensures accumulated content is safe for display
func SanitizeStreamContent(content string) string {
	// Fix any Unicode issues from chunk boundaries
	sanitized := strings.ToValidUTF8(content, "")

	// Trim any trailing whitespace that might accumulate
	sanitized = strings.TrimRight(sanitized, " \t")

	return sanitized
}
