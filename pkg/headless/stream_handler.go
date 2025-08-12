package headless

import (
	"fmt"
	"strings"
	"sync"
)

// headlessStreamHandler handles streaming output for headless mode
// It prints chunks to stdout and accumulates the content
type headlessStreamHandler struct {
	content strings.Builder
	mu      sync.Mutex
}

// newHeadlessStreamHandler creates a handler for headless streaming output
func newHeadlessStreamHandler() *headlessStreamHandler {
	return &headlessStreamHandler{}
}

// OnChunk prints chunk to stdout and accumulates it
func (h *headlessStreamHandler) OnChunk(chunk []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Print to stdout for immediate output
	fmt.Print(string(chunk))

	// Also accumulate for final content
	h.content.Write(chunk)
	return nil
}

// OnComplete handles completion
func (h *headlessStreamHandler) OnComplete(finalContent string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if finalContent != "" && finalContent != h.content.String() {
		// If final content differs from accumulated content, use final
		h.content.Reset()
		h.content.WriteString(finalContent)
	}
	return nil
}

// OnError handles streaming errors
func (h *headlessStreamHandler) OnError(err error) {
	// In headless mode, errors are handled by the runner
	// This could log to stderr if needed
}

// GetContent returns the accumulated content
func (h *headlessStreamHandler) GetContent() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.content.String()
}
