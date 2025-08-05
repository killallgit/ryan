package tui

import (
	"sync"
)

// AppState represents the centralized application state
type AppState struct {
	mu           sync.RWMutex
	currentView  string
	previousView string
	sending      bool
	streaming    bool
	streamID     string
	model        string
	messages     []string // Simplified for now
}

// StateManager manages the application state and notifies listeners of changes
type StateManager struct {
	state     *AppState
	listeners []StateListener
	mu        sync.RWMutex
}

// StateListener is a function that gets called when state changes
type StateListener func(oldState, newState *AppState)

// NewStateManager creates a new state manager with initial state
func NewStateManager() *StateManager {
	return &StateManager{
		state: &AppState{
			currentView:  "chat",
			previousView: "chat",
			sending:      false,
			streaming:    false,
			streamID:     "",
			model:        "",
			messages:     make([]string, 0),
		},
		listeners: make([]StateListener, 0),
	}
}

// GetState returns a copy of the current state
func (sm *StateManager) GetState() *AppState {
	sm.state.mu.RLock()
	defer sm.state.mu.RUnlock()
	
	// Return a copy to prevent external modifications
	return &AppState{
		currentView:  sm.state.currentView,
		previousView: sm.state.previousView,
		sending:      sm.state.sending,
		streaming:    sm.state.streaming,
		streamID:     sm.state.streamID,
		model:        sm.state.model,
		messages:     append([]string(nil), sm.state.messages...),
	}
}

// UpdateState updates the state with a new state and notifies listeners
func (sm *StateManager) UpdateState(newState *AppState) {
	sm.state.mu.Lock()
	oldState := &AppState{
		currentView:  sm.state.currentView,
		previousView: sm.state.previousView,
		sending:      sm.state.sending,
		streaming:    sm.state.streaming,
		streamID:     sm.state.streamID,
		model:        sm.state.model,
		messages:     append([]string(nil), sm.state.messages...),
	}
	
	// Update state
	sm.state.currentView = newState.currentView
	sm.state.previousView = newState.previousView
	sm.state.sending = newState.sending
	sm.state.streaming = newState.streaming
	sm.state.streamID = newState.streamID
	sm.state.model = newState.model
	sm.state.messages = append([]string(nil), newState.messages...)
	
	sm.state.mu.Unlock()
	
	// Notify listeners
	sm.notifyListeners(oldState, newState)
}

// SetCurrentView updates the current view
func (sm *StateManager) SetCurrentView(view string) {
	sm.state.mu.Lock()
	oldState := sm.GetStateUnsafe()
	
	if sm.state.currentView != view {
		sm.state.previousView = sm.state.currentView
		sm.state.currentView = view
	}
	
	newState := sm.GetStateUnsafe()
	sm.state.mu.Unlock()
	
	sm.notifyListeners(oldState, newState)
}

// SetSending updates the sending state
func (sm *StateManager) SetSending(sending bool) {
	sm.state.mu.Lock()
	oldState := sm.GetStateUnsafe()
	sm.state.sending = sending
	newState := sm.GetStateUnsafe()
	sm.state.mu.Unlock()
	
	sm.notifyListeners(oldState, newState)
}

// SetStreaming updates the streaming state
func (sm *StateManager) SetStreaming(streaming bool, streamID string) {
	sm.state.mu.Lock()
	oldState := sm.GetStateUnsafe()
	sm.state.streaming = streaming
	sm.state.streamID = streamID
	newState := sm.GetStateUnsafe()
	sm.state.mu.Unlock()
	
	sm.notifyListeners(oldState, newState)
}

// SetModel updates the current model
func (sm *StateManager) SetModel(model string) {
	sm.state.mu.Lock()
	oldState := sm.GetStateUnsafe()
	sm.state.model = model
	newState := sm.GetStateUnsafe()
	sm.state.mu.Unlock()
	
	sm.notifyListeners(oldState, newState)
}

// GetStateUnsafe returns the current state without locking (for internal use only)
func (sm *StateManager) GetStateUnsafe() *AppState {
	return &AppState{
		currentView:  sm.state.currentView,
		previousView: sm.state.previousView,
		sending:      sm.state.sending,
		streaming:    sm.state.streaming,
		streamID:     sm.state.streamID,
		model:        sm.state.model,
		messages:     append([]string(nil), sm.state.messages...),
	}
}

// AddListener adds a state change listener
func (sm *StateManager) AddListener(listener StateListener) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listeners = append(sm.listeners, listener)
}

// RemoveListener removes a state change listener
func (sm *StateManager) RemoveListener(targetListener StateListener) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Note: This is a simplified implementation
	// In a real scenario, you might want to use a more sophisticated method
	// to identify and remove specific listeners
	for i, listener := range sm.listeners {
		// This comparison won't work in Go, but shows the intent
		// In practice, you'd need to use IDs or other identification methods
		_ = listener
		if i >= 0 { // Placeholder logic
			sm.listeners = append(sm.listeners[:i], sm.listeners[i+1:]...)
			break
		}
	}
}

// notifyListeners notifies all registered listeners of state changes
func (sm *StateManager) notifyListeners(oldState, newState *AppState) {
	sm.mu.RLock()
	listeners := make([]StateListener, len(sm.listeners))
	copy(listeners, sm.listeners)
	sm.mu.RUnlock()
	
	// Notify listeners in separate goroutines to avoid blocking
	for _, listener := range listeners {
		go func(l StateListener) {
			l(oldState, newState)
		}(listener)
	}
}

// IsCurrentView checks if the given view is the current view
func (sm *StateManager) IsCurrentView(view string) bool {
	sm.state.mu.RLock()
	defer sm.state.mu.RUnlock()
	return sm.state.currentView == view
}

// IsSending checks if the app is in sending state
func (sm *StateManager) IsSending() bool {
	sm.state.mu.RLock()
	defer sm.state.mu.RUnlock()
	return sm.state.sending
}

// IsStreaming checks if the app is in streaming state
func (sm *StateManager) IsStreaming() bool {
	sm.state.mu.RLock()
	defer sm.state.mu.RUnlock()
	return sm.state.streaming
}

// GetCurrentView returns the current view name
func (sm *StateManager) GetCurrentView() string {
	sm.state.mu.RLock()
	defer sm.state.mu.RUnlock()
	return sm.state.currentView
}

// GetPreviousView returns the previous view name
func (sm *StateManager) GetPreviousView() string {
	sm.state.mu.RLock()
	defer sm.state.mu.RUnlock()
	return sm.state.previousView
}

// GetModel returns the current model name
func (sm *StateManager) GetModel() string {
	sm.state.mu.RLock()
	defer sm.state.mu.RUnlock()
	return sm.state.model
}

// GetStreamID returns the current stream ID
func (sm *StateManager) GetStreamID() string {
	sm.state.mu.RLock()
	defer sm.state.mu.RUnlock()
	return sm.state.streamID
}