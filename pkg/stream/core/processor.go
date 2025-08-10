package core

// Processor transforms stream chunks before they reach handlers
// This enables middleware-style processing pipelines
type Processor interface {
	// Process transforms a chunk before it's passed to the next handler
	Process(chunk string) (string, error)

	// OnComplete is called when streaming completes
	OnComplete(finalContent string) (string, error)
}

// ProcessorFunc is a simple processor that applies a function to chunks
type ProcessorFunc func(chunk string) (string, error)

// Process implements Processor
func (f ProcessorFunc) Process(chunk string) (string, error) {
	return f(chunk)
}

// OnComplete implements Processor
func (f ProcessorFunc) OnComplete(finalContent string) (string, error) {
	// By default, just return the final content unchanged
	return finalContent, nil
}

// ChainProcessor chains multiple processors together
type ChainProcessor struct {
	processors []Processor
}

// NewChainProcessor creates a processor that chains multiple processors
func NewChainProcessor(processors ...Processor) *ChainProcessor {
	return &ChainProcessor{
		processors: processors,
	}
}

// Process runs chunk through all processors in sequence
func (c *ChainProcessor) Process(chunk string) (string, error) {
	result := chunk
	for _, p := range c.processors {
		var err error
		result, err = p.Process(result)
		if err != nil {
			return "", err
		}
	}
	return result, nil
}

// OnComplete runs final content through all processors
func (c *ChainProcessor) OnComplete(finalContent string) (string, error) {
	result := finalContent
	for _, p := range c.processors {
		var err error
		result, err = p.OnComplete(result)
		if err != nil {
			return "", err
		}
	}
	return result, nil
}
