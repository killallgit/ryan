package llm

// This file is deprecated. All streaming functionality has been moved to pkg/stream.
// The types below are provided as aliases for backward compatibility during migration.

import (
	"fmt"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/stream"
)

// BufferedStreamHandler is deprecated. Use stream.WriterHandler or create a custom handler.
// Deprecated: Use github.com/killallgit/ryan/pkg/stream package
type BufferedStreamHandler = stream.HandlerFunc

// NewBufferedStreamHandler creates a new buffered stream handler
// Deprecated: Use stream.NewWriterHandler or stream.HandlerFunc
func NewBufferedStreamHandler() *stream.HandlerFunc {
	logger.Warn("BufferedStreamHandler is deprecated. Use stream.WriterHandler instead.")
	return &stream.HandlerFunc{}
}

// ConsoleStreamHandler is deprecated. Use stream/core.ConsoleHandler
// Deprecated: Use github.com/killallgit/ryan/pkg/stream/core.ConsoleHandler
type ConsoleStreamHandler struct {
	*stream.HandlerFunc
}

// NewConsoleStreamHandler creates a stream handler that outputs to console
// Deprecated: Use stream/core.NewConsoleHandler
func NewConsoleStreamHandler() *ConsoleStreamHandler {
	logger.Warn("ConsoleStreamHandler is deprecated. Use stream/core.ConsoleHandler instead.")
	handler := &ConsoleStreamHandler{
		HandlerFunc: &stream.HandlerFunc{
			ChunkFunc: func(chunk []byte) error {
				fmt.Print(string(chunk))
				return nil
			},
		},
	}
	return handler
}

// OnComplete handles completion
func (h *ConsoleStreamHandler) OnComplete(finalContent string) error {
	fmt.Println() // Add newline after streaming
	return nil
}
