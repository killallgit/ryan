package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)


// CompatibilityResult represents the result of a compatibility test
type CompatibilityResult struct {
	ToolName           string                `json:"tool_name"`
	ModelName          string                `json:"model_name"`
	Status             ToolCompatibilityStatus   `json:"status"`
	SupportsToolCalls  bool                  `json:"supports_tool_calls"`
	SupportsJSONSchema bool                  `json:"supports_json_schema"`
	TestDuration       time.Duration         `json:"test_duration"`
	TestedAt           time.Time             `json:"tested_at"`
	ErrorMessage       string                `json:"error_message,omitempty"`
	Details            map[string]interface{} `json:"details,omitempty"`
}

// ModelCapabilityTester is a test interface for model capabilities
type ModelCapabilityTester interface {
	// TestToolSupport tests if a model supports tool calling with a specific tool
	TestToolSupport(ctx context.Context, modelName string, toolDefinition map[string]any) (*CompatibilityResult, error)
	
	// TestModelCapabilities tests general model capabilities (tool calling, JSON schema support)
	TestModelCapabilities(ctx context.Context, modelName string) (bool, bool, error)
}

// CompatibilityTester manages background testing of tool compatibility with models
type CompatibilityTester struct {
	registry       *Registry
	modelTester    ModelCapabilityTester
	results        map[string]map[string]*CompatibilityResult // [modelName][toolName] -> result
	testQueue      chan testRequest
	resultHandlers []CompatibilityResultHandler
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	workers        int
}

type testRequest struct {
	modelName string
	toolName  string
	tool      Tool
	priority  int // Higher priority = tested first
}

// CompatibilityResultHandler is called when a compatibility test completes
type CompatibilityResultHandler func(result *CompatibilityResult)

// NewCompatibilityTester creates a new compatibility tester
func NewCompatibilityTester(registry *Registry, modelTester ModelCapabilityTester) *CompatibilityTester {
	ctx, cancel := context.WithCancel(context.Background())
	
	tester := &CompatibilityTester{
		registry:       registry,
		modelTester:    modelTester,
		results:        make(map[string]map[string]*CompatibilityResult),
		testQueue:      make(chan testRequest, 100), // Buffered queue
		resultHandlers: make([]CompatibilityResultHandler, 0),
		ctx:            ctx,
		cancel:         cancel,
		workers:        2, // Run 2 background workers by default
	}
	
	// Start background workers
	for i := 0; i < tester.workers; i++ {
		tester.wg.Add(1)
		go tester.worker()
	}
	
	return tester
}

// AddResultHandler adds a handler that will be called when compatibility results are available
func (ct *CompatibilityTester) AddResultHandler(handler CompatibilityResultHandler) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.resultHandlers = append(ct.resultHandlers, handler)
}

// TestModelCompatibility queues compatibility tests for all tools with the given model
func (ct *CompatibilityTester) TestModelCompatibility(modelName string, priority int) {
	log := logger.WithComponent("compatibility_tester")
	log.Debug("Queuing compatibility tests for model", "model", modelName, "priority", priority)
	
	tools := ct.registry.GetTools()
	
	for toolName, tool := range tools {
		// Mark as testing
		ct.setResult(modelName, toolName, &CompatibilityResult{
			ToolName:  toolName,
			ModelName: modelName,
			Status:    CompatibilityTesting,
			TestedAt:  time.Now(),
		})
		
		// Queue for testing
		select {
		case ct.testQueue <- testRequest{
			modelName: modelName,
			toolName:  toolName,
			tool:      tool,
			priority:  priority,
		}:
			log.Debug("Queued compatibility test", "model", modelName, "tool", toolName)
		case <-ct.ctx.Done():
			log.Debug("Context cancelled, not queuing test", "model", modelName, "tool", toolName)
			return
		default:
			log.Warn("Test queue full, skipping test", "model", modelName, "tool", toolName)
		}
	}
}

// GetCompatibilityResult returns the compatibility result for a specific model and tool
func (ct *CompatibilityTester) GetCompatibilityResult(modelName, toolName string) *CompatibilityResult {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	
	if modelResults, exists := ct.results[modelName]; exists {
		return modelResults[toolName]
	}
	return nil
}

// GetModelCompatibility returns all compatibility results for a specific model
func (ct *CompatibilityTester) GetModelCompatibility(modelName string) map[string]*CompatibilityResult {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	
	if modelResults, exists := ct.results[modelName]; exists {
		// Return a copy to prevent external modification
		results := make(map[string]*CompatibilityResult, len(modelResults))
		for toolName, result := range modelResults {
			resultCopy := *result // Create a copy
			results[toolName] = &resultCopy
		}
		return results
	}
	return make(map[string]*CompatibilityResult)
}

// GetAllCompatibilityResults returns all compatibility results
func (ct *CompatibilityTester) GetAllCompatibilityResults() map[string]map[string]*CompatibilityResult {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	
	// Return a deep copy to prevent external modification
	results := make(map[string]map[string]*CompatibilityResult, len(ct.results))
	for modelName, modelResults := range ct.results {
		results[modelName] = make(map[string]*CompatibilityResult, len(modelResults))
		for toolName, result := range modelResults {
			resultCopy := *result // Create a copy
			results[modelName][toolName] = &resultCopy
		}
	}
	return results
}

// setResult stores a compatibility result
func (ct *CompatibilityTester) setResult(modelName, toolName string, result *CompatibilityResult) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	if ct.results[modelName] == nil {
		ct.results[modelName] = make(map[string]*CompatibilityResult)
	}
	ct.results[modelName][toolName] = result
	
	// Notify handlers
	for _, handler := range ct.resultHandlers {
		go handler(result) // Run handlers asynchronously
	}
}

// worker runs background compatibility tests
func (ct *CompatibilityTester) worker() {
	defer ct.wg.Done()
	log := logger.WithComponent("compatibility_tester_worker")
	
	for {
		select {
		case req := <-ct.testQueue:
			ct.processTestRequest(req)
		case <-ct.ctx.Done():
			log.Debug("Worker stopping due to context cancellation")
			return
		}
	}
}

// processTestRequest processes a single compatibility test request
func (ct *CompatibilityTester) processTestRequest(req testRequest) {
	log := logger.WithComponent("compatibility_tester")
	log.Debug("Processing compatibility test", "model", req.modelName, "tool", req.toolName)
	
	startTime := time.Now()
	
	// Create a timeout context for this test
	testCtx, cancel := context.WithTimeout(ct.ctx, 30*time.Second)
	defer cancel()
	
	result := &CompatibilityResult{
		ToolName:     req.toolName,
		ModelName:    req.modelName,
		Status:       CompatibilityTesting,
		TestDuration: 0,
		TestedAt:     startTime,
		Details:      make(map[string]interface{}),
	}
	
	// Test model capabilities first (tool calling support, JSON schema support)
	supportsToolCalls, supportsJSONSchema, err := ct.modelTester.TestModelCapabilities(testCtx, req.modelName)
	if err != nil {
		log.Error("Failed to test model capabilities", "model", req.modelName, "error", err)
		result.Status = CompatibilityError
		result.ErrorMessage = fmt.Sprintf("Failed to test model capabilities: %v", err)
		result.TestDuration = time.Since(startTime)
		ct.setResult(req.modelName, req.toolName, result)
		return
	}
	
	result.SupportsToolCalls = supportsToolCalls
	result.SupportsJSONSchema = supportsJSONSchema
	result.Details["model_capabilities_test"] = map[string]interface{}{
		"supports_tool_calls":  supportsToolCalls,
		"supports_json_schema": supportsJSONSchema,
	}
	
	// If model doesn't support tool calls, mark as unsupported
	if !supportsToolCalls {
		log.Debug("Model does not support tool calls", "model", req.modelName)
		result.Status = CompatibilityUnsupported
		result.ErrorMessage = "Model does not support tool calling"
		result.TestDuration = time.Since(startTime)
		ct.setResult(req.modelName, req.toolName, result)
		return
	}
	
	// Test specific tool compatibility
	toolSchema := req.tool.JSONSchema()
	testResult, err := ct.modelTester.TestToolSupport(testCtx, req.modelName, toolSchema)
	if err != nil {
		log.Error("Failed to test tool support", "model", req.modelName, "tool", req.toolName, "error", err)
		result.Status = CompatibilityError
		result.ErrorMessage = fmt.Sprintf("Failed to test tool support: %v", err)
		result.TestDuration = time.Since(startTime)
		ct.setResult(req.modelName, req.toolName, result)
		return
	}
	
	// Merge test result
	if testResult != nil {
		result.Status = testResult.Status
		result.ErrorMessage = testResult.ErrorMessage
		if testResult.Details != nil {
			for k, v := range testResult.Details {
				result.Details[k] = v
			}
		}
	} else {
		// Default to supported if no specific test result
		result.Status = CompatibilitySupported
	}
	
	result.TestDuration = time.Since(startTime)
	log.Debug("Completed compatibility test", 
		"model", req.modelName, 
		"tool", req.toolName, 
		"status", result.Status.String(),
		"duration", result.TestDuration)
	
	ct.setResult(req.modelName, req.toolName, result)
}

// Shutdown stops the compatibility tester and waits for workers to finish
func (ct *CompatibilityTester) Shutdown() {
	log := logger.WithComponent("compatibility_tester")
	log.Debug("Shutting down compatibility tester")
	
	ct.cancel()
	ct.wg.Wait()
	close(ct.testQueue)
	
	log.Debug("Compatibility tester shutdown complete")
}

// MockModelTester provides a mock implementation for testing
type MockModelTester struct{}

// TestToolSupport implements a mock test that randomly determines tool support
func (m *MockModelTester) TestToolSupport(ctx context.Context, modelName string, toolDefinition map[string]any) (*CompatibilityResult, error) {
	// Simulate some testing time
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	
	// Mock logic: assume tools are generally supported unless they have complex schemas
	status := CompatibilitySupported
	
	// Check if tool has complex parameters that might not be supported
	if props, ok := toolDefinition["properties"].(map[string]any); ok {
		if len(props) > 5 {
			// Complex tools might not be supported by all models
			if strings.Contains(strings.ToLower(modelName), "small") || 
			   strings.Contains(strings.ToLower(modelName), "mini") {
				status = CompatibilityUnsupported
			}
		}
	}
	
	result := &CompatibilityResult{
		Status: status,
		Details: map[string]interface{}{
			"mock_test": true,
			"schema_complexity": len(toolDefinition),
		},
	}
	
	if status == CompatibilityUnsupported {
		result.ErrorMessage = "Mock: Model appears to have limited tool calling capabilities"
	}
	
	return result, nil
}

// TestModelCapabilities implements a mock test for general model capabilities
func (m *MockModelTester) TestModelCapabilities(ctx context.Context, modelName string) (bool, bool, error) {
	// Simulate some testing time
	select {
	case <-time.After(50 * time.Millisecond):
	case <-ctx.Done():
		return false, false, ctx.Err()
	}
	
	// Mock logic: assume most models support tool calls and JSON schema
	supportsToolCalls := true
	supportsJSONSchema := true
	
	// Some models might not support advanced features
	if strings.Contains(strings.ToLower(modelName), "base") ||
	   strings.Contains(strings.ToLower(modelName), "instruct") {
		supportsToolCalls = false
		supportsJSONSchema = false
	}
	
	return supportsToolCalls, supportsJSONSchema, nil
}

// GetQueueLength returns the current number of tests in the queue
func (ct *CompatibilityTester) GetQueueLength() int {
	return len(ct.testQueue)
}

// ClearResults clears all compatibility test results
func (ct *CompatibilityTester) ClearResults() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.results = make(map[string]map[string]*CompatibilityResult)
}

// IsModelTested returns true if all tools have been tested for the given model
func (ct *CompatibilityTester) IsModelTested(modelName string) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	
	modelResults, exists := ct.results[modelName]
	if !exists {
		return false
	}
	
	tools := ct.registry.GetTools()
	for toolName := range tools {
		result, exists := modelResults[toolName]
		if !exists || result.Status == CompatibilityTesting || result.Status == CompatibilityUnknown {
			return false
		}
	}
	
	return true
}

// GetCompatibilitySummary returns a summary of compatibility for a model
func (ct *CompatibilityTester) GetCompatibilitySummary(modelName string) (supported, unsupported, testing, total int) {
	results := ct.GetModelCompatibility(modelName)
	total = len(ct.registry.GetTools())
	
	for _, result := range results {
		switch result.Status {
		case CompatibilitySupported:
			supported++
		case CompatibilityUnsupported, CompatibilityError:
			unsupported++
		case CompatibilityTesting:
			testing++
		}
	}
	
	return supported, unsupported, testing, total
}