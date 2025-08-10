package prompt

import (
	"fmt"
	"sync"
)

// DefaultRegistry is the global prompt template registry
var DefaultRegistry = NewRegistry()

// registry is a concrete implementation of the Registry interface
type registry struct {
	mu        sync.RWMutex
	templates map[string]Template
}

// NewRegistry creates a new prompt template registry
func NewRegistry() Registry {
	return &registry{
		templates: make(map[string]Template),
	}
}

// Register registers a template with a name
func (r *registry) Register(name string, template Template) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.templates[name]; exists {
		return fmt.Errorf("template %s already registered", name)
	}

	r.templates[name] = template
	return nil
}

// Get retrieves a template by name
func (r *registry) Get(name string) (Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	template, exists := r.templates[name]
	if !exists {
		return nil, fmt.Errorf("template %s not found", name)
	}

	return template, nil
}

// List returns all registered template names
func (r *registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}

	return names
}

// Clear removes all registered templates
func (r *registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.templates = make(map[string]Template)
}

// MustRegister registers a template and panics if it fails
func MustRegister(name string, template Template) {
	if err := DefaultRegistry.Register(name, template); err != nil {
		panic(fmt.Sprintf("failed to register template %s: %v", name, err))
	}
}

// MustGet retrieves a template and panics if not found
func MustGet(name string) Template {
	template, err := DefaultRegistry.Get(name)
	if err != nil {
		panic(fmt.Sprintf("template %s not found", name))
	}
	return template
}
