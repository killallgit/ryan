package react

import (
	"strings"
)

// DecisionMaker determines when to stop the ReAct loop
type DecisionMaker struct {
	maxIterations      int
	requireObservation bool
}

// NewDecisionMaker creates a new decision maker
func NewDecisionMaker() *DecisionMaker {
	return &DecisionMaker{
		maxIterations:      5,
		requireObservation: true,
	}
}

// ShouldStop determines if the ReAct loop should stop
func (dm *DecisionMaker) ShouldStop(state *State) bool {
	// Stop if we've reached max iterations
	if state.CurrentIter >= dm.maxIterations {
		return true
	}

	// Stop if the agent is stuck in a loop
	sm := &StateManager{state: state}
	if sm.IsStuck() {
		return true
	}

	// Check if we have a satisfactory answer
	if dm.hasGoodAnswer(state) {
		return true
	}

	return false
}

// hasGoodAnswer checks if we have a good answer to return
func (dm *DecisionMaker) hasGoodAnswer(state *State) bool {
	if len(state.Iterations) == 0 {
		return false
	}

	lastIter := state.Iterations[len(state.Iterations)-1]

	// If we have an observation from a tool, that's usually good enough
	if lastIter.Observation != "" && !strings.HasPrefix(lastIter.Observation, "Error:") {
		// Check if the observation actually answers the question
		return dm.answersQuestion(lastIter.Observation, state.Input)
	}

	// If the thought indicates completion
	thoughtLower := strings.ToLower(lastIter.Thought)
	if strings.Contains(thoughtLower, "the answer is") ||
		strings.Contains(thoughtLower, "therefore") ||
		strings.Contains(thoughtLower, "in conclusion") ||
		strings.Contains(thoughtLower, "final answer") {
		return true
	}

	return false
}

// answersQuestion checks if the response answers the original question
func (dm *DecisionMaker) answersQuestion(response, question string) bool {
	// Simple heuristic: check if response is substantial
	if len(response) < 10 {
		return false
	}

	// Check if it's not an error or empty response
	if strings.HasPrefix(response, "Error:") ||
		strings.Contains(response, "not found") ||
		strings.Contains(response, "failed") {
		return false
	}

	// For questions asking for counts/numbers
	questionLower := strings.ToLower(question)
	if strings.Contains(questionLower, "how many") ||
		strings.Contains(questionLower, "count") ||
		strings.Contains(questionLower, "number of") {
		// Check if response contains a number
		for _, char := range response {
			if char >= '0' && char <= '9' {
				return true
			}
		}
	}

	// For file/directory operations
	if strings.Contains(questionLower, "file") ||
		strings.Contains(questionLower, "directory") ||
		strings.Contains(questionLower, "list") {
		// Any non-error response is probably good
		return len(response) > 20
	}

	// Default: accept substantial responses
	return len(response) > 30
}

// SetMaxIterations sets the maximum number of iterations
func (dm *DecisionMaker) SetMaxIterations(max int) {
	dm.maxIterations = max
}

// SetRequireObservation sets whether observations are required
func (dm *DecisionMaker) SetRequireObservation(require bool) {
	dm.requireObservation = require
}
