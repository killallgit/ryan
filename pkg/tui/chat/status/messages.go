package status

import "time"

// StatusUpdateMsg updates the status text
type StatusUpdateMsg struct {
	Status string
}

// StartStreamingMsg indicates streaming has started
type StartStreamingMsg struct {
	Icon string
}

// StopStreamingMsg indicates streaming has stopped
type StopStreamingMsg struct{}

// UpdateTokensMsg updates token counts
type UpdateTokensMsg struct {
	Sent int
	Recv int
}

// TickMsg updates the timer
type TickMsg time.Time
