package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// CompatibilityChecker manages background testing of tool compatibility with models
type CompatibilityChecker struct {
	registry     *Registry
	ollamaClient OllamaClientInterface
	currentModel string
	mu           sync.RWMutex
	stopChan     chan struct{}
	isRunning    bool
	testInterval time.Duration
	logger       *logger.Logger
}

// OllamaClientInterface defines the interface needed to test model compatibility
type OllamaClientInterface interface {
	ChatWithTools(ctx context.Context, model string, messages []map[string]interface{}, tools []map[string]interface{}) (map[string]interface{}, error)
}

// NewCompatibilityChecker creates a new tool compatibility checker
func NewCompatibilityChecker(registry *Registry, ollamaClient OllamaClientInterface) *CompatibilityChecker {
	return &CompatibilityChecker{
		registry:     registry,
		ollamaClient: ollamaClient,
		currentModel: "",
		stopChan:     make(chan struct{}),
		isRunning:    false,
		testInterval: 30 * time.Second, // Test every 30 seconds
		logger:       logger.WithComponent("compatibility_checker"),
	}
}

// SetModel updates the current model and triggers compatibility checks
func (cc *CompatibilityChecker) SetModel(modelName string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.currentModel != modelName {
		cc.currentModel = modelName
		cc.logger.Debug("Model changed, will test tool compatibility", "model", modelName)

		// Mark all tools as unknown for the new model
		tools := cc.registry.GetTools()
		for toolName := range tools {
			cc.registry.SetToolCompatibility(toolName, modelName, CompatibilityUnknown)
		}
	}
}

// Start begins the background compatibility checking process
func (cc *CompatibilityChecker) Start() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.isRunning {
		return
	}

	cc.isRunning = true
	cc.stopChan = make(chan struct{})

	go cc.runBackgroundChecker()
	cc.logger.Debug("Tool compatibility checker started")
}

// Stop halts the background compatibility checking process
func (cc *CompatibilityChecker) Stop() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if !cc.isRunning {
		return
	}

	close(cc.stopChan)
	cc.isRunning = false
	cc.logger.Debug("Tool compatibility checker stopped")
}

// runBackgroundChecker is the main background loop for checking tool compatibility
func (cc *CompatibilityChecker) runBackgroundChecker() {
	ticker := time.NewTicker(cc.testInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cc.stopChan:
			return
		case <-ticker.C:
			cc.checkAllToolsCompatibility()
		}
	}
}

// checkAllToolsCompatibility tests all registered tools with the current model
func (cc *CompatibilityChecker) checkAllToolsCompatibility() {
	cc.mu.RLock()
	model := cc.currentModel
	cc.mu.RUnlock()

	if model == "" {
		return // No model set yet
	}

	tools := cc.registry.GetTools()
	for toolName, tool := range tools {
		// Skip if recently tested
		stats := cc.registry.GetToolStats(toolName)
		if lastTested, exists := stats.LastTested[model]; exists {
			if time.Since(lastTested) < cc.testInterval {
				continue // Recently tested, skip
			}
		}

		// Check current compatibility status
		currentStatus := cc.registry.GetToolCompatibility(toolName, model)
		if currentStatus == CompatibilityTesting {
			continue // Already being tested
		}

		// Start testing in a separate goroutine to avoid blocking
		go cc.testToolCompatibility(model, toolName, tool)
	}
}

