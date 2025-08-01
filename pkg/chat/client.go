package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ChatResponse struct {
	Model              string        `json:"model"`
	CreatedAt          time.Time     `json:"created_at"`
	Message            Message       `json:"message"`
	Done               bool          `json:"done"`
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
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
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
		return ChatResponse{}, fmt.Errorf("request failed with status %d", resp.StatusCode)
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
