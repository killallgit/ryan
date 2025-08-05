package tui_test

import (
	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/tui"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ViewManager", func() {
	var (
		screen      *tui.TestScreen
		viewManager *tui.ViewManager
	)

	BeforeEach(func() {
		screen = tui.NewTestScreen()
		err := screen.Init()
		Expect(err).ToNot(HaveOccurred())
		screen.SetSize(80, 24)
		screen.Clear()
		
		viewManager = tui.NewViewManager()
	})

	AfterEach(func() {
		screen.Fini()
	})

	Describe("View Registration", func() {
		It("should register views correctly", func() {
			mockView := &MockViewImpl{
				name:        "test-view",
				description: "Test View",
			}
			
			viewManager.RegisterView("test", mockView)
			
			// The view should be registered and available
			// We can verify this by triggering the menu and checking if it appears
			viewManager.ToggleMenu()
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 24}
			viewManager.Render(screen, area)
			
			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring("Test View"))
		})

		It("should handle multiple view registrations", func() {
			views := []struct {
				key         string
				name        string
				description string
			}{
				{"chat", "chat-view", "Chat with AI"},
				{"models", "model-view", "Manage Models"},
				{"tools", "tools-view", "Tools Registry"},
			}
			
			for _, v := range views {
				mockView := &MockViewImpl{
					name:        v.name,
					description: v.description,
				}
				viewManager.RegisterView(v.key, mockView)
			}
			
			// Toggle menu to see all views
			viewManager.ToggleMenu()
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 24}
			viewManager.Render(screen, area)
			
			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring("Chat with AI"))
			Expect(content).To(ContainSubstring("Manage Models"))
			Expect(content).To(ContainSubstring("Tools Registry"))
		})
	})

	Describe("View Switching", func() {
		var (
			chatView  *MockViewImpl
			modelView *MockViewImpl
			toolsView *MockViewImpl
		)

		BeforeEach(func() {
			chatView = &MockViewImpl{
				name:        "chat-view",
				description: "Chat with AI",
			}
			modelView = &MockViewImpl{
				name:        "model-view",
				description: "Manage Models",
			}
			toolsView = &MockViewImpl{
				name:        "tools-view",
				description: "Tools Registry",
			}
			
			viewManager.RegisterView("chat", chatView)
			viewManager.RegisterView("models", modelView)
			viewManager.RegisterView("tools", toolsView)
		})

		It("should switch between views", func() {
			// Initially chat view is active (first registered)
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 24}
			
			// Current view should be "chat"
			Expect(viewManager.GetCurrentViewName()).To(Equal("chat"))
			
			// Switch to models view
			result := viewManager.SetCurrentView("models")
			Expect(result).To(BeTrue())
			Expect(viewManager.GetCurrentViewName()).To(Equal("models"))
			
			// Render to trigger view rendering
			viewManager.Render(screen, area)
			Expect(modelView.renderCalled).To(BeTrue())
			Expect(chatView.renderCalled).To(BeFalse())
		})

		It("should return false for non-existent view", func() {
			result := viewManager.SetCurrentView("non-existent")
			Expect(result).To(BeFalse())
			// Should stay on current view
			Expect(viewManager.GetCurrentViewName()).To(Equal("chat"))
		})

		It("should maintain view state when switching", func() {
			// Simulate some state in chat view
			chatView.state = "some chat state"
			
			// Switch to model view and back
			viewManager.SetCurrentView("models")
			viewManager.SetCurrentView("chat")
			
			// State should be preserved
			Expect(chatView.state).To(Equal("some chat state"))
		})
	})

	Describe("Menu Navigation", func() {
		BeforeEach(func() {
			// Register some views
			for i := 0; i < 5; i++ {
				mockView := &MockViewImpl{
					name:        string(rune('a' + i)),
					description: string(rune('A'+i)) + " View",
				}
				viewManager.RegisterView(string(rune('a'+i)), mockView)
			}
		})

		It("should show/hide menu on toggle", func() {
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 24}
			
			// Initially menu is hidden
			Expect(viewManager.IsMenuVisible()).To(BeFalse())
			
			// Toggle menu
			viewManager.ToggleMenu()
			Expect(viewManager.IsMenuVisible()).To(BeTrue())
			
			// Render to see menu
			viewManager.Render(screen, area)
			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring("A View"))
			
			// Toggle again to hide
			viewManager.ToggleMenu()
			Expect(viewManager.IsMenuVisible()).To(BeFalse())
			
			// Render again
			screen.Clear()
			viewManager.Render(screen, area)
			content = screen.CaptureContent()
			Expect(content).NotTo(ContainSubstring("A View"))
		})

		It("should navigate menu with keyboard", func() {
			viewManager.ToggleMenu()
			
			// Navigate down
			ev := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			handled := viewManager.HandleMenuKeyEvent(ev)
			Expect(handled).To(BeTrue())
			
			// Navigate up
			ev = tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			handled = viewManager.HandleMenuKeyEvent(ev)
			Expect(handled).To(BeTrue())
			
			// Select with Enter
			ev = tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
			handled = viewManager.HandleMenuKeyEvent(ev)
			Expect(handled).To(BeTrue())
			
			// Menu should be hidden after selection
			Expect(viewManager.IsMenuVisible()).To(BeFalse())
		})

		It("should close menu on Escape", func() {
			viewManager.ToggleMenu()
			Expect(viewManager.IsMenuVisible()).To(BeTrue())
			
			// Press Escape
			ev := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
			handled := viewManager.HandleMenuKeyEvent(ev)
			Expect(handled).To(BeTrue())
			
			// Menu should be closed
			Expect(viewManager.IsMenuVisible()).To(BeFalse())
		})

		It("should handle input mode toggle with Tab", func() {
			viewManager.ToggleMenu()
			
			// Press Tab to toggle input mode
			ev := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
			handled := viewManager.HandleMenuKeyEvent(ev)
			Expect(handled).To(BeTrue())
		})

		It("should handle character input in input mode", func() {
			viewManager.ToggleMenu()
			
			// Type a character
			ev := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
			handled := viewManager.HandleMenuKeyEvent(ev)
			Expect(handled).To(BeTrue())
		})
	})

	Describe("Event Handling", func() {
		var mockView *MockViewImpl

		BeforeEach(func() {
			mockView = &MockViewImpl{
				name:        "test-view",
				description: "Test View",
			}
			viewManager.RegisterView("test", mockView)
		})

		It("should return false for key events when menu not visible", func() {
			// Menu is not visible
			ev := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
			handled := viewManager.HandleMenuKeyEvent(ev)
			Expect(handled).To(BeFalse())
		})

		It("should handle menu events when menu is visible", func() {
			viewManager.ToggleMenu()
			
			// Menu navigation should be handled
			ev := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			handled := viewManager.HandleMenuKeyEvent(ev)
			Expect(handled).To(BeTrue())
		})

		It("should handle mouse events on menu", func() {
			viewManager.ToggleMenu()
			Expect(viewManager.IsMenuVisible()).To(BeTrue())
			
			// Any mouse event should close menu
			ev := tcell.NewEventMouse(10, 10, tcell.Button1, tcell.ModNone)
			handled := viewManager.HandleMenuMouseEvent(ev)
			Expect(handled).To(BeTrue())
			Expect(viewManager.IsMenuVisible()).To(BeFalse())
		})
	})

	Describe("Resize Handling", func() {
		var mockView *MockViewImpl

		BeforeEach(func() {
			mockView = &MockViewImpl{
				name:        "test-view",
				description: "Test View",
			}
			viewManager.RegisterView("test", mockView)
		})

		It("should forward resize events to all views", func() {
			viewManager.HandleResize(100, 30)
			
			Expect(mockView.lastResizeWidth).To(Equal(100))
			Expect(mockView.lastResizeHeight).To(Equal(30))
		})

		It("should handle resize when menu is open", func() {
			viewManager.ToggleMenu()
			
			// Resize should still work
			Expect(func() {
				viewManager.HandleResize(100, 30)
			}).NotTo(Panic())
		})
	})

	Describe("Rendering", func() {
		It("should render with no views", func() {
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 24}
			
			// Should not panic
			Expect(func() {
				viewManager.Render(screen, area)
			}).NotTo(Panic())
		})

		It("should render current view", func() {
			mockView := &MockViewImpl{
				name:        "test",
				description: "Test View",
			}
			viewManager.RegisterView("test", mockView)
			
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 24}
			viewManager.Render(screen, area)
			
			Expect(mockView.renderCalled).To(BeTrue())
		})

		It("should render menu overlay correctly", func() {
			// Register views and show menu
			viewManager.RegisterView("test1", &MockViewImpl{name: "test1", description: "Test 1"})
			viewManager.RegisterView("test2", &MockViewImpl{name: "test2", description: "Test 2"})
			viewManager.ToggleMenu()
			
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 24}
			viewManager.Render(screen, area)
			
			content := screen.CaptureContent()
			// Menu should be centered and visible
			Expect(content).To(ContainSubstring("Test 1"))
			Expect(content).To(ContainSubstring("Test 2"))
			// Should have menu borders
			Expect(content).To(ContainSubstring("┌"))
			Expect(content).To(ContainSubstring("┘"))
		})

		It("should handle small screen sizes", func() {
			screen.SetSize(20, 10)
			area := tui.Rect{X: 0, Y: 0, Width: 20, Height: 10}
			
			viewManager.RegisterView("test", &MockViewImpl{name: "test", description: "Test"})
			viewManager.ToggleMenu()
			
			// Should not panic
			Expect(func() {
				viewManager.Render(screen, area)
			}).NotTo(Panic())
		})
	})

	Describe("GetCurrentView", func() {
		It("should return nil when no views registered", func() {
			view := viewManager.GetCurrentView()
			Expect(view).To(BeNil())
		})

		It("should return current view when views are registered", func() {
			mockView := &MockViewImpl{
				name:        "test",
				description: "Test View",
			}
			viewManager.RegisterView("test", mockView)
			
			view := viewManager.GetCurrentView()
			Expect(view).To(Equal(mockView))
		})
	})
})

// MockViewImpl implements the View interface for testing
// Using a different name to avoid conflicts with menu_test.go
type MockViewImpl struct {
	name             string
	description      string
	renderCalled     bool
	lastKeyEvent     *tcell.EventKey
	lastResizeWidth  int
	lastResizeHeight int
	state            string
}

func (mv *MockViewImpl) Name() string        { return mv.name }
func (mv *MockViewImpl) Description() string { return mv.description }

func (mv *MockViewImpl) Render(screen tcell.Screen, area tui.Rect) {
	mv.renderCalled = true
	// Optionally render something to verify it was called
	if area.Width > 0 && area.Height > 0 {
		// Use the internal renderText function
		for i, r := range mv.name + " rendered" {
			if area.X+i < area.X+area.Width {
				screen.SetContent(area.X+i, area.Y, r, nil, tcell.StyleDefault)
			}
		}
	}
}

func (mv *MockViewImpl) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool {
	mv.lastKeyEvent = ev
	return false // Don't consume events by default
}

func (mv *MockViewImpl) HandleResize(width, height int) {
	mv.lastResizeWidth = width
	mv.lastResizeHeight = height
}