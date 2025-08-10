package providers

import (
	"context"
	"strings"

	"github.com/killallgit/ryan/pkg/stream/core"
	"github.com/tmc/langchaingo/llms"
)

// LangChainSource provides real streaming using LangChain
type LangChainSource struct {
	llm llms.Model
}

// NewLangChainSource creates a streaming source from LangChain LLM
func NewLangChainSource(llm llms.Model) *LangChainSource {
	return &LangChainSource{
		llm: llm,
	}
}

// Stream initiates a streaming response with a single prompt
func (l *LangChainSource) Stream(ctx context.Context, prompt string, handler core.Handler) error {
	messages := []core.Message{
		{Role: "user", Content: prompt},
	}
	return l.StreamWithHistory(ctx, messages, handler)
}

// StreamWithHistory streams with conversation history using real LangChain streaming
func (l *LangChainSource) StreamWithHistory(ctx context.Context, messages []core.Message, handler core.Handler) error {
	// Convert messages to LangChain format
	llmMessages := make([]llms.MessageContent, 0, len(messages))
	for _, msg := range messages {
		messageType := llms.ChatMessageTypeHuman
		switch msg.Role {
		case "system":
			messageType = llms.ChatMessageTypeSystem
		case "assistant":
			messageType = llms.ChatMessageTypeAI
		case "user":
			messageType = llms.ChatMessageTypeHuman
		case "tool":
			messageType = llms.ChatMessageTypeTool
		}
		llmMessages = append(llmMessages, llms.TextParts(messageType, msg.Content))
	}

	// Buffer to accumulate chunks for final content
	var contentBuilder strings.Builder

	// Create streaming function that calls our handler
	streamingFunc := func(ctx context.Context, chunk []byte) error {
		chunkStr := string(chunk)
		contentBuilder.WriteString(chunkStr)

		// Pass chunk to handler
		if err := handler.OnChunk(chunkStr); err != nil {
			return err
		}

		return nil
	}

	// Call LangChain with real streaming
	response, err := l.llm.GenerateContent(ctx, llmMessages, llms.WithStreamingFunc(streamingFunc))
	if err != nil {
		handler.OnError(err)
		return err
	}

	// Get final content
	finalContent := contentBuilder.String()

	// If no streaming occurred, use response content
	if finalContent == "" && len(response.Choices) > 0 {
		finalContent = response.Choices[0].Content
		// Send as single chunk if we didn't stream
		if err := handler.OnChunk(finalContent); err != nil {
			return err
		}
	}

	// Notify completion
	return handler.OnComplete(finalContent)
}

// Ensure LangChainSource implements Source
var _ core.Source = (*LangChainSource)(nil)
