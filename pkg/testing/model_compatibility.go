package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/spf13/viper"
)

// OllamaVersion represents version information from Ollama server
type OllamaVersion struct {
	Version string `json:"version"`
}

// ModelTestResult represents the test results for a specific model
type ModelTestResult struct {
	ModelName           string
	ToolCallSupported   bool
	BasicToolCallPassed bool
	FileReadPassed      bool
	ErrorHandlingPassed bool
	MultiToolPassed     bool
	AverageResponseTime time.Duration
	TotalTests          int
	PassedTests         int
	Errors              []string
	OllamaVersion       string
}

// ModelCompatibilityTester provides tools for testing model compatibility
type ModelCompatibilityTester struct {
	ollamaURL    string
	toolRegistry *tools.Registry
	timeout      time.Duration
}

// NewModelCompatibilityTester creates a new model compatibility tester
func NewModelCompatibilityTester(ollamaURL string) *ModelCompatibilityTester {
	toolRegistry := tools.NewRegistry()
	if err := toolRegistry.RegisterBuiltinTools(); err != nil {
		log.Printf("Failed to register built-in tools: %v", err)
	}
	ollamaTimeout := viper.GetDuration("ollama.timeout")
	return &ModelCompatibilityTester{
		ollamaURL:    ollamaURL,
		toolRegistry: toolRegistry,
		timeout:      ollamaTimeout,
	}
}

// CheckOllamaVersion checks if the Ollama server supports tool calling
func (mct *ModelCompatibilityTester) CheckOllamaVersion() (string, bool, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/version", mct.ollamaURL))
	if err != nil {
		return "", false, fmt.Errorf("failed to check Ollama version: %w", err)
	}
	defer resp.Body.Close()

	var version OllamaVersion
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return "", false, fmt.Errorf("failed to decode version response: %w", err)
	}

	// Tool calling was introduced in Ollama 0.4.x, became more stable in 1.0+
	supported := mct.versionSupportsTools(version.Version)
	return version.Version, supported, nil
}

// versionSupportsTools checks if a version string indicates tool calling support
func (mct *ModelCompatibilityTester) versionSupportsTools(version string) bool {
	// Extract major and minor version numbers
	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) < 3 {
		return false // Can't parse version, assume no support
	}

	major, err1 := strconv.Atoi(matches[1])
	minor, err2 := strconv.Atoi(matches[2])
	if err1 != nil || err2 != nil {
		return false
	}

	// Tool calling support introduced in 0.4.x, more stable in 1.0+
	if major > 1 {
		return true
	}
	if major == 1 {
		return true
	}
	if major == 0 && minor >= 4 {
		return true
	}

	return false
}

// TestModel runs comprehensive compatibility tests on a specific model
func (mct *ModelCompatibilityTester) TestModel(modelName string) ModelTestResult {
	result := ModelTestResult{
		ModelName: modelName,
		Errors:    make([]string, 0),
	}

	// Check Ollama version first
	version, versionSupported, err := mct.CheckOllamaVersion()
	result.OllamaVersion = version
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Version check failed: %v", err))
		return result
	}

	if !versionSupported {
		result.Errors = append(result.Errors, fmt.Sprintf("Ollama version %s does not support tool calling (requires 0.4.0+)", version))
		return result
	}

	client := chat.NewClient(mct.ollamaURL)
	controller := controllers.NewChatController(client, modelName, mct.toolRegistry)

	ctx, cancel := context.WithTimeout(context.Background(), mct.timeout)
	defer cancel()

	log.Printf("Testing model: %s (Ollama v%s)", modelName, version)

	// Test 1: Basic tool call support detection
	result.ToolCallSupported = mct.testToolCallSupport(ctx, controller, &result)

	if !result.ToolCallSupported {
		result.Errors = append(result.Errors, "Model does not support tool calling")
		return result
	}

	// Test 2: Basic bash command execution
	result.BasicToolCallPassed = mct.testBasicToolCall(ctx, controller, &result)

	// Test 3: File reading functionality
	result.FileReadPassed = mct.testFileRead(ctx, controller, &result)

	// Test 4: Error handling
	result.ErrorHandlingPassed = mct.testErrorHandling(ctx, controller, &result)

	// Test 5: Multi-tool sequence
	result.MultiToolPassed = mct.testMultiToolSequence(ctx, controller, &result)

	// Calculate pass rate
	if result.BasicToolCallPassed {
		result.PassedTests++
	}
	if result.FileReadPassed {
		result.PassedTests++
	}
	if result.ErrorHandlingPassed {
		result.PassedTests++
	}
	if result.MultiToolPassed {
		result.PassedTests++
	}
	result.TotalTests = 4

	log.Printf("Model %s test completed: %d/%d tests passed", modelName, result.PassedTests, result.TotalTests)
	return result
}

