package core

import "github.com/killallgit/ryan/pkg/stream"

// Handler is deprecated. Use stream.Handler instead.
// This type alias is provided for backward compatibility during migration.
// Deprecated: Use github.com/killallgit/ryan/pkg/stream.Handler
type Handler = stream.Handler

// HandlerFunc is deprecated. Use stream.HandlerFunc instead.
// Deprecated: Use github.com/killallgit/ryan/pkg/stream.HandlerFunc
type HandlerFunc = stream.HandlerFunc

// LegacyStringHandler provides compatibility for old string-based handlers
// Deprecated: Update code to use []byte chunks instead
type LegacyStringHandler struct {
	ChunkFunc    func(chunk string) error
	CompleteFunc func(finalContent string) error
	ErrorFunc    func(err error)
}

// OnChunk implements Handler by converting []byte to string
func (h LegacyStringHandler) OnChunk(chunk []byte) error {
	if h.ChunkFunc != nil {
		return h.ChunkFunc(string(chunk))
	}
	return nil
}

// OnComplete implements Handler
func (h LegacyStringHandler) OnComplete(finalContent string) error {
	if h.CompleteFunc != nil {
		return h.CompleteFunc(finalContent)
	}
	return nil
}

// OnError implements Handler
func (h LegacyStringHandler) OnError(err error) {
	if h.ErrorFunc != nil {
		h.ErrorFunc(err)
	}
}

// Ensure LegacyStringHandler implements Handler
var _ Handler = LegacyStringHandler{}
