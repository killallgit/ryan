package chat

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/schema"
)

// LangChainMemory wraps LangChain Go's memory system and provides compatibility with our Conversation type
type LangChainMemory struct {
	buffer       *memory.ConversationBuffer
	conversation *Conversation
}

// NewLangChainMemory creates a new LangChain memory wrapper
func NewLangChainMemory() *LangChainMemory {
	emptyConv := NewConversation("")
	return &LangChainMemory{
		buffer:       memory.NewConversationBuffer(),
		conversation: &emptyConv,
	}
}

// NewLangChainMemoryWithConversation creates a new LangChain memory wrapper with existing conversation
func NewLangChainMemoryWithConversation(conv Conversation) (*LangChainMemory, error) {
	lm := &LangChainMemory{
		buffer:       memory.NewConversationBuffer(),
		conversation: &conv,
	}

	// Convert existing messages to LangChain format
	ctx := context.Background()
	messages := GetMessages(conv)
	for _, msg := range messages {
		if err := lm.addMessageToBuffer(ctx, msg); err != nil {
			return nil, fmt.Errorf("failed to convert message to LangChain format: %w", err)
		}
	}

	return lm, nil
}

// addMessageToBuffer adds a message to the LangChain buffer
func (lm *LangChainMemory) addMessageToBuffer(ctx context.Context, msg Message) error {
	switch msg.Role {
	case RoleUser:
		return lm.buffer.ChatHistory.AddUserMessage(ctx, msg.Content)
	case RoleAssistant:
		return lm.buffer.ChatHistory.AddAIMessage(ctx, msg.Content)
	case RoleSystem:
		// LangChain doesn't have a direct system message in ConversationBuffer
		// We'll store it as a special formatted message
		return lm.buffer.ChatHistory.AddMessage(ctx, llms.SystemChatMessage{
			Content: msg.Content,
		})
	case RoleTool:
		// Tool messages can be stored as generic messages
		content := fmt.Sprintf("Tool (%s): %s", msg.ToolName, msg.Content)
		return lm.buffer.ChatHistory.AddMessage(ctx, llms.GenericChatMessage{
			Role:    "tool",
			Content: content,
		})
	case RoleError:
		// Error messages stored as generic
		return lm.buffer.ChatHistory.AddMessage(ctx, llms.GenericChatMessage{
			Role:    "error",
			Content: msg.Content,
		})
	default:
		return fmt.Errorf("unknown message role: %s", msg.Role)
	}
}

// AddMessage adds a message to both conversation and LangChain memory
func (lm *LangChainMemory) AddMessage(ctx context.Context, msg Message) error {
	// Add to our conversation
	newConv := AddMessage(*lm.conversation, msg)
	lm.conversation = &newConv

	// Add to LangChain buffer
	return lm.addMessageToBuffer(ctx, msg)
}

// GetConversation returns the current conversation
func (lm *LangChainMemory) GetConversation() Conversation {
	return *lm.conversation
}

// GetBuffer returns the underlying LangChain memory buffer
func (lm *LangChainMemory) GetBuffer() *memory.ConversationBuffer {
	return lm.buffer
}

// GetMemoryVariables returns the memory variables (for LangChain chains)
func (lm *LangChainMemory) GetMemoryVariables(ctx context.Context) (map[string]any, error) {
	return lm.buffer.LoadMemoryVariables(ctx, map[string]any{})
}

// SaveContext saves the context of the conversation (for LangChain chains)
func (lm *LangChainMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	// Save to LangChain buffer
	if err := lm.buffer.SaveContext(ctx, inputs, outputs); err != nil {
		return err
	}

	// Also add messages to our conversation for consistency
	if input, ok := inputs["input"].(string); ok && input != "" {
		userMsg := NewUserMessage(input)
		*lm.conversation = AddMessage(*lm.conversation, userMsg)
	}

	if output, ok := outputs["output"].(string); ok && output != "" {
		assistantMsg := NewAssistantMessage(output)
		*lm.conversation = AddMessage(*lm.conversation, assistantMsg)
	}

	return nil
}

// Clear clears both the conversation and LangChain memory
func (lm *LangChainMemory) Clear(ctx context.Context) error {
	// Clear our conversation
	clearedConv := NewConversation(lm.conversation.Model)
	lm.conversation = &clearedConv

	// Clear LangChain buffer
	return lm.buffer.Clear(ctx)
}

// ConvertToLangChainMessages converts our messages to LangChain message format
func ConvertToLangChainMessages(messages []Message) []llms.ChatMessage {
	result := make([]llms.ChatMessage, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			result = append(result, llms.HumanChatMessage{Content: msg.Content})
		case RoleAssistant:
			result = append(result, llms.AIChatMessage{Content: msg.Content})
		case RoleSystem:
			result = append(result, llms.SystemChatMessage{Content: msg.Content})
		case RoleTool:
			result = append(result, llms.GenericChatMessage{
				Role:    "tool",
				Content: fmt.Sprintf("Tool (%s): %s", msg.ToolName, msg.Content),
			})
		case RoleError:
			result = append(result, llms.GenericChatMessage{
				Role:    "error",
				Content: msg.Content,
			})
		}
	}

	return result
}

// ConvertFromLangChainMessages converts LangChain messages to our message format
func ConvertFromLangChainMessages(messages []llms.ChatMessage) []Message {
	result := make([]Message, 0, len(messages))

	for _, msg := range messages {
		switch m := msg.(type) {
		case llms.HumanChatMessage:
			result = append(result, NewUserMessage(m.Content))
		case llms.AIChatMessage:
			result = append(result, NewAssistantMessage(m.Content))
		case llms.SystemChatMessage:
			result = append(result, NewSystemMessage(m.Content))
		case llms.GenericChatMessage:
			switch m.Role {
			case "tool":
				result = append(result, NewToolResultMessage("", m.Content))
			case "error":
				result = append(result, NewErrorMessage(m.Content))
			default:
				// Fallback to assistant message
				result = append(result, NewAssistantMessage(m.Content))
			}
		}
	}

	return result
}

// MemoryAdapter allows using LangChainMemory as schema.Memory
type MemoryAdapter struct {
	*LangChainMemory
}

// Ensure MemoryAdapter implements schema.Memory
var _ schema.Memory = (*MemoryAdapter)(nil)

// GetMemoryKey returns the memory key
func (ma *MemoryAdapter) GetMemoryKey(ctx context.Context) string {
	return "history"
}

// MemoryVariables returns the memory variables
func (ma *MemoryAdapter) MemoryVariables(ctx context.Context) []string {
	return []string{"history"}
}

// LoadMemoryVariables loads the memory variables
func (ma *MemoryAdapter) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	return ma.LangChainMemory.GetMemoryVariables(ctx)
}

// SaveContext saves the context
func (ma *MemoryAdapter) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	return ma.LangChainMemory.SaveContext(ctx, inputs, outputs)
}

// Clear clears the memory
func (ma *MemoryAdapter) Clear(ctx context.Context) error {
	return ma.LangChainMemory.Clear(ctx)
}
