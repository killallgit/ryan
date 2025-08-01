package tui

import (
	"fmt"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
)

func TestMenuComponent_Render_SafeRunes(t *testing.T) {
	tests := []struct {
		name        string
		description string
		area        Rect
		expectPanic bool
	}{
		{
			name:        "Normal ASCII text",
			description: "Chat with AI",
			area:        Rect{X: 0, Y: 0, Width: 50, Height: 10},
			expectPanic: false,
		},
		{
			name:        "Unicode text with emojis",
			description: "🚀 Model Management 🤖",
			area:        Rect{X: 0, Y: 0, Width: 50, Height: 10},
			expectPanic: false,
		},
		{
			name:        "Very small area",
			description: "Test",
			area:        Rect{X: 0, Y: 0, Width: 5, Height: 3},
			expectPanic: false,
		},
		{
			name:        "Zero width area",
			description: "Test",
			area:        Rect{X: 0, Y: 0, Width: 0, Height: 10},
			expectPanic: false,
		},
		{
			name:        "Negative area",
			description: "Test",  
			area:        Rect{X: 0, Y: 0, Width: -5, Height: -3},
			expectPanic: false,
		},
		{
			name:        "Long text with Unicode",
			description: "This is a very long description with émojis 🌟 and spëcial chars",
			area:        Rect{X: 0, Y: 0, Width: 30, Height: 10},
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
			
			menu := NewMenuComponent().WithOption("test", tt.description)
			
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
		menu := NewMenuComponent()
		area := Rect{X: 0, Y: 0, Width: 50, Height: 10}
		
		assert.NotPanics(t, func() {
			menu.Render(screen, area)
		})
	})
	
	t.Run("Menu with many options", func(t *testing.T) {
		menu := NewMenuComponent()
		for i := 0; i < 20; i++ {
			menu = menu.WithOption("option"+string(rune('0'+i)), "Description "+string(rune('0'+i)))
		}
		
		area := Rect{X: 0, Y: 0, Width: 50, Height: 10}
		
		assert.NotPanics(t, func() {
			menu.Render(screen, area)
		})
	})
	
	t.Run("Unicode in option names and descriptions", func(t *testing.T) {
		menu := NewMenuComponent().
			WithOption("чат", "Чат с ИИ").
			WithOption("모델", "모델 관리").
			WithOption("設定", "設定管理")
		
		area := Rect{X: 0, Y: 0, Width: 50, Height: 10}
		
		assert.NotPanics(t, func() {
			menu.Render(screen, area)
		})
	})
}

func TestMenuComponent_SafeStringHandling(t *testing.T) {
	// Test our string truncation function
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"🚀🤖", 5, "🚀🤖"},
		{"", 5, ""},
		{"test", 0, ""},
		{"test", 3, ""},  // Should handle edge case
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.LessOrEqual(t, len(result), tt.maxLen)
		})
	}
}

func TestViewManager_RenderMenu_SafeDimensions(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	assert.NoError(t, err)
	defer screen.Fini()
	
	screen.SetSize(100, 50)
	
	vm := NewViewManager()
	
	// Create a mock view
	mockView := &MockView{
		name:        "test",
		description: "Test View with 🚀 emojis",
	}
	vm.RegisterView("test", mockView)
	vm.ToggleMenu()
	
	tests := []struct {
		name string
		area Rect
	}{
		{"Normal area", Rect{X: 0, Y: 0, Width: 80, Height: 24}},
		{"Small area", Rect{X: 0, Y: 0, Width: 10, Height: 5}},
		{"Zero width", Rect{X: 0, Y: 0, Width: 0, Height: 24}},
		{"Zero height", Rect{X: 0, Y: 0, Width: 80, Height: 0}},
		{"Negative dimensions", Rect{X: 0, Y: 0, Width: -10, Height: -5}},
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
	menu := NewMenuComponent().WithOption("test", "Chat with AI")
	
	// These area dimensions were causing the panic
	problematicAreas := []Rect{
		{X: 0, Y: 0, Width: 50, Height: 10},   // Original failing case
		{X: 5, Y: 5, Width: 40, Height: 8},    // Offset position
		{X: 10, Y: 3, Width: 20, Height: 6},   // Small dimensions
		{X: 0, Y: 0, Width: 4, Height: 4},     // At boundary
		{X: 0, Y: 0, Width: 3, Height: 3},     // Below boundary (should be safe)
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

func (mv *MockView) Name() string { return mv.name }
func (mv *MockView) Description() string { return mv.description }
func (mv *MockView) Render(screen tcell.Screen, area Rect) {}
func (mv *MockView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool { return false }
func (mv *MockView) HandleResize(width, height int) {}