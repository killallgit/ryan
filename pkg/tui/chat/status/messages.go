package status

import "time"

// ProcessState represents the current processing state
type ProcessState string

const (
	StateIdle      ProcessState = ""
	StateSending   ProcessState = "sending"
	StateReceiving ProcessState = "receiving"
	StateThinking  ProcessState = "thinking"
	StateToolUse   ProcessState = "tool"
)

// StatusUpdateMsg updates the status text
type StatusUpdateMsg struct {
	Status string
	State  ProcessState
}

// StartStreamingMsg indicates streaming has started
type StartStreamingMsg struct {
	Icon  string
	State ProcessState
}

// StopStreamingMsg indicates streaming has stopped
type StopStreamingMsg struct{}

// UpdateTokensMsg updates token counts
type UpdateTokensMsg struct {
	Sent int
	Recv int
}

// SetProcessStateMsg sets the current process state and icon
type SetProcessStateMsg struct {
	State ProcessState
}

// TickMsg updates the timer
type TickMsg time.Time
