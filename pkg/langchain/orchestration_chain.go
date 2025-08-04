package langchain

import (
	"context"
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	langchaintools "github.com/tmc/langchaingo/tools"
)

// ContextAwareOrchestrationChain orchestrates all file and conversation operations
type ContextAwareOrchestrationChain struct {
	// Core components
	llm                  llms.Model
	agent                agents.Agent
	executor             *agents.Executor
	
	// Memory systems
	fileMemory           *FileContextMemory
	conversationMemory   *BranchingConversationMemory
	
	// Processing chains
	fileProcessor        *FileProcessingChain
	fileEditor           *FileEditChain
	fileSearcher         *FileSearchChain
	vectorManager        *VectorStoreManager
	
	// Output processing
	thinkingParser       *ThinkingOutputParser
	streamingParser      *StreamingThinkingParser
	
	// Configuration
	config               *config.Config
	enableThinking       bool
	enableFileContext    bool
	enableBranching      bool
	
	log                  *logger.Logger
}

// NewContextAwareOrchestrationChain creates the master orchestration chain
func NewContextAwareOrchestrationChain(llm llms.Model, toolRegistry *tools.Registry, cfg *config.Config) (*ContextAwareOrchestrationChain, error) {
	log := logger.WithComponent("orchestration_chain")
	
	// Create branching conversation
	branchingConv := chat.NewBranchingConversation("enhanced")
	
	// Create file context memory
	baseMemory := NewConversationBuffer() // Simple base memory
	fileMemory := NewFileContextMemory(baseMemory, branchingConv)
	
	// Create conversation memory
	conversationMemory := NewBranchingConversationMemory(branchingConv, baseMemory)
	
	// Create vector store manager
	vectorManager, err := NewVectorStoreManager(llm, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating vector store manager: %w", err)
	}
	vectorManager.WithFileContextMemory(fileMemory).WithConversationMemory(conversationMemory)
	
	// Create processing chains
	fileProcessor, err := NewFileProcessingChain(llm, fileMemory, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating file processing chain: %w", err)
	}
	fileProcessor.WithVectorStore(vectorManager.vectorStore)
	
	fileEditor := NewFileEditChain(fileMemory)
	fileSearcher := NewFileSearchChain(vectorManager.embedder, fileMemory)
	fileSearcher.WithVectorStore(vectorManager.vectorStore)
	
	// Create thinking parser
	thinkingParser := NewThinkingOutputParser(cfg != nil && cfg.ShowThinking)
	streamingParser := NewStreamingThinkingParser(cfg != nil && cfg.ShowThinking)
	
	// Create orchestration chain
	chain := &ContextAwareOrchestrationChain{
		llm:                  llm,
		fileMemory:           fileMemory,
		conversationMemory:   conversationMemory,
		fileProcessor:        fileProcessor,
		fileEditor:           fileEditor,
		fileSearcher:         fileSearcher,
		vectorManager:        vectorManager,
		thinkingParser:       thinkingParser,
		streamingParser:      streamingParser,
		config:               cfg,
		enableThinking:       cfg != nil && cfg.ShowThinking,
		enableFileContext:    true,
		enableBranching:      true,
		log:                  log,
	}
	
	// Initialize agent with enhanced tools
	if err := chain.initializeEnhancedAgent(toolRegistry); err != nil {
		return nil, fmt.Errorf("initializing enhanced agent: %w", err)
	}
	
	return chain, nil
}

// initializeEnhancedAgent creates an agent with file-aware tools
func (c *ContextAwareOrchestrationChain) initializeEnhancedAgent(toolRegistry *tools.Registry) error {
	// Create enhanced tools that are context-aware
	enhancedTools := make([]langchaintools.Tool, 0)
	
	// Add original tools with enhancement
	if toolRegistry != nil {
		for _, tool := range toolRegistry.GetTools() {
			enhancedTool := &EnhancedToolAdapter{
				baseTool:      tool,
				fileMemory:    c.fileMemory,
				vectorManager: c.vectorManager,
				log:           c.log,
			}
			enhancedTools = append(enhancedTools, enhancedTool)
		}
	}
	
	// Add file-specific tools
	enhancedTools = append(enhancedTools, &FileProcessorTool{
		processor: c.fileProcessor,
		log:       c.log,
	})
	
	enhancedTools = append(enhancedTools, &FileEditorTool{
		editor: c.fileEditor,
		log:    c.log,
	})
	
	enhancedTools = append(enhancedTools, &FileSearchTool{
		searcher: c.fileSearcher,
		log:      c.log,
	})
	
	enhancedTools = append(enhancedTools, &ConversationBranchTool{
		memory: c.conversationMemory,
		log:    c.log,
	})
	
	// Create conversational agent with enhanced memory
	c.agent = agents.NewConversationalAgent(c.llm, enhancedTools,
		agents.WithMemory(c.fileMemory))
	
	// Create executor
	c.executor = agents.NewExecutor(c.agent)
	
	c.log.Info("Enhanced agent initialized", "tools_count", len(enhancedTools))
	return nil
}

