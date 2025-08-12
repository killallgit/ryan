package integration

import (
	"context"
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/killallgit/ryan/pkg/orchestrator"
	"github.com/killallgit/ryan/pkg/orchestrator/agents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestratorRoutingFileOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup viper for test
	setupViperForTest(t)

	// Create LLM
	ollamaClient := ollama.NewClient()
	llm := ollamaClient.LLM

	// Create orchestrator
	orch, err := orchestrator.New(llm, orchestrator.WithMaxIterations(3))
	require.NoError(t, err)

	// Register real agents
	err = agents.RegisterRealAgents(orch, llm, true) // skipPermissions = true
	require.NoError(t, err)

	testCases := []struct {
		name           string
		query          string
		expectedAgent  string
		shouldContain  string
	}{
		{
			name:          "List files command",
			query:         "list files in current directory",
			expectedAgent: "tool_caller",
			shouldContain: "ls",
		},
		{
			name:          "Read file command",
			query:         "read README.md file",
			expectedAgent: "tool_caller",
			shouldContain: "README",
		},
		{
			name:          "Execute bash command",
			query:         "run bash command pwd",
			expectedAgent: "tool_caller",
			shouldContain: "pwd",
		},
		{
			name:          "Check directory",
			query:         "check what's in the src folder",
			expectedAgent: "tool_caller",
			shouldContain: "src",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Execute the query
			result, err := orch.Execute(ctx, tc.query)
			require.NoError(t, err, "Should execute successfully")

			// Check that we used the correct agent
			foundExpectedAgent := false
			for _, response := range result.History {
				if string(response.AgentType) == tc.expectedAgent {
					foundExpectedAgent = true
					// Check if the response contains expected content
					if tc.shouldContain != "" {
						assert.Contains(t, response.Response, tc.shouldContain,
							"Response should contain expected content")
					}
					break
				}
			}

			assert.True(t, foundExpectedAgent,
				"Should have used %s agent for query: %s", tc.expectedAgent, tc.query)
		})
	}
}

func TestOrchestratorIntentAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup viper for test
	setupViperForTest(t)

	// Create LLM
	ollamaClient := ollama.NewClient()
	llm := ollamaClient.LLM

	orch, err := orchestrator.New(llm)
	require.NoError(t, err)

	testCases := []struct {
		name         string
		query        string
		expectedType string
	}{
		{
			name:         "File operation",
			query:        "read a random file",
			expectedType: "tool_use",
		},
		{
			name:         "List files",
			query:        "list all files",
			expectedType: "tool_use",
		},
		{
			name:         "Run command",
			query:        "execute ls command",
			expectedType: "tool_use",
		},
		{
			name:         "Git operation",
			query:        "check git status",
			expectedType: "tool_use",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			intent, err := orch.AnalyzeIntent(ctx, tc.query)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedType, intent.Type,
				"Query '%s' should be recognized as %s", tc.query, tc.expectedType)
		})
	}
}
