package llm

import (
	"fmt"
	"sync"
)

// ProviderRegistry manages LLM providers
type ProviderRegistry struct {
	providers       map[string]Provider
	defaultProvider string
	mu              sync.RWMutex
}

// NewRegistry creates a new provider registry
func NewRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register registers a new provider
func (r *ProviderRegistry) Register(name string, provider Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	r.providers[name] = provider

	// Set as default if it's the first provider
	if len(r.providers) == 1 {
		r.defaultProvider = name
	}

	return nil
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// List returns all registered provider names
func (r *ProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// SetDefault sets the default provider
func (r *ProviderRegistry) SetDefault(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	r.defaultProvider = name
	return nil
}

// GetDefault returns the default provider
func (r *ProviderRegistry) GetDefault() (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.defaultProvider == "" {
		return nil, fmt.Errorf("no default provider set")
	}

	provider, exists := r.providers[r.defaultProvider]
	if !exists {
		return nil, fmt.Errorf("default provider %s not found", r.defaultProvider)
	}

	return provider, nil
}
