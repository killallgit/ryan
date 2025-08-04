package vectorstore

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPClientRetry(t *testing.T) {
	t.Run("SuccessOnFirstAttempt", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})
		server := httptest.NewServer(handler)
		defer server.Close()

		client := newHTTPClient(DefaultHTTPClientConfig())
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("RetryOnServerError", func(t *testing.T) {
		var attempts int32
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&attempts, 1)
			if count < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})
		server := httptest.NewServer(handler)
		defer server.Close()

		client := newHTTPClient(HTTPClientConfig{
			Timeout:     5 * time.Second,
			MaxRetries:  3,
			BackoffBase: 10 * time.Millisecond,
		})

		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		if atomic.LoadInt32(&attempts) != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("NoRetryOnClientError", func(t *testing.T) {
		var attempts int32
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusBadRequest)
		})
		server := httptest.NewServer(handler)
		defer server.Close()

		client := newHTTPClient(DefaultHTTPClientConfig())
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}

		if atomic.LoadInt32(&attempts) != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})
		server := httptest.NewServer(handler)
		defer server.Close()

		client := newHTTPClient(DefaultHTTPClientConfig())
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		_, err := client.Do(req)
		if err == nil {
			t.Fatal("expected timeout error")
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context deadline exceeded, got %v", err)
		}
	})
}
