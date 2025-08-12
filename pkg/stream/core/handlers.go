package core

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ChannelHandler sends stream events through a channel
type ChannelHandler struct {
	streamID string
	channel  chan<- Event
	buffer   strings.Builder
	mu       sync.Mutex
}

// NewChannelHandler creates a handler that sends events through a channel
func NewChannelHandler(streamID string, channel chan<- Event) *ChannelHandler {
	return &ChannelHandler{
		streamID: streamID,
		channel:  channel,
	}
}

// OnChunk sends chunk event
func (c *ChannelHandler) OnChunk(chunk []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.buffer.Write(chunk)

	select {
	case c.channel <- Event{
		StreamID:  c.streamID,
		State:     StateStreaming,
		Timestamp: time.Now(),
		Data:      string(chunk),
	}:
		return nil
	default:
		return fmt.Errorf("channel full or closed")
	}
}

// OnComplete sends completion event
func (c *ChannelHandler) OnComplete(finalContent string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if finalContent == "" {
		finalContent = c.buffer.String()
	}

	select {
	case c.channel <- Event{
		StreamID:  c.streamID,
		State:     StateComplete,
		Timestamp: time.Now(),
		Data:      finalContent,
	}:
		return nil
	default:
		return fmt.Errorf("channel full or closed")
	}
}

// OnError sends error event
func (c *ChannelHandler) OnError(err error) {
	select {
	case c.channel <- Event{
		StreamID:  c.streamID,
		State:     StateError,
		Timestamp: time.Now(),
		Data:      err,
	}:
	default:
		// Best effort - don't block on error
	}
}

// BufferHandler accumulates content in a buffer
type BufferHandler struct {
	buffer strings.Builder
	mu     sync.Mutex
}

// NewBufferHandler creates a handler that buffers content
func NewBufferHandler() *BufferHandler {
	return &BufferHandler{}
}

// OnChunk adds chunk to buffer
func (b *BufferHandler) OnChunk(chunk []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buffer.Write(chunk)
	return nil
}

// OnComplete finalizes buffer
func (b *BufferHandler) OnComplete(finalContent string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if finalContent != "" && finalContent != b.buffer.String() {
		b.buffer.Reset()
		b.buffer.WriteString(finalContent)
	}
	return nil
}

// OnError does nothing for buffer handler
func (b *BufferHandler) OnError(err error) {}

// GetContent returns the buffered content
func (b *BufferHandler) GetContent() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

// Reset clears the buffer
func (b *BufferHandler) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buffer.Reset()
}
