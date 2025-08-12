package core

import (
	"sync"
	"time"
)

// State represents the current state of a stream
type State int

const (
	StateIdle State = iota
	StateStreaming
	StateComplete
	StateError
	StateCancelled
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateStreaming:
		return "streaming"
	case StateComplete:
		return "complete"
	case StateError:
		return "error"
	case StateCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Event represents a stream state change event
type Event struct {
	StreamID  string
	State     State
	Timestamp time.Time
	Data      any // Can be chunk, error, completion data, etc.
}

// Tracker tracks the state of multiple streams
type Tracker struct {
	states map[string]*StreamInfo
	mu     sync.RWMutex
}

// StreamInfo holds information about a stream
type StreamInfo struct {
	ID        string
	State     State
	StartTime time.Time
	EndTime   time.Time
	Error     error
	Buffer    string
}

// NewTracker creates a new state tracker
func NewTracker() *Tracker {
	return &Tracker{
		states: make(map[string]*StreamInfo),
	}
}

// StartStream marks a stream as started
func (t *Tracker) StartStream(streamID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.states[streamID] = &StreamInfo{
		ID:        streamID,
		State:     StateStreaming,
		StartTime: time.Now(),
	}
}

// UpdateState updates the state of a stream
func (t *Tracker) UpdateState(streamID string, state State) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if info, exists := t.states[streamID]; exists {
		info.State = state
		if state == StateComplete || state == StateError || state == StateCancelled {
			info.EndTime = time.Now()
		}
	}
}

// SetError sets an error for a stream
func (t *Tracker) SetError(streamID string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if info, exists := t.states[streamID]; exists {
		info.Error = err
		info.State = StateError
		info.EndTime = time.Now()
	}
}

// GetState returns the current state of a stream
func (t *Tracker) GetState(streamID string) (State, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if info, exists := t.states[streamID]; exists {
		return info.State, true
	}
	return StateIdle, false
}

// GetStreamInfo returns full information about a stream
func (t *Tracker) GetStreamInfo(streamID string) (*StreamInfo, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	info, exists := t.states[streamID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	copy := *info
	return &copy, true
}

// AppendBuffer appends content to a stream's buffer
func (t *Tracker) AppendBuffer(streamID string, content string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if info, exists := t.states[streamID]; exists {
		info.Buffer += content
	}
}

// Cleanup removes completed/errored streams older than the specified duration
func (t *Tracker) Cleanup(olderThan time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for id, info := range t.states {
		if info.State == StateComplete || info.State == StateError || info.State == StateCancelled {
			if now.Sub(info.EndTime) > olderThan {
				delete(t.states, id)
			}
		}
	}
}

// StatefulHandler wraps a Handler with state tracking
type StatefulHandler struct {
	handler  Handler
	tracker  *Tracker
	streamID string
}

// NewStatefulHandler creates a handler that tracks state
func NewStatefulHandler(handler Handler, tracker *Tracker, streamID string) *StatefulHandler {
	return &StatefulHandler{
		handler:  handler,
		tracker:  tracker,
		streamID: streamID,
	}
}

// OnChunk implements Handler with state tracking
func (s *StatefulHandler) OnChunk(chunk []byte) error {
	s.tracker.AppendBuffer(s.streamID, string(chunk))
	return s.handler.OnChunk(chunk)
}

// OnComplete implements Handler with state tracking
func (s *StatefulHandler) OnComplete(finalContent string) error {
	s.tracker.UpdateState(s.streamID, StateComplete)
	return s.handler.OnComplete(finalContent)
}

// OnError implements Handler with state tracking
func (s *StatefulHandler) OnError(err error) {
	s.tracker.SetError(s.streamID, err)
	s.handler.OnError(err)
}
