package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/logger"
)

type View interface {
	Render(screen tcell.Screen, area Rect)
	HandleKeyEvent(ev *tcell.EventKey, sending bool) bool
	HandleResize(width, height int)
	Name() string
	Description() string
}

type ViewManager struct {
	views       map[string]View
	currentView string
	menuVisible bool
	menu        MenuComponent
}

func NewViewManager() *ViewManager {
	return &ViewManager{
		views:       make(map[string]View),
		currentView: "",
		menuVisible: false,
		menu:        NewMenuComponent(),
	}
}

func (vm *ViewManager) RegisterView(name string, view View) {
	vm.views[name] = view
	vm.menu = vm.menu.WithOption(name, view.Description())

	if vm.currentView == "" {
		vm.currentView = name
	}
}

func (vm *ViewManager) GetCurrentView() View {
	if view, exists := vm.views[vm.currentView]; exists {
		return view
	}
	return nil
}

func (vm *ViewManager) GetCurrentViewName() string {
	return vm.currentView
}

func (vm *ViewManager) SetCurrentView(name string) bool {
	log := logger.WithComponent("view_manager")
	log.Debug("SetCurrentView called", "requested_view", name, "current_view", vm.currentView)

	if _, exists := vm.views[name]; exists {
		oldView := vm.currentView
		vm.currentView = name
		vm.menuVisible = false

		log.Debug("View switched", "from", oldView, "to", name)

		// If switching to ModelView, activate it to start loading data
		if name == "models" {
			if modelView, ok := vm.views[name].(*ModelView); ok {
				log.Debug("Activating ModelView")
				modelView.Activate()
			}
		}
		
		// If switching to context tree view, update it with current conversation data
		if name == "context-tree" {
			log.Debug("Switching to context tree view")
			// This will be called by the app when it detects context tree view is active
		}

		return true
	}
	log.Debug("View not found", "requested_view", name)
	return false
}

func (vm *ViewManager) SyncViewState(sending bool) {
	log := logger.WithComponent("view_manager")
	log.Debug("Syncing view state", "sending", sending, "current_view", vm.currentView)

	// Sync ChatView state when it's the current view
	if vm.currentView == "chat" {
		if chatView, ok := vm.views["chat"].(*ChatView); ok {
			log.Debug("Syncing ChatView state")
			chatView.SyncWithAppState(sending)
		}
	}
}

func (vm *ViewManager) ToggleMenu() {
	vm.menuVisible = !vm.menuVisible
}

func (vm *ViewManager) IsMenuVisible() bool {
	return vm.menuVisible
}

func (vm *ViewManager) HideMenu() {
	vm.menuVisible = false
}

func (vm *ViewManager) HandleMenuKeyEvent(ev *tcell.EventKey) bool {
	if !vm.menuVisible {
		return false
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		vm.menuVisible = false
		return true

	case tcell.KeyEnter:
		selectedView := vm.menu.GetSelectedOption()
		if selectedView != "" {
			vm.SetCurrentView(selectedView)
			return true
		}
		return false

	case tcell.KeyUp, tcell.KeyCtrlP:
		vm.menu = vm.menu.SelectPrevious()
		return true

	case tcell.KeyDown, tcell.KeyCtrlN:
		vm.menu = vm.menu.SelectNext()
		return true

	default:
		if ev.Rune() != 0 {
			switch ev.Rune() {
			case 'j', 'J':
				vm.menu = vm.menu.SelectNext()
				return true
			case 'k', 'K':
				vm.menu = vm.menu.SelectPrevious()
				return true
			}
		}
	}

	return false
}

func (vm *ViewManager) HandleMenuMouseEvent(ev *tcell.EventMouse) bool {
	if !vm.menuVisible {
		return false
	}

	// For now, just hide menu on any click - could be enhanced later
	// to handle specific menu item clicks
	vm.menuVisible = false
	return true
}

func (vm *ViewManager) HandleResize(width, height int) {
	for _, view := range vm.views {
		view.HandleResize(width, height)
	}
	vm.menu = vm.menu.WithSize(width, height)
}

func (vm *ViewManager) Render(screen tcell.Screen, area Rect) {
	currentView := vm.GetCurrentView()
	if currentView != nil {
		currentView.Render(screen, area)
	}

	if vm.menuVisible {
		vm.renderMenu(screen, area)
	}
}

func (vm *ViewManager) renderMenu(screen tcell.Screen, area Rect) {
	// Don't render menu if area is too small
	if area.Width < 10 || area.Height < 6 {
		return
	}

	// Calculate optimal menu width based on content - wider command palette style
	maxDescLen := 0
	for _, view := range vm.views {
		descLen := len(view.Description())
		if descLen > maxDescLen {
			maxDescLen = descLen
		}
	}

	// Command palette style: wider menu with more padding
	menuWidth := maxDescLen + 8 // description + padding
	if menuWidth < 60 {
		menuWidth = 60 // minimum width for command palette
	}
	if menuWidth > area.Width-6 {
		menuWidth = area.Width - 6 // minimal margin
	}

	// Ensure minimum menu width
	if menuWidth < 30 {
		menuWidth = 30
	}

	menuHeight := len(vm.views) + 2 // options + borders only (no header/footer)

	// Ensure minimum menu height
	if menuHeight < 6 {
		menuHeight = 6
	}
	if menuHeight > area.Height-2 {
		menuHeight = area.Height - 2 // leave margin
	}

	menuX := (area.Width - menuWidth) / 2
	menuY := (area.Height - menuHeight) / 2

	if menuX < 0 {
		menuX = 0
		menuWidth = area.Width
	}
	if menuY < 0 {
		menuY = 0
		menuHeight = area.Height
	}

	menuArea := Rect{
		X:      menuX,
		Y:      menuY,
		Width:  menuWidth,
		Height: menuHeight,
	}

	vm.menu.Render(screen, menuArea)
}
