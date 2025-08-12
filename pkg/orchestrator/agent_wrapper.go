package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/stream/core"
)

// AgentWrapper wraps the Orchestrator to implement the agent.Agent interface
// This allows the orchestrator to be used with the TUI
type AgentWrapper struct {
	orchestrator   *Orchestrator
	historyManager *HistoryManager
	tokensSent     int
	tokensReceived int
}

// NewAgentWrapper creates a new orchestrator agent wrapper
func NewAgentWrapper(orch *Orchestrator) (*AgentWrapper, error) {
	historyManager, err := NewHistoryManager("orchestrator_tui_session")
	if err != nil {
		logger.Warn("Could not initialize history manager: %v", err)
		// Continue without history - not critical for operation
		historyManager = nil
	}

	return &AgentWrapper{
		orchestrator:   orch,
		historyManager: historyManager,
	}, nil
}

// Execute handles a request and returns a response (blocking)
func (w *AgentWrapper) Execute(ctx context.Context, prompt string) (string, error) {
	logger.Debug("🎯 Orchestrator wrapper executing: %s", prompt)

	// Execute through orchestrator
	result, err := w.orchestrator.Execute(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("orchestrator execution failed: %w", err)
	}

	// Save to history
	if w.historyManager != nil {
		if err := w.historyManager.SaveTaskExecution(result); err != nil {
			logger.Warn("Could not save task execution to history: %v", err)
		}
	}

	// Update token stats (approximate - we don't have exact counts from orchestrator)
	w.tokensSent += len(prompt) / 4 // Rough estimate
	w.tokensReceived += len(result.Result) / 4

	// Format the response to include routing information
	response := w.formatResponse(result)
	return response, nil
}

// ExecuteStream handles a request with streaming response
func (w *AgentWrapper) ExecuteStream(ctx context.Context, prompt string, handler core.Handler) error {
	logger.Debug("🎯 Orchestrator wrapper executing stream: %s", prompt)

	// Use the new streaming execution
	result, err := w.orchestrator.ExecuteStream(ctx, prompt, handler)
	if err != nil {
		return fmt.Errorf("orchestrator streaming execution failed: %w", err)
	}

	// Save to history
	if w.historyManager != nil {
		if err := w.historyManager.SaveTaskExecution(result); err != nil {
			logger.Warn("Could not save task execution to history: %v", err)
		}
	}

	// Update token stats (approximate)
	w.tokensSent += len(prompt) / 4
	w.tokensReceived += len(result.Result) / 4

	return nil
}

// formatResponse formats the orchestrator result for display
func (w *AgentWrapper) formatResponse(result *TaskResult) string {
	var sb strings.Builder

	// Add routing header
	sb.WriteString("🤖 **Orchestrator Routing Decision**\n\n")

	// Show agent flow
	if len(result.History) > 0 {
		sb.WriteString("**Agent Flow:**\n")
		for i, resp := range result.History {
			emoji := "✅"
			if resp.Status == "failed" {
				emoji = "❌"
			} else if resp.Status == "in_progress" {
				emoji = "⏳"
			}

			sb.WriteString(fmt.Sprintf("%s %d. **%s** - %s\n",
				emoji, i+1, resp.AgentType, resp.Status))

			// Show tools used
			if len(resp.ToolsCalled) > 0 {
				sb.WriteString("   🔧 Tools: ")
				toolNames := make([]string, len(resp.ToolsCalled))
				for j, tool := range resp.ToolsCalled {
					toolNames[j] = tool.Name
				}
				sb.WriteString(strings.Join(toolNames, ", "))
				sb.WriteString("\n")
			}

			// Show error if any
			if resp.Error != nil && *resp.Error != "" {
				sb.WriteString(fmt.Sprintf("   ⚠️ Error: %s\n", *resp.Error))
			}
		}
		sb.WriteString("\n")
	}

	// Add the actual result
	sb.WriteString("**Result:**\n")
	sb.WriteString(result.Result)

	return sb.String()
}

// ClearMemory clears the conversation memory
func (w *AgentWrapper) ClearMemory() error {
	if w.historyManager != nil {
		return w.historyManager.Clear()
	}
	return nil
}

// GetTokenStats returns the cumulative token usage statistics
func (w *AgentWrapper) GetTokenStats() (int, int) {
	return w.tokensSent, w.tokensReceived
}

// Close cleans up resources
func (w *AgentWrapper) Close() error {
	if w.historyManager != nil {
		return w.historyManager.Close()
	}
	return nil
}
