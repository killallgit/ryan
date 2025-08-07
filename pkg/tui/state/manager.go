package state

import (
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/logger"
)

// AppState represents the complete application state
type AppState struct {
	// UI State
	CurrentView   string
	PreviousView  string
	Sending       bool
	Streaming     bool
	StreamID      string
	CurrentAgent  string
	CurrentAction string
	UIState       string // idle, sending, thinking, streaming, executing, preparing_tools

	// Model State
	CurrentModel    string
	AvailableModels []string
	ModelValidating bool
	ModelError      string

	// Chat State
	Messages     []chat.Message
	StreamBuffer string
	TokenUsage   TokenUsage

	// View State
	ModelViewLoading bool
	ToolsViewData    interface{}
	VectorStoreData  interface{}

	// System State
	Connected  bool
	LastError  string
	LastUpdate time.Time
}

// TokenUsage represents token usage statistics
type TokenUsage struct {
	PromptTokens   int
	ResponseTokens int
}

// StateManager manages the centralized application state
type StateManager struct {
	state     *AppState
	mutex     sync.RWMutex
	observers map[string][]Observer
	log       *logger.Logger
}

// Observer defines the interface for state observers
type Observer interface {
	OnStateChanged(change StateChange)
}

// StateChange represents a state mutation
type StateChange struct {
	Type     string
	Field    string
	OldValue interface{}
	NewValue interface{}
	State    *AppState
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		state: &AppState{
			CurrentView:  "chat",
			PreviousView: "chat",
			UIState:      "idle",
			LastUpdate:   time.Now(),
		},
		observers: make(map[string][]Observer),
		log:       logger.WithComponent("state_manager"),
	}
}

// GetState returns a copy of the current state (read-only)
func (sm *StateManager) GetState() AppState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Return a copy to prevent external modifications
	return *sm.state
}

// Subscribe adds an observer for specific state changes
func (sm *StateManager) Subscribe(changeType string, observer Observer) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.observers[changeType] = append(sm.observers[changeType], observer)
	sm.log.Debug("Observer subscribed", "changeType", changeType)
}

// Unsubscribe removes an observer
func (sm *StateManager) Unsubscribe(changeType string, observer Observer) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	observers := sm.observers[changeType]
	for i, obs := range observers {
		if obs == observer {
			sm.observers[changeType] = append(observers[:i], observers[i+1:]...)
			break
		}
	}
	sm.log.Debug("Observer unsubscribed", "changeType", changeType)
}

// notifyObservers sends state changes to all relevant observers
func (sm *StateManager) notifyObservers(change StateChange) {
	observers := sm.observers[change.Type]
	observers = append(observers, sm.observers["*"]...) // Global observers

	for _, observer := range observers {
		go func(obs Observer) {
			obs.OnStateChanged(change)
		}(observer)
	}
}

