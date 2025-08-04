package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// WebFetchTool implements HTTP requests with caching and rate limiting
type WebFetchTool struct {
	client       *http.Client
	cache        *WebFetchCache
	rateLimiter  *RateLimiter
	log          *logger.Logger
	userAgent    string
	maxBodySize  int64
	allowedHosts []string
}

// WebFetchCache provides simple in-memory caching for HTTP responses
type WebFetchCache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
	maxAge  time.Duration
	maxSize int
}

// CacheEntry represents a cached HTTP response
type CacheEntry struct {
	URL         string
	Content     string
	ContentType string
	StatusCode  int
	Timestamp   time.Time
	Size        int
}

// RateLimiter implements simple rate limiting for HTTP requests
type RateLimiter struct {
	requests     chan time.Time
	maxPerMinute int
}

// NewWebFetchTool creates a new WebFetch tool with default configuration
func NewWebFetchTool() *WebFetchTool {
	// Create HTTP client with reasonable timeouts
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			IdleConnTimeout:    60 * time.Second,
			DisableCompression: false,
		},
	}

	// Create cache with 15-minute TTL and max 100 entries
	cache := &WebFetchCache{
		entries: make(map[string]*CacheEntry),
		maxAge:  15 * time.Minute,
		maxSize: 100,
	}

	// Create rate limiter (60 requests per minute)
	rateLimiter := &RateLimiter{
		requests:     make(chan time.Time, 60),
		maxPerMinute: 60,
	}

	return &WebFetchTool{
		client:      client,
		cache:       cache,
		rateLimiter: rateLimiter,
		log:         logger.WithComponent("webfetch_tool"),
		userAgent:   "Ryan-AI-Assistant/1.0",
		maxBodySize: 10 * 1024 * 1024, // 10MB limit
		allowedHosts: []string{
			// Common safe domains for AI assistants
			"github.com",
			"githubusercontent.com",
			"docs.python.org",
			"nodejs.org",
			"developer.mozilla.org",
			"stackoverflow.com",
			"wikipedia.org",
			"wikimedia.org",
		},
	}
}

// Name returns the tool name
func (wft *WebFetchTool) Name() string {
	return "web_fetch"
}

// Description returns the tool description
func (wft *WebFetchTool) Description() string {
	return "Fetch content from web URLs with caching and rate limiting. Supports HTTP and HTTPS requests to common developer and reference sites. Use for fetching documentation, code examples, or public information."
}

// JSONSchema returns the JSON schema for the tool parameters
func (wft *WebFetchTool) JSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "The URL to fetch content from (HTTP or HTTPS)",
				"pattern":     "^https?://",
			},
			"method": map[string]interface{}{
				"type":        "string",
				"description": "HTTP method to use (GET, POST, PUT, DELETE)",
				"enum":        []string{"GET", "POST", "PUT", "DELETE"},
				"default":     "GET",
			},
			"headers": map[string]interface{}{
				"type":        "object",
				"description": "Optional HTTP headers to include in the request",
				"additionalProperties": map[string]interface{}{
					"type": "string",
				},
			},
			"body": map[string]interface{}{
				"type":        "string",
				"description": "Request body for POST/PUT requests",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Request timeout in seconds (1-120)",
				"minimum":     1,
				"maximum":     120,
				"default":     30,
			},
		},
		"required": []string{"url"},
	}
}

// Execute performs the HTTP request with caching and rate limiting
func (wft *WebFetchTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	startTime := time.Now()

	// Extract and validate URL parameter
	urlInterface, exists := params["url"]
	if !exists {
		return wft.createErrorResult(startTime, "url parameter is required"), nil
	}

	urlStr, ok := urlInterface.(string)
	if !ok {
		return wft.createErrorResult(startTime, "url parameter must be a string"), nil
	}

	// Validate and parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return wft.createErrorResult(startTime, fmt.Sprintf("invalid URL: %v", err)), nil
	}

	// Security check: only allow HTTP/HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return wft.createErrorResult(startTime, "only HTTP and HTTPS URLs are allowed"), nil
	}

	// Security check: validate allowed hosts
	if !wft.isHostAllowed(parsedURL.Host) {
		return wft.createErrorResult(startTime, fmt.Sprintf("host %s is not in the allowed list", parsedURL.Host)), nil
	}

	// Check cache first
	if cached := wft.cache.Get(urlStr); cached != nil {
		wft.log.Debug("Cache hit", "url", urlStr, "age", time.Since(cached.Timestamp))
		return ToolResult{
			Success: true,
			Content: cached.Content,
			Error:   "",
			Data: map[string]interface{}{
				"url":          urlStr,
				"status_code":  cached.StatusCode,
				"content_type": cached.ContentType,
				"cached":       true,
				"cache_age":    time.Since(cached.Timestamp).String(),
			},
			Metadata: ToolMetadata{
				ExecutionTime: time.Since(startTime),
				StartTime:     startTime,
				EndTime:       time.Now(),
				ToolName:      wft.Name(),
				Parameters:    params,
			},
		}, nil
	}

	// Apply rate limiting
	if err := wft.rateLimiter.Wait(ctx); err != nil {
		return wft.createErrorResult(startTime, fmt.Sprintf("rate limiting error: %v", err)), nil
	}

	// Extract optional parameters
	method := wft.getStringParam(params, "method", "GET")
	body := wft.getStringParam(params, "body", "")
	timeout := wft.getIntParam(params, "timeout", 30)

	// Create HTTP request
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return wft.createErrorResult(startTime, fmt.Sprintf("failed to create request: %v", err)), nil
	}

	// Set default headers
	req.Header.Set("User-Agent", wft.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,text/plain;q=0.8,*/*;q=0.1")

	// Add custom headers if provided
	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if valueStr, ok := value.(string); ok {
				req.Header.Set(key, valueStr)
			}
		}
	}

	// Set request timeout
	clientWithTimeout := *wft.client
	clientWithTimeout.Timeout = time.Duration(timeout) * time.Second

	wft.log.Debug("Making HTTP request", "method", method, "url", urlStr, "timeout", timeout)

	// Execute request
	resp, err := clientWithTimeout.Do(req)
	if err != nil {
		return wft.createErrorResult(startTime, fmt.Sprintf("request failed: %v", err)), nil
	}
	defer resp.Body.Close()

	// Read response body with size limit
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, wft.maxBodySize))
	if err != nil {
		return wft.createErrorResult(startTime, fmt.Sprintf("failed to read response: %v", err)), nil
	}

	content := string(bodyBytes)
	contentType := resp.Header.Get("Content-Type")

	// Cache successful responses
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		wft.cache.Set(urlStr, &CacheEntry{
			URL:         urlStr,
			Content:     content,
			ContentType: contentType,
			StatusCode:  resp.StatusCode,
			Timestamp:   time.Now(),
			Size:        len(content),
		})
	}

	wft.log.Debug("HTTP request completed",
		"url", urlStr,
		"status", resp.StatusCode,
		"content_length", len(content),
		"duration", time.Since(startTime))

	// Return successful result
	return ToolResult{
		Success: resp.StatusCode >= 200 && resp.StatusCode < 400,
		Content: content,
		Error:   "",
		Data: map[string]interface{}{
			"url":          urlStr,
			"method":       method,
			"status_code":  resp.StatusCode,
			"content_type": contentType,
			"content_size": len(content),
			"cached":       false,
		},
		Metadata: ToolMetadata{
			ExecutionTime: time.Since(startTime),
			StartTime:     startTime,
			EndTime:       time.Now(),
			ToolName:      wft.Name(),
			Parameters:    params,
		},
	}, nil
}

// Helper methods

func (wft *WebFetchTool) createErrorResult(startTime time.Time, errorMsg string) ToolResult {
	wft.log.Error("WebFetch tool error", "error", errorMsg)
	return ToolResult{
		Success: false,
		Content: "",
		Error:   errorMsg,
		Metadata: ToolMetadata{
			ExecutionTime: time.Since(startTime),
			StartTime:     startTime,
			EndTime:       time.Now(),
			ToolName:      wft.Name(),
		},
	}
}

func (wft *WebFetchTool) getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if value, exists := params[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

func (wft *WebFetchTool) getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, exists := params[key]; exists {
		if intValue, ok := value.(int); ok {
			return intValue
		}
		if floatValue, ok := value.(float64); ok {
			return int(floatValue)
		}
	}
	return defaultValue
}

func (wft *WebFetchTool) isHostAllowed(host string) bool {
	// Remove port if present
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	for _, allowedHost := range wft.allowedHosts {
		if host == allowedHost || strings.HasSuffix(host, "."+allowedHost) {
			return true
		}
	}
	return false
}

// Cache methods

func (c *WebFetchCache) Get(url string) *CacheEntry {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.entries[url]
	if !exists {
		return nil
	}

	// Check if entry is expired
	if time.Since(entry.Timestamp) > c.maxAge {
		delete(c.entries, url)
		return nil
	}

	return entry
}

func (c *WebFetchCache) Set(url string, entry *CacheEntry) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Clean up old entries if cache is full
	if len(c.entries) >= c.maxSize {
		c.cleanup()
	}

	c.entries[url] = entry
}

func (c *WebFetchCache) cleanup() {
	// Remove expired entries first
	now := time.Now()
	for url, entry := range c.entries {
		if now.Sub(entry.Timestamp) > c.maxAge {
			delete(c.entries, url)
		}
	}

	// If still too many entries, remove oldest 25%
	if len(c.entries) >= c.maxSize {
		type urlTime struct {
			url  string
			time time.Time
		}

		var entries []urlTime
		for url, entry := range c.entries {
			entries = append(entries, urlTime{url, entry.Timestamp})
		}

		// Sort by timestamp (oldest first)
		for i := 0; i < len(entries)-1; i++ {
			for j := i + 1; j < len(entries); j++ {
				if entries[i].time.After(entries[j].time) {
					entries[i], entries[j] = entries[j], entries[i]
				}
			}
		}

		// Remove oldest 25%
		removeCount := len(entries) / 4
		for i := 0; i < removeCount; i++ {
			delete(c.entries, entries[i].url)
		}
	}
}

// Rate limiter methods

func (rl *RateLimiter) Wait(ctx context.Context) error {
	// Clean old requests
	now := time.Now()
	cutoff := now.Add(-time.Minute)

	var remaining []time.Time
	for len(rl.requests) > 0 {
		select {
		case req := <-rl.requests:
			if req.After(cutoff) {
				remaining = append(remaining, req)
			}
		default:
			break
		}
	}

	// Put back remaining requests
	for _, req := range remaining {
		select {
		case rl.requests <- req:
		default:
			break
		}
	}

	// Check if we can make a request
	if len(rl.requests) >= rl.maxPerMinute {
		return fmt.Errorf("rate limit exceeded: %d requests per minute", rl.maxPerMinute)
	}

	// Add current request
	select {
	case rl.requests <- now:
		return nil
	default:
		return fmt.Errorf("rate limiter channel full")
	}
}
