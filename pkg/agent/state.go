package agent

import (
	"sync"
	"time"
)

// ExecutionState represents the current state of agent execution
type ExecutionState struct {
	mu sync.RWMutex

	// Current phase of execution
	Phase ExecutionPhase `json:"phase"`

	// Current tool being executed (if any)
	CurrentTool *ToolExecution `json:"current_tool,omitempty"`

	// History of tool executions for this request
	ToolHistory []ToolExecution `json:"tool_history"`

	// Agent's current reasoning/thought process
	CurrentThought string `json:"current_thought,omitempty"`

	// Timestamp of last update
	LastUpdated time.Time `json:"last_updated"`
}

// ExecutionPhase represents the current phase of agent execution
type ExecutionPhase string

const (
	PhaseIdle       ExecutionPhase = "idle"
	PhaseThinking   ExecutionPhase = "thinking"
	PhaseToolUse    ExecutionPhase = "tool_use"
	PhaseResponding ExecutionPhase = "responding"
	PhaseComplete   ExecutionPhase = "complete"
	PhaseError      ExecutionPhase = "error"
)

// ToolExecution represents a single tool execution
type ToolExecution struct {
	// Name of the tool
	Name string `json:"name"`

	// Arguments passed to the tool
	Arguments map[string]interface{} `json:"arguments,omitempty"`

	// Output from the tool (truncated for display)
	Output string `json:"output,omitempty"`

	// Full output (for logging/debugging)
	FullOutput string `json:"full_output,omitempty"`

	// Error if tool execution failed
	Error string `json:"error,omitempty"`

	// Timestamps
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`

	// Duration of execution
	Duration time.Duration `json:"duration,omitempty"`
}

// NewExecutionState creates a new execution state
func NewExecutionState() *ExecutionState {
	return &ExecutionState{
		Phase:       PhaseIdle,
		ToolHistory: make([]ToolExecution, 0),
		LastUpdated: time.Now(),
	}
}

// SetPhase updates the execution phase
func (s *ExecutionState) SetPhase(phase ExecutionPhase) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Phase = phase
	s.LastUpdated = time.Now()
}

// StartToolExecution marks the beginning of a tool execution
func (s *ExecutionState) StartToolExecution(name string, args map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tool := &ToolExecution{
		Name:      name,
		Arguments: args,
		StartTime: time.Now(),
	}

	s.CurrentTool = tool
	s.Phase = PhaseToolUse
	s.LastUpdated = time.Now()
}

// CompleteToolExecution marks the completion of a tool execution
func (s *ExecutionState) CompleteToolExecution(output string, fullOutput string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.CurrentTool != nil {
		s.CurrentTool.EndTime = time.Now()
		s.CurrentTool.Duration = s.CurrentTool.EndTime.Sub(s.CurrentTool.StartTime)
		s.CurrentTool.Output = truncateOutput(output, 200)
		s.CurrentTool.FullOutput = fullOutput

		// Add to history
		s.ToolHistory = append(s.ToolHistory, *s.CurrentTool)
		s.CurrentTool = nil
	}

	s.Phase = PhaseThinking
	s.LastUpdated = time.Now()
}

// FailToolExecution marks a tool execution as failed
func (s *ExecutionState) FailToolExecution(err string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.CurrentTool != nil {
		s.CurrentTool.EndTime = time.Now()
		s.CurrentTool.Duration = s.CurrentTool.EndTime.Sub(s.CurrentTool.StartTime)
		s.CurrentTool.Error = err

		// Add to history
		s.ToolHistory = append(s.ToolHistory, *s.CurrentTool)
		s.CurrentTool = nil
	}

	s.LastUpdated = time.Now()
}

// SetThought updates the current reasoning/thought
func (s *ExecutionState) SetThought(thought string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentThought = thought
	s.Phase = PhaseThinking
	s.LastUpdated = time.Now()
}

// GetSnapshot returns a snapshot of the current state
func (s *ExecutionState) GetSnapshot() ExecutionStateSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := ExecutionStateSnapshot{
		Phase:          s.Phase,
		CurrentThought: s.CurrentThought,
		LastUpdated:    s.LastUpdated,
	}

	if s.CurrentTool != nil {
		toolCopy := *s.CurrentTool
		snapshot.CurrentTool = &toolCopy
	}

	// Copy tool history
	snapshot.ToolHistory = make([]ToolExecution, len(s.ToolHistory))
	copy(snapshot.ToolHistory, s.ToolHistory)

	return snapshot
}

// ExecutionStateSnapshot is a thread-safe copy of the execution state
type ExecutionStateSnapshot struct {
	Phase          ExecutionPhase  `json:"phase"`
	CurrentTool    *ToolExecution  `json:"current_tool,omitempty"`
	ToolHistory    []ToolExecution `json:"tool_history"`
	CurrentThought string          `json:"current_thought,omitempty"`
	LastUpdated    time.Time       `json:"last_updated"`
}

// Reset clears the execution state
func (s *ExecutionState) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Phase = PhaseIdle
	s.CurrentTool = nil
	s.ToolHistory = make([]ToolExecution, 0)
	s.CurrentThought = ""
	s.LastUpdated = time.Now()
}

// truncateOutput truncates output to a maximum length for display
func truncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen-3] + "..."
}
