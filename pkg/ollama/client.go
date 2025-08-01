package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	log := logger.WithComponent("ollama_client")
	log.Debug("Creating new ollama client", "base_url", baseURL, "timeout", "30s")

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Tags() (*TagsResponse, error) {
	log := logger.WithComponent("ollama_client")
	url := fmt.Sprintf("%s/api/tags", c.baseURL)

	log.Debug("Making HTTP GET request to tags endpoint", "url", url)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		log.Error("HTTP GET to tags endpoint failed", "url", url, "error", err)
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	defer resp.Body.Close()

	log.Debug("Received HTTP response from tags endpoint",
		"status_code", resp.StatusCode,
		"content_length", resp.ContentLength)

	if resp.StatusCode != http.StatusOK {
		log.Error("Tags endpoint returned non-200 status", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("tags request failed with status: %d", resp.StatusCode)
	}

	var tagsResponse TagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
		log.Error("Failed to decode JSON response from tags endpoint", "error", err)
		return nil, fmt.Errorf("failed to decode tags response: %w", err)
	}

	log.Debug("Successfully decoded tags response", "model_count", len(tagsResponse.Models))
	return &tagsResponse, nil
}

func (c *Client) Ps() (*PsResponse, error) {
	log := logger.WithComponent("ollama_client")
	url := fmt.Sprintf("%s/api/ps", c.baseURL)

	log.Debug("Making HTTP GET request to ps endpoint", "url", url)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		log.Error("HTTP GET to ps endpoint failed", "url", url, "error", err)
		return nil, fmt.Errorf("failed to get ps: %w", err)
	}
	defer resp.Body.Close()

	log.Debug("Received HTTP response from ps endpoint",
		"status_code", resp.StatusCode,
		"content_length", resp.ContentLength)

	if resp.StatusCode != http.StatusOK {
		log.Error("Ps endpoint returned non-200 status", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ps request failed with status: %d", resp.StatusCode)
	}

	var psResponse PsResponse
	if err := json.NewDecoder(resp.Body).Decode(&psResponse); err != nil {
		log.Error("Failed to decode JSON response from ps endpoint", "error", err)
		return nil, fmt.Errorf("failed to decode ps response: %w", err)
	}

	log.Debug("Successfully decoded ps response", "running_count", len(psResponse.Models))
	return &psResponse, nil
}

func (c *Client) Pull(modelName string) error {
	log := logger.WithComponent("ollama_client")
	url := fmt.Sprintf("%s/api/pull", c.baseURL)

	pullRequest := PullRequest{
		Name: modelName,
	}

	requestBody, err := json.Marshal(pullRequest)
	if err != nil {
		log.Error("Failed to marshal pull request", "model", modelName, "error", err)
		return fmt.Errorf("failed to marshal pull request: %w", err)
	}

	// Create a client with longer timeout for model pulling
	pullClient := &http.Client{
		Timeout: 30 * time.Minute, // Model pulls can take a long time
	}

	log.Debug("Making HTTP POST request to pull endpoint", "url", url, "model", modelName)
	resp, err := pullClient.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Error("HTTP POST to pull endpoint failed", "url", url, "model", modelName, "error", err)
		return fmt.Errorf("failed to pull model: %w", err)
	}
	defer resp.Body.Close()

	log.Debug("Received HTTP response from pull endpoint",
		"status_code", resp.StatusCode,
		"content_length", resp.ContentLength)

	if resp.StatusCode != http.StatusOK {
		log.Error("Pull endpoint returned non-200 status", "status_code", resp.StatusCode, "model", modelName)
		return fmt.Errorf("pull request failed with status: %d", resp.StatusCode)
	}

	// Read and parse streaming response
	decoder := json.NewDecoder(resp.Body)
	for {
		var pullResponse PullResponse
		if err := decoder.Decode(&pullResponse); err != nil {
			if err == io.EOF {
				log.Error("Pull response stream ended unexpectedly", "model", modelName)
				return fmt.Errorf("pull stream ended unexpectedly")
			}
			log.Error("Failed to decode pull response", "model", modelName, "error", err)
			return fmt.Errorf("failed to decode pull response: %w", err)
		}

		log.Debug("Pull progress", "model", modelName, "status", pullResponse.Status)

		// Handle errors in the response
		if pullResponse.Error != "" {
			log.Error("Pull failed with error", "model", modelName, "error", pullResponse.Error)
			return fmt.Errorf("pull failed: %s", pullResponse.Error)
		}

		// Check for completion
		if pullResponse.Status == "success" {
			log.Debug("Successfully pulled model", "model", modelName)
			return nil
		}
	}
}

func (c *Client) Delete(modelName string) error {
	log := logger.WithComponent("ollama_client")
	url := fmt.Sprintf("%s/api/delete", c.baseURL)

	deleteRequest := DeleteRequest{
		Name: modelName,
	}

	requestBody, err := json.Marshal(deleteRequest)
	if err != nil {
		log.Error("Failed to marshal delete request", "model", modelName, "error", err)
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	log.Debug("Making HTTP DELETE request to delete endpoint", "url", url, "model", modelName)
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Error("Failed to create delete request", "url", url, "model", modelName, "error", err)
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("HTTP DELETE to delete endpoint failed", "url", url, "model", modelName, "error", err)
		return fmt.Errorf("failed to delete model: %w", err)
	}
	defer resp.Body.Close()

	log.Debug("Received HTTP response from delete endpoint",
		"status_code", resp.StatusCode,
		"content_length", resp.ContentLength)

	if resp.StatusCode != http.StatusOK {
		log.Error("Delete endpoint returned non-200 status", "status_code", resp.StatusCode, "model", modelName)
		return fmt.Errorf("delete request failed with status: %d", resp.StatusCode)
	}

	log.Debug("Successfully deleted model", "model", modelName)
	return nil
}
