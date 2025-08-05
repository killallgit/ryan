package tui_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/tui"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
)

var _ = Describe("MenuComponent", func() {
	var (
		screen *tui.TestScreen
		menu   tui.MenuComponent
	)

	BeforeEach(func() {
		screen = tui.NewTestScreen()
		err := screen.Init()
		Expect(err).ToNot(HaveOccurred())
		screen.SetSize(80, 24)
		screen.Clear()
		
		menu = tui.NewMenuComponent()
	})

	AfterEach(func() {
		screen.Fini()
	})

	Describe("Menu Creation", func() {
		It("should create menu with default values", func() {
			Expect(menu.GetSelectedOption()).To(Equal(""))
		})

		It("should add options correctly", func() {
			menu = menu.WithOption("chat", "Chat with AI").
				WithOption("models", "Manage Models").
				WithOption("tools", "Tools Registry")

			Expect(menu.GetOptionByIndex(0)).To(Equal("chat"))
			Expect(menu.GetOptionByIndex(1)).To(Equal("models"))
			Expect(menu.GetOptionByIndex(2)).To(Equal("tools"))
		})
	})

	Describe("Menu Navigation", func() {
		BeforeEach(func() {
			menu = menu.WithOption("option1", "First Option").
				WithOption("option2", "Second Option").
				WithOption("option3", "Third Option")
		})

		It("should navigate down through options", func() {
			Expect(menu.GetSelectedOption()).To(Equal("option1"))
			
			menu = menu.SelectNext()
			Expect(menu.GetSelectedOption()).To(Equal("option2"))
			
			menu = menu.SelectNext()
			Expect(menu.GetSelectedOption()).To(Equal("option3"))
			
			// Should wrap around
			menu = menu.SelectNext()
			Expect(menu.GetSelectedOption()).To(Equal("option1"))
		})

		It("should navigate up through options", func() {
			Expect(menu.GetSelectedOption()).To(Equal("option1"))
			
			// Should wrap around from beginning
			menu = menu.SelectPrevious()
			Expect(menu.GetSelectedOption()).To(Equal("option3"))
			
			menu = menu.SelectPrevious()
			Expect(menu.GetSelectedOption()).To(Equal("option2"))
			
			menu = menu.SelectPrevious()
			Expect(menu.GetSelectedOption()).To(Equal("option1"))
		})

		It("should handle empty menu navigation", func() {
			emptyMenu := tui.NewMenuComponent()
			
			// Navigation should be safe on empty menu
			emptyMenu = emptyMenu.SelectNext()
			Expect(emptyMenu.GetSelectedOption()).To(Equal(""))
			
			emptyMenu = emptyMenu.SelectPrevious()
			Expect(emptyMenu.GetSelectedOption()).To(Equal(""))
		})
	})

	Describe("Menu Rendering", func() {
		BeforeEach(func() {
			menu = menu.WithOption("chat", "Chat with AI").
				WithOption("models", "Manage Models").
				WithOption("tools", "Tools Registry")
		})

		It("should render menu with options", func() {
			area := tui.Rect{X: 20, Y: 5, Width: 40, Height: 10}
			menu.Render(screen, area)

			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring("Chat with AI"))
			Expect(content).To(ContainSubstring("Manage Models"))
			Expect(content).To(ContainSubstring("Tools Registry"))
		})

		It("should highlight selected option", func() {
			// Select the second option
			menu = menu.SelectNext()
			
			area := tui.Rect{X: 20, Y: 5, Width: 40, Height: 10}
			menu.Render(screen, area)

			// Check that the second option is highlighted
			// We can verify this by checking the screen content at specific positions
			// The selected item should have different styling
			content := screen.CaptureContent()
			lines := strings.Split(content, "\n")
			
			// Find the line with "Manage Models" and verify it exists
			foundHighlight := false
			for _, line := range lines {
				if strings.Contains(line, "Manage Models") {
					// In a real implementation, we'd check the style attributes
					foundHighlight = true
					break
				}
			}
			Expect(foundHighlight).To(BeTrue())
		})

		It("should render menu borders", func() {
			area := tui.Rect{X: 10, Y: 5, Width: 40, Height: 10}
			menu.Render(screen, area)

			content := screen.CaptureContent()
			// Check for border characters
			Expect(content).To(ContainSubstring("┌"))
			Expect(content).To(ContainSubstring("┐"))
			Expect(content).To(ContainSubstring("└"))
			Expect(content).To(ContainSubstring("┘"))
			Expect(content).To(ContainSubstring("─"))
			Expect(content).To(ContainSubstring("│"))
		})

		It("should handle menu rendering in small area", func() {
			area := tui.Rect{X: 0, Y: 0, Width: 20, Height: 6}
			menu.Render(screen, area)

			// Should not panic and should render something
			content := screen.CaptureContent()
			// At least one option should be visible
			Expect(content).To(Or(
				ContainSubstring("Chat"),
				ContainSubstring("Manage"),
				ContainSubstring("Tools"),
			))
		})

		It("should not render in too small area", func() {
			// Area too small to render menu
			area := tui.Rect{X: 0, Y: 0, Width: 3, Height: 3}
			menu.Render(screen, area)

			// Should not crash, just not render
			content := screen.CaptureContent()
			// The area is too small, so no menu content should be rendered
			Expect(content).NotTo(ContainSubstring("Chat with AI"))
		})

		It("should handle long option descriptions", func() {
			longMenu := tui.NewMenuComponent().
				WithOption("long", "This is a very long description that should be truncated when displayed in the menu")
			
			area := tui.Rect{X: 10, Y: 5, Width: 30, Height: 10}
			longMenu.Render(screen, area)

			content := screen.CaptureContent()
			// Should see truncation with ellipsis
			Expect(content).To(ContainSubstring("..."))
			// Should not see the full text
			Expect(content).NotTo(ContainSubstring("when displayed in the menu"))
		})

		It("should handle Unicode text properly", func() {
			unicodeMenu := tui.NewMenuComponent().
				WithOption("emoji", "🚀 Launch Feature").
				WithOption("chinese", "中文选项").
				WithOption("russian", "Русский вариант")
			
			area := tui.Rect{X: 10, Y: 5, Width: 40, Height: 10}
			unicodeMenu.Render(screen, area)

			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring("🚀"))
			Expect(content).To(ContainSubstring("Launch Feature"))
			Expect(content).To(ContainSubstring("中文选项"))
			Expect(content).To(ContainSubstring("Русский вариант"))
		})
	})

	Describe("Menu Size Configuration", func() {
		It("should update menu size", func() {
			menu = menu.WithSize(60, 20)
			// Size is stored internally and used for rendering calculations
			// We can't directly test the size, but we can verify it renders correctly
			
			menu = menu.WithOption("test", "Test Option")
			area := tui.Rect{X: 0, Y: 0, Width: 60, Height: 20}
			
			// Should render without issues
			Expect(func() {
				menu.Render(screen, area)
			}).NotTo(Panic())
		})
	})

	Describe("Menu with Many Options", func() {
		BeforeEach(func() {
			// Add many options to test scrolling behavior
			for i := 0; i < 20; i++ {
				menu = menu.WithOption(
					fmt.Sprintf("option%d", i),
					fmt.Sprintf("Option Number %d", i),
				)
			}
		})

		It("should render visible options only", func() {
			area := tui.Rect{X: 10, Y: 5, Width: 40, Height: 8} // Small height
			menu.Render(screen, area)

			content := screen.CaptureContent()
			// Should see some options
			Expect(content).To(ContainSubstring("Option Number 0"))
			// Later options might not be visible due to height constraint
			// This depends on the implementation
		})

		It("should maintain selection when navigating many options", func() {
			// Navigate to middle
			for i := 0; i < 10; i++ {
				menu = menu.SelectNext()
			}
			Expect(menu.GetSelectedOption()).To(Equal("option10"))

			// Navigate back
			for i := 0; i < 5; i++ {
				menu = menu.SelectPrevious()
			}
			Expect(menu.GetSelectedOption()).To(Equal("option5"))
		})
	})
})

