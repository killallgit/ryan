package ollama

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// HealthStatus represents the health status of Ollama service
type HealthStatus struct {
	Available bool
	Error     error
	Models    []Model
}

// CheckHealth checks if Ollama service is available and responsive
func (c *Client) CheckHealth(ctx context.Context) (*HealthStatus, error) {
	log := logger.WithComponent("ollama_health")
	log.Debug("Checking Ollama health", "base_url", c.baseURL)

	// Create a request with context for timeout control
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/tags", c.baseURL), nil)
	if err != nil {
		return &HealthStatus{Available: false, Error: err}, err
	}

	// Try to get the list of models (this also checks connectivity)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("Failed to connect to Ollama", "error", err)
		return &HealthStatus{
			Available: false,
			Error:     fmt.Errorf("cannot connect to Ollama at %s: %w", c.baseURL, err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("Ollama returned non-OK status", "status_code", resp.StatusCode)
		return &HealthStatus{
			Available: false,
			Error:     fmt.Errorf("Ollama returned status %d", resp.StatusCode),
		}, nil
	}

	// Try to get the list of available models
	tagsResp, err := c.Tags()
	if err != nil {
		log.Error("Failed to get model list", "error", err)
		return &HealthStatus{
			Available: true, // Ollama is running but we couldn't get models
			Error:     fmt.Errorf("failed to get model list: %w", err),
			Models:    []Model{},
		}, nil
	}

	log.Debug("Ollama health check successful", "model_count", len(tagsResp.Models))
	return &HealthStatus{
		Available: true,
		Error:     nil,
		Models:    tagsResp.Models,
	}, nil
}

// CheckModel checks if a specific model is available
func (c *Client) CheckModel(ctx context.Context, modelName string) (bool, error) {
	log := logger.WithComponent("ollama_health")
	log.Debug("Checking for model", "model", modelName)

	health, err := c.CheckHealth(ctx)
	if err != nil {
		return false, err
	}

	if !health.Available {
		return false, health.Error
	}

	// Check if the model exists in the list
	for _, model := range health.Models {
		if model.Name == modelName {
			log.Debug("Model found", "model", modelName)
			return true, nil
		}
	}

	log.Debug("Model not found", "model", modelName)
	return false, nil
}

// CheckHealthWithTimeout performs a health check with a specific timeout
func (c *Client) CheckHealthWithTimeout(timeout time.Duration) (*HealthStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.CheckHealth(ctx)
}

// CheckModelWithTimeout checks if a model is available with a specific timeout
func (c *Client) CheckModelWithTimeout(modelName string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.CheckModel(ctx, modelName)
}
