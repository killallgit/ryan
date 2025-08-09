package streaming

import (
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Manager struct {
	Registry      *Registry
	activeStreams map[string]*ActiveStream
	mu            sync.RWMutex
	program       *tea.Program // Reference to the TUI program for sending updates
}

type ActiveStream struct {
	ID         string
	SourceType string
	Buffer     strings.Builder
	StartTime  time.Time
	NodeType   string       // Display element type
	Prompt     string       // Store the prompt for the stream
	Callback   func(string) // Callback for chunk notifications
	LastSent   string       // Track last sent content for incremental updates
	Error      error        // Store any error that occurred during streaming
}

func NewManager(registry *Registry) *Manager {
	return &Manager{
		Registry:      registry,
		activeStreams: make(map[string]*ActiveStream),
	}
}

func (m *Manager) StartStream(streamID, sourceType, nodeType, prompt string) *ActiveStream {
	m.mu.Lock()
	defer m.mu.Unlock()

	stream := &ActiveStream{
		ID:         streamID,
		SourceType: sourceType,
		StartTime:  time.Now(),
		NodeType:   nodeType,
		Prompt:     prompt,
		Callback:   func(string) {}, // Set a dummy callback to indicate stream is active
	}
	m.activeStreams[streamID] = stream
	return stream
}

func (m *Manager) GetStream(streamID string) (*ActiveStream, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stream, exists := m.activeStreams[streamID]
	return stream, exists
}

func (m *Manager) EndStream(streamID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.activeStreams, streamID)
}

func (m *Manager) AppendToStream(streamID string, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stream, exists := m.activeStreams[streamID]; exists {
		stream.Buffer.WriteString(content)
		// Call callback if set
		if stream.Callback != nil {
			stream.Callback(content)
		}
	}
}

// SetProgram sets the tea.Program reference for sending updates
func (m *Manager) SetProgram(program *tea.Program) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.program = program
}

// GetProgram returns the tea.Program reference
func (m *Manager) GetProgram() *tea.Program {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.program
}

// SetStreamCallback sets a callback for when chunks are received
func (m *Manager) SetStreamCallback(streamID string, callback func(string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stream, exists := m.activeStreams[streamID]; exists {
		stream.Callback = callback
	}
}

// GetLastSentContent gets the last sent content for a stream
func (m *Manager) GetLastSentContent(streamID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if stream, exists := m.activeStreams[streamID]; exists {
		return stream.LastSent
	}
	return ""
}

// SetLastSentContent sets the last sent content for a stream
func (m *Manager) SetLastSentContent(streamID string, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stream, exists := m.activeStreams[streamID]; exists {
		stream.LastSent = content
	}
}

// SetStreamError sets an error for a stream
func (m *Manager) SetStreamError(streamID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stream, exists := m.activeStreams[streamID]; exists {
		stream.Error = err
	}
}

// GetStreamError gets the error for a stream
func (m *Manager) GetStreamError(streamID string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if stream, exists := m.activeStreams[streamID]; exists {
		return stream.Error
	}
	return nil
}
