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
	menu        FilterableMenuComponent
}

func NewViewManager() *ViewManager {
	return &ViewManager{
		views:       make(map[string]View),
		currentView: "",
		menuVisible: false,
		menu:        NewFilterableMenuComponent(),
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
	if vm.menuVisible {
		vm.menu = vm.menu.WithInputText("").WithInputMode(true)
	}
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
		vm.menu = vm.menu.WithInputText("").WithInputMode(true)
		return true

	case tcell.KeyEnter:
		selectedView := vm.menu.GetSelectedOption()
		if selectedView != "" {
			vm.SetCurrentView(selectedView)
			vm.menu = vm.menu.WithInputText("").WithInputMode(true)
			return true
		}
		return false

	case tcell.KeyTab:
		vm.menu = vm.menu.WithInputMode(!vm.menu.IsInputMode())
		return true

	case tcell.KeyUp, tcell.KeyCtrlP:
		if !vm.menu.IsInputMode() {
			vm.menu = vm.menu.SelectPrevious()
		}
		return true

	case tcell.KeyDown, tcell.KeyCtrlN:
		if !vm.menu.IsInputMode() {
			vm.menu = vm.menu.SelectNext()
		} else {
			vm.menu = vm.menu.WithInputMode(false)
		}
		return true

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if vm.menu.IsInputMode() {
			vm.menu = vm.menu.DeleteChar()
		}
		return true

	case tcell.KeyLeft:
		if vm.menu.IsInputMode() {
			vm.menu = vm.menu.MoveCursorLeft()
		}
		return true

	case tcell.KeyRight:
		if vm.menu.IsInputMode() {
			vm.menu = vm.menu.MoveCursorRight()
		}
		return true

	default:
		if ev.Rune() != 0 && vm.menu.IsInputMode() {
			switch ev.Rune() {
			case 'j', 'J', 'k', 'K':
				if !vm.menu.IsInputMode() {
					if ev.Rune() == 'j' || ev.Rune() == 'J' {
						vm.menu = vm.menu.SelectNext()
					} else {
						vm.menu = vm.menu.SelectPrevious()
					}
					return true
				}
				fallthrough
			default:
				vm.menu = vm.menu.AddChar(ev.Rune())
				return true
			}
		} else if !vm.menu.IsInputMode() {
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

	// Command palette style: larger, centered modal
	menuWidth := 70 // Fixed width for consistency
	if menuWidth > area.Width-4 {
		menuWidth = area.Width - 4 // minimal margin
	}

	// Fixed height for 5 items + input + borders + separator + more indicator
	maxItems := 5
	actualItems := len(vm.views)
	if actualItems > maxItems {
		actualItems = maxItems
	}

	menuHeight := actualItems + 4 // items + input + borders + separator
	if len(vm.views) > maxItems {
		menuHeight += 1 // space for "more" indicator
	}

	// Ensure minimum menu size
	if menuWidth < 30 {
		menuWidth = 30
	}

	// Ensure minimum menu height for the input field
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

	// Update command palette size and render
	vm.menu = vm.menu.WithSize(menuWidth, menuHeight)
	vm.menu.Render(screen, menuArea)
}