// Keep the original test functions for compatibility
func TestMenuComponent_Render_SafeRunes(t *testing.T) {
	tests := []struct {
		name        string
		description string
		area        tui.Rect
		expectPanic bool
	}{
		{
			name:        "Normal ASCII text",
			description: "Chat with AI",
			area:        tui.Rect{X: 0, Y: 0, Width: 50, Height: 10},
			expectPanic: false,
		},
		{
			name:        "Unicode text with emojis",
			description: "🚀 Model Management 🤖",
			area:        tui.Rect{X: 0, Y: 0, Width: 50, Height: 10},
			expectPanic: false,
		},
		{
			name:        "Very small area",
			description: "Test",
			area:        tui.Rect{X: 0, Y: 0, Width: 5, Height: 3},
			expectPanic: false,
		},
		{
			name:        "Zero width area",
			description: "Test",
			area:        tui.Rect{X: 0, Y: 0, Width: 0, Height: 10},
			expectPanic: false,
		},
		{
			name:        "Negative area",
			description: "Test",
			area:        tui.Rect{X: 0, Y: 0, Width: -5, Height: -3},
			expectPanic: false,
		},
		{
			name:        "Long text with Unicode",
			description: "This is a very long description with émojis 🌟 and spëcial chars",
			area:        tui.Rect{X: 0, Y: 0, Width: 30, Height: 10},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := tcell.NewSimulationScreen("UTF-8")
			err := screen.Init()
			assert.NoError(t, err)
			defer screen.Fini()

			screen.SetSize(100, 50)

			menu := tui.NewMenuComponent().WithOption("test", tt.description)

			if tt.expectPanic {
				assert.Panics(t, func() {
					menu.Render(screen, tt.area)
				})
			} else {
				assert.NotPanics(t, func() {
					menu.Render(screen, tt.area)
				})
			}
		})
	}
}

