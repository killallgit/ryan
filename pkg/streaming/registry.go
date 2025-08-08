package streaming

import "sync"

type StreamSource struct {
	ID       string
	Type     string      // "ollama", "openai", "agent", etc.
	Provider interface{} // The actual LLM client
}

type Registry struct {
	sources map[string]*StreamSource
	mu      sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		sources: make(map[string]*StreamSource),
	}
}

func (r *Registry) Register(id string, sourceType string, provider interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources[id] = &StreamSource{
		ID:       id,
		Type:     sourceType,
		Provider: provider,
	}
}

func (r *Registry) Get(id string) (*StreamSource, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	source, exists := r.sources[id]
	return source, exists
}

func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sources, id)
}
