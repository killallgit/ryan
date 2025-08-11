package orchestrator

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/killallgit/ryan/pkg/logger"
)

// StateManager manages task states throughout their lifecycle
type StateManager struct {
	states map[string]*TaskState
	mu     sync.RWMutex
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		states: make(map[string]*TaskState),
	}
}

// CreateState creates a new task state
func (sm *StateManager) CreateState(query string) *TaskState {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state := &TaskState{
		ID:           uuid.New().String(),
		Query:        query,
		CurrentPhase: PhaseAnalysis,
		History:      make([]AgentResponse, 0),
		Status:       StatusPending,
		Metadata:     make(map[string]interface{}),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	sm.states[state.ID] = state
	logger.Debug("Created new task state: %s", state.ID)
	return state
}

// GetState retrieves a task state by ID
func (sm *StateManager) GetState(id string) (*TaskState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[id]
	if !exists {
		return nil, fmt.Errorf("state not found: %s", id)
	}
	return state, nil
}

// UpdateState updates an existing task state
func (sm *StateManager) UpdateState(state *TaskState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.states[state.ID]; !exists {
		return fmt.Errorf("state not found: %s", state.ID)
	}

	state.UpdatedAt = time.Now()
	sm.states[state.ID] = state
	logger.Debug("Updated task state: %s", state.ID)
	return nil
}

// DeleteState removes a task state
func (sm *StateManager) DeleteState(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.states[id]; !exists {
		return fmt.Errorf("state not found: %s", id)
	}

	delete(sm.states, id)
	logger.Debug("Deleted task state: %s", id)
	return nil
}

// ListStates returns all task states
func (sm *StateManager) ListStates() []*TaskState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	states := make([]*TaskState, 0, len(sm.states))
	for _, state := range sm.states {
		states = append(states, state)
	}
	return states
}

// GetActiveStates returns all active (in-progress) task states
func (sm *StateManager) GetActiveStates() []*TaskState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	states := make([]*TaskState, 0)
	for _, state := range sm.states {
		if state.Status == StatusInProgress {
			states = append(states, state)
		}
	}
	return states
}

// CleanupOldStates removes states older than the specified duration
func (sm *StateManager) CleanupOldStates(maxAge time.Duration) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, state := range sm.states {
		if state.UpdatedAt.Before(cutoff) && state.Status != StatusInProgress {
			delete(sm.states, id)
			removed++
		}
	}

	if removed > 0 {
		logger.Info("Cleaned up %d old task states", removed)
	}
	return removed
}

// UpdatePhase updates the current phase of a task
func (state *TaskState) UpdatePhase(phase Phase) {
	state.CurrentPhase = phase
	state.UpdatedAt = time.Now()
	logger.Debug("Task %s phase updated to: %s", state.ID, phase)
}

// UpdateStatus updates the status of a task
func (state *TaskState) UpdateStatus(status Status) {
	state.Status = status
	state.UpdatedAt = time.Now()
	logger.Debug("Task %s status updated to: %s", state.ID, status)
}

// AddMetadata adds metadata to the task state
func (state *TaskState) AddMetadata(key string, value interface{}) {
	if state.Metadata == nil {
		state.Metadata = make(map[string]interface{})
	}
	state.Metadata[key] = value
	state.UpdatedAt = time.Now()
}

// GetMetadata retrieves metadata from the task state
func (state *TaskState) GetMetadata(key string) (interface{}, bool) {
	if state.Metadata == nil {
		return nil, false
	}
	value, exists := state.Metadata[key]
	return value, exists
}

// UpdateWithFeedback updates the state based on agent feedback
func (state *TaskState) UpdateWithFeedback(feedback *AgentResponse) {
	// Update phase based on feedback status
	switch feedback.Status {
	case "success":
		if feedback.NextAction != nil {
			state.CurrentPhase = PhaseRouting
		} else {
			state.CurrentPhase = PhaseComplete
		}
	case "failed":
		// Stay in feedback phase for retry logic
		state.CurrentPhase = PhaseFeedback
	case "partial":
		// Stay in execution phase
		state.CurrentPhase = PhaseExecution
	}

	// Add any tool calls to metadata
	if len(feedback.ToolsCalled) > 0 {
		var toolNames []string
		for _, tool := range feedback.ToolsCalled {
			toolNames = append(toolNames, tool.Name)
		}
		state.AddMetadata("tools_used", toolNames)
	}

	state.UpdatedAt = time.Now()
}

// GetLastResponse returns the most recent agent response
func (state *TaskState) GetLastResponse() *AgentResponse {
	if len(state.History) == 0 {
		return nil
	}
	return &state.History[len(state.History)-1]
}

// GetResponsesByAgent returns all responses from a specific agent type
func (state *TaskState) GetResponsesByAgent(agentType AgentType) []AgentResponse {
	var responses []AgentResponse
	for _, resp := range state.History {
		if resp.AgentType == agentType {
			responses = append(responses, resp)
		}
	}
	return responses
}
