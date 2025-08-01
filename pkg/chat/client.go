package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type ChatRequest struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	Stream   bool             `json:"stream"`
	Tools    []map[string]any `json:"tools,omitempty"`
}

type ChatResponse struct {
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

type ChatClient interface {
	SendMessage(req ChatRequest) (Message, error)
	SendMessageWithResponse(req ChatRequest) (ChatResponse, error)
}

func NewClient(baseURL string) *Client {
	return NewClientWithTimeout(baseURL, 60*time.Second)
}

func NewClientWithTimeout(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) SendMessage(req ChatRequest) (Message, error) {
	resp, err := c.SendMessageWithResponse(req)
	if err != nil {
		return Message{}, err
	}
	return resp.Message, nil
}

func (c *Client) SendMessageWithResponse(req ChatRequest) (ChatResponse, error) {
	req.Stream = false

	reqBody, err := json.Marshal(req)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/chat", c.baseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read the error response body for detailed error information
		errorBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return ChatResponse{}, fmt.Errorf("request failed with status %d (failed to read error response: %w)", resp.StatusCode, err)
		}

		// Try to parse as JSON error response
		var errorResp struct {
			Error string `json:"error"`
		}

		if json.Unmarshal(errorBody, &errorResp) == nil && errorResp.Error != "" {
			return ChatResponse{}, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, errorResp.Error)
		}

		// Fallback to raw body if JSON parsing fails
		return ChatResponse{}, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(errorBody))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return ChatResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return chatResp, nil
}

func CreateChatRequest(conversation Conversation, userMessage string) ChatRequest {
	conv := AddMessage(conversation, NewUserMessage(userMessage))

	return ChatRequest{
		Model:    conv.Model,
		Messages: conv.Messages,
		Stream:   false,
	}
}

func CreateChatRequestWithTools(conversation Conversation, userMessage string, tools []map[string]any) ChatRequest {
	conv := AddMessage(conversation, NewUserMessage(userMessage))

	return ChatRequest{
		Model:    conv.Model,
		Messages: conv.Messages,
		Stream:   false,
		Tools:    tools,
	}
}
