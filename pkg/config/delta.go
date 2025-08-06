package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"
)

// DeltaStorage implements Claude CLI's delta storage pattern
// Only changed values are persisted to minimize file size and disk I/O
type DeltaStorage struct {
	defaults interface{}
}

// NewDeltaStorage creates a new delta storage instance with default values
func NewDeltaStorage(defaults interface{}) *DeltaStorage {
	return &DeltaStorage{
		defaults: defaults,
	}
}

// ComputeDelta computes the delta between current values and defaults
// Returns only the values that differ from defaults
func (ds *DeltaStorage) ComputeDelta(current interface{}) (map[string]interface{}, error) {
	// Convert both to JSON for easy comparison
	currentJSON, err := json.Marshal(current)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal current values: %w", err)
	}

	defaultsJSON, err := json.Marshal(ds.defaults)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal defaults: %w", err)
	}

	// Unmarshal to maps for comparison
	var currentMap map[string]interface{}
	if err := json.Unmarshal(currentJSON, &currentMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal current values: %w", err)
	}

	var defaultsMap map[string]interface{}
	if err := json.Unmarshal(defaultsJSON, &defaultsMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal defaults: %w", err)
	}

	// Compute differences
	delta := make(map[string]interface{})
	ds.computeDeltaRecursive(currentMap, defaultsMap, delta, "")

	return delta, nil
}

// computeDeltaRecursive recursively computes differences between current and default values
func (ds *DeltaStorage) computeDeltaRecursive(current, defaults map[string]interface{}, delta map[string]interface{}, prefix string) {
	for key, currentValue := range current {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		defaultValue, hasDefault := defaults[key]

		// If key doesn't exist in defaults, it's a new value
		if !hasDefault {
			delta[fullKey] = currentValue
			continue
		}

		// Handle nested objects
		if currentMap, ok := currentValue.(map[string]interface{}); ok {
			if defaultMap, ok := defaultValue.(map[string]interface{}); ok {
				ds.computeDeltaRecursive(currentMap, defaultMap, delta, fullKey)
				continue
			}
		}

		// Compare values
		if !ds.valuesEqual(currentValue, defaultValue) {
			delta[fullKey] = currentValue
		}
	}
}

// valuesEqual compares two values for equality, handling special cases
func (ds *DeltaStorage) valuesEqual(a, b interface{}) bool {
	// Handle time.Time specially for configuration timestamps
	if timeA, ok := a.(time.Time); ok {
		if timeB, ok := b.(time.Time); ok {
			return timeA.Equal(timeB)
		}
		return false
	}

	// Handle nil values
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Use reflection for deep comparison
	return reflect.DeepEqual(a, b)
}

