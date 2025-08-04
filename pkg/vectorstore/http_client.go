package vectorstore

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// HTTPClientConfig configures the HTTP client with retry logic
type HTTPClientConfig struct {
	Timeout     time.Duration
	MaxRetries  int
	BackoffBase time.Duration
}

// DefaultHTTPClientConfig returns sensible defaults for HTTP client
func DefaultHTTPClientConfig() HTTPClientConfig {
	return HTTPClientConfig{
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		BackoffBase: 100 * time.Millisecond,
	}
}

// httpTransport implements http.RoundTripper with built-in retry logic
type httpTransport struct {
	base       http.RoundTripper
	maxRetries int
	backoff    time.Duration
}

// RoundTrip executes a single HTTP transaction with retry logic
func (t *httpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		// Clone the request for each attempt
		reqCopy := req.Clone(req.Context())
		if req.Body != nil {
			// Body needs to be reset for retry
			reqCopy.Body = req.Body
		}

		resp, err = t.base.RoundTrip(reqCopy)

		// Don't retry on success or client errors
		if err == nil && resp.StatusCode < 500 {
			return resp, err
		}

		// Don't retry on the last attempt
		if attempt < t.maxRetries {
			// Close the response body if we got one
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}

			// Calculate backoff with exponential increase and jitter
			wait := t.backoff * time.Duration(1<<uint(attempt))
			jitter := time.Duration(rand.Int63n(int64(wait / 4)))
			sleepDuration := wait + jitter

			// Check if context would expire before retry
			if deadline, ok := req.Context().Deadline(); ok {
				if time.Until(deadline) < sleepDuration {
					// Not enough time for retry
					break
				}
			}

			select {
			case <-time.After(sleepDuration):
				// Continue to next retry
			case <-req.Context().Done():
				// Context cancelled
				return nil, req.Context().Err()
			}
		}
	}

	// Return the last response and error
	if err != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", t.maxRetries+1, err)
	}
	return resp, nil
}

// newHTTPClient creates an HTTP client with built-in retry logic
func newHTTPClient(config HTTPClientConfig) *http.Client {
	if config.Timeout == 0 {
		config = DefaultHTTPClientConfig()
	}

	transport := &httpTransport{
		base:       http.DefaultTransport,
		maxRetries: config.MaxRetries,
		backoff:    config.BackoffBase,
	}

	return &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}
}