// Call implements the Chain interface for orchestration
func (c *ContextAwareOrchestrationChain) Call(ctx context.Context, inputs map[string]any, options ...chains.ChainCallOption) (map[string]any, error) {
	userInput, ok := inputs["input"].(string)
	if !ok {
		return nil, fmt.Errorf("input required")
	}
	
	// Determine if this requires file context
	needsFileContext := c.shouldUseFileContext(userInput)
	
	// Build enhanced context if needed
	if needsFileContext {
		contextResult, err := c.vectorManager.CreateContextualSearch(ctx, userInput, 10)
		if err != nil {
			c.log.Error("Failed to create contextual search", "error", err)
		} else {
			// Add context to memory
			contextStr := c.vectorManager.BuildContextFromSearch(contextResult)
			inputs["context"] = contextStr
		}
	}
	
	// Process with agent
	result, err := c.processWithAgent(ctx, userInput, inputs)
	if err != nil {
		return nil, err
	}
	
	// Parse thinking blocks if present
	if output, ok := result["output"].(string); ok {
		parsed, parseErr := c.thinkingParser.Parse(output)
		if parseErr == nil {
			result["thinking"] = parsed.Thinking
			result["has_thinking"] = parsed.HasThinking
			result["tool_calls"] = parsed.ToolCalls
			result["output"] = parsed.Content
		}
	}
	
	return result, nil
}

// processWithAgent processes input using the enhanced agent
func (c *ContextAwareOrchestrationChain) processWithAgent(ctx context.Context, userInput string, inputs map[string]any) (map[string]any, error) {
	// Add file contexts to input
	if c.enableFileContext {
		relevantContexts := c.fileMemory.GetRelevantFileContexts(userInput)
		if len(relevantContexts) > 0 {
			inputs["file_contexts"] = relevantContexts
		}
	}
	
	// Execute with agent
	result, err := c.executor.Call(ctx, inputs)
	if err != nil {
		// Check if error is due to thinking blocks
		if strings.Contains(err.Error(), "unable to parse agent output") && strings.Contains(err.Error(), "<think>") {
			c.log.Warn("Agent failed due to thinking blocks, falling back to direct LLM")
			return c.fallbackToDirectLLM(ctx, userInput, inputs)
		}
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}
	
	return result, nil
}

// fallbackToDirectLLM provides direct LLM interaction when agent fails
func (c *ContextAwareOrchestrationChain) fallbackToDirectLLM(ctx context.Context, userInput string, inputs map[string]any) (map[string]any, error) {
	// Build messages with context
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, userInput),
	}
	
	// Add file context if available
	if contexts, ok := inputs["file_contexts"].([]chat.FileContext); ok && len(contexts) > 0 {
		contextStr := c.buildFileContextString(contexts)
		messages = append([]llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, contextStr),
		}, messages...)
	}
	
	// Add search context if available
	if contextStr, ok := inputs["context"].(string); ok {
		messages = append([]llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, contextStr),
		}, messages...)
	}
	
	// Generate response
	response, err := c.llm.GenerateContent(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("direct LLM generation failed: %w", err)
	}
	
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response choices available")
	}
	
	return map[string]any{
		"output": response.Choices[0].Content,
	}, nil
}

// StreamingCall provides streaming response with thinking awareness
func (c *ContextAwareOrchestrationChain) StreamingCall(ctx context.Context, userInput string, outputChan chan<- string) error {
	// Reset streaming parser
	c.streamingParser.Reset()
	
	// Build context similar to regular call
	contextResult, err := c.vectorManager.CreateContextualSearch(ctx, userInput, 5)
	if err != nil {
		c.log.Error("Failed to create contextual search for streaming", "error", err)
	}
	
	// Build messages
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, userInput),
	}
	
	if contextResult != nil {
		contextStr := c.vectorManager.BuildContextFromSearch(contextResult)
		messages = append([]llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, contextStr),
		}, messages...)
	}
	
	// Stream with thinking awareness
	_, err = c.llm.GenerateContent(ctx, messages,
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			chunkStr := string(chunk)
			
			// Process chunk through thinking parser
			content, thinking, isComplete := c.streamingParser.ProcessChunk(chunkStr)
			
			// Send appropriate content
			if c.enableThinking && thinking != "" {
				select {
				case outputChan <- fmt.Sprintf("<think>\n%s\n</think>\n", thinking):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			
			if content != "" {
				select {
				case outputChan <- content:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			
			if isComplete {
				// Finalize and save to memory
				parsed := c.streamingParser.Finalize()
				if parsed.HasThinking || parsed.Content != "" {
					// Save to conversation memory
					msg := c.thinkingParser.CreateMessage(parsed, chat.RoleAssistant)
					if err := c.saveMessageToMemory(ctx, userInput, msg); err != nil {
						c.log.Error("Failed to save streaming message to memory", "error", err)
					}
				}
			}
			
			return nil
		}),
	)
	
	return err
}

// Helper methods

func (c *ContextAwareOrchestrationChain) shouldUseFileContext(input string) bool {
	// Simple heuristics to determine if file context is needed
	fileKeywords := []string{"file", "code", "edit", "read", "write", "search", "function", "class", "import"}
	
	inputLower := strings.ToLower(input)
	for _, keyword := range fileKeywords {
		if strings.Contains(inputLower, keyword) {
			return true
		}
	}
	
	return false
}

func (c *ContextAwareOrchestrationChain) buildFileContextString(contexts []chat.FileContext) string {
	var builder strings.Builder
	builder.WriteString("## Current File Context:\n\n")
	
	for _, ctx := range contexts {
		builder.WriteString(fmt.Sprintf("### File: %s\n", ctx.Path))
		builder.WriteString(fmt.Sprintf("Last edited: %s\n", ctx.LastEdit.Format("2006-01-02 15:04:05")))
		builder.WriteString("```\n")
		
		// Truncate content if too long
		content := ctx.Content
		if len(content) > 2000 {
			content = content[:2000] + "\n... (truncated)"
		}
		
		builder.WriteString(content)
		builder.WriteString("\n```\n\n")
	}
	
	return builder.String()
}

func (c *ContextAwareOrchestrationChain) saveMessageToMemory(ctx context.Context, userInput string, assistantMsg chat.Message) error {
	// Save to file memory
	inputs := map[string]any{
		"input": userInput,
	}
	outputs := map[string]any{
		"output": assistantMsg.Content,
	}
	
	return c.fileMemory.SaveContext(ctx, inputs, outputs)
}

// BranchConversation creates a new conversation branch
func (c *ContextAwareOrchestrationChain) BranchConversation(messageID, branchName string) error {
	_, err := c.conversationMemory.branchingConv.BranchFrom(messageID, branchName)
	return err
}

// SwitchBranch switches to a different conversation branch
func (c *ContextAwareOrchestrationChain) SwitchBranch(branchID string) error {
	_, err := c.conversationMemory.branchingConv.SwitchBranch(branchID)
	return err
}

// GetCurrentBranch returns the current branch information
func (c *ContextAwareOrchestrationChain) GetCurrentBranch() chat.ConversationBranch {
	return c.conversationMemory.branchingConv.Branches[c.conversationMemory.branchingConv.CurrentBranch]
}

// GetAllBranches returns all conversation branches
func (c *ContextAwareOrchestrationChain) GetAllBranches() []chat.ConversationBranch {
	return c.conversationMemory.branchingConv.GetAllBranches()
}

// Chain interface methods
func (c *ContextAwareOrchestrationChain) GetMemory() schema.Memory {
	return c.fileMemory
}

func (c *ContextAwareOrchestrationChain) GetInputKeys() []string {
	return []string{"input"}
}

func (c *ContextAwareOrchestrationChain) GetOutputKeys() []string {
	return []string{"output", "thinking", "has_thinking", "tool_calls"}
}