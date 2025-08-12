package react

import (
	"strings"
)

// State represents the current state of the ReAct loop
type State struct {
	Input       string
	Iterations  []Iteration
	CurrentIter int
}

// Iteration represents one iteration of the ReAct loop
type Iteration struct {
	Thought     string
	Action      string
	ActionInput string
	Observation string
}

// StateManager manages the state of the ReAct loop
type StateManager struct {
	state       *State
	maxHistory  int
	lastAnswers []string
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		state:       &State{},
		maxHistory:  10,
		lastAnswers: make([]string, 0),
	}
}

// Reset clears the state for a new query
func (sm *StateManager) Reset() {
	sm.state = &State{
		Iterations: make([]Iteration, 0),
	}
	sm.lastAnswers = make([]string, 0)
}

// SetInput sets the user input
func (sm *StateManager) SetInput(input string) {
	sm.state.Input = input
}

// AddIteration adds a parsed response as a new iteration
func (sm *StateManager) AddIteration(parsed *ParsedResponse) {
	iter := Iteration{
		Thought:     parsed.Thought,
		Action:      parsed.Action,
		ActionInput: parsed.ActionInput,
	}

	sm.state.Iterations = append(sm.state.Iterations, iter)
	sm.state.CurrentIter++

	// Track potential answers from thoughts
	if parsed.Thought != "" && !strings.Contains(strings.ToLower(parsed.Thought), "i need to") {
		sm.lastAnswers = append(sm.lastAnswers, parsed.Thought)
	}

	// Trim history if it gets too long
	if len(sm.state.Iterations) > sm.maxHistory {
		sm.state.Iterations = sm.state.Iterations[len(sm.state.Iterations)-sm.maxHistory:]
	}
}

// AddObservation adds an observation to the current iteration
func (sm *StateManager) AddObservation(observation string) {
	if len(sm.state.Iterations) > 0 {
		sm.state.Iterations[len(sm.state.Iterations)-1].Observation = observation

		// Track observations as potential answers
		if observation != "" && !strings.HasPrefix(observation, "Error:") {
			sm.lastAnswers = append(sm.lastAnswers, observation)
		}
	}
}

// GetState returns the current state
func (sm *StateManager) GetState() *State {
	return sm.state
}

// GetCurrentIteration returns the current iteration number
func (sm *StateManager) GetCurrentIteration() int {
	return sm.state.CurrentIter
}

// GetLastThought returns the most recent thought
func (sm *StateManager) GetLastThought() string {
	if len(sm.state.Iterations) > 0 {
		return sm.state.Iterations[len(sm.state.Iterations)-1].Thought
	}
	return ""
}

// GetLastObservation returns the most recent observation
func (sm *StateManager) GetLastObservation() string {
	if len(sm.state.Iterations) > 0 {
		return sm.state.Iterations[len(sm.state.Iterations)-1].Observation
	}
	return ""
}

// GetBestAnswer returns the best answer found so far
func (sm *StateManager) GetBestAnswer() string {
	// First, check for observations (tool outputs)
	for i := len(sm.state.Iterations) - 1; i >= 0; i-- {
		if sm.state.Iterations[i].Observation != "" &&
			!strings.HasPrefix(sm.state.Iterations[i].Observation, "Error:") {
			return sm.state.Iterations[i].Observation
		}
	}

	// Then check last thoughts
	if len(sm.lastAnswers) > 0 {
		return sm.lastAnswers[len(sm.lastAnswers)-1]
	}

	// Finally, return last thought
	return sm.GetLastThought()
}

// HasAction checks if the current iteration has an action
func (sm *StateManager) HasAction() bool {
	if len(sm.state.Iterations) > 0 {
		return sm.state.Iterations[len(sm.state.Iterations)-1].Action != ""
	}
	return false
}

// IsStuck checks if the agent seems stuck in a loop
func (sm *StateManager) IsStuck() bool {
	if len(sm.state.Iterations) < 3 {
		return false
	}

	// Check if last 3 actions are the same
	lastActions := make([]string, 0)
	for i := len(sm.state.Iterations) - 1; i >= 0 && len(lastActions) < 3; i-- {
		if sm.state.Iterations[i].Action != "" {
			lastActions = append(lastActions, sm.state.Iterations[i].Action)
		}
	}

	if len(lastActions) == 3 &&
		lastActions[0] == lastActions[1] &&
		lastActions[1] == lastActions[2] {
		return true
	}

	return false
}
