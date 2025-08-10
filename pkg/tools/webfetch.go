package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebFetchTool implements web content fetching with permission checking
type WebFetchTool struct {
	*SecuredTool
	client      *http.Client
	maxBodySize int64
}

// NewWebFetchTool creates a new web fetch tool
func NewWebFetchTool() *WebFetchTool {
	return NewWebFetchToolWithBypass(false)
}

// NewWebFetchToolWithBypass creates a new web fetch tool with optional permission bypass
func NewWebFetchToolWithBypass(bypass bool) *WebFetchTool {
	return &WebFetchTool{
		SecuredTool: NewSecuredToolWithBypass(bypass),
		client: &http.Client{
			Timeout: 30 * time.Second,
			// Disable following redirects automatically to check permissions
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		maxBodySize: 10 * 1024 * 1024, // 10MB limit
	}
}

// Name returns the tool name
func (t *WebFetchTool) Name() string {
	return "web_fetch"
}

// Description returns the tool description
func (t *WebFetchTool) Description() string {
	return "Fetch content from a URL. Input: URL string"
}

// Call executes the web fetch operation
func (t *WebFetchTool) Call(ctx context.Context, input string) (string, error) {
	// Trim whitespace from input
	urlStr := strings.TrimSpace(input)
	if urlStr == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}

	// Parse URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Ensure scheme is present
	if u.Scheme == "" {
		u.Scheme = "https"
		urlStr = u.String()
	}

	// Validate scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme: %s (only http/https allowed)", u.Scheme)
	}

	// Check ACL with domain:path format
	aclInput := fmt.Sprintf("%s:%s", u.Host, u.Path)
	if err := t.ValidateAccess("WebFetch", aclInput); err != nil {
		// Also try with just the domain
		if err := t.ValidateAccess("WebFetch", u.Host); err != nil {
			return "", fmt.Errorf("permission denied for URL %s: %w", urlStr, err)
		}
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "Ryan-AI-Assistant/1.0")

	// Execute request
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read a small amount of the error body if available
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("HTTP %d: %s\n%s", resp.StatusCode, resp.Status, string(body))
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	// Read body with size limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, t.maxBodySize))
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if we hit the size limit
	if int64(len(body)) >= t.maxBodySize {
		return string(body) + fmt.Sprintf("\n\n[Content truncated at %d MB]", t.maxBodySize/(1024*1024)), nil
	}

	// Add metadata about the fetch
	result := string(body)
	if result == "" {
		result = fmt.Sprintf("[Empty response from %s]", urlStr)
	}

	// Add content type info if it's not HTML or text
	if !strings.Contains(contentType, "text/") && !strings.Contains(contentType, "application/json") {
		result = fmt.Sprintf("[Content-Type: %s]\n\n%s", contentType, result)
	}

	return result, nil
}
