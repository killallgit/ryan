package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MessageChunk represents a single chunk from a streaming response
type MessageChunk struct {
	ID        string    `json:"-"` // Unique chunk identifier
	Content   string    `json:"-"` // Incremental text content
	Done      bool      `json:"-"` // Stream completion indicator
	Timestamp time.Time `json:"-"` // When chunk was received
	StreamID  string    `json:"-"` // Which stream this belongs to
	Error     error     `json:"-"` // Error if chunk processing failed

	// Raw Ollama response fields
	Model              string        `json:"model"`
	CreatedAt          time.Time     `json:"created_at"`
	Message            Message       `json:"message"`
	DoneFlag           bool          `json:"done"`
	DoneReason         string        `json:"done_reason,omitempty"`
	PromptEvalCount    int           `json:"prompt_eval_count"`
	EvalCount          int           `json:"eval_count"`
	PromptEvalDuration time.Duration `json:"prompt_eval_duration"`
	EvalDuration       time.Duration `json:"eval_duration"`
}

// StreamingClient extends the basic client with streaming capabilities
type StreamingClient struct {
	*Client
}

// StreamingChatClient interface extends ChatClient with streaming support
type StreamingChatClient interface {
	ChatClient
	StreamMessage(ctx context.Context, req ChatRequest) (<-chan MessageChunk, error)
}

// NewStreamingClient creates a new streaming client
func NewStreamingClient(baseURL string) *StreamingClient {
	return &StreamingClient{
		Client: NewClient(baseURL),
	}
}

// NewStreamingClientWithTimeout creates a new streaming client with custom timeout
func NewStreamingClientWithTimeout(baseURL string, timeout time.Duration) *StreamingClient {
	return &StreamingClient{
		Client: NewClientWithTimeout(baseURL, timeout),
	}
}

// StreamMessage sends a chat request and returns a channel of message chunks
func (sc *StreamingClient) StreamMessage(ctx context.Context, req ChatRequest) (<-chan MessageChunk, error) {
	// Force streaming mode
	req.Stream = true

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/chat", sc.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := sc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Read the error response body for detailed error information
		errorBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		if err != nil {
			return nil, fmt.Errorf("request failed with status %d (failed to read error response: %w)", resp.StatusCode, err)
		}
		
		// Try to parse as JSON error response
		var errorResp struct {
			Error string `json:"error"`
		}
		
		if json.Unmarshal(errorBody, &errorResp) == nil && errorResp.Error != "" {
			return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, errorResp.Error)
		}
		
		// Fallback to raw body if JSON parsing fails
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(errorBody))
	}

	// Create channel for chunks
	chunks := make(chan MessageChunk, 100) // Buffered for performance
	streamID := generateStreamID()

	// Start goroutine to read stream
	go sc.readStream(ctx, resp.Body, chunks, streamID)

	return chunks, nil
}

// readStream processes the HTTP response body stream
func (sc *StreamingClient) readStream(ctx context.Context, body io.ReadCloser, chunks chan<- MessageChunk, streamID string) {
	defer close(chunks)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	chunkIndex := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			// Send cancellation error
			chunks <- MessageChunk{
				ID:       fmt.Sprintf("%s-cancelled", streamID),
				StreamID: streamID,
				Error:    ctx.Err(),
			}
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse JSON chunk
		var ollamaResp struct {
			Model              string        `json:"model"`
			CreatedAt          time.Time     `json:"created_at"`
			Message            Message       `json:"message"`
			Done               bool          `json:"done"`
			DoneReason         string        `json:"done_reason,omitempty"`
			PromptEvalCount    int           `json:"prompt_eval_count"`
			EvalCount          int           `json:"eval_count"`
			PromptEvalDuration time.Duration `json:"prompt_eval_duration"`
			EvalDuration       time.Duration `json:"eval_duration"`
		}

		if err := json.Unmarshal(line, &ollamaResp); err != nil {
			chunks <- MessageChunk{
				ID:       fmt.Sprintf("%s-error-%d", streamID, chunkIndex),
				StreamID: streamID,
				Error:    fmt.Errorf("failed to parse JSON chunk: %w", err),
			}
			continue
		}

		// Create message chunk
		chunk := MessageChunk{
			ID:                 fmt.Sprintf("%s-%d", streamID, chunkIndex),
			Content:            ollamaResp.Message.Content,
			Done:               ollamaResp.Done,
			Timestamp:          time.Now(),
			StreamID:           streamID,
			Model:              ollamaResp.Model,
			CreatedAt:          ollamaResp.CreatedAt,
			Message:            ollamaResp.Message,
			DoneFlag:           ollamaResp.Done,
			DoneReason:         ollamaResp.DoneReason,
			PromptEvalCount:    ollamaResp.PromptEvalCount,
			EvalCount:          ollamaResp.EvalCount,
			PromptEvalDuration: ollamaResp.PromptEvalDuration,
			EvalDuration:       ollamaResp.EvalDuration,
		}

		chunks <- chunk
		chunkIndex++

		// Exit on completion
		if ollamaResp.Done {
			return
		}
	}

	// Handle scanner errors
	if err := scanner.Err(); err != nil {
		chunks <- MessageChunk{
			ID:       fmt.Sprintf("%s-scan-error", streamID),
			StreamID: streamID,
			Error:    fmt.Errorf("stream reading error: %w", err),
		}
	}
}

// generateStreamID creates a unique identifier for this stream
func generateStreamID() string {
	return fmt.Sprintf("stream-%d", time.Now().UnixNano())
}

// CreateStreamingChatRequest creates a streaming-enabled chat request
func CreateStreamingChatRequest(conversation Conversation, userMessage string) ChatRequest {
	conv := AddMessage(conversation, NewUserMessage(userMessage))

	return ChatRequest{
		Model:    conv.Model,
		Messages: conv.Messages,
		Stream:   true,
	}
}

// CreateStreamingChatRequestWithTools creates a streaming-enabled chat request with tools
func CreateStreamingChatRequestWithTools(conversation Conversation, userMessage string, tools []map[string]any) ChatRequest {
	conv := AddMessage(conversation, NewUserMessage(userMessage))

	return ChatRequest{
		Model:    conv.Model,
		Messages: conv.Messages,
		Stream:   true,
		Tools:    tools,
	}
}
