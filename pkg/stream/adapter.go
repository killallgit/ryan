package stream

import (
	"bytes"
	"context"
	"io"
	"strings"
)

// WriterHandler adapts an io.Writer to implement the Handler interface.
// This enables using standard Go io.Writer implementations as stream handlers.
type WriterHandler struct {
	writer io.Writer
	buffer bytes.Buffer
}

// NewWriterHandler creates a new handler that writes to an io.Writer
func NewWriterHandler(w io.Writer) *WriterHandler {
	return &WriterHandler{
		writer: w,
	}
}

// OnChunk writes the chunk to the underlying writer
func (w *WriterHandler) OnChunk(chunk []byte) error {
	// Write to both the writer and buffer
	if _, err := w.writer.Write(chunk); err != nil {
		return err
	}
	w.buffer.Write(chunk)
	return nil
}

// OnComplete handles completion (no-op for writers)
func (w *WriterHandler) OnComplete(finalContent string) error {
	// If finalContent differs from buffer, write the difference
	if finalContent != "" && finalContent != w.buffer.String() {
		diff := strings.TrimPrefix(finalContent, w.buffer.String())
		if diff != "" {
			_, err := w.writer.Write([]byte(diff))
			return err
		}
	}
	return nil
}

// OnError handles errors (no-op for basic writers)
func (w *WriterHandler) OnError(err error) {
	// Writers typically don't have error callbacks
	// Errors are returned from Write operations
}

// GetContent returns the accumulated content
func (w *WriterHandler) GetContent() string {
	return w.buffer.String()
}

// PipeHandler creates a pipe between a source and handler.
// This enables advanced streaming patterns like transformations and multiplexing.
func PipeHandler(ctx context.Context, handler Handler) (*io.PipeReader, Handler) {
	pr, pw := io.Pipe()

	// Create a handler that writes to the pipe
	pipeHandler := HandlerFunc{
		ChunkFunc: func(chunk []byte) error {
			select {
			case <-ctx.Done():
				pw.CloseWithError(ctx.Err())
				return ctx.Err()
			default:
				_, err := pw.Write(chunk)
				if err != nil {
					handler.OnError(err)
				}
				// Also forward to the original handler
				return handler.OnChunk(chunk)
			}
		},
		CompleteFunc: func(finalContent string) error {
			pw.Close()
			return handler.OnComplete(finalContent)
		},
		ErrorFunc: func(err error) {
			pw.CloseWithError(err)
			handler.OnError(err)
		},
	}

	return pr, pipeHandler
}

// MultiHandler broadcasts chunks to multiple handlers.
// Similar to io.MultiWriter but for our Handler interface.
type MultiHandler struct {
	handlers []Handler
}

// NewMultiHandler creates a handler that forwards to multiple handlers
func NewMultiHandler(handlers ...Handler) *MultiHandler {
	return &MultiHandler{
		handlers: handlers,
	}
}

// OnChunk forwards the chunk to all handlers
func (m *MultiHandler) OnChunk(chunk []byte) error {
	for _, h := range m.handlers {
		if err := h.OnChunk(chunk); err != nil {
			return err
		}
	}
	return nil
}

// OnComplete forwards completion to all handlers
func (m *MultiHandler) OnComplete(finalContent string) error {
	for _, h := range m.handlers {
		if err := h.OnComplete(finalContent); err != nil {
			return err
		}
	}
	return nil
}

// OnError forwards errors to all handlers
func (m *MultiHandler) OnError(err error) {
	for _, h := range m.handlers {
		h.OnError(err)
	}
}

// Ensure implementations satisfy the interface
var (
	_ Handler = (*WriterHandler)(nil)
	_ Handler = (*MultiHandler)(nil)
)
