package core

// Pipeline wraps a Handler with processing middleware
type Pipeline struct {
	handler    Handler
	processors []Processor
	buffer     string
}

// NewPipeline creates a handler with middleware processors
func NewPipeline(handler Handler, processors ...Processor) *Pipeline {
	return &Pipeline{
		handler:    handler,
		processors: processors,
	}
}

// OnChunk processes chunk through pipeline before passing to handler
func (p *Pipeline) OnChunk(chunk []byte) error {
	processedChunk := string(chunk)
	var err error

	// Run through all processors in order
	for _, processor := range p.processors {
		processedChunk, err = processor.Process(processedChunk)
		if err != nil {
			return err
		}
	}

	p.buffer += processedChunk
	return p.handler.OnChunk([]byte(processedChunk))
}

// OnComplete processes final content through pipeline
func (p *Pipeline) OnComplete(finalContent string) error {
	// Use buffer if finalContent is empty (streaming case)
	if finalContent == "" {
		finalContent = p.buffer
	}

	processedContent := finalContent
	var err error

	// Run through all processors' completion handlers
	for _, processor := range p.processors {
		processedContent, err = processor.OnComplete(processedContent)
		if err != nil {
			return err
		}
	}

	return p.handler.OnComplete(processedContent)
}

// OnError passes error to handler
func (p *Pipeline) OnError(err error) {
	p.handler.OnError(err)
}

// Ensure Pipeline implements Handler
var _ Handler = (*Pipeline)(nil)
