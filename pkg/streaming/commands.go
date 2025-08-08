package streaming

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/killallgit/ryan/pkg/tokens"
	"github.com/killallgit/ryan/pkg/tui/chat/status"
	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/llms"
)

// StreamFromProvider creates a command to stream from a registered provider
// If sourceID is empty, it will use the router to determine the provider
func StreamFromProvider(mgr *Manager, sourceID string, prompt string, nodeType string) tea.Cmd {
	return func() tea.Msg {
		// Use router if sourceID not specified
		if sourceID == "" {
			sourceID = mgr.Router.Route(prompt)
		}

		source, exists := mgr.Registry.Get(sourceID)
		if !exists {
			return StreamEndMsg{
				StreamID: sourceID,
				Error:    fmt.Errorf("source %s not found", sourceID),
			}
		}

		// Create unique stream ID
		streamID := fmt.Sprintf("%s-%d", sourceID, time.Now().UnixNano())

		// Start the stream with prompt
		mgr.StartStream(streamID, source.Type, nodeType, prompt)

		// Return start message with prompt
		return StreamStartMsg{
			StreamID:   streamID,
			SourceType: nodeType,
			Prompt:     prompt,
		}
	}
}

// StreamContent handles the actual streaming from the provider
func StreamContent(mgr *Manager, streamID string, sourceID string, prompt string) tea.Cmd {
	return func() tea.Msg {
		source, exists := mgr.Registry.Get(sourceID)
		if !exists {
			return StreamEndMsg{
				StreamID: streamID,
				Error:    fmt.Errorf("source %s not found", sourceID),
			}
		}

		// If prompt is empty, try to get it from the stream
		if prompt == "" {
			if stream, exists := mgr.GetStream(streamID); exists {
				prompt = stream.Prompt
			}
		}

		// Validate prompt
		if prompt == "" {
			return StreamEndMsg{
				StreamID: streamID,
				Error:    fmt.Errorf("empty prompt provided"),
			}
		}

		ctx := context.Background()

		// Initialize token counter
		modelName := viper.GetString("ollama.default_model")
		tokenCounter, err := tokens.NewTokenCounter(modelName)
		if err != nil {
			// Log warning but continue without token counting
			fmt.Printf("Warning: Could not initialize token counter: %v\n", err)
			tokenCounter = nil
		}

		// Count and send input tokens
		if tokenCounter != nil && mgr.GetProgram() != nil {
			inputTokens := tokenCounter.CountTokens(prompt)
			mgr.GetProgram().Send(status.UpdateTokensMsg{Sent: inputTokens, Recv: 0})
		}

		// Track tokens received during streaming
		var receivedContent string
		lastTokenCount := 0

		// Generic streaming function that sends chunks and counts tokens
		streamFunc := func(ctx context.Context, chunk []byte) error {
			// Append to manager's buffer
			mgr.AppendToStream(streamID, string(chunk))

			// Track received content for token counting
			receivedContent += string(chunk)

			// Count tokens incrementally
			if tokenCounter != nil && mgr.GetProgram() != nil {
				currentTokens := tokenCounter.CountTokens(receivedContent)
				if currentTokens > lastTokenCount {
					tokenDiff := currentTokens - lastTokenCount
					mgr.GetProgram().Send(status.UpdateTokensMsg{Sent: 0, Recv: tokenDiff})
					lastTokenCount = currentTokens
				}
			}

			// Send chunk message (will be handled by update)
			// For now, we'll return the chunk in the final message
			// In a real implementation, we'd use a channel or program.Send
			return nil
		}

		// Call appropriate provider
		var genErr error
		switch provider := source.Provider.(type) {
		case *ollama.OllamaClient:
			messages := []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeHuman, prompt),
			}
			_, genErr = provider.GenerateContent(ctx, messages,
				llms.WithStreamingFunc(streamFunc))
		default:
			genErr = fmt.Errorf("unsupported provider type: %T", provider)
		}

		// Get final content from stream
		stream, _ := mgr.GetStream(streamID)
		finalContent := ""
		if stream != nil {
			finalContent = stream.Buffer.String()
		}

		// Final token count check
		if tokenCounter != nil && mgr.GetProgram() != nil && finalContent != "" {
			finalTokens := tokenCounter.CountTokens(finalContent)
			if finalTokens > lastTokenCount {
				tokenDiff := finalTokens - lastTokenCount
				mgr.GetProgram().Send(status.UpdateTokensMsg{Sent: 0, Recv: tokenDiff})
			}
		}

		// End the stream
		mgr.EndStream(streamID)

		return StreamEndMsg{
			StreamID:     streamID,
			Error:        genErr,
			FinalContent: finalContent,
		}
	}
}

// Message types for streaming (moved here to avoid circular import)
type StreamStartMsg struct {
	StreamID   string
	SourceType string
	Prompt     string
}

type StreamChunkMsg struct {
	StreamID   string
	Content    string
	SourceType string
}

type StreamEndMsg struct {
	StreamID     string
	Error        error
	FinalContent string
}
