package status

import (
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/killallgit/ryan/pkg/process"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

// testWrapper wraps StatusModel with immediate quit behavior for teatest
type testWrapper struct {
	StatusModel
	shouldQuit bool
}

func (m testWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.shouldQuit {
		return m, tea.Quit
	}

	// For teatest, quit immediately after initial render
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		m.shouldQuit = true
	}

	// Update the underlying status model
	updatedModel, cmd := m.StatusModel.Update(msg)
	m.StatusModel = updatedModel.(StatusModel)

	return m, cmd
}

// createInitialModel creates a status model for testing
func createInitialModel() testWrapper {
	// Set ASCII color profile for consistent test output
	lipgloss.SetColorProfile(termenv.Ascii)
	return testWrapper{
		StatusModel: NewStatusModel(),
		shouldQuit:  false,
	}
}

// TestStatusBarInactiveOutput tests the output of an inactive status bar
func TestStatusBarInactiveOutput(t *testing.T) {
	model := createInitialModel()

	// Create a test model that will terminate immediately
	tm := teatest.NewTestModel(
		t,
		model,
		teatest.WithInitialTermSize(80, 24),
	)

	// Get the final output - should be empty for inactive status
	out, err := io.ReadAll(tm.FinalOutput(t))
	require.NoError(t, err)

	// Verify with golden file
	teatest.RequireEqualOutput(t, out)
}

// TestStatusBarActiveOutput tests active status bar output
func TestStatusBarActiveOutput(t *testing.T) {
	// Create a model that's already in active state
	model := createInitialModel()
	model.isActive = true
	model.width = 80
	model.status = "Streaming"
	model.icon = "↓"
	model.timer = 5 * time.Second // Fixed timer for consistent output
	model.tokensSent = 10
	model.tokensRecv = 15

	// Create test model
	tm := teatest.NewTestModel(
		t,
		model,
		teatest.WithInitialTermSize(80, 24),
	)

	// Get the output
	out, err := io.ReadAll(tm.FinalOutput(t))
	require.NoError(t, err)

	// Verify with golden file
	teatest.RequireEqualOutput(t, out)
}

// TestStatusBarWithTokens tests status bar with token display
func TestStatusBarWithTokens(t *testing.T) {
	model := createInitialModel()
	model.isActive = true
	model.width = 80
	model.status = "Thinking"
	model.icon = "🤔"
	model.timer = 12 * time.Second // 00:12 display
	model.tokensSent = 25
	model.tokensRecv = 30

	tm := teatest.NewTestModel(
		t,
		model,
		teatest.WithInitialTermSize(80, 24),
	)

	out, err := io.ReadAll(tm.FinalOutput(t))
	require.NoError(t, err)

	teatest.RequireEqualOutput(t, out)
}

// TestStatusBarDifferentStates tests various status bar states
func TestStatusBarDifferentStates(t *testing.T) {
	tests := []struct {
		name   string
		status string
		icon   string
		state  process.State
	}{
		{
			name:   "sending",
			status: "Sending",
			icon:   "↑",
			state:  process.StateSending,
		},
		{
			name:   "receiving",
			status: "Receiving",
			icon:   "↓",
			state:  process.StateReceiving,
		},
		{
			name:   "thinking",
			status: "Thinking",
			icon:   "🤔",
			state:  process.StateThinking,
		},
		{
			name:   "tool_use",
			status: "Using tools",
			icon:   "🔨",
			state:  process.StateToolUse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createInitialModel()
			model.isActive = true
			model.width = 80
			model.status = tt.status
			model.icon = tt.icon
			model.timer = 8 * time.Second
			model.processState = tt.state

			tm := teatest.NewTestModel(
				t,
				model,
				teatest.WithInitialTermSize(80, 24),
			)

			out, err := io.ReadAll(tm.FinalOutput(t))
			require.NoError(t, err)

			teatest.RequireEqualOutput(t, out)
		})
	}
}

// TestStatusBarViewMethod tests the View method directly
func TestStatusBarViewMethod(t *testing.T) {
	// Ensure ASCII colors for consistent testing
	lipgloss.SetColorProfile(termenv.Ascii)

	t.Run("inactive_status_empty", func(t *testing.T) {
		model := NewStatusModel()
		// Model is inactive by default
		view := model.View()

		if view != "" {
			t.Errorf("Expected empty view for inactive status, got: %q", view)
		}
	})

	t.Run("zero_width_empty", func(t *testing.T) {
		model := NewStatusModel()
		model.isActive = true
		model.width = 0 // Zero width should return empty

		view := model.View()
		if view != "" {
			t.Errorf("Expected empty view for zero width, got: %q", view)
		}
	})

	t.Run("active_status_has_content", func(t *testing.T) {
		model := NewStatusModel()
		model.isActive = true
		model.width = 80
		model.status = "Test Status"
		model.icon = "🔥"

		view := model.View()
		if view == "" {
			t.Error("Expected non-empty view for active status")
		}

		// Should end with newline as per the View implementation
		if view[len(view)-1:] != "\n" {
			t.Error("Expected view to end with newline")
		}
	})
}