// testToolCompatibility tests if a specific tool is compatible with a model
func (cc *CompatibilityChecker) testToolCompatibility(model, toolName string, tool Tool) {
	cc.logger.Debug("Testing tool compatibility", "tool", toolName, "model", model)

	// Mark as testing
	cc.registry.SetToolCompatibility(toolName, model, CompatibilityTesting)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a simple test message that would likely trigger tool use
	testMessage := fmt.Sprintf("Test if you can use the %s tool. Just respond with 'yes' if you can use it, 'no' if you cannot.", toolName)

	messages := []map[string]interface{}{
		{
			"role":    "user",
			"content": testMessage,
		},
	}

	// Get tool definition for this model
	toolDef, err := cc.getToolDefinition(tool)
	if err != nil {
		cc.logger.Debug("Failed to get tool definition", "tool", toolName, "error", err)
		cc.registry.SetToolCompatibility(toolName, model, CompatibilityUnsupported)
		return
	}

	toolDefs := []map[string]interface{}{toolDef}

	// Test with the model
	response, err := cc.ollamaClient.ChatWithTools(ctx, model, messages, toolDefs)
	if err != nil {
		cc.logger.Debug("Tool compatibility test failed", "tool", toolName, "model", model, "error", err)

		// Check if error suggests tool support issues
		if cc.isToolSupportError(err) {
			cc.registry.SetToolCompatibility(toolName, model, CompatibilityUnsupported)
		} else {
			// Network or other issues - keep as unknown
			cc.registry.SetToolCompatibility(toolName, model, CompatibilityUnknown)
		}
		return
	}

	// Analyze response to determine compatibility
	compatible := cc.analyzeToolResponse(response)
	if compatible {
		cc.registry.SetToolCompatibility(toolName, model, CompatibilitySupported)
		cc.logger.Debug("Tool compatibility confirmed", "tool", toolName, "model", model)
	} else {
		cc.registry.SetToolCompatibility(toolName, model, CompatibilityUnsupported)
		cc.logger.Debug("Tool not compatible", "tool", toolName, "model", model)
	}
}

// getToolDefinition converts a tool to the format expected by Ollama
func (cc *CompatibilityChecker) getToolDefinition(tool Tool) (map[string]interface{}, error) {
	// Convert to Ollama/OpenAI tool format
	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  tool.JSONSchema(),
		},
	}, nil
}

// isToolSupportError determines if an error indicates lack of tool support
func (cc *CompatibilityChecker) isToolSupportError(err error) bool {
	errorStr := strings.ToLower(err.Error())
	toolErrorKeywords := []string{
		"tool",
		"function",
		"unsupported",
		"not supported",
		"invalid",
		"unknown parameter",
	}

	for _, keyword := range toolErrorKeywords {
		if strings.Contains(errorStr, keyword) {
			return true
		}
	}

	return false
}

// analyzeToolResponse analyzes the model's response to determine tool compatibility
func (cc *CompatibilityChecker) analyzeToolResponse(response map[string]interface{}) bool {
	// Check if response contains tool calls
	if toolCalls, exists := response["tool_calls"]; exists {
		if calls, ok := toolCalls.([]interface{}); ok && len(calls) > 0 {
			return true // Model made tool calls, so it supports tools
		}
	}

	// Check message content for positive indicators
	if message, exists := response["message"]; exists {
		if msg, ok := message.(map[string]interface{}); ok {
			if content, exists := msg["content"]; exists {
				if contentStr, ok := content.(string); ok {
					contentLower := strings.ToLower(contentStr)

					// Look for positive indicators
					positiveIndicators := []string{
						"yes",
						"can use",
						"available",
						"supported",
					}

					negativeIndicators := []string{
						"no",
						"cannot",
						"can't",
						"not available",
						"not supported",
						"unsupported",
					}

					// Check for negative indicators first (they're more definitive)
					for _, neg := range negativeIndicators {
						if strings.Contains(contentLower, neg) {
							return false
						}
					}

					// Check for positive indicators
					for _, pos := range positiveIndicators {
						if strings.Contains(contentLower, pos) {
							return true
						}
					}
				}
			}
		}
	}

	// If we get a valid response without errors, assume basic compatibility
	return true
}

// IsRunning returns whether the compatibility checker is currently running
func (cc *CompatibilityChecker) IsRunning() bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.isRunning
}

// GetCurrentModel returns the current model being tested
func (cc *CompatibilityChecker) GetCurrentModel() string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.currentModel
}

// SetTestInterval updates the interval between compatibility checks
func (cc *CompatibilityChecker) SetTestInterval(interval time.Duration) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.testInterval = interval
}
