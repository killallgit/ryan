package orchestrator

import (
	"encoding/json"
	"fmt"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/memory"
	"github.com/tmc/langchaingo/llms"
)

// HistoryManager manages the orchestrator's conversation history
type HistoryManager struct {
	memory *memory.Memory
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(sessionID string) (*HistoryManager, error) {
	mem, err := memory.New(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}

	return &HistoryManager{memory: mem}, nil
}

// SaveTaskExecution saves a complete task execution to history
func (h *HistoryManager) SaveTaskExecution(result *TaskResult) error {
	logger.Debug("💾 Saving orchestrator task execution to history")

	// Create a structured summary of the orchestrator execution
	summary := createExecutionSummary(result)

	// Save the user query
	if err := h.memory.AddUserMessage(result.Query); err != nil {
		return fmt.Errorf("failed to save user query: %w", err)
	}

	// Save the orchestrator response with detailed breakdown
	if err := h.memory.AddAssistantMessage(summary); err != nil {
		return fmt.Errorf("failed to save orchestrator response: %w", err)
	}

	// Optionally save the full detailed history as metadata
	// Note: We'll skip system messages for now as the interface doesn't support them directly
	fullHistoryJSON, err := json.Marshal(result)
	if err != nil {
		logger.Warn("Could not serialize full task result for history: %v", err)
	} else {
		logger.Debug("Full task result serialized: %d bytes", len(fullHistoryJSON))
		// Could be saved to a separate debug file if needed
	}

	logger.Info("✅ Task execution saved to history: %d agent interactions", len(result.History))
	return nil
}

// createExecutionSummary creates a human-readable summary of the orchestrator execution
func createExecutionSummary(result *TaskResult) string {
	summary := fmt.Sprintf("🤖 **Orchestrator Execution Summary**\n\n")
	summary += fmt.Sprintf("**Query:** %s\n", result.Query)
	summary += fmt.Sprintf("**Status:** %s\n", result.Status)
	summary += fmt.Sprintf("**Duration:** %v\n", result.Duration)
	summary += fmt.Sprintf("**Agents Used:** %d\n\n", len(result.History))

	if len(result.History) > 0 {
		summary += "**Execution Flow:**\n"
		for i, response := range result.History {
			summary += fmt.Sprintf("%d. **%s** (%s)", i+1, response.AgentType, response.Status)

			// Add tool usage info
			if len(response.ToolsCalled) > 0 {
				toolNames := make([]string, len(response.ToolsCalled))
				for j, tool := range response.ToolsCalled {
					toolNames[j] = tool.Name
				}
				summary += fmt.Sprintf(" - Used tools: %v", toolNames)
			}
			summary += "\n"

			// Add error info if present
			if response.Error != nil {
				summary += fmt.Sprintf("   Error: %s\n", *response.Error)
			}
		}
		summary += "\n"
	}

	// Add final result
	summary += "**Final Result:**\n"
	summary += result.Result

	return summary
}

// GetConversationHistory returns the conversation history
func (h *HistoryManager) GetConversationHistory() ([]llms.ChatMessage, error) {
	return h.memory.GetMessages()
}

// Clear clears the conversation history
func (h *HistoryManager) Clear() error {
	return h.memory.Clear()
}

// Close closes the history manager
func (h *HistoryManager) Close() error {
	if h.memory != nil {
		return h.memory.Close()
	}
	return nil
}