// setState is the internal method for state mutations
func (sm *StateManager) setState(changeType, field string, newValue interface{}) {
	sm.mutex.Lock()

	var oldValue interface{}

	// Get old value and set new value based on field
	switch field {
	case "CurrentView":
		oldValue = sm.state.CurrentView
		sm.state.CurrentView = newValue.(string)
	case "PreviousView":
		oldValue = sm.state.PreviousView
		sm.state.PreviousView = newValue.(string)
	case "Sending":
		oldValue = sm.state.Sending
		sm.state.Sending = newValue.(bool)
	case "Streaming":
		oldValue = sm.state.Streaming
		sm.state.Streaming = newValue.(bool)
	case "StreamID":
		oldValue = sm.state.StreamID
		sm.state.StreamID = newValue.(string)
	case "CurrentAgent":
		oldValue = sm.state.CurrentAgent
		sm.state.CurrentAgent = newValue.(string)
	case "CurrentAction":
		oldValue = sm.state.CurrentAction
		sm.state.CurrentAction = newValue.(string)
	case "UIState":
		oldValue = sm.state.UIState
		sm.state.UIState = newValue.(string)
	case "CurrentModel":
		oldValue = sm.state.CurrentModel
		sm.state.CurrentModel = newValue.(string)
	case "AvailableModels":
		oldValue = sm.state.AvailableModels
		sm.state.AvailableModels = newValue.([]string)
	case "ModelValidating":
		oldValue = sm.state.ModelValidating
		sm.state.ModelValidating = newValue.(bool)
	case "ModelError":
		oldValue = sm.state.ModelError
		sm.state.ModelError = newValue.(string)
	case "Messages":
		oldValue = sm.state.Messages
		sm.state.Messages = newValue.([]chat.Message)
	case "StreamBuffer":
		oldValue = sm.state.StreamBuffer
		sm.state.StreamBuffer = newValue.(string)
	case "ModelViewLoading":
		oldValue = sm.state.ModelViewLoading
		sm.state.ModelViewLoading = newValue.(bool)
	case "Connected":
		oldValue = sm.state.Connected
		sm.state.Connected = newValue.(bool)
	case "LastError":
		oldValue = sm.state.LastError
		sm.state.LastError = newValue.(string)
	}

	sm.state.LastUpdate = time.Now()

	// Create change notification
	change := StateChange{
		Type:     changeType,
		Field:    field,
		OldValue: oldValue,
		NewValue: newValue,
		State:    sm.state,
	}

	sm.mutex.Unlock()

	sm.log.Debug("State changed", "type", changeType, "field", field, "newValue", newValue)

	// Notify observers (outside of lock to prevent deadlocks)
	sm.notifyObservers(change)
}

// Public methods for state mutations

// SetView changes the current view
func (sm *StateManager) SetView(newView string) {
	currentState := sm.GetState()
	if currentState.CurrentView != newView {
		sm.setState("view_change", "PreviousView", currentState.CurrentView)
		sm.setState("view_change", "CurrentView", newView)
	}
}

// SetSending changes the sending state
func (sm *StateManager) SetSending(sending bool) {
	sm.setState("sending_change", "Sending", sending)
}

// SetStreaming changes the streaming state
func (sm *StateManager) SetStreaming(streaming bool, streamID string) {
	sm.setState("streaming_change", "Streaming", streaming)
	sm.setState("streaming_change", "StreamID", streamID)
}

// SetUIState changes the UI state
func (sm *StateManager) SetUIState(uiState string) {
	sm.setState("ui_state_change", "UIState", uiState)
}

// SetCurrentAgent sets the current agent
func (sm *StateManager) SetCurrentAgent(agent string) {
	sm.setState("agent_change", "CurrentAgent", agent)
}

// SetCurrentAction sets the current action
func (sm *StateManager) SetCurrentAction(action string) {
	sm.setState("action_change", "CurrentAction", action)
}

// SetCurrentModel changes the current model
func (sm *StateManager) SetCurrentModel(model string) {
	sm.setState("model_change", "CurrentModel", model)
}

// SetAvailableModels updates the list of available models
func (sm *StateManager) SetAvailableModels(models []string) {
	sm.setState("models_change", "AvailableModels", models)
}

// SetModelValidating sets model validation state
func (sm *StateManager) SetModelValidating(validating bool) {
	sm.setState("model_validation_change", "ModelValidating", validating)
}

// SetModelError sets model error state
func (sm *StateManager) SetModelError(err string) {
	sm.setState("model_error_change", "ModelError", err)
}

// SetMessages updates the chat messages
func (sm *StateManager) SetMessages(messages []chat.Message) {
	sm.setState("messages_change", "Messages", messages)
}

// SetStreamBuffer updates the stream buffer
func (sm *StateManager) SetStreamBuffer(buffer string) {
	sm.setState("stream_change", "StreamBuffer", buffer)
}

// SetModelViewLoading sets model view loading state
func (sm *StateManager) SetModelViewLoading(loading bool) {
	sm.setState("model_view_change", "ModelViewLoading", loading)
}

// SetConnected sets connection state
func (sm *StateManager) SetConnected(connected bool) {
	sm.setState("connection_change", "Connected", connected)
}

// SetError sets the last error
func (sm *StateManager) SetError(err string) {
	sm.setState("error_change", "LastError", err)
}