// testToolCallSupport checks if the model can make tool calls
func (mct *ModelCompatibilityTester) testToolCallSupport(ctx context.Context, controller *controllers.ChatController, result *ModelTestResult) bool {
	startTime := time.Now()

	response, err := controller.SendUserMessageWithContext(ctx, "Run the command 'echo hello world' using the execute_bash tool")

	duration := time.Since(startTime)
	result.AverageResponseTime = duration

	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Tool call support test failed: %v", err))
		return false
	}

	// Check if response mentions tool execution or contains expected output
	responseContent := strings.ToLower(response.Content)
	if strings.Contains(responseContent, "hello world") ||
		strings.Contains(responseContent, "executed") ||
		strings.Contains(responseContent, "command") {
		return true
	}

	// Check conversation history for tool calls
	history := controller.GetHistory()
	for _, msg := range history {
		if msg.HasToolCalls() {
			return true
		}
		if msg.IsTool() && msg.ToolName == "execute_bash" {
			return true
		}
	}

	result.Errors = append(result.Errors, "No evidence of tool calling in response or conversation history")
	return false
}

// testBasicToolCall tests basic bash command execution
func (mct *ModelCompatibilityTester) testBasicToolCall(ctx context.Context, controller *controllers.ChatController, result *ModelTestResult) bool {
	startTime := time.Now()

	response, err := controller.SendUserMessageWithContext(ctx, "Use the execute_bash tool to run 'pwd' and tell me the current directory")

	duration := time.Since(startTime)
	if result.AverageResponseTime == 0 {
		result.AverageResponseTime = duration
	} else {
		result.AverageResponseTime = (result.AverageResponseTime + duration) / 2
	}

	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Basic tool call test failed: %v", err))
		return false
	}

	// Check if response contains directory information
	if strings.Contains(response.Content, "/") {
		return true
	}

	result.Errors = append(result.Errors, "Basic tool call test did not return expected directory information")
	return false
}

// testFileRead tests file reading functionality
func (mct *ModelCompatibilityTester) testFileRead(ctx context.Context, controller *controllers.ChatController, result *ModelTestResult) bool {
	startTime := time.Now()

	// Create a test file first
	testFilePath := "/tmp/ryan_test_file.txt"
	testContent := "This is a test file for Ryan tool testing"

	// Create the test file using bash tool
	_, err := controller.SendUserMessageWithContext(ctx, fmt.Sprintf("Use execute_bash to create a test file: echo '%s' > %s", testContent, testFilePath))
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to create test file: %v", err))
		return false
	}

	// Now test file reading
	response, err := controller.SendUserMessageWithContext(ctx, fmt.Sprintf("Use the read_file tool to read the contents of %s", testFilePath))

	duration := time.Since(startTime)
	if result.AverageResponseTime == 0 {
		result.AverageResponseTime = duration
	} else {
		result.AverageResponseTime = (result.AverageResponseTime + duration) / 2
	}

	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("File read test failed: %v", err))
		return false
	}

	// Check if response contains the test content
	if strings.Contains(response.Content, testContent) {
		// Clean up test file
		controller.SendUserMessageWithContext(ctx, fmt.Sprintf("Use execute_bash to clean up: rm %s", testFilePath))
		return true
	}

	result.Errors = append(result.Errors, "File read test did not return expected content")
	return false
}

// testErrorHandling tests how the model handles tool errors
func (mct *ModelCompatibilityTester) testErrorHandling(ctx context.Context, controller *controllers.ChatController, result *ModelTestResult) bool {
	startTime := time.Now()

	response, err := controller.SendUserMessageWithContext(ctx, "Use execute_bash to run an invalid command: 'nonexistentcommand12345'")

	duration := time.Since(startTime)
	if result.AverageResponseTime == 0 {
		result.AverageResponseTime = duration
	} else {
		result.AverageResponseTime = (result.AverageResponseTime + duration) / 2
	}

	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Error handling test failed: %v", err))
		return false
	}

	// Check if response indicates error handling
	responseContent := strings.ToLower(response.Content)
	if strings.Contains(responseContent, "error") ||
		strings.Contains(responseContent, "failed") ||
		strings.Contains(responseContent, "not found") ||
		strings.Contains(responseContent, "command not found") {
		return true
	}

	result.Errors = append(result.Errors, "Error handling test did not show proper error recognition")
	return false
}

// testMultiToolSequence tests using multiple tools in sequence
func (mct *ModelCompatibilityTester) testMultiToolSequence(ctx context.Context, controller *controllers.ChatController, result *ModelTestResult) bool {
	startTime := time.Now()

	_, err := controller.SendUserMessageWithContext(ctx, "First use execute_bash to create a file with 'echo testing > /tmp/multitest.txt', then use read_file to read it back, and finally use execute_bash to delete it")

	duration := time.Since(startTime)
	if result.AverageResponseTime == 0 {
		result.AverageResponseTime = duration
	} else {
		result.AverageResponseTime = (result.AverageResponseTime + duration) / 2
	}

	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Multi-tool test failed: %v", err))
		return false
	}

	// Check conversation history for multiple tool calls
	history := controller.GetHistory()
	bashCalls := 0
	fileCalls := 0

	for _, msg := range history {
		if msg.IsTool() {
			if msg.ToolName == "execute_bash" {
				bashCalls++
			} else if msg.ToolName == "read_file" {
				fileCalls++
			}
		}
	}

	// Should have at least 2 bash calls (create, delete) and 1 file read
	if bashCalls >= 2 && fileCalls >= 1 {
		return true
	}

	result.Errors = append(result.Errors, fmt.Sprintf("Multi-tool test incomplete: bash calls=%d, file calls=%d", bashCalls, fileCalls))
	return false
}

// TestMultipleModels tests a list of models and returns results
func (mct *ModelCompatibilityTester) TestMultipleModels(models []string) []ModelTestResult {
	results := make([]ModelTestResult, 0, len(models))

	log.Printf("Starting compatibility testing for %d models", len(models))

	for i, model := range models {
		log.Printf("Testing model %d/%d: %s", i+1, len(models), model)
		result := mct.TestModel(model)
		results = append(results, result)

		// Brief pause between tests to avoid overwhelming Ollama
		time.Sleep(2 * time.Second)
	}

	return results
}

// PrintResults prints a formatted summary of test results
func (mct *ModelCompatibilityTester) PrintResults(results []ModelTestResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("MODEL COMPATIBILITY TEST RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	// Show Ollama version info if available
	if len(results) > 0 && results[0].OllamaVersion != "" {
		fmt.Printf("\n🔗 Ollama Server: v%s\n", results[0].OllamaVersion)
		versionSupported := mct.versionSupportsTools(results[0].OllamaVersion)
		if versionSupported {
			fmt.Printf("   Tool Support: ✅ Compatible (v0.4.0+ required)\n")
		} else {
			fmt.Printf("   Tool Support: ❌ Incompatible (v0.4.0+ required, found v%s)\n", results[0].OllamaVersion)
		}
	}

	for _, result := range results {
		fmt.Printf("\n📊 Model: %s\n", result.ModelName)
		fmt.Printf("   Tool Support: %v\n", result.ToolCallSupported)
		if result.ToolCallSupported {
			fmt.Printf("   Tests Passed: %d/%d (%.1f%%)\n", result.PassedTests, result.TotalTests,
				float64(result.PassedTests)/float64(result.TotalTests)*100)
			fmt.Printf("   Avg Response: %v\n", result.AverageResponseTime.Round(time.Millisecond))
			fmt.Printf("   Basic Tool:   %v\n", result.BasicToolCallPassed)
			fmt.Printf("   File Read:    %v\n", result.FileReadPassed)
			fmt.Printf("   Error Handle: %v\n", result.ErrorHandlingPassed)
			fmt.Printf("   Multi-tool:   %v\n", result.MultiToolPassed)
		}

		if len(result.Errors) > 0 {
			fmt.Printf("   Errors:\n")
			for _, err := range result.Errors {
				fmt.Printf("     - %s\n", err)
			}
		}
	}

	// Summary statistics
	totalModels := len(results)
	supportedModels := 0
	totalPassRate := 0.0

	for _, result := range results {
		if result.ToolCallSupported {
			supportedModels++
			if result.TotalTests > 0 {
				totalPassRate += float64(result.PassedTests) / float64(result.TotalTests)
			}
		}
	}

	fmt.Printf("\n" + strings.Repeat("-", 80))
	fmt.Printf("\n📈 SUMMARY:\n")
	fmt.Printf("   Models Tested: %d\n", totalModels)
	fmt.Printf("   Tool Compatible: %d (%.1f%%)\n", supportedModels, float64(supportedModels)/float64(totalModels)*100)
	if supportedModels > 0 {
		fmt.Printf("   Avg Pass Rate: %.1f%%\n", totalPassRate/float64(supportedModels)*100)
	}
	fmt.Println(strings.Repeat("=", 80))
}
