package llm

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tokens"
	"github.com/killallgit/ryan/pkg/tui/chat/status"
)

// TokenTrackingAdapter wraps an LLM provider and tracks token usage
type TokenTrackingAdapter struct {
	provider     Provider
	tokenCounter *tokens.TokenCounter
	program      *tea.Program // Reference to the TUI program for sending updates
}

// NewTokenTrackingAdapter creates a new adapter that tracks tokens
func NewTokenTrackingAdapter(provider Provider, modelName string, program *tea.Program) (*TokenTrackingAdapter, error) {
	counter, err := tokens.NewTokenCounter(modelName)
	if err != nil {
		// Don't fail if token counter can't be initialized, just log warning
		logger.Warn("Could not initialize token counter: %v", err)
		counter = nil
	}

	return &TokenTrackingAdapter{
		provider:     provider,
		tokenCounter: counter,
		program:      program,
	}, nil
}

// Generate generates a response and tracks tokens
func (a *TokenTrackingAdapter) Generate(ctx context.Context, prompt string) (string, error) {
	// Count input tokens
	inputTokens := 0
	if a.tokenCounter != nil {
		inputTokens = a.tokenCounter.CountTokens(prompt)
		// Send token update for sent tokens
		if a.program != nil {
			a.program.Send(status.UpdateTokensMsg{Sent: inputTokens, Recv: 0})
		}
	}

	// Call underlying provider
	response, err := a.provider.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}

	// Count output tokens
	if a.tokenCounter != nil && a.program != nil {
		outputTokens := a.tokenCounter.CountTokens(response)
		// Send token update for received tokens
		a.program.Send(status.UpdateTokensMsg{Sent: 0, Recv: outputTokens})
	}

	return response, nil
}

// GenerateStream generates a streaming response and tracks tokens
func (a *TokenTrackingAdapter) GenerateStream(ctx context.Context, prompt string, handler StreamHandler) error {
	// Count input tokens
	if a.tokenCounter != nil && a.program != nil {
		inputTokens := a.tokenCounter.CountTokens(prompt)
		a.program.Send(status.UpdateTokensMsg{Sent: inputTokens, Recv: 0})
	}

	// Create a wrapper handler that tracks tokens
	wrappedHandler := &tokenTrackingStreamHandler{
		handler:    handler,
		adapter:    a,
		buffer:     "",
		lastTokens: 0,
	}

	return a.provider.GenerateStream(ctx, prompt, wrappedHandler)
}

// GetName returns the provider name
func (a *TokenTrackingAdapter) GetName() string {
	return a.provider.GetName()
}

// GetModel returns the model name
func (a *TokenTrackingAdapter) GetModel() string {
	return a.provider.GetModel()
}

// tokenTrackingStreamHandler wraps a stream handler to track tokens incrementally
type tokenTrackingStreamHandler struct {
	handler    StreamHandler
	adapter    *TokenTrackingAdapter
	buffer     string
	lastTokens int
}

func (h *tokenTrackingStreamHandler) OnChunk(chunk []byte) error {
	// Accumulate chunks
	h.buffer += string(chunk)

	// Count tokens in accumulated buffer
	if h.adapter.tokenCounter != nil && h.adapter.program != nil {
		currentTokens := h.adapter.tokenCounter.CountTokens(h.buffer)
		// Only send update if token count changed
		if currentTokens > h.lastTokens {
			tokenDiff := currentTokens - h.lastTokens
			h.adapter.program.Send(status.UpdateTokensMsg{Sent: 0, Recv: tokenDiff})
			h.lastTokens = currentTokens
		}
	}

	// Forward to original handler
	return h.handler.OnChunk(chunk)
}

func (h *tokenTrackingStreamHandler) OnComplete(finalText string) error {
	// Final token count (in case there's any discrepancy)
	if h.adapter.tokenCounter != nil && h.adapter.program != nil && finalText != "" {
		finalTokens := h.adapter.tokenCounter.CountTokens(finalText)
		if finalTokens > h.lastTokens {
			tokenDiff := finalTokens - h.lastTokens
			h.adapter.program.Send(status.UpdateTokensMsg{Sent: 0, Recv: tokenDiff})
		}
	}

	return h.handler.OnComplete(finalText)
}

func (h *tokenTrackingStreamHandler) OnError(err error) {
	h.handler.OnError(err)
}

// SetProgram sets the tea.Program reference for sending updates
func (a *TokenTrackingAdapter) SetProgram(program *tea.Program) {
	a.program = program
}