// ApplyDelta merges delta values with defaults to reconstruct full configuration
func (ds *DeltaStorage) ApplyDelta(delta map[string]interface{}) (interface{}, error) {
	// Start with defaults
	result, err := ds.deepCopy(ds.defaults)
	if err != nil {
		return nil, fmt.Errorf("failed to copy defaults: %w", err)
	}

	// Convert to map for easier manipulation
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(resultJSON, &resultMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	// Apply delta values
	for key, value := range delta {
		ds.setNestedValue(resultMap, key, value)
	}

	// Convert back to original type
	finalJSON, err := json.Marshal(resultMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal final result: %w", err)
	}

	// Create instance of same type as defaults
	resultType := reflect.TypeOf(ds.defaults)
	if resultType.Kind() == reflect.Ptr {
		resultType = resultType.Elem()
	}

	resultPtr := reflect.New(resultType).Interface()
	if err := json.Unmarshal(finalJSON, resultPtr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to final type: %w", err)
	}

	return resultPtr, nil
}

// setNestedValue sets a nested value in a map using dot notation
func (ds *DeltaStorage) setNestedValue(m map[string]interface{}, key string, value interface{}) {
	parts := ds.splitKey(key)
	current := m

	// Navigate to the parent of the final key
	for _, part := range parts[:len(parts)-1] {
		if _, exists := current[part]; !exists {
			current[part] = make(map[string]interface{})
		}

		if nextMap, ok := current[part].(map[string]interface{}); ok {
			current = nextMap
		} else {
			// Create intermediate map
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}

	// Set the final value
	finalKey := parts[len(parts)-1]
	current[finalKey] = value
}

// splitKey splits a dot-notation key into parts
func (ds *DeltaStorage) splitKey(key string) []string {
	// Simple split for now - could be improved to handle escaped dots
	parts := []string{}
	current := ""

	for _, char := range key {
		if char == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// deepCopy creates a deep copy of an interface{}
func (ds *DeltaStorage) deepCopy(src interface{}) (interface{}, error) {
	// Use JSON marshal/unmarshal for deep copy
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}

	var dst interface{}
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, err
	}

	return dst, nil
}

// GlobalConfigDeltaStorage provides delta storage for global configuration
type GlobalConfigDeltaStorage struct {
	*DeltaStorage
}

// NewGlobalConfigDeltaStorage creates delta storage for global configuration
func NewGlobalConfigDeltaStorage() *GlobalConfigDeltaStorage {
	defaults := &GlobalConfig{
		NumStartups:              0,
		Theme:                    "dark",
		PreferredNotifChannel:    "auto",
		Verbose:                  false,
		EditorMode:               "normal",
		AutoCompactEnabled:       true,
		HasSeenTasksHint:         false,
		QueuedCommandUpHintCount: 0,
		DiffTool:                 "auto",
		CustomAPIKeyResponses: struct {
			Approved []string `json:"approved"`
			Rejected []string `json:"rejected"`
		}{
			Approved: []string{},
			Rejected: []string{},
		},
		Env:          make(map[string]string),
		TipsHistory:  make(map[string]interface{}),
		Projects:     make(map[string]*ProjectConfig),
		LastModified: time.Time{}, // Zero time for defaults
		Version:      "1.0.0",
	}

	return &GlobalConfigDeltaStorage{
		DeltaStorage: NewDeltaStorage(defaults),
	}
}

// SaveDelta saves only the changed values from global configuration
func (gcds *GlobalConfigDeltaStorage) SaveDelta(config *GlobalConfig, configPath string) error {
	// Compute delta from defaults
	delta, err := gcds.ComputeDelta(config)
	if err != nil {
		return fmt.Errorf("failed to compute delta: %w", err)
	}

	// Add metadata to delta
	deltaWithMeta := map[string]interface{}{
		"version":      "1.0.0",
		"lastModified": time.Now().Format(time.RFC3339),
		"delta":        delta,
	}

	// Marshal delta to JSON
	data, err := json.MarshalIndent(deltaWithMeta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal delta: %w", err)
	}

	// Use atomic write with file locking
	return AtomicWrite(configPath, data, 0600)
}

// LoadFromDelta loads configuration from delta file
func (gcds *GlobalConfigDeltaStorage) LoadFromDelta(configPath string) (*GlobalConfig, error) {
	// Read delta file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse delta with metadata
	var deltaWithMeta map[string]interface{}
	if err := json.Unmarshal(data, &deltaWithMeta); err != nil {
		return nil, fmt.Errorf("failed to parse delta file: %w", err)
	}

	// Extract delta
	delta, ok := deltaWithMeta["delta"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid delta format in file")
	}

	// Apply delta to defaults
	result, err := gcds.ApplyDelta(delta)
	if err != nil {
		return nil, fmt.Errorf("failed to apply delta: %w", err)
	}

	// Convert to GlobalConfig
	config, ok := result.(*GlobalConfig)
	if !ok {
		return nil, fmt.Errorf("failed to convert result to GlobalConfig")
	}

	return config, nil
}

// ProjectConfigDeltaStorage provides delta storage for project configuration
type ProjectConfigDeltaStorage struct {
	*DeltaStorage
}

// NewProjectConfigDeltaStorage creates delta storage for project configuration
func NewProjectConfigDeltaStorage() *ProjectConfigDeltaStorage {
	defaults := &ProjectConfig{
		AllowedTools:                            []string{},
		History:                                 []ConversationMessage{},
		MCPContextUris:                          []string{},
		MCPServers:                              make(map[string]interface{}),
		EnabledMCPServers:                       []string{},
		DisabledMCPServers:                      []string{},
		HasTrustDialogAccepted:                  false,
		IgnorePatterns:                          []string{},
		ProjectOnboardingSeenCount:              0,
		HasClaudeMdExternalIncludesApproved:     false,
		HasClaudeMdExternalIncludesWarningShown: false,
		ProjectRoot:                             "",
		LastModified:                            time.Time{}, // Zero time for defaults
	}

	return &ProjectConfigDeltaStorage{
		DeltaStorage: NewDeltaStorage(defaults),
	}
}