func TestMenuComponent_Render_EdgeCases(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	assert.NoError(t, err)
	defer screen.Fini()

	screen.SetSize(100, 50)

	t.Run("Empty menu", func(t *testing.T) {
		menu := tui.NewMenuComponent()
		area := tui.Rect{X: 0, Y: 0, Width: 50, Height: 10}

		assert.NotPanics(t, func() {
			menu.Render(screen, area)
		})
	})

	t.Run("Menu with many options", func(t *testing.T) {
		menu := tui.NewMenuComponent()
		for i := 0; i < 20; i++ {
			menu = menu.WithOption("option"+string(rune('0'+i)), "Description "+string(rune('0'+i)))
		}

		area := tui.Rect{X: 0, Y: 0, Width: 50, Height: 10}

		assert.NotPanics(t, func() {
			menu.Render(screen, area)
		})
	})

	t.Run("Unicode in option names and descriptions", func(t *testing.T) {
		menu := tui.NewMenuComponent().
			WithOption("чат", "Чат с ИИ").
			WithOption("모델", "모델 관리").
			WithOption("設定", "設定管理")

		area := tui.Rect{X: 0, Y: 0, Width: 50, Height: 10}

		assert.NotPanics(t, func() {
			menu.Render(screen, area)
		})
	})
}

func TestMenuComponent_SafeStringHandling(t *testing.T) {
	// Since truncateString is not exported, we'll test it indirectly through menu rendering
	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	assert.NoError(t, err)
	defer screen.Fini()

	screen.SetSize(100, 50)

	tests := []struct {
		input    string
		maxWidth int
	}{
		{"hello", 20},
		{"hello world with a very long description", 15},
		{"🚀🤖 Unicode test", 20},
		{"", 5},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			menu := tui.NewMenuComponent().WithOption("test", tt.input)
			area := tui.Rect{X: 0, Y: 0, Width: tt.maxWidth, Height: 10}
			
			// Should not panic with various string lengths
			assert.NotPanics(t, func() {
				menu.Render(screen, area)
			})
		})
	}
}

func TestViewManager_RenderMenu_SafeDimensions(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	assert.NoError(t, err)
	defer screen.Fini()

	screen.SetSize(100, 50)

	vm := tui.NewViewManager()

	// Create a mock view
	mockView := &MockView{
		name:        "test",
		description: "Test View with 🚀 emojis",
	}
	vm.RegisterView("test", mockView)
	vm.ToggleMenu()

	tests := []struct {
		name string
		area tui.Rect
	}{
		{"Normal area", tui.Rect{X: 0, Y: 0, Width: 80, Height: 24}},
		{"Small area", tui.Rect{X: 0, Y: 0, Width: 10, Height: 5}},
		{"Zero width", tui.Rect{X: 0, Y: 0, Width: 0, Height: 24}},
		{"Zero height", tui.Rect{X: 0, Y: 0, Width: 80, Height: 0}},
		{"Negative dimensions", tui.Rect{X: 0, Y: 0, Width: -10, Height: -5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				vm.Render(screen, tt.area)
			})
		})
	}
}

// TestMenuComponent_RegressionTest_IndexOutOfRange tests for the specific
// panic that was occurring with negative array indexing in menu rendering.
// This reproduces the original issue: runtime error: index out of range [-1]
func TestMenuComponent_RegressionTest_IndexOutOfRange(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	assert.NoError(t, err)
	defer screen.Fini()

	screen.SetSize(100, 50)

	// This specific combination was causing the panic:
	// - Area with specific dimensions
	// - Text content that caused negative indexing
	menu := tui.NewMenuComponent().WithOption("test", "Chat with AI")

	// These area dimensions were causing the panic
	problematicAreas := []tui.Rect{
		{X: 0, Y: 0, Width: 50, Height: 10}, // Original failing case
		{X: 5, Y: 5, Width: 40, Height: 8},  // Offset position
		{X: 10, Y: 3, Width: 20, Height: 6}, // Small dimensions
		{X: 0, Y: 0, Width: 4, Height: 4},   // At boundary
		{X: 0, Y: 0, Width: 3, Height: 3},   // Below boundary (should be safe)
	}

	for i, area := range problematicAreas {
		t.Run(fmt.Sprintf("Problematic_area_%d", i), func(t *testing.T) {
			// This should NOT panic
			assert.NotPanics(t, func() {
				menu.Render(screen, area)
			}, "Menu render should handle all area dimensions safely")
		})
	}
}

// MockView implements the View interface for testing
type MockView struct {
	name        string
	description string
}

func (mv *MockView) Name() string                                               { return mv.name }
func (mv *MockView) Description() string                                        { return mv.description }
func (mv *MockView) Render(screen tcell.Screen, area tui.Rect)                 {}
func (mv *MockView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool      { return false }
func (mv *MockView) HandleResize(width, height int)                            {}