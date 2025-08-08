package streaming

import (
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Manager struct {
	Registry      *Registry
	Router        *Router
	activeStreams map[string]*ActiveStream
	mu            sync.RWMutex
	program       *tea.Program // Reference to the TUI program for sending updates
}

type ActiveStream struct {
	ID         string
	SourceType string
	Buffer     strings.Builder
	StartTime  time.Time
	NodeType   string // Display element type
	Prompt     string // Store the prompt for the stream
}

func NewManager(registry *Registry) *Manager {
	return &Manager{
		Registry:      registry,
		Router:        NewRouter(registry, "ollama-main"),
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
