package tui

import (
	"sync"

	"github.com/rivo/tview"
)

// ComponentLifecycle defines the lifecycle methods for components
type ComponentLifecycle interface {
	OnMount() error
	OnUnmount() error
	OnUpdate(state *AppState) error
	OnEvent(event Event) error
}

// Component represents a UI component with lifecycle management
type Component interface {
	ComponentLifecycle

	// UI methods
	Render() tview.Primitive
	GetName() string
	IsVisible() bool
	SetVisible(visible bool)

	// State management
	NeedsUpdate() bool
	MarkUpdated()

	// Event handling
	CanHandleEvent(eventType EventType) bool
}

// BaseComponent provides a basic implementation of the Component interface
type BaseComponent struct {
	name          string
	visible       bool
	needsUpdate   bool
	primitive     tview.Primitive
	eventBus      *EventBus
	stateManager  *StateManager
	handledEvents []EventType
	mu            sync.RWMutex
}

// NewBaseComponent creates a new base component
func NewBaseComponent(name string, primitive tview.Primitive, eventBus *EventBus, stateManager *StateManager) *BaseComponent {
	return &BaseComponent{
		name:          name,
		visible:       true,
		needsUpdate:   true,
		primitive:     primitive,
		eventBus:      eventBus,
		stateManager:  stateManager,
		handledEvents: make([]EventType, 0),
	}
}

// Lifecycle methods (default implementations)

// OnMount is called when the component is mounted
func (bc *BaseComponent) OnMount() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Subscribe to events this component can handle
	for _, eventType := range bc.handledEvents {
		bc.eventBus.Subscribe(eventType, func(event Event) {
			_ = bc.OnEvent(event) // Ignore error in event handler
		})
	}

	return nil
}

// OnUnmount is called when the component is unmounted
func (bc *BaseComponent) OnUnmount() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Unsubscribe from events
	for _, eventType := range bc.handledEvents {
		bc.eventBus.Unsubscribe(eventType, func(event Event) {
			_ = bc.OnEvent(event) // Ignore error in event handler
		})
	}

	return nil
}

// OnUpdate is called when the application state changes
func (bc *BaseComponent) OnUpdate(state *AppState) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Default implementation marks as needing update
	bc.needsUpdate = true
	return nil
}

// OnEvent is called when an event occurs that this component handles
func (bc *BaseComponent) OnEvent(event Event) error {
	// Default implementation does nothing
	return nil
}

// UI methods

// Render returns the tview primitive for this component
func (bc *BaseComponent) Render() tview.Primitive {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.primitive
}

// GetName returns the component name
func (bc *BaseComponent) GetName() string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.name
}

// IsVisible returns whether the component is visible
func (bc *BaseComponent) IsVisible() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.visible
}

// SetVisible sets the component visibility
func (bc *BaseComponent) SetVisible(visible bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.visible != visible {
		bc.visible = visible
		bc.needsUpdate = true
	}
}

// State management

// NeedsUpdate returns whether the component needs to be updated
func (bc *BaseComponent) NeedsUpdate() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.needsUpdate
}

// MarkUpdated marks the component as updated
func (bc *BaseComponent) MarkUpdated() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.needsUpdate = false
}

// Event handling

// CanHandleEvent returns whether this component can handle the given event type
func (bc *BaseComponent) CanHandleEvent(eventType EventType) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	for _, et := range bc.handledEvents {
		if et == eventType {
			return true
		}
	}
	return false
}

// SetHandledEvents sets the event types this component can handle
func (bc *BaseComponent) SetHandledEvents(eventTypes []EventType) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.handledEvents = make([]EventType, len(eventTypes))
	copy(bc.handledEvents, eventTypes)
}

// AddHandledEvent adds an event type this component can handle
func (bc *BaseComponent) AddHandledEvent(eventType EventType) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Check if already exists
	for _, et := range bc.handledEvents {
		if et == eventType {
			return
		}
	}

	bc.handledEvents = append(bc.handledEvents, eventType)
}

// ComponentManager manages the lifecycle of all UI components
type ComponentManager struct {
	components   map[string]Component
	eventBus     *EventBus
	stateManager *StateManager
	mu           sync.RWMutex
}

// NewComponentManager creates a new component manager
func NewComponentManager(eventBus *EventBus, stateManager *StateManager) *ComponentManager {
	return &ComponentManager{
		components:   make(map[string]Component),
		eventBus:     eventBus,
		stateManager: stateManager,
	}
}

// Register registers a component with the manager
func (cm *ComponentManager) Register(component Component) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	name := component.GetName()
	if _, exists := cm.components[name]; exists {
		return &ComponentError{
			Component: name,
			Message:   "component already registered",
		}
	}

	cm.components[name] = component

	// Mount the component
	if err := component.OnMount(); err != nil {
		delete(cm.components, name)
		return &ComponentError{
			Component: name,
			Message:   "failed to mount component",
			Cause:     err,
		}
	}

	return nil
}

// Unregister unregisters a component from the manager
func (cm *ComponentManager) Unregister(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	component, exists := cm.components[name]
	if !exists {
		return &ComponentError{
			Component: name,
			Message:   "component not found",
		}
	}

	// Unmount the component
	if err := component.OnUnmount(); err != nil {
		return &ComponentError{
			Component: name,
			Message:   "failed to unmount component",
			Cause:     err,
		}
	}

	delete(cm.components, name)
	return nil
}

// Get retrieves a component by name
func (cm *ComponentManager) Get(name string) (Component, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	component, exists := cm.components[name]
	return component, exists
}

// GetAll returns all registered components
func (cm *ComponentManager) GetAll() map[string]Component {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make(map[string]Component, len(cm.components))
	for name, component := range cm.components {
		result[name] = component
	}

	return result
}

// UpdateAll updates all components with the current state
func (cm *ComponentManager) UpdateAll(state *AppState) {
	cm.mu.RLock()
	components := make([]Component, 0, len(cm.components))
	for _, component := range cm.components {
		components = append(components, component)
	}
	cm.mu.RUnlock()

	// Update components in parallel
	var wg sync.WaitGroup
	for _, component := range components {
		wg.Add(1)
		go func(c Component) {
			defer wg.Done()
			if err := c.OnUpdate(state); err != nil {
				// Log error but don't fail the entire update
				_ = err
			}
		}(component)
	}

	wg.Wait()
}

// GetVisibleComponents returns all visible components
func (cm *ComponentManager) GetVisibleComponents() []Component {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]Component, 0)
	for _, component := range cm.components {
		if component.IsVisible() {
			result = append(result, component)
		}
	}

	return result
}

// GetComponentsNeedingUpdate returns components that need updating
func (cm *ComponentManager) GetComponentsNeedingUpdate() []Component {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]Component, 0)
	for _, component := range cm.components {
		if component.NeedsUpdate() {
			result = append(result, component)
		}
	}

	return result
}

// ComponentError represents an error related to component management
type ComponentError struct {
	Component string
	Message   string
	Cause     error
}

// Error implements the error interface
func (ce *ComponentError) Error() string {
	if ce.Cause != nil {
		return ce.Component + ": " + ce.Message + " (" + ce.Cause.Error() + ")"
	}
	return ce.Component + ": " + ce.Message
}

// Unwrap returns the underlying error
func (ce *ComponentError) Unwrap() error {
	return ce.Cause
}
