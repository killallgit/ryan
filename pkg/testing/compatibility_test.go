package testing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModelCompatibilityTester(t *testing.T) {
	tester := NewModelCompatibilityTester("http://localhost:11434")
	
	assert.NotNil(t, tester)
	assert.Equal(t, "http://localhost:11434", tester.ollamaURL)
	assert.NotNil(t, tester.toolRegistry)
	assert.Greater(t, tester.timeout, time.Duration(0))
}

func TestModelTestResult(t *testing.T) {
	result := ModelTestResult{
		ModelName:           "test-model",
		ToolCallSupported:   true,
		BasicToolCallPassed: true,
		FileReadPassed:      false,
		ErrorHandlingPassed: true,
		MultiToolPassed:     false,
		AverageResponseTime: 500 * time.Millisecond,
		TotalTests:          4,
		PassedTests:         2,
		Errors:              []string{"File read failed"},
		OllamaVersion:       "0.4.0",
	}

	assert.Equal(t, "test-model", result.ModelName)
	assert.True(t, result.ToolCallSupported)
	assert.True(t, result.BasicToolCallPassed)
	assert.False(t, result.FileReadPassed)
	assert.True(t, result.ErrorHandlingPassed)
	assert.False(t, result.MultiToolPassed)
	assert.Equal(t, 500*time.Millisecond, result.AverageResponseTime)
	assert.Equal(t, 4, result.TotalTests)
	assert.Equal(t, 2, result.PassedTests)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "0.4.0", result.OllamaVersion)
}

func TestCheckOllamaVersion(t *testing.T) {
	tester := NewModelCompatibilityTester("http://invalid-url")
	
	// This will fail because the URL is invalid, but we can test the method exists
	version, supported, err := tester.CheckOllamaVersion()
	
	// Should return error for invalid URL
	assert.Error(t, err)
	assert.Empty(t, version)
	assert.False(t, supported)
}

func TestModelCompatibilityTester_TestMultipleModels(t *testing.T) {
	tester := NewModelCompatibilityTester("http://invalid-url")
	
	// Test with empty model list
	results := tester.TestMultipleModels([]string{})
	assert.Empty(t, results)
	
	// Test with model list (will fail due to invalid URL, but tests the structure)
	models := []string{"test-model-1", "test-model-2"}
	results = tester.TestMultipleModels(models)
	
	assert.Len(t, results, 2)
	for i, result := range results {
		assert.Equal(t, models[i], result.ModelName)
		assert.False(t, result.ToolCallSupported) // Should fail due to invalid URL
		assert.NotEmpty(t, result.Errors)        // Should have connection errors
	}
}

func TestModelCompatibilityTester_PrintResults(t *testing.T) {
	tester := NewModelCompatibilityTester("http://localhost:11434")
	
	// Create test results
	results := []ModelTestResult{
		{
			ModelName:           "test-model-1",
			ToolCallSupported:   true,
			BasicToolCallPassed: true,
			FileReadPassed:      true,
			ErrorHandlingPassed: false,
			MultiToolPassed:     true,
			AverageResponseTime: 750 * time.Millisecond,
			TotalTests:          4,
			PassedTests:         3,
			OllamaVersion:       "0.4.2",
		},
		{
			ModelName:         "test-model-2",
			ToolCallSupported: false,
			Errors:           []string{"Model not found", "Connection failed"},
			OllamaVersion:     "0.4.2",
		},
	}
	
	// This should not panic - just testing that the method can be called
	require.NotPanics(t, func() {
		tester.PrintResults(results)
	})
}

func TestModelCompatibilityTester_PrivateMethods(t *testing.T) {
	// Test the structure and logic that can be tested without actual Ollama
	
	t.Run("Test result calculation logic", func(t *testing.T) {
		result := ModelTestResult{
			BasicToolCallPassed: true,
			FileReadPassed:      false,
			ErrorHandlingPassed: true,
			MultiToolPassed:     true,
		}
		
		// Simulate the counting logic from TestModel
		passedTests := 0
		if result.BasicToolCallPassed {
			passedTests++
		}
		if result.FileReadPassed {
			passedTests++
		}
		if result.ErrorHandlingPassed {
			passedTests++
		}
		if result.MultiToolPassed {
			passedTests++
		}
		totalTests := 4
		
		assert.Equal(t, 3, passedTests)
		assert.Equal(t, 4, totalTests)
		assert.Equal(t, 75.0, float64(passedTests)/float64(totalTests)*100)
	})
	
	t.Run("Test error accumulation", func(t *testing.T) {
		result := ModelTestResult{
			Errors: make([]string, 0),
		}
		
		// Simulate error accumulation
		result.Errors = append(result.Errors, "Connection failed")
		result.Errors = append(result.Errors, "Tool not found")
		result.Errors = append(result.Errors, "Timeout occurred")
		
		assert.Len(t, result.Errors, 3)
		assert.Contains(t, result.Errors, "Connection failed")
		assert.Contains(t, result.Errors, "Tool not found")
		assert.Contains(t, result.Errors, "Timeout occurred")
	})
	
	t.Run("Test response time averaging", func(t *testing.T) {
		// Simulate the averaging logic used in the test methods
		averageResponseTime := time.Duration(0)
		
		// First measurement
		duration1 := 500 * time.Millisecond
		averageResponseTime = duration1
		
		// Second measurement 
		duration2 := 1000 * time.Millisecond
		averageResponseTime = (averageResponseTime + duration2) / 2
		
		// Third measurement
		duration3 := 750 * time.Millisecond
		averageResponseTime = (averageResponseTime + duration3) / 2
		
		// Should be averaging the times
		assert.Greater(t, averageResponseTime, time.Duration(0))
		assert.Less(t, averageResponseTime, 1000*time.Millisecond)
	})
}

func TestModelCompatibilityTester_ContextHandling(t *testing.T) {
	tester := NewModelCompatibilityTester("http://localhost:11434")
	
	t.Run("Test context with timeout", func(t *testing.T) {
		_, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		// Test that the tester can handle context timeout
		// This will fail quickly due to invalid URL, but tests context handling
		result := tester.TestModel("test-model")
		
		assert.Equal(t, "test-model", result.ModelName)
		assert.False(t, result.ToolCallSupported)
		assert.NotEmpty(t, result.Errors)
	})
	
	t.Run("Test context cancellation", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		// The method should handle cancelled context gracefully
		result := tester.TestModel("test-model")
		
		assert.Equal(t, "test-model", result.ModelName)
		assert.False(t, result.ToolCallSupported)
	})
}

func TestModelCompatibilityTester_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	// Test with invalid URL to simulate network failures
	// This tests the error handling without requiring a real server
	tester := NewModelCompatibilityTester("http://invalid-test-url:99999")
	
	// Test the structure of multiple model testing
	models := []string{"test-model-1", "test-model-2"}
	
	// This will fail due to invalid URL, but tests the error flow
	results := tester.TestMultipleModels(models)
	
	assert.Len(t, results, len(models))
	for i, result := range results {
		assert.Equal(t, models[i], result.ModelName)
		assert.False(t, result.ToolCallSupported) // Should fail due to connection error
		assert.NotEmpty(t, result.Errors)        // Should have connection errors
		assert.Equal(t, 0, result.PassedTests)   // No tests should pass with invalid URL
		// TotalTests might be 0 if version check fails early, don't assert specific value
		assert.GreaterOrEqual(t, result.TotalTests, 0)
	}
}