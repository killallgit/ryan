package tui

import (
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
)

// TestScreen wraps SimulationScreen with additional test utilities
type TestScreen struct {
	tcell.SimulationScreen
	mu            sync.Mutex
	eventHistory  []tcell.Event
	screenHistory []string
}

// NewTestScreen creates a new test screen
func NewTestScreen() *TestScreen {
	return &TestScreen{
		SimulationScreen: tcell.NewSimulationScreen("UTF-8"),
		eventHistory:     make([]tcell.Event, 0),
		screenHistory:    make([]string, 0),
	}
}

// CaptureContent returns the current screen content as a string
func (ts *TestScreen) CaptureContent() string {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	width, height := ts.Size()
	var content strings.Builder

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ch, _, _, _ := ts.GetContent(x, y)
			if ch != 0 {
				content.WriteRune(ch)
			} else {
				content.WriteRune(' ')
			}
		}
		content.WriteRune('\n')
	}

	result := content.String()
	ts.screenHistory = append(ts.screenHistory, result)
	return result
}

// PostEventAndWait posts an event and waits for processing
func (ts *TestScreen) PostEventAndWait(ev tcell.Event, wait time.Duration) {
	ts.mu.Lock()
	ts.eventHistory = append(ts.eventHistory, ev)
	ts.mu.Unlock()

	ts.PostEvent(ev)
	time.Sleep(wait)
}

// InjectKeyAndWait injects a key press and waits
func (ts *TestScreen) InjectKeyAndWait(key tcell.Key, ch rune, mod tcell.ModMask, wait time.Duration) {
	ts.InjectKey(key, ch, mod)
	time.Sleep(wait)
}

// GetEventHistory returns all events posted to the screen
func (ts *TestScreen) GetEventHistory() []tcell.Event {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return append([]tcell.Event{}, ts.eventHistory...)
}

// GetScreenHistory returns all captured screen states
func (ts *TestScreen) GetScreenHistory() []string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return append([]string{}, ts.screenHistory...)
}

// FindInContent searches for text in the current screen content
func (ts *TestScreen) FindInContent(text string) bool {
	content := ts.CaptureContent()
	return strings.Contains(content, text)
}

// GetRegion extracts content from a specific screen region
func (ts *TestScreen) GetRegion(x, y, width, height int) string {
	var content strings.Builder

	for row := y; row < y+height; row++ {
		for col := x; col < x+width; col++ {
			ch, _, _, _ := ts.GetContent(col, row)
			if ch != 0 {
				content.WriteRune(ch)
			} else {
				content.WriteRune(' ')
			}
		}
		if row < y+height-1 {
			content.WriteRune('\n')
		}
	}

	return content.String()
}

// StreamingTestHelper provides utilities for testing streaming scenarios
type StreamingTestHelper struct {
	screen *TestScreen
	Chunks chan string // Exported for test access
	Errors chan error  // Exported for test access
	done   chan struct{}
	wg     sync.WaitGroup
}

// NewStreamingTestHelper creates a new streaming test helper
func NewStreamingTestHelper(screen *TestScreen) *StreamingTestHelper {
	return &StreamingTestHelper{
		screen: screen,
		Chunks: make(chan string, 100),
		Errors: make(chan error, 10),
		done:   make(chan struct{}),
	}
}

// SimulateStreaming simulates a streaming operation
func (sth *StreamingTestHelper) SimulateStreaming(chunks []string, delays []time.Duration) {
	sth.wg.Add(1)
	go func() {
		defer sth.wg.Done()

		for i, chunk := range chunks {
			select {
			case <-sth.done:
				return
			default:
				sth.Chunks <- chunk
				if i < len(delays) {
					time.Sleep(delays[i])
				}
			}
		}
	}()
}

// Stop stops the streaming simulation
func (sth *StreamingTestHelper) Stop() {
	close(sth.done)
	sth.wg.Wait()
}

// GetNextChunk gets the next chunk from the stream
func (sth *StreamingTestHelper) GetNextChunk(timeout time.Duration) (string, bool) {
	select {
	case chunk := <-sth.Chunks:
		return chunk, true
	case <-time.After(timeout):
		return "", false
	}
}

// MockComponent provides a base for testing TUI components
type MockComponent struct {
	renderCalls    int
	lastRenderArea Rect
	mu             sync.Mutex
}

// Render implements a mock render method
func (mc *MockComponent) Render(screen tcell.Screen, area Rect) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.renderCalls++
	mc.lastRenderArea = area
}

// GetRenderCalls returns the number of times Render was called
func (mc *MockComponent) GetRenderCalls() int {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.renderCalls
}

// GetLastRenderArea returns the last area passed to Render
func (mc *MockComponent) GetLastRenderArea() Rect {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.lastRenderArea
}

// EventRecorder records events for testing
type EventRecorder struct {
	events []tcell.Event
	mu     sync.Mutex
}

// NewEventRecorder creates a new event recorder
func NewEventRecorder() *EventRecorder {
	return &EventRecorder{
		events: make([]tcell.Event, 0),
	}
}

// Record records an event
func (er *EventRecorder) Record(ev tcell.Event) {
	er.mu.Lock()
	defer er.mu.Unlock()
	er.events = append(er.events, ev)
}

// GetEvents returns all recorded events
func (er *EventRecorder) GetEvents() []tcell.Event {
	er.mu.Lock()
	defer er.mu.Unlock()
	return append([]tcell.Event{}, er.events...)
}

// CountEventType counts events of a specific type
func (er *EventRecorder) CountEventType(eventType string) int {
	er.mu.Lock()
	defer er.mu.Unlock()

	count := 0
	for _, ev := range er.events {
		switch ev.(type) {
		case *tcell.EventKey:
			if eventType == "key" {
				count++
			}
		case *tcell.EventResize:
			if eventType == "resize" {
				count++
			}
		case *tcell.EventMouse:
			if eventType == "mouse" {
				count++
			}
		}
	}
	return count
}
