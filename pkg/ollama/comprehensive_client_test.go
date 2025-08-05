package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_NewClientWithTimeout(t *testing.T) {
	baseURL := "http://localhost:11434"
	timeout := 60 * time.Second
	
	client := ollama.NewClientWithTimeout(baseURL, timeout)
	
	assert.NotNil(t, client)
	// Note: Cannot directly test internal fields as they're private
}

func TestClient_NewClient(t *testing.T) {
	baseURL := "http://localhost:11434"
	
	client := ollama.NewClient(baseURL)
	
	assert.NotNil(t, client)
}

func TestClient_Pull(t *testing.T) {
	tests := []struct {
		name           string
		modelName      string
		serverResponse string
		statusCode     int
		expectError    bool
		errorContains  string
	}{
		{
			name:      "successful pull",
			modelName: "llama2:latest",
			serverResponse: `{"status":"downloading digestsha256:abc123","digest":"sha256:abc123","total":1000,"completed":0}
{"status":"downloading digestsha256:abc123","digest":"sha256:abc123","total":1000,"completed":500}
{"status":"success"}`,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:           "pull with error response",
			modelName:      "invalid-model",
			serverResponse: `{"error":"model not found"}`,
			statusCode:     http.StatusOK,
			expectError:    true,
			errorContains:  "model not found",
		},
		{
			name:           "server error status",
			modelName:      "llama2:latest",
			serverResponse: "",
			statusCode:     http.StatusInternalServerError,
			expectError:    true,
			errorContains:  "failed with status: 500",
		},
		{
			name:           "unexpected stream end",
			modelName:      "llama2:latest",
			serverResponse: `{"status":"downloading"}`, // Missing success status
			statusCode:     http.StatusOK,
			expectError:    true,
			errorContains:  "stream ended unexpectedly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/pull", r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Verify request body
				var pullReq ollama.PullRequest
				err := json.NewDecoder(r.Body).Decode(&pullReq)
				require.NoError(t, err)
				assert.Equal(t, tt.modelName, pullReq.Name)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				
				if tt.statusCode == http.StatusOK {
					// Write streaming responses line by line
					lines := strings.Split(tt.serverResponse, "\n")
					for _, line := range lines {
						if line != "" {
							w.Write([]byte(line + "\n"))
						}
					}
				}
			}))
			defer server.Close()

			client := ollama.NewClient(server.URL)
			err := client.Pull(tt.modelName)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_PullWithProgress(t *testing.T) {
	tests := []struct {
		name           string
		modelName      string
		serverResponse string
		statusCode     int
		expectError    bool
		errorContains  string
		expectProgress []ollama.PullResponse
	}{
		{
			name:      "successful pull with progress",
			modelName: "llama2:latest", 
			serverResponse: `{"status":"downloading","digest":"sha256:abc123","total":1000,"completed":0}
{"status":"downloading","digest":"sha256:abc123","total":1000,"completed":500}
{"status":"success"}`,
			statusCode:  http.StatusOK,
			expectError: false,
			expectProgress: []ollama.PullResponse{
				{Status: "downloading", Digest: "sha256:abc123", Total: 1000, Completed: 0},
				{Status: "downloading", Digest: "sha256:abc123", Total: 1000, Completed: 500},
				{Status: "success"},
			},
		},
		{
			name:           "pull with context cancellation",
			modelName:      "llama2:latest",
			serverResponse: `{"status":"downloading","total":1000,"completed":0}`,
			statusCode:     http.StatusOK,
			expectError:    true,
			errorContains:  "context deadline exceeded",
		},
		{
			name:           "pull with error in response",
			modelName:      "invalid-model",
			serverResponse: `{"error":"model not found"}`,
			statusCode:     http.StatusOK,
			expectError:    true,
			errorContains:  "model not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/pull", r.URL.Path)
				assert.Equal(t, "POST", r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				
				if tt.statusCode == http.StatusOK {
					lines := strings.Split(tt.serverResponse, "\n")
					for _, line := range lines {
						if line != "" {
							w.Write([]byte(line + "\n"))
							// Add delay for context cancellation test
							if tt.name == "pull with context cancellation" {
								time.Sleep(200 * time.Millisecond)
							}
						}
					}
				}
			}))
			defer server.Close()

			client := ollama.NewClient(server.URL)
			
			var progressCalls []ollama.PullResponse
			progressCallback := func(status string, completed, total int64) {
				progressCalls = append(progressCalls, ollama.PullResponse{
					Status:    status,
					Total:     total,
					Completed: completed,
				})
			}

			var ctx context.Context
			var cancel context.CancelFunc
			if tt.name == "pull with context cancellation" {
				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
			} else {
				ctx = context.Background()
			}

			err := client.PullWithProgress(ctx, tt.modelName, progressCallback)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, progressCalls, len(tt.expectProgress))
				for i, expected := range tt.expectProgress {
					if i < len(progressCalls) {
						assert.Equal(t, expected.Status, progressCalls[i].Status)
						assert.Equal(t, expected.Total, progressCalls[i].Total)
						assert.Equal(t, expected.Completed, progressCalls[i].Completed)
					}
				}
			}
		})
	}
}

