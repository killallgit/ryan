package stream

import (
	"fmt"
	"os"
)

// ConsoleHandler writes streaming content to console
type ConsoleHandler struct{}

// NewConsoleHandler creates a new console handler
func NewConsoleHandler() *ConsoleHandler {
	return &ConsoleHandler{}
}

// OnChunk writes chunk to stdout
func (h *ConsoleHandler) OnChunk(chunk []byte) error {
	_, err := os.Stdout.Write(chunk)
	return err
}

// OnComplete signals completion
func (h *ConsoleHandler) OnComplete(finalContent string) error {
	// Print newline if final content doesn't end with one
	if len(finalContent) > 0 && finalContent[len(finalContent)-1] != '\n' {
		fmt.Println()
	}
	return nil
}

// OnError prints error to stderr
func (h *ConsoleHandler) OnError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

// Ensure ConsoleHandler implements Handler
var _ Handler = (*ConsoleHandler)(nil)
