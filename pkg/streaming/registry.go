package streaming

type StreamSource struct {
	ID       string
	Type     string      // Always "ollama" now
	Provider interface{} // The Ollama client
}

type Registry struct {
	source *StreamSource
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Register(id string, sourceType string, provider interface{}) {
	// Now we only keep a single source
	r.source = &StreamSource{
		ID:       id,
		Type:     sourceType,
		Provider: provider,
	}
}

func (r *Registry) Get(id string) (*StreamSource, bool) {
	// Return the single source regardless of ID for backwards compatibility
	if r.source != nil {
		return r.source, true
	}
	return nil, false
}

func (r *Registry) GetSource() *StreamSource {
	return r.source
}
