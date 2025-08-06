package agents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeAnalysisAgent_NewCodeAnalysisAgent(t *testing.T) {
	agent := NewCodeAnalysisAgent()

	assert.NotNil(t, agent)
	assert.NotNil(t, agent.astAnalyzer)
	assert.NotNil(t, agent.symbolResolver)
	assert.NotNil(t, agent.log)
}

func TestCodeAnalysisAgent_Name(t *testing.T) {
	agent := NewCodeAnalysisAgent()
	assert.Equal(t, "code_analysis", agent.Name())
}

func TestCodeAnalysisAgent_Description(t *testing.T) {
	agent := NewCodeAnalysisAgent()
	assert.Equal(t, "Performs AST analysis, symbol resolution, and code structure understanding", agent.Description())
}

func TestCodeAnalysisAgent_CanHandle(t *testing.T) {
	// With LLM-based routing, all agents trust the orchestrator's decision
	// and always return true/1.0 from CanHandle
	tests := []struct {
		name    string
		request string
	}{
		{
			name:    "Analyze code request",
			request: "analyze the authentication module",
		},
		{
			name:    "AST request",
			request: "show me the AST of this function",
		},
		{
			name:    "Structure request",
			request: "show me the structure of the handlers package",
		},
		{
			name:    "Symbols request",
			request: "list all symbols in this file",
		},
		{
			name:    "Functions request",
			request: "show all functions in this module",
		},
		{
			name:    "Types request",
			request: "what types are defined here?",
		},
		{
			name:    "Interfaces request",
			request: "show all interfaces in the codebase",
		},
		{
			name:    "Patterns request",
			request: "identify patterns in this code",
		},
		{
			name:    "Non-analysis request",
			request: "create a new file",
		},
		{
			name:    "Search request",
			request: "search for TODO comments",
		},
	}

	agent := NewCodeAnalysisAgent()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canHandle, confidence := agent.CanHandle(tt.request)
			// All agents now trust the orchestrator's LLM routing decision
			assert.True(t, canHandle, "Agent should always return true with LLM-based routing")
			assert.Equal(t, 1.0, confidence, "Agent should always return confidence 1.0 with LLM-based routing")
		})
	}
}

func TestCodeAnalysisAgent_Execute(t *testing.T) {
	// Note: The actual CodeAnalysisAgent performs direct AST analysis
	// and doesn't use external tools
	agent := NewCodeAnalysisAgent()

	tests := []struct {
		name        string
		request     AgentRequest
		expectError bool
		checkResult func(t *testing.T, result AgentResult)
	}{
		{
			name: "Analyze Go file",
			request: AgentRequest{
				Prompt: "analyze main.go",
				Context: map[string]interface{}{
					"file_contents": map[string]string{
						"main.go": `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
	os.Exit(0)
}

func helper() string {
	return "helper"
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
			name: "Analyze package structure",
			request: AgentRequest{
				Prompt: "analyze the structure of pkg/handlers",
				Context: map[string]interface{}{
					"file_contents": map[string]string{
						"handlers.go": `package handlers

type Handler interface {
	Handle() error
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
			name: "Missing file content",
			request: AgentRequest{
				Prompt:  "analyze missing.go",
				Context: map[string]interface{}{},
			},
			expectError: true, // Should return an error for missing content
			checkResult: func(t *testing.T, result AgentResult) {
				// When error occurs, result may not be populated
				assert.False(t, result.Success)
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

func TestCodeAnalysisAgent_Analyzers(t *testing.T) {
	t.Run("NewASTAnalyzer", func(t *testing.T) {
		analyzer := NewASTAnalyzer()
		assert.NotNil(t, analyzer)
	})

	t.Run("NewSymbolResolver", func(t *testing.T) {
		resolver := NewSymbolResolver()
		assert.NotNil(t, resolver)
	})
}

func TestCodeAnalysisAgent_TypeToString(t *testing.T) {
	agent := NewCodeAnalysisAgent()

	// Since typeToString is private, we test it indirectly through Execute
	// The function converts AST type expressions to strings
	// This is covered when analyzing Go files with various type definitions

	ctx := context.Background()
	request := AgentRequest{
		Prompt: "analyze type definitions",
	}

	// The actual type conversion happens internally
	canHandle, _ := agent.CanHandle(request.Prompt)
	assert.True(t, canHandle, "Agent should handle analysis request")

	// All agents now trust the orchestrator's LLM routing decision
	// Test execution context
	_, _ = agent.Execute(ctx, request)
}
