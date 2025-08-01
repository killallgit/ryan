package ollama

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Tags() (*TagsResponse, error) {
	url := fmt.Sprintf("%s/api/tags", c.baseURL)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tags request failed with status: %d", resp.StatusCode)
	}

	var tagsResponse TagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode tags response: %w", err)
	}

	return &tagsResponse, nil
}

func (c *Client) Ps() (*PsResponse, error) {
	url := fmt.Sprintf("%s/api/ps", c.baseURL)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get ps: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ps request failed with status: %d", resp.StatusCode)
	}

	var psResponse PsResponse
	if err := json.NewDecoder(resp.Body).Decode(&psResponse); err != nil {
		return nil, fmt.Errorf("failed to decode ps response: %w", err)
	}

	return &psResponse, nil
}