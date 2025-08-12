package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// Model represents an Ollama model from the API
type Model struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    Details   `json:"details"`
}

// Details contains model details
type Details struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// ModelsResponse represents the response from /api/tags
type ModelsResponse struct {
	Models []Model `json:"models"`
}

// PullProgress represents the progress of a model pull operation
type PullProgress struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

// PullRequest represents a request to pull a model
type PullRequest struct {
	Model string `json:"model"`
}

// DeleteRequest represents a request to delete a model
type DeleteRequest struct {
	Model string `json:"model"`
}

// APIClient provides direct API access to Ollama
type APIClient struct {
	baseURL string
	client  *http.Client
}

// NewAPIClient creates a new Ollama API client
func NewAPIClient() *APIClient {
	baseURL := os.Getenv("OLLAMA_HOST")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &APIClient{
		baseURL: baseURL,
		client: &http.Client{
			// Use longer timeout for model operations which can take a while
			Timeout: 30 * time.Minute, // 30 minutes should be enough for most downloads
		},
	}
}

// ListModels fetches the list of available models
func (c *APIClient) ListModels() ([]Model, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/api/tags", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return modelsResp.Models, nil
}

// PullModel downloads a model from the Ollama library
func (c *APIClient) PullModel(modelName string) (chan PullProgress, chan error, error) {
	logger.Debug("Starting pull for model: %s", modelName)
	progressChan := make(chan PullProgress, 10)
	errorChan := make(chan error, 1)

	req := PullRequest{Model: modelName}
	reqBody, err := json.Marshal(req)
	if err != nil {
		logger.Error("Failed to marshal pull request: %v", err)
		close(progressChan)
		close(errorChan)
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	logger.Debug("Sending pull request to %s/api/pull", c.baseURL)

	go func() {
		defer close(progressChan)
		defer close(errorChan)

		resp, err := c.client.Post(fmt.Sprintf("%s/api/pull", c.baseURL), "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			logger.Error("Failed to send pull request: %v", err)
			errorChan <- fmt.Errorf("failed to pull model: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logger.Error("Pull request returned status code: %d", resp.StatusCode)
			errorChan <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			return
		}

		logger.Debug("Pull request successful, starting to read progress stream")
		decoder := json.NewDecoder(resp.Body)
		for {
			var progress PullProgress
			if err := decoder.Decode(&progress); err != nil {
				if err.Error() != "EOF" {
					logger.Error("Failed to decode progress: %v", err)
					errorChan <- fmt.Errorf("failed to decode progress: %w", err)
				} else {
					logger.Debug("Reached end of progress stream")
				}
				return
			}

			logger.Debug("Pull progress: status=%s, total=%d, completed=%d", progress.Status, progress.Total, progress.Completed)
			progressChan <- progress

			if progress.Status == "success" {
				logger.Debug("Pull completed successfully")
				return
			}
		}
	}()

	return progressChan, errorChan, nil
}

// DeleteModel removes a model and its associated data
func (c *APIClient) DeleteModel(modelName string) error {
	logger.Debug("Starting delete for model: %s", modelName)
	deleteReq := DeleteRequest{Model: modelName}
	reqBody, err := json.Marshal(deleteReq)
	if err != nil {
		logger.Error("Failed to marshal delete request: %v", err)
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	logger.Debug("Sending delete request to %s/api/delete", c.baseURL)

	// Create DELETE request with JSON body
	httpReq, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/delete", c.baseURL), bytes.NewBuffer(reqBody))
	if err != nil {
		logger.Error("Failed to create delete request: %v", err)
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		logger.Error("Failed to send delete request: %v", err)
		return fmt.Errorf("failed to delete model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("Delete request returned status code: %d", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	logger.Debug("Delete completed successfully")
	return nil
}

// FormatSize formats bytes to human readable string
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
