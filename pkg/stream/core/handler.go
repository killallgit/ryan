package core

// Handler is the core interface for handling streaming responses
// This provides a unified contract for all streaming implementations
type Handler interface {
	// OnChunk is called when a new chunk of content is received
	OnChunk(chunk string) error

	// OnComplete is called when streaming is complete with final content
	OnComplete(finalContent string) error

	// OnError is called when an error occurs during streaming
	OnError(err error)
}

// HandlerFunc is a function adapter for Handler interface
type HandlerFunc struct {
	ChunkFunc    func(chunk string) error
	CompleteFunc func(finalContent string) error
	ErrorFunc    func(err error)
}

// OnChunk implements Handler
func (h HandlerFunc) OnChunk(chunk string) error {
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

// Ensure HandlerFunc implements Handler
var _ Handler = HandlerFunc{}
