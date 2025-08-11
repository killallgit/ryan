package stream

import "context"

// Handler is the unified interface for handling streaming responses.
// This provides a consistent contract for all streaming implementations,
// aligned with LangChain-Go's streaming patterns and Go best practices.
type Handler interface {
	// OnChunk is called when a new chunk of content is received.
	// Using []byte for efficiency and alignment with io.Writer patterns.
	OnChunk(chunk []byte) error

	// OnComplete is called when streaming is complete with final content.
	OnComplete(finalContent string) error

	// OnError is called when an error occurs during streaming.
	OnError(err error)
}

// HandlerFunc is a function adapter for Handler interface
type HandlerFunc struct {
	ChunkFunc    func(chunk []byte) error
	CompleteFunc func(finalContent string) error
	ErrorFunc    func(err error)
}

// OnChunk implements Handler
func (h HandlerFunc) OnChunk(chunk []byte) error {
	if h.ChunkFunc != nil {
		return h.ChunkFunc(chunk)
	}
	return nil
}

// OnComplete implements Handler
func (h HandlerFunc) OnComplete(finalContent string) error {
	if h.CompleteFunc != nil {
		return h.CompleteFunc(finalContent)
	}
	return nil
}

// OnError implements Handler
func (h HandlerFunc) OnError(err error) {
	if h.ErrorFunc != nil {
		h.ErrorFunc(err)
	}
}

// ToStreamingFunc converts a Handler to LangChain's streaming function signature.
// This enables seamless integration with LangChain-Go's llms.WithStreamingFunc.
func ToStreamingFunc(handler Handler) func(context.Context, []byte) error {
	return func(ctx context.Context, chunk []byte) error {
		select {
		case <-ctx.Done():
			handler.OnError(ctx.Err())
			return ctx.Err()
		default:
			return handler.OnChunk(chunk)
		}
	}
}

// FromStreamingFunc creates a Handler from LangChain's streaming function.
// This allows wrapping LangChain streaming functions as Handlers.
func FromStreamingFunc(fn func(context.Context, []byte) error, ctx context.Context) Handler {
	return HandlerFunc{
		ChunkFunc: func(chunk []byte) error {
			return fn(ctx, chunk)
		},
		CompleteFunc: func(finalContent string) error {
			return nil // Streaming funcs don't have explicit completion
		},
		ErrorFunc: func(err error) {
			// Error handling is typically done by the caller
		},
	}
}

// StringHandler wraps a Handler to work with string chunks for backward compatibility
type StringHandler struct {
	Handler Handler
}

// OnChunk converts string to []byte and forwards to wrapped handler
func (s *StringHandler) OnChunk(chunk string) error {
	return s.Handler.OnChunk([]byte(chunk))
}

// OnComplete forwards to wrapped handler
func (s *StringHandler) OnComplete(finalContent string) error {
	return s.Handler.OnComplete(finalContent)
}

// OnError forwards to wrapped handler
func (s *StringHandler) OnError(err error) {
	s.Handler.OnError(err)
}

// Ensure implementations satisfy the interface
var _ Handler = HandlerFunc{}
