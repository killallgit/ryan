package langchain

import (
	"context"

	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/schema"
)

// SimpleConversationBuffer provides a basic conversation buffer
type SimpleConversationBuffer struct {
	buffer *memory.ConversationBuffer
}

// NewConversationBuffer creates a new simple conversation buffer
func NewConversationBuffer() schema.Memory {
	return &SimpleConversationBuffer{
		buffer: memory.NewConversationBuffer(),
	}
}

// MemoryVariables returns the memory variables
func (scb *SimpleConversationBuffer) MemoryVariables(ctx context.Context) []string {
	return []string{"history"}
}

// LoadMemoryVariables loads the memory variables
func (scb *SimpleConversationBuffer) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	return scb.buffer.LoadMemoryVariables(ctx, inputs)
}

// SaveContext saves the context to memory
func (scb *SimpleConversationBuffer) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
	return scb.buffer.SaveContext(ctx, inputs, outputs)
}

// Clear clears the memory
func (scb *SimpleConversationBuffer) Clear(ctx context.Context) error {
	return scb.buffer.Clear(ctx)
}

// GetMemoryKey returns the memory key
func (scb *SimpleConversationBuffer) GetMemoryKey(ctx context.Context) string {
	return "history"
}