package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/stream"
)

// StreamingHandler wraps a stream.Handler to intercept and format orchestrator output
type StreamingHandler struct {
	inner        stream.Handler
	showRouting  bool
	currentAgent string
	buffer       strings.Builder
}

// NewStreamingHandler creates a new streaming handler for orchestrator
func NewStreamingHandler(handler stream.Handler, showRouting bool) *StreamingHandler {
	return &StreamingHandler{
		inner:       handler,
		showRouting: showRouting,
	}
}

// OnChunk handles streaming chunks from agents
func (h *StreamingHandler) OnChunk(chunk []byte) error {
	// Buffer the chunk
	h.buffer.Write(chunk)

	// Pass through to inner handler
	return h.inner.OnChunk(chunk)
}

// OnComplete handles completion of streaming
func (h *StreamingHandler) OnComplete(finalContent string) error {
	return h.inner.OnComplete(finalContent)
}

// OnError handles streaming errors
func (h *StreamingHandler) OnError(err error) {
	h.inner.OnError(err)
}

// OnRoutingDecision sends routing information to the stream
func (h *StreamingHandler) OnRoutingDecision(decision *RouteDecision) error {
	if !h.showRouting {
		return nil
	}

	h.currentAgent = string(decision.TargetAgent)

	// Format routing info as markdown
	routingInfo := fmt.Sprintf("\n🎯 **Routing to %s agent**\n", decision.TargetAgent)
	// Add instruction if available
	if decision.Instruction != "" {
		routingInfo += fmt.Sprintf("*Task: %s*\n\n", decision.Instruction)
	}

	return h.inner.OnChunk([]byte(routingInfo))
}

// OnAgentStart notifies when an agent starts processing
func (h *StreamingHandler) OnAgentStart(agentType AgentType) error {
	h.currentAgent = string(agentType)

	// Simple agent name display
	info := fmt.Sprintf("[%s]\n", agentType)
	return h.inner.OnChunk([]byte(info))
}

// OnAgentComplete notifies when an agent completes
func (h *StreamingHandler) OnAgentComplete(agentType AgentType, status string) error {
	// Don't show completion messages, just the agent name at start is enough
	return nil
}

// ExecuteStream executes a task with streaming output
func (o *Orchestrator) ExecuteStream(ctx context.Context, query string, handler stream.Handler) (*TaskResult, error) {
	logger.Debug("🎯 Orchestrator executing with streaming: %s", query)

	// Wrap the handler to intercept orchestrator events
	streamHandler := NewStreamingHandler(handler, true)

	// Create initial task state
	state := o.stateManager.CreateState(query)

	// No fancy formatting, just start processing
	// Analyze intent

	intent, err := o.AnalyzeIntent(ctx, query)
	if err != nil {
		handler.OnError(fmt.Errorf("failed to analyze intent: %w", err))
		return nil, err
	}
	state.Intent = intent

	// Execute with streaming feedback loop
	result, err := o.streamingFeedbackLoop(ctx, state, streamHandler)
	if err != nil {
		handler.OnError(err)
		return nil, err
	}

	// Complete the stream (result already streamed during execution)
	if err := handler.OnComplete(""); err != nil {
		return nil, err
	}

	return result, nil
}

// streamingFeedbackLoop runs the feedback loop with streaming support
func (o *Orchestrator) streamingFeedbackLoop(ctx context.Context, state *TaskState, handler *StreamingHandler) (*TaskResult, error) {
	startTime := time.Now()

	// Initialize state
	state.CurrentPhase = PhaseRouting
	state.Status = StatusInProgress

	for i := 0; i < o.maxIterations; i++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			state.Status = StatusCancelled
			return o.buildStreamingResult(state, startTime), ctx.Err()
		default:
		}

		// Route to agent
		decision, err := o.Route(ctx, state.Intent, state)
		if err != nil {
			state.Status = StatusFailed
			return o.buildStreamingResult(state, startTime), err
		}

		if decision == nil {
			state.Status = StatusCompleted
			state.CurrentPhase = PhaseComplete
			return o.buildStreamingResult(state, startTime), nil
		}

		// Notify about routing decision
		if err := handler.OnRoutingDecision(decision); err != nil {
			return nil, err
		}

		// Get the agent from registry
		agent, err := o.registry.GetAgent(decision.TargetAgent)
		if err != nil {
			return nil, fmt.Errorf("agent not found: %s", decision.TargetAgent)
		}

		// Notify agent start
		if err := handler.OnAgentStart(decision.TargetAgent); err != nil {
			return nil, err
		}

		// Execute agent (agents should support streaming internally)
		state.CurrentPhase = PhaseExecution
		response, err := agent.Execute(ctx, decision, state)
		if err != nil {
			handler.OnError(err)
			return nil, err
		}

		// Stream the agent's response
		if response.Response != "" {
			if err := handler.OnChunk([]byte(response.Response)); err != nil {
				return nil, err
			}
		}

		// Notify agent completion
		if err := handler.OnAgentComplete(decision.TargetAgent, response.Status); err != nil {
			return nil, err
		}

		// Update state
		state.History = append(state.History, *response)

		// Process feedback
		state.CurrentPhase = PhaseFeedback
		nextStep, err := o.ProcessFeedback(ctx, response, state)
		if err != nil {
			return nil, err
		}

		// Check if complete
		if nextStep.Action == ActionComplete {
			state.Status = StatusCompleted
			state.CurrentPhase = PhaseComplete
			return o.buildStreamingResult(state, startTime), nil
		}

		// Continue with next iteration if needed
		if nextStep.Action == ActionRetry || nextStep.Action == ActionContinue {
			continue
		}
	}

	// Max iterations reached
	state.Status = StatusFailed
	return o.buildStreamingResult(state, startTime), fmt.Errorf("max iterations reached")
}

// buildStreamingResult builds the final result from state
func (o *Orchestrator) buildStreamingResult(state *TaskState, startTime time.Time) *TaskResult {
	// Collect all agent responses
	var resultBuilder strings.Builder
	for _, resp := range state.History {
		if resp.Response != "" {
			resultBuilder.WriteString(resp.Response)
			resultBuilder.WriteString("\n")
		}
	}

	return &TaskResult{
		ID:        state.ID,
		Query:     state.Query,
		Result:    strings.TrimSpace(resultBuilder.String()),
		Status:    state.Status,
		History:   state.History,
		Duration:  time.Since(startTime),
		StartTime: startTime,
		EndTime:   time.Now(),
	}
}

// formatStreamingSummary formats a summary for the streaming output
func (o *Orchestrator) formatStreamingSummary(result *TaskResult) string {
	var sb strings.Builder

	sb.WriteString("\n---\n")
	sb.WriteString("📋 **Execution Summary**\n")
	sb.WriteString(fmt.Sprintf("⏱️ Duration: %v\n", result.Duration))
	sb.WriteString(fmt.Sprintf("🔄 Agent interactions: %d\n", len(result.History)))
	sb.WriteString(fmt.Sprintf("✅ Status: %s\n", result.Status))

	return sb.String()
}
