package status

import (
	"github.com/killallgit/ryan/pkg/process"
	"time"
)

// StatusUpdateMsg updates the status text
type StatusUpdateMsg struct {
	Status string
	State  process.State
}

// StartStreamingMsg indicates streaming has started
type StartStreamingMsg struct {
	Icon  string
	State process.State
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
	State process.State
}

// TickMsg updates the timer
type TickMsg time.Time