func TestClient_Delete(t *testing.T) {
	tests := []struct {
		name          string
		modelName     string
		statusCode    int
		expectError   bool
		errorContains string
	}{
		{
			name:        "successful delete",
			modelName:   "llama2:latest",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:          "server error",
			modelName:     "llama2:latest",
			statusCode:    http.StatusInternalServerError,
			expectError:   true,
			errorContains: "failed with status: 500",
		},
		{
			name:          "not found",
			modelName:     "nonexistent:latest",
			statusCode:    http.StatusNotFound,
			expectError:   true,
			errorContains: "failed with status: 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/delete", r.URL.Path)
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Verify request body
				var deleteReq ollama.DeleteRequest
				err := json.NewDecoder(r.Body).Decode(&deleteReq)
				require.NoError(t, err)
				assert.Equal(t, tt.modelName, deleteReq.Name)

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := ollama.NewClient(server.URL)
			err := client.Delete(tt.modelName)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_Tags_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectError   bool
		errorContains string
	}{
		{
			name:          "server returns 500",
			statusCode:    http.StatusInternalServerError,
			responseBody:  "",
			expectError:   true,
			errorContains: "failed with status: 500",
		},
		{
			name:          "invalid JSON response",
			statusCode:    http.StatusOK,
			responseBody:  `{"models": [invalid json}`,
			expectError:   true,
			errorContains: "failed to decode tags response",
		},
		{
			name:        "empty models array",
			statusCode:  http.StatusOK,
			responseBody: `{"models": []}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/tags", r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := ollama.NewClient(server.URL)
			response, err := client.Tags()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, response)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
			}
		})
	}
}

func TestClient_Ps_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectError   bool
		errorContains string
	}{
		{
			name:          "server returns 404",
			statusCode:    http.StatusNotFound,
			responseBody:  "",
			expectError:   true,
			errorContains: "failed with status: 404",
		},
		{
			name:          "malformed JSON response",
			statusCode:    http.StatusOK,
			responseBody:  `{"models": [{"name": "incomplete"`,
			expectError:   true,
			errorContains: "failed to decode ps response",
		},
		{
			name:        "no running models",
			statusCode:  http.StatusOK,
			responseBody: `{"models": []}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/ps", r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := ollama.NewClient(server.URL)
			response, err := client.Ps()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, response)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
			}
		})
	}
}

func TestClient_NetworkErrors(t *testing.T) {
	// Test with unreachable server
	client := ollama.NewClient("http://non-existent-server:99999")

	t.Run("Tags network error", func(t *testing.T) {
		_, err := client.Tags()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get tags")
	})

	t.Run("Ps network error", func(t *testing.T) {
		_, err := client.Ps()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get ps")
	})

	t.Run("Pull network error", func(t *testing.T) {
		err := client.Pull("test-model")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to pull model")
	})

	t.Run("Delete network error", func(t *testing.T) {
		err := client.Delete("test-model")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete model")
	})
}

func TestClient_PullWithProgress_NilCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	ctx := context.Background()
	
	// Test with nil progress callback - should not crash
	err := client.PullWithProgress(ctx, "test-model", nil)
	assert.NoError(t, err)
}

func TestProgressCallback(t *testing.T) {
	var capturedStatus string
	var capturedCompleted, capturedTotal int64

	callback := func(status string, completed, total int64) {
		capturedStatus = status
		capturedCompleted = completed
		capturedTotal = total
	}

	// Test the callback directly
	callback("downloading", 500, 1000)
	
	assert.Equal(t, "downloading", capturedStatus)
	assert.Equal(t, int64(500), capturedCompleted)
	assert.Equal(t, int64(1000), capturedTotal)
}

func TestClient_RequestCreationErrors(t *testing.T) {
	// Test with invalid request body scenarios
	client := ollama.NewClient("http://localhost:11434")
	
	t.Run("PullWithProgress request creation error", func(t *testing.T) {
		// Create context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		
		err := client.PullWithProgress(ctx, "test-model", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}