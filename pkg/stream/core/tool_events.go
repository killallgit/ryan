package core

import (
	"time"
)

// ToolEvent represents a tool execution event in the stream
type ToolEvent struct {
	Type      ToolEventType          `json:"type"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Output    string                 `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ToolEventType represents the type of tool event
type ToolEventType string

const (
	ToolEventStart    ToolEventType = "tool_start"
	ToolEventOutput   ToolEventType = "tool_output"
	ToolEventComplete ToolEventType = "tool_complete"
	ToolEventError    ToolEventType = "tool_error"
)

// NewToolStartEvent creates a new tool start event
func NewToolStartEvent(name string, args map[string]interface{}) ToolEvent {
	return ToolEvent{
		Type:      ToolEventStart,
		Name:      name,
		Arguments: args,
		Timestamp: time.Now(),
	}
}

// NewToolOutputEvent creates a new tool output event
func NewToolOutputEvent(name string, output string) ToolEvent {
	return ToolEvent{
		Type:      ToolEventOutput,
		Name:      name,
		Output:    output,
		Timestamp: time.Now(),
	}
}

// NewToolCompleteEvent creates a new tool complete event
func NewToolCompleteEvent(name string, output string) ToolEvent {
	return ToolEvent{
		Type:      ToolEventComplete,
		Name:      name,
		Output:    output,
		Timestamp: time.Now(),
	}
}

// NewToolErrorEvent creates a new tool error event
func NewToolErrorEvent(name string, err string) ToolEvent {
	return ToolEvent{
		Type:      ToolEventError,
		Name:      name,
		Error:     err,
		Timestamp: time.Now(),
	}
}
