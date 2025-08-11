package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
)

// FeedbackLoop manages the execution and feedback cycle for tasks
type FeedbackLoop struct {
	orchestrator  *Orchestrator
	maxIterations int
}

// NewFeedbackLoop creates a new feedback loop manager
func NewFeedbackLoop(orchestrator *Orchestrator) *FeedbackLoop {
	return &FeedbackLoop{
		orchestrator:  orchestrator,
		maxIterations: orchestrator.maxIterations,
	}
}

// Run executes the feedback loop for a task
func (fl *FeedbackLoop) Run(ctx context.Context, state *TaskState) (*TaskResult, error) {
	logger.Debug("Starting feedback loop for task: %s", state.ID)
	startTime := time.Now()

	// Initialize state
	state.CurrentPhase = PhaseRouting
	state.Status = StatusInProgress

	for i := 0; i < fl.maxIterations; i++ {
		logger.Debug("Feedback loop iteration %d/%d", i+1, fl.maxIterations)

		// Check context cancellation
		select {
		case <-ctx.Done():
			state.Status = StatusCancelled
			return fl.buildResult(state, startTime), ctx.Err()
		default:
		}

		// Get routing decision
		decision, err := fl.getNextAction(ctx, state)
		if err != nil {
			logger.Error("Failed to get next action: %v", err)
			state.Status = StatusFailed
			return fl.buildResult(state, startTime), err
		}

		// Check if task is complete
		if decision == nil {
			logger.Info("Task completed successfully")
			state.Status = StatusCompleted
			state.CurrentPhase = PhaseComplete
			return fl.buildResult(state, startTime), nil
		}

		// Execute with selected agent
		state.CurrentPhase = PhaseExecution
		response, err := fl.executeWithAgent(ctx, decision, state)
		if err != nil {
			logger.Error("Agent execution failed: %v", err)
			errorStr := err.Error()
			response = &AgentResponse{
				AgentType: decision.TargetAgent,
				Status:    "failed",
				Error:     &errorStr,
				Timestamp: time.Now(),
			}
		}

		// Add response to history
		state.History = append(state.History, *response)
		state.UpdatedAt = time.Now()

		// Process feedback
		state.CurrentPhase = PhaseFeedback
		nextStep, err := fl.orchestrator.ProcessFeedback(ctx, response, state)
		if err != nil {
			logger.Error("Failed to process feedback: %v", err)
			state.Status = StatusFailed
			return fl.buildResult(state, startTime), err
		}

		// Handle next step action
		switch nextStep.Action {
		case ActionComplete:
			logger.Info("Task marked as complete")
			state.Status = StatusCompleted
			state.CurrentPhase = PhaseComplete
			return fl.buildResult(state, startTime), nil

		case ActionFail:
			logger.Warn("Task marked as failed")
			state.Status = StatusFailed
			return fl.buildResult(state, startTime), fmt.Errorf("task failed after %d iterations", i+1)

		case ActionContinue:
			// Continue to next iteration
			logger.Debug("Continuing to next iteration")
			continue

		case ActionRetry:
			// Retry with modified parameters
			logger.Debug("Retrying with modified parameters")
			continue
		}
	}

	// Max iterations reached
	logger.Warn("Max iterations reached without completion")
	state.Status = StatusFailed
	return fl.buildResult(state, startTime), fmt.Errorf("max iterations (%d) reached", fl.maxIterations)
}

// getNextAction determines the next action based on current state
func (fl *FeedbackLoop) getNextAction(ctx context.Context, state *TaskState) (*RouteDecision, error) {
	// If this is the first iteration, route based on intent
	if len(state.History) == 0 {
		if state.Intent == nil {
			return nil, fmt.Errorf("no intent available for routing")
		}
		return fl.orchestrator.Route(ctx, state.Intent, state)
	}

	// Check last response for next action
	lastResponse := state.History[len(state.History)-1]
	if lastResponse.NextAction != nil {
		logger.Debug("Using next action from previous response")
		return lastResponse.NextAction, nil
	}

	// Check if task should be considered complete
	if lastResponse.Status == "success" {
		return nil, nil // Signal completion
	}

	// Create retry decision if last attempt failed
	if lastResponse.Status == "failed" {
		return fl.createRetryDecision(&lastResponse, state), nil
	}

	// Default: task is complete
	return nil, nil
}

// executeWithAgent executes a task with the specified agent
func (fl *FeedbackLoop) executeWithAgent(ctx context.Context, decision *RouteDecision, state *TaskState) (*AgentResponse, error) {
	logger.Debug("Executing with agent: %s", decision.TargetAgent)

	// Get agent from registry
	agent, err := fl.orchestrator.registry.GetAgent(decision.TargetAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Execute agent
	response, err := agent.Execute(ctx, decision, state)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	return response, nil
}

// createRetryDecision creates a retry decision for failed tasks
func (fl *FeedbackLoop) createRetryDecision(lastResponse *AgentResponse, state *TaskState) *RouteDecision {
	retryCount := 0
	for _, resp := range state.History {
		if resp.AgentType == lastResponse.AgentType && resp.Status == "failed" {
			retryCount++
		}
	}

	return &RouteDecision{
		TargetAgent: lastResponse.AgentType,
		Instruction: fmt.Sprintf("Retry attempt %d: %s", retryCount+1, state.Query),
		Parameters: map[string]interface{}{
			"retry_count":    retryCount + 1,
			"previous_error": lastResponse.Error,
		},
	}
}

// buildResult creates the final task result
func (fl *FeedbackLoop) buildResult(state *TaskState, startTime time.Time) *TaskResult {
	endTime := time.Now()

	// Get final response text
	var resultText string
	if len(state.History) > 0 {
		// Combine all successful responses
		for _, resp := range state.History {
			if resp.Status == "success" && resp.Response != "" {
				if resultText != "" {
					resultText += "\n\n"
				}
				resultText += resp.Response
			}
		}
		// If no successful responses, use the last response
		if resultText == "" {
			resultText = state.History[len(state.History)-1].Response
		}
	}

	return &TaskResult{
		ID:        state.ID,
		Query:     state.Query,
		Result:    resultText,
		Status:    state.Status,
		History:   state.History,
		Metadata:  state.Metadata,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
	}
}
