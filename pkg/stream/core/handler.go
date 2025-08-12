package core

import "github.com/killallgit/ryan/pkg/stream"

// Handler re-exports the stream.Handler interface for convenience
// This allows packages to import just stream/core and get the handler
type Handler = stream.Handler

// HandlerFunc re-exports for convenience
type HandlerFunc = stream.HandlerFunc
