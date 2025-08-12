package cmd

import (
	"context"
	"testing"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/stream/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCustomizableAgent implements a simple mock for testing prompt customization
type MockCustomizableAgent struct {
	customPrompt string
}

func NewMockCustomizableAgent() *MockCustomizableAgent {
	return &MockCustomizableAgent{}
}

func (a *MockCustomizableAgent) Execute(ctx context.Context, prompt string) (string, error) {
	return "mock response", nil
}

func (a *MockCustomizableAgent) ExecuteStream(ctx context.Context, prompt string, handler core.Handler) error {
	return nil
}

func (a *MockCustomizableAgent) ClearMemory() error {
	return nil
}

func (a *MockCustomizableAgent) GetTokenStats() (int, int) {
	return 0, 0
}

func (a *MockCustomizableAgent) Close() error {
	return nil
}

func (a *MockCustomizableAgent) SetCustomPrompt(customPrompt string) {
	a.customPrompt = customPrompt
}

// TestRootCommandFlags tests that all expected CLI flags are present
func TestRootCommandFlags(t *testing.T) {
	// Test that CLI-only flags exist
	continueFlag := rootCmd.PersistentFlags().Lookup("continue")
	assert.NotNil(t, continueFlag)
	assert.Equal(t, "bool", continueFlag.Value.Type())

	promptFlag := rootCmd.PersistentFlags().Lookup("prompt")
	assert.NotNil(t, promptFlag)
	assert.Equal(t, "string", promptFlag.Value.Type())

	headlessFlag := rootCmd.PersistentFlags().Lookup("headless")
	assert.NotNil(t, headlessFlag)
	assert.Equal(t, "bool", headlessFlag.Value.Type())

	skipPermissionsFlag := rootCmd.PersistentFlags().Lookup("skip-permissions")
	assert.NotNil(t, skipPermissionsFlag)
	assert.Equal(t, "bool", skipPermissionsFlag.Value.Type())

	// Check Claude Code-style prompt control flags
	systemPromptFlag := rootCmd.PersistentFlags().Lookup("system-prompt")
	assert.NotNil(t, systemPromptFlag)
	assert.Equal(t, "string", systemPromptFlag.Value.Type())

	appendSystemPromptFlag := rootCmd.PersistentFlags().Lookup("append-system-prompt")
	assert.NotNil(t, appendSystemPromptFlag)
	assert.Equal(t, "string", appendSystemPromptFlag.Value.Type())

	planFlag := rootCmd.PersistentFlags().Lookup("plan")
	assert.NotNil(t, planFlag)
	assert.Equal(t, "bool", planFlag.Value.Type())

	// Check logging flag
	loggingPersistFlag := rootCmd.PersistentFlags().Lookup("logging.persist")
	assert.NotNil(t, loggingPersistFlag)
	assert.Equal(t, "bool", loggingPersistFlag.Value.Type())
}

// TestFlagDefaults tests default values of CLI flags
func TestFlagDefaults(t *testing.T) {
	// Test default values
	continueFlag := rootCmd.PersistentFlags().Lookup("continue")
	assert.Equal(t, "false", continueFlag.DefValue)

	headlessFlag := rootCmd.PersistentFlags().Lookup("headless")
	assert.Equal(t, "false", headlessFlag.DefValue)

	skipPermissionsFlag := rootCmd.PersistentFlags().Lookup("skip-permissions")
	assert.Equal(t, "false", skipPermissionsFlag.DefValue)

	systemPromptFlag := rootCmd.PersistentFlags().Lookup("system-prompt")
	assert.Equal(t, "", systemPromptFlag.DefValue)

	appendSystemPromptFlag := rootCmd.PersistentFlags().Lookup("append-system-prompt")
	assert.Equal(t, "", appendSystemPromptFlag.DefValue)

	planFlag := rootCmd.PersistentFlags().Lookup("plan")
	assert.Equal(t, "false", planFlag.DefValue)

	loggingPersistFlag := rootCmd.PersistentFlags().Lookup("logging.persist")
	assert.Equal(t, "false", loggingPersistFlag.DefValue)
}

// TestFlagHelp tests that flags have appropriate usage descriptions
func TestFlagHelp(t *testing.T) {
	systemPromptFlag := rootCmd.PersistentFlags().Lookup("system-prompt")
	assert.Contains(t, systemPromptFlag.Usage, "override the default system prompt entirely")

	appendSystemPromptFlag := rootCmd.PersistentFlags().Lookup("append-system-prompt")
	assert.Contains(t, appendSystemPromptFlag.Usage, "append additional instructions to the system prompt")

	planFlag := rootCmd.PersistentFlags().Lookup("plan")
	assert.Contains(t, planFlag.Usage, "encourage planning behavior for complex tasks")

	loggingPersistFlag := rootCmd.PersistentFlags().Lookup("logging.persist")
	assert.Contains(t, loggingPersistFlag.Usage, "persist system logs across sessions")
}

// TestApplyPromptCustomizations tests the prompt customization logic
func TestApplyPromptCustomizations(t *testing.T) {
	tests := []struct {
		name           string
		customPrompt   string
		appendPrompt   string
		planningBias   bool
		expectedPrompt string
	}{
		{
			name:           "custom_prompt_only",
			customPrompt:   "You are a helpful assistant.",
			appendPrompt:   "",
			planningBias:   false,
			expectedPrompt: "You are a helpful assistant.",
		},
		{
			name:           "append_prompt_only",
			customPrompt:   "",
			appendPrompt:   "Always be thorough.",
			planningBias:   false,
			expectedPrompt: "Always be thorough.",
		},
		{
			name:           "plan_flag_only",
			customPrompt:   "",
			appendPrompt:   "",
			planningBias:   true,
			expectedPrompt: "IMPORTANT: For complex or ambiguous tasks, always plan first before executing. Ask for user confirmation before proceeding with multi-step operations.",
		},
		{
			name:           "custom_and_append",
			customPrompt:   "You are a coding assistant.",
			appendPrompt:   "Always explain your code.",
			planningBias:   false,
			expectedPrompt: "You are a coding assistant.\n\nAlways explain your code.",
		},
		{
			name:           "all_flags_combined",
			customPrompt:   "You are an expert.",
			appendPrompt:   "Always double-check.",
			planningBias:   true,
			expectedPrompt: "You are an expert.\n\nAlways double-check.\n\nIMPORTANT: For complex or ambiguous tasks, always plan first before executing. Ask for user confirmation before proceeding with multi-step operations.",
		},
		{
			name:           "no_customizations",
			customPrompt:   "",
			appendPrompt:   "",
			planningBias:   false,
			expectedPrompt: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAgent := NewMockCustomizableAgent()

			// Apply prompt customizations
			applyPromptCustomizations(mockAgent, tt.customPrompt, tt.appendPrompt, tt.planningBias)

			// Check that the custom prompt was set correctly
			assert.Equal(t, tt.expectedPrompt, mockAgent.customPrompt)
		})
	}
}

// TestApplyPromptCustomizationsWithNonCustomizableAgent tests handling of non-customizable agents
func TestApplyPromptCustomizationsWithNonCustomizableAgent(t *testing.T) {
	// Create a mock agent that doesn't implement CustomizableAgent
	type NonCustomizableAgent struct {
		*MockCustomizableAgent
	}

	// Remove the SetCustomPrompt method to make it non-customizable
	nonCustomizableAgent := &NonCustomizableAgent{
		MockCustomizableAgent: NewMockCustomizableAgent(),
	}

	// This should not panic even with a non-customizable agent
	require.NotPanics(t, func() {
		applyPromptCustomizations(nonCustomizableAgent, "test prompt", "append", true)
	})
}

// TestPromptCustomizationInterface tests the interface casting
func TestPromptCustomizationInterface(t *testing.T) {
	mockAgent := NewMockCustomizableAgent()

	// Test that mockAgent can be used as CustomizableAgent
	var customizable agent.CustomizableAgent = mockAgent
	assert.NotNil(t, customizable)

	// Test SetCustomPrompt
	customizable.SetCustomPrompt("test prompt")
	assert.Equal(t, "test prompt", mockAgent.customPrompt)
}
