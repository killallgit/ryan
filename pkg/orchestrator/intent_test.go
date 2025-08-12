package orchestrator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

// MockLLM for testing intent analysis
type MockIntentLLM struct {
	responses map[string]string
}

func (m *MockIntentLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	// Return predefined responses based on the query in the prompt
	for key, response := range m.responses {
		if contains(prompt, key) {
			return response, nil
		}
	}
	// Default response
	return `{"type": "reasoning", "confidence": 0.5, "required_capabilities": ["general"], "reasoning": "default"}`, nil
}

func (m *MockIntentLLM) Generate(ctx context.Context, prompts []string, options ...llms.CallOption) ([]string, error) {
	results := make([]string, len(prompts))
	for i, prompt := range prompts {
		result, _ := m.Call(ctx, prompt, options...)
		results[i] = result
	}
	return results, nil
}

func (m *MockIntentLLM) CreateEmbedding(ctx context.Context, inputTexts []string) ([][]float32, error) {
	return nil, nil
}

func (m *MockIntentLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	// For testing, just use the last message
	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		for _, part := range lastMsg.Parts {
			if textPart, ok := part.(llms.TextContent); ok {
				response, err := m.Call(ctx, textPart.Text, options...)
				if err != nil {
					return nil, err
				}
				return &llms.ContentResponse{
					Choices: []*llms.ContentChoice{
						{
							Content: response,
						},
					},
				}, nil
			}
		}
	}
	return &llms.ContentResponse{}, nil
}

func contains(text, substr string) bool {
	return len(text) >= len(substr) && text[len(text)-len(substr):] == substr ||
		len(text) >= len(substr) && containsSubstring(text, substr)
}

func containsSubstring(text, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAnalyzeIntent_FileOperations(t *testing.T) {
	tests := []struct {
		name               string
		query              string
		llmResponse        string
		expectedType       string
		expectedConfidence float64
		shouldContainCap   string
	}{
		{
			name:  "Read file request",
			query: "read a random file",
			llmResponse: `{
				"type": "tool_use",
				"confidence": 0.9,
				"required_capabilities": ["file_read", "bash"],
				"reasoning": "User wants to read a file which requires file system tools"
			}`,
			expectedType:       "tool_use",
			expectedConfidence: 0.9,
			shouldContainCap:   "file_read",
		},
		{
			name:  "List files request",
			query: "list all files in the current directory",
			llmResponse: `{
				"type": "tool_use",
				"confidence": 0.95,
				"required_capabilities": ["bash", "list", "filesystem"],
				"reasoning": "User wants to list files which requires bash command"
			}`,
			expectedType:       "tool_use",
			expectedConfidence: 0.95,
			shouldContainCap:   "bash",
		},
		{
			name:  "Check directory contents",
			query: "check what's in the src folder",
			llmResponse: `{
				"type": "tool_use",
				"confidence": 0.85,
				"required_capabilities": ["bash", "directory", "ls"],
				"reasoning": "User wants to check directory contents"
			}`,
			expectedType:       "tool_use",
			expectedConfidence: 0.85,
			shouldContainCap:   "directory",
		},
		{
			name:  "Write file request",
			query: "create a new file called test.txt",
			llmResponse: `{
				"type": "tool_use",
				"confidence": 0.9,
				"required_capabilities": ["file_write", "create"],
				"reasoning": "User wants to create a file which requires file_write tool"
			}`,
			expectedType:       "tool_use",
			expectedConfidence: 0.9,
			shouldContainCap:   "file_write",
		},
		{
			name:  "Run command request",
			query: "run npm install",
			llmResponse: `{
				"type": "tool_use",
				"confidence": 0.95,
				"required_capabilities": ["bash", "command", "npm"],
				"reasoning": "User wants to execute a bash command"
			}`,
			expectedType:       "tool_use",
			expectedConfidence: 0.95,
			shouldContainCap:   "bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock LLM with response
			mockLLM := &MockIntentLLM{
				responses: map[string]string{
					tt.query: tt.llmResponse,
				},
			}

			// Create orchestrator with mock LLM
			orch := &Orchestrator{
				llm:           mockLLM,
				config:        DefaultConfig(),
				maxIterations: 5,
			}

			// Test intent analysis
			ctx := context.Background()
			intent, err := orch.AnalyzeIntent(ctx, tt.query)
			require.NoError(t, err)

			// Verify results
			assert.Equal(t, tt.expectedType, intent.Type, "Intent type should match")
			assert.Equal(t, tt.expectedConfidence, intent.Confidence, "Confidence should match")

			// Check for required capability
			hasCapability := false
			for _, cap := range intent.RequiredCapabilities {
				if cap == tt.shouldContainCap {
					hasCapability = true
					break
				}
			}
			assert.True(t, hasCapability, "Should contain capability: %s", tt.shouldContainCap)
		})
	}
}

func TestSelectAgentForIntent_FileOperations(t *testing.T) {
	tests := []struct {
		name          string
		intent        *TaskIntent
		expectedAgent AgentType
	}{
		{
			name: "High confidence tool_use",
			intent: &TaskIntent{
				Type:                 "tool_use",
				Confidence:           0.9,
				RequiredCapabilities: []string{"file_read", "bash"},
			},
			expectedAgent: AgentToolCaller,
		},
		{
			name: "Low confidence with file keyword",
			intent: &TaskIntent{
				Type:                 "reasoning",
				Confidence:           0.5,
				RequiredCapabilities: []string{"file", "read"},
			},
			expectedAgent: AgentToolCaller,
		},
		{
			name: "Reasoning with filesystem capability",
			intent: &TaskIntent{
				Type:                 "reasoning",
				Confidence:           0.8,
				RequiredCapabilities: []string{"filesystem", "analysis"},
			},
			expectedAgent: AgentToolCaller,
		},
		{
			name: "Pure reasoning without tool keywords",
			intent: &TaskIntent{
				Type:                 "reasoning",
				Confidence:           0.9,
				RequiredCapabilities: []string{"analysis", "explanation"},
			},
			expectedAgent: AgentReasoner,
		},
		{
			name: "Search intent",
			intent: &TaskIntent{
				Type:                 "search",
				Confidence:           0.85,
				RequiredCapabilities: []string{"code_search", "pattern_matching"},
			},
			expectedAgent: AgentSearcher,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create orchestrator
			orch := &Orchestrator{
				config: DefaultConfig(),
			}

			// Test agent selection
			selectedAgent := orch.selectAgentForIntent(tt.intent)
			assert.Equal(t, tt.expectedAgent, selectedAgent, "Selected agent should match expected")
		})
	}
}

func TestShouldUseToolCallerForCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []string
		expected     bool
	}{
		{
			name:         "File operations",
			capabilities: []string{"file_read", "bash"},
			expected:     true,
		},
		{
			name:         "Directory operations",
			capabilities: []string{"directory", "listing"},
			expected:     true,
		},
		{
			name:         "System commands",
			capabilities: []string{"command", "execute"},
			expected:     true,
		},
		{
			name:         "Git operations",
			capabilities: []string{"git", "version_control"},
			expected:     true,
		},
		{
			name:         "Web operations",
			capabilities: []string{"web", "fetch"},
			expected:     true,
		},
		{
			name:         "Pure reasoning",
			capabilities: []string{"analysis", "explanation", "understanding"},
			expected:     false,
		},
		{
			name:         "Code generation",
			capabilities: []string{"implementation", "algorithm", "function"},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orch := &Orchestrator{}
			result := orch.shouldUseToolCallerForCapabilities(tt.capabilities)
			assert.Equal(t, tt.expected, result, "Tool caller detection should match expected")
		})
	}
}
