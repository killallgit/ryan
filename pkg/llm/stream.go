package llm

import (
	"fmt"
	"strings"
	"sync"

	"github.com/killallgit/ryan/pkg/logger"
)

// BufferedStreamHandler buffers streaming content
type BufferedStreamHandler struct {
	buffer  strings.Builder
	onChunk func(string) error
	onError func(error)
	mu      sync.Mutex
}

// NewBufferedStreamHandler creates a new buffered stream handler
func NewBufferedStreamHandler() *BufferedStreamHandler {
	return &BufferedStreamHandler{}
}

// WithChunkCallback sets the chunk callback
func (h *BufferedStreamHandler) WithChunkCallback(fn func(string) error) *BufferedStreamHandler {
	h.onChunk = fn
	return h
}

// WithErrorCallback sets the error callback
func (h *BufferedStreamHandler) WithErrorCallback(fn func(error)) *BufferedStreamHandler {
	h.onError = fn
	return h
}

// OnChunk handles a new chunk of content
func (h *BufferedStreamHandler) OnChunk(chunk string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.buffer.WriteString(chunk)

	if h.onChunk != nil {
		return h.onChunk(chunk)
	}
	return nil
}

// OnComplete handles completion of streaming
func (h *BufferedStreamHandler) OnComplete(finalContent string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// If finalContent is provided, use it; otherwise use buffer
	if finalContent == "" {
		finalContent = h.buffer.String()
	}
	return nil
}

// OnError handles streaming errors
func (h *BufferedStreamHandler) OnError(err error) {
	if h.onError != nil {
		h.onError(err)
	}
}

// GetContent returns the buffered content
func (h *BufferedStreamHandler) GetContent() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.buffer.String()
}

// ConsoleStreamHandler outputs streaming content to console
type ConsoleStreamHandler struct {
	*BufferedStreamHandler
}

// NewConsoleStreamHandler creates a stream handler that outputs to console
func NewConsoleStreamHandler() *ConsoleStreamHandler {
	handler := &ConsoleStreamHandler{
		BufferedStreamHandler: NewBufferedStreamHandler(),
	}

	// Set chunk callback to print to console without overwriting
	handler.WithChunkCallback(func(chunk string) error {
		// Print chunk directly to stdout without any special formatting
		fmt.Print(chunk)
		return nil
	})

	handler.WithErrorCallback(func(err error) {
		logger.Error("LLM streaming error: %v", err)
	})

	return handler
}

// OnComplete handles completion
func (h *ConsoleStreamHandler) OnComplete(finalContent string) error {
	fmt.Println() // Add newline after streaming
	return h.BufferedStreamHandler.OnComplete(finalContent)
}
