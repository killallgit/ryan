package agents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeReviewAgent_NewCodeReviewAgent(t *testing.T) {
	agent := NewCodeReviewAgent()
	
	assert.NotNil(t, agent)
	// Agent doesn't use tool registry directly
	assert.NotNil(t, agent.log)
}

func TestCodeReviewAgent_Name(t *testing.T) {
	agent := NewCodeReviewAgent()
	assert.Equal(t, "code_review", agent.Name())
}

func TestCodeReviewAgent_Description(t *testing.T) {
	agent := NewCodeReviewAgent()
	assert.Equal(t, "Performs comprehensive code reviews with architectural analysis and best practices", agent.Description())
}

func TestCodeReviewAgent_CanHandle(t *testing.T) {
	tests := []struct {
		name          string
		request       string
		shouldHandle  bool
		minConfidence float64
	}{
		{
			name:          "Code review request",
			request:       "code review this pull request",
			shouldHandle:  true,
			minConfidence: 0.9,
		},
		{
			name:          "Review request",
			request:       "review this implementation",
			shouldHandle:  true,
			minConfidence: 0.9,
		},
		{
			name:          "Critique request",
			request:       "critique this code",
			shouldHandle:  true,
			minConfidence: 0.9,
		},
		{
			name:          "Feedback request",
			request:       "give feedback on this implementation",
			shouldHandle:  true,
			minConfidence: 0.9,
		},
		{
			name:          "Improve code request",
			request:       "how can I improve this function",
			shouldHandle:  true,
			minConfidence: 0.9,
		},
		{
			name:          "Suggestions request",
			request:       "any suggestions for this code?",
			shouldHandle:  true,
			minConfidence: 0.9,
		},
		{
			name:          "Best practices request",
			request:       "does this follow best practices?",
			shouldHandle:  true,
			minConfidence: 0.9,
		},
		{
			name:          "Non-review request",
			request:       "create a new file",
			shouldHandle:  false,
			minConfidence: 0.0,
		},
		{
			name:          "Search request",
			request:       "search for TODO comments",
			shouldHandle:  false,
			minConfidence: 0.0,
		},
	}

	agent := NewCodeReviewAgent()
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canHandle, confidence := agent.CanHandle(tt.request)
			assert.Equal(t, tt.shouldHandle, canHandle)
			if tt.shouldHandle {
				assert.GreaterOrEqual(t, confidence, tt.minConfidence)
			} else {
				assert.Equal(t, 0.0, confidence)
			}
		})
	}
}

func TestCodeReviewAgent_Execute(t *testing.T) {
	// Code review agent performs analysis without external tools
	agent := NewCodeReviewAgent()
	
	tests := []struct {
		name        string
		request     AgentRequest
		expectError bool
		checkResult func(t *testing.T, result AgentResult)
	}{
		{
			name: "Review git diff",
			request: AgentRequest{
				Prompt: "review the current changes",
				Context: map[string]interface{}{
					"file_contents": map[string]string{
						"main.go": `package main
func main() {
    fmt.Println("Hello, World!")
}`,
					},
				},
			},
			expectError: false,
			checkResult: func(t *testing.T, result AgentResult) {
				assert.True(t, result.Success)
				assert.NotEmpty(t, result.Details)
				assert.NotEmpty(t, result.Summary)
			},
		},
		{
			name: "Review specific file",
			request: AgentRequest{
				Prompt: "review main.go for issues",
				Context: map[string]interface{}{
					"file_contents": map[string]string{
						"main.go": `package main

import "os"

func uncheckedError() {
    file, _ := os.Open("test.txt")  // Error not handled
    defer file.Close()
}`,
					},
				},
			},
			expectError: false,
			checkResult: func(t *testing.T, result AgentResult) {
				assert.True(t, result.Success)
				assert.NotEmpty(t, result.Details)
				assert.NotEmpty(t, result.Summary)
			},
		},
		{
			name: "Suggest improvements",
			request: AgentRequest{
				Prompt: "suggest improvements for the code",
				Context: map[string]interface{}{
					"file_contents": map[string]string{
						"example.go": `package example
func process() {
    // Long function that could be refactored
    x := 1
    y := 2
    z := x + y
    return z
}`,
					},
				},
			},
			expectError: false,
			checkResult: func(t *testing.T, result AgentResult) {
				assert.True(t, result.Success)
				assert.NotEmpty(t, result.Details)
			},
		},
		{
			name: "Security audit",
			request: AgentRequest{
				Prompt: "audit this code for security issues",
				Context: map[string]interface{}{
					"file_contents": map[string]string{
						"security.go": `package security
import "os/exec"
func runCommand(cmd string) {
    exec.Command("sh", "-c", cmd).Run() // Potential command injection
}`,
					},
				},
			},
			expectError: false,
			checkResult: func(t *testing.T, result AgentResult) {
				assert.True(t, result.Success)
				assert.NotEmpty(t, result.Details)
				assert.NotEmpty(t, result.Summary)
			},
		},
	}

	ctx := context.Background()
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := agent.Execute(ctx, tt.request)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}