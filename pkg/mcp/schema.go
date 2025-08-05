package mcp

import (
	"container/list"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

// SchemaCache implements LRU caching for JSON schema validators
// This provides the performance optimization that Claude CLI uses
type SchemaCache struct {
	mu       sync.RWMutex
	maxSize  int
	ttl      time.Duration
	items    map[string]*schemaCacheItem
	lru      *list.List // For LRU eviction
}

// schemaCacheItem represents a cached schema validator
type schemaCacheItem struct {
	key         string
	validator   *gojsonschema.Schema
	createdAt   time.Time
	lastUsed    time.Time
	element     *list.Element // For LRU list
}

// NewSchemaCache creates a new schema cache with the specified size and TTL
func NewSchemaCache(maxSize int, ttl time.Duration) *SchemaCache {
	if maxSize <= 0 {
		maxSize = 100 // Default cache size
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute // Default TTL
	}
	
	return &SchemaCache{
		maxSize: maxSize,
		ttl:     ttl,
		items:   make(map[string]*schemaCacheItem),
		lru:     list.New(),
	}
}

// GetValidator retrieves a cached schema validator
func (sc *SchemaCache) GetValidator(toolName string, schemaType string) (*gojsonschema.Schema, bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	key := sc.makeKey(toolName, schemaType)
	item, exists := sc.items[key]
	
	if !exists {
		return nil, false
	}
	
	// Check if item has expired
	if time.Since(item.createdAt) > sc.ttl {
		sc.removeItem(item)
		return nil, false
	}
	
	// Update last used time and move to front of LRU list
	item.lastUsed = time.Now()
	sc.lru.MoveToFront(item.element)
	
	return item.validator, true
}

// SetValidator stores a schema validator in the cache
func (sc *SchemaCache) SetValidator(toolName string, schemaType string, schema *json.RawMessage) (*gojsonschema.Schema, error) {
	// Compile the schema
	schemaLoader := gojsonschema.NewBytesLoader(*schema)
	validator, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}
	
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	key := sc.makeKey(toolName, schemaType)
	
	// Check if we need to evict items
	sc.evictIfNecessary()
	
	// Remove existing item if it exists
	if existingItem, exists := sc.items[key]; exists {
		sc.removeItem(existingItem)
	}
	
	// Create new item
	item := &schemaCacheItem{
		key:       key,
		validator: validator,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
	}
	
	// Add to cache
	item.element = sc.lru.PushFront(item)
	sc.items[key] = item
	
	return validator, nil
}

// CacheToolSchemas caches schemas for multiple tools
func (sc *SchemaCache) CacheToolSchemas(tools []ToolDefinition) error {
	for _, tool := range tools {
		// Cache input schema if present
		if tool.InputSchema != nil {
			if _, err := sc.SetValidator(tool.Name, "input", tool.InputSchema); err != nil {
				return fmt.Errorf("failed to cache input schema for tool %s: %w", tool.Name, err)
			}
		}
		
		// Cache output schema if present
		if tool.OutputSchema != nil {
			if _, err := sc.SetValidator(tool.Name, "output", tool.OutputSchema); err != nil {
				return fmt.Errorf("failed to cache output schema for tool %s: %w", tool.Name, err)
			}
		}
	}
	
	return nil
}

// evictIfNecessary evicts items if the cache is full
func (sc *SchemaCache) evictIfNecessary() {
	for len(sc.items) >= sc.maxSize {
		// Remove least recently used item
		oldest := sc.lru.Back()
		if oldest != nil {
			item := oldest.Value.(*schemaCacheItem)
			sc.removeItem(item)
		} else {
			break
		}
	}
}

// removeItem removes an item from the cache
func (sc *SchemaCache) removeItem(item *schemaCacheItem) {
	delete(sc.items, item.key)
	sc.lru.Remove(item.element)
}

// makeKey creates a cache key for a tool schema
func (sc *SchemaCache) makeKey(toolName string, schemaType string) string {
	return fmt.Sprintf("%s:%s", toolName, schemaType)
}

// Clear removes all cached schemas
func (sc *SchemaCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	sc.items = make(map[string]*schemaCacheItem)
	sc.lru = list.New()
}

// Size returns the current number of cached schemas
func (sc *SchemaCache) Size() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.items)
}

// Stats returns cache statistics
func (sc *SchemaCache) Stats() SchemaCacheStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	stats := SchemaCacheStats{
		Size:    len(sc.items),
		MaxSize: sc.maxSize,
		TTL:     sc.ttl,
	}
	
	// Calculate expired items
	now := time.Now()
	for _, item := range sc.items {
		if now.Sub(item.createdAt) > sc.ttl {
			stats.ExpiredItems++
		}
	}
	
	return stats
}

// SchemaCacheStats represents cache statistics
type SchemaCacheStats struct {
	Size         int           `json:"size"`
	MaxSize      int           `json:"maxSize"`
	TTL          time.Duration `json:"ttl"`
	ExpiredItems int           `json:"expiredItems"`
}

// JSONSchemaValidator implements the ToolValidator interface using JSON Schema
type JSONSchemaValidator struct {
	cache  *SchemaCache
	client MCPClient // Reference to get schemas from tools
}

// NewJSONSchemaValidator creates a new JSON schema validator
func NewJSONSchemaValidator() *JSONSchemaValidator {
	return &JSONSchemaValidator{
		cache: NewSchemaCache(100, 30*time.Minute),
	}
}

// SetClient sets the MCP client for schema retrieval
func (v *JSONSchemaValidator) SetClient(client MCPClient) {
	v.client = client
}

// ValidateInput validates tool input against its schema
func (v *JSONSchemaValidator) ValidateInput(toolName string, input map[string]interface{}) error {
	// Get input schema validator
	validator, found := v.cache.GetValidator(toolName, "input")
	if !found {
		// Try to load schema from client
		if v.client != nil {
			schema, err := v.client.GetToolSchema("", toolName) // Server name empty - find automatically
			if err != nil {
				return fmt.Errorf("failed to get schema for tool %s: %w", toolName, err)
			}
			
			if schema != nil {
				validator, err = v.cache.SetValidator(toolName, "input", schema)
				if err != nil {
					return fmt.Errorf("failed to cache schema for tool %s: %w", toolName, err)
				}
			}
		}
		
		if validator == nil {
			// No schema available, skip validation
			return nil
		}
	}
	
	// Validate input
	inputLoader := gojsonschema.NewGoLoader(input)
	result, err := validator.Validate(inputLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	
	if !result.Valid() {
		errors := make([]string, len(result.Errors()))
		for i, err := range result.Errors() {
			errors[i] = err.String()
		}
		return fmt.Errorf("input validation failed: %v", errors)
	}
	
	return nil
}

// ValidateOutput validates tool output against its schema
func (v *JSONSchemaValidator) ValidateOutput(toolName string, output interface{}) error {
	// Get output schema validator
	validator, found := v.cache.GetValidator(toolName, "output")
	if !found {
		// Try to load schema from client
		if v.client != nil {
			schema, err := v.client.GetToolSchema("", toolName) // Server name empty - find automatically
			if err != nil {
				return fmt.Errorf("failed to get schema for tool %s: %w", toolName, err)
			}
			
			if schema != nil {
				validator, err = v.cache.SetValidator(toolName, "output", schema)
				if err != nil {
					return fmt.Errorf("failed to cache schema for tool %s: %w", toolName, err)
				}
			}
		}
		
		if validator == nil {
			// No schema available, skip validation
			return nil
		}
	}
	
	// Validate output
	outputLoader := gojsonschema.NewGoLoader(output)
	result, err := validator.Validate(outputLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	
	if !result.Valid() {
		errors := make([]string, len(result.Errors()))
		for i, err := range result.Errors() {
			errors[i] = err.String()
		}
		return fmt.Errorf("output validation failed: %v", errors)
	}
	
	return nil
}

// GetInputSchema retrieves the input schema for a tool
func (v *JSONSchemaValidator) GetInputSchema(toolName string) (*json.RawMessage, error) {
	if v.client == nil {
		return nil, fmt.Errorf("no MCP client configured")
	}
	
	return v.client.GetToolSchema("", toolName)
}

// GetOutputSchema retrieves the output schema for a tool
func (v *JSONSchemaValidator) GetOutputSchema(toolName string) (*json.RawMessage, error) {
	if v.client == nil {
		return nil, fmt.Errorf("no MCP client configured")
	}
	
	return v.client.GetToolSchema("", toolName)
}

// Additional methods needed by the MCPClient interface

// GetToolSchema retrieves a tool's schema (implementation for MCPClient interface)
func (c *Client) GetToolSchema(serverName, toolName string) (*json.RawMessage, error) {
	// Find the server and tool
	var conn *ServerConnection
	var tool ToolDefinition
	
	if serverName != "" {
		// Look for specific server
		c.serversMu.RLock()
		var exists bool
		conn, exists = c.servers[serverName]
		c.serversMu.RUnlock()
		
		if !exists {
			return nil, fmt.Errorf("server not found: %s", serverName)
		}
		
		conn.toolsMu.RLock()
		tool, exists = conn.tools[toolName]
		conn.toolsMu.RUnlock()
		
		if !exists {
			return nil, fmt.Errorf("tool not found on server %s: %s", serverName, toolName)
		}
	} else {
		// Find any server that has this tool
		conn, err := c.findToolServer(toolName)
		if err != nil {
			return nil, err
		}
		
		conn.toolsMu.RLock()
		tool = conn.tools[toolName]
		conn.toolsMu.RUnlock()
	}
	
	// Return input schema (could be extended to specify input vs output)
	return tool.InputSchema, nil
}

// CacheToolSchemas caches schemas for multiple tools (implementation for MCPClient interface)
func (c *Client) CacheToolSchemas(tools []ToolDefinition) error {
	if c.schemaCache == nil {
		return nil // Caching disabled
	}
	
	return c.schemaCache.CacheToolSchemas(tools)
}

// ValidateToolOutput validates tool output (implementation for MCPClient interface)
func (c *Client) ValidateToolOutput(toolName string, output interface{}) error {
	if c.validator == nil {
		return nil // Validation disabled
	}
	
	return c.validator.ValidateOutput(toolName, output)
}