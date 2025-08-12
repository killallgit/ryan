package stream

import (
	"fmt"
	"sync"
)

// RegisteredSource represents a registered streaming provider
type RegisteredSource struct {
	ID       string
	Type     string      // e.g., "ollama", "openai", "anthropic"
	Provider interface{} // The actual provider client
}

// Registry manages multiple streaming sources with thread-safe operations
type Registry struct {
	sources   map[string]*RegisteredSource
	defaultID string // Default source ID
	mu        sync.RWMutex
}

// NewRegistry creates a new registry for streaming sources
func NewRegistry() *Registry {
	return &Registry{
		sources: make(map[string]*RegisteredSource),
	}
}

// Register adds a new streaming source to the registry
func (r *Registry) Register(id string, sourceType string, provider interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if id == "" {
		return fmt.Errorf("source ID cannot be empty")
	}

	r.sources[id] = &RegisteredSource{
		ID:       id,
		Type:     sourceType,
		Provider: provider,
	}

	// Set as default if it's the first source
	if len(r.sources) == 1 {
		r.defaultID = id
	}

	return nil
}

// Get retrieves a source by ID
func (r *Registry) Get(id string) (*RegisteredSource, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	source, exists := r.sources[id]
	return source, exists
}

// GetOrDefault retrieves a source by ID, falling back to default if not found
func (r *Registry) GetOrDefault(id string) (*RegisteredSource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try to get the requested source
	if id != "" {
		if source, exists := r.sources[id]; exists {
			return source, nil
		}
	}

	// Fall back to default
	if r.defaultID != "" {
		if source, exists := r.sources[r.defaultID]; exists {
			return source, nil
		}
	}

	return nil, fmt.Errorf("no source found with ID %q and no default available", id)
}

// SetDefault sets the default source ID
func (r *Registry) SetDefault(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sources[id]; !exists {
		return fmt.Errorf("source %q not found", id)
	}

	r.defaultID = id
	return nil
}

// GetDefault returns the default source
func (r *Registry) GetDefault() (*RegisteredSource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.defaultID == "" {
		return nil, fmt.Errorf("no default source set")
	}

	source, exists := r.sources[r.defaultID]
	if !exists {
		return nil, fmt.Errorf("default source %q not found", r.defaultID)
	}

	return source, nil
}

// List returns all registered source IDs
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.sources))
	for id := range r.sources {
		ids = append(ids, id)
	}
	return ids
}

// Remove unregisters a source
func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sources[id]; !exists {
		return fmt.Errorf("source %q not found", id)
	}

	delete(r.sources, id)

	// Clear default if it was removed
	if r.defaultID == id {
		r.defaultID = ""
		// Set a new default if sources remain
		for id := range r.sources {
			r.defaultID = id
			break
		}
	}

	return nil
}

// Clear removes all sources
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sources = make(map[string]*RegisteredSource)
	r.defaultID = ""
}
