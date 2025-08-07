package tui

import (
	"github.com/killallgit/ryan/pkg/tui/async"
	"github.com/killallgit/ryan/pkg/tui/events"
	"github.com/killallgit/ryan/pkg/tui/state"
	"github.com/rivo/tview"
)

// ReactiveComponent defines the interface for components that react to state changes
type ReactiveComponent interface {
	// OnStateChanged is called when relevant state changes occur
	OnStateChanged(change state.StateChange)

	// Subscribe should set up subscriptions to relevant state changes
	Subscribe(stateManager *state.StateManager, eventBus *events.EventBus)

	// Unsubscribe should clean up subscriptions
	Unsubscribe(stateManager *state.StateManager, eventBus *events.EventBus)
}

// AsyncComponent defines the interface for components that perform async operations
type AsyncComponent interface {
	// SetAsyncManager provides access to the async manager
	SetAsyncManager(asyncManager *async.AsyncManager)
}

// TUIContext provides shared services to all components
type TUIContext struct {
	App          *tview.Application
	StateManager *state.StateManager
	EventBus     *events.EventBus
	AsyncManager *async.AsyncManager
}

// NewTUIContext creates a new TUI context with all services
func NewTUIContext(app *tview.Application) *TUIContext {
	stateManager := state.NewStateManager()
	eventBus := events.NewEventBus()
	asyncManager := async.NewAsyncManager(app, eventBus, 5) // 5 workers

	return &TUIContext{
		App:          app,
		StateManager: stateManager,
		EventBus:     eventBus,
		AsyncManager: asyncManager,
	}
}

// Close cleans up all services
func (ctx *TUIContext) Close() {
	ctx.AsyncManager.Close()
	ctx.EventBus.Close()
}

// ViewComponent defines the interface for view components
type ViewComponent interface {
	ReactiveComponent
	AsyncComponent
	tview.Primitive

	// GetName returns the name of the view
	GetName() string

	// SetFocused is called when the view gains/loses focus
	SetFocused(focused bool)
}

// ModalComponent defines the interface for modal components
type ModalComponent interface {
	// Show displays the modal
	Show(pages *tview.Pages)

	// Hide removes the modal
	Hide(pages *tview.Pages)

	// GetName returns the modal name
	GetName() string
}

// NavigationAction represents a navigation action
type NavigationAction struct {
	Type        string // "switch_view", "show_modal", "hide_modal"
	Target      string // View name or modal name
	Data        interface{}
	ReturnFocus bool
}

// UIAction represents a UI action that can be performed
type UIAction struct {
	Type    string // "select_model", "send_message", "refresh_models", etc.
	Payload interface{}
	Source  string
}

// Progress represents operation progress
type Progress struct {
	Current       int64
	Total         int64
	Percentage    float64
	Message       string
	Indeterminate bool
}

// ViewState represents the state of a specific view
type ViewState struct {
	Name    string
	Focused bool
	Loading bool
	Error   string
	Data    interface{}
}

// Constants for common action types
const (
	ActionSwitchView      = "switch_view"
	ActionShowModal       = "show_modal"
	ActionHideModal       = "hide_modal"
	ActionSelectModel     = "select_model"
	ActionSendMessage     = "send_message"
	ActionRefreshModels   = "refresh_models"
	ActionDeleteModel     = "delete_model"
	ActionDownloadModel   = "download_model"
	ActionCancelOperation = "cancel_operation"
)

// Constants for view names
const (
	ViewChat        = "chat"
	ViewModels      = "models"
	ViewTools       = "tools"
	ViewVectorStore = "vectorstore"
	ViewContextTree = "context-tree"
)

// Constants for modal names
const (
	ModalViewSwitcher  = "view-switcher"
	ModalModelDownload = "model-download"
	ModalConfirm       = "confirm"
	ModalError         = "error"
)
