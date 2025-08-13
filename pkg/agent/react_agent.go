package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/embeddings"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/memory"
	"github.com/killallgit/ryan/pkg/prompt"
	"github.com/killallgit/ryan/pkg/retrieval"
	"github.com/killallgit/ryan/pkg/stream/core"
	"github.com/killallgit/ryan/pkg/stream/providers"
	"github.com/killallgit/ryan/pkg/tokens"
	_ "github.com/killallgit/ryan/pkg/tools" // Import for init() registration
	"github.com/killallgit/ryan/pkg/tools/registry"
	"github.com/killallgit/ryan/pkg/vectorstore"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
	lcmemory "github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/tools"
)

// ReactAgent is a LangChain ReAct-based agent implementation
// It wraps LangChain's conversational agent with an executor for handling requests
type ReactAgent struct {
	llm          llms.Model
	executor     *agents.Executor
	memory       *memory.Memory
	tools        []tools.Tool
	tokenCounter *tokens.TokenCounter
	tokensSent   int
	tokensRecv   int
	tokensMu     sync.RWMutex

	// Observable execution state
	state *ExecutionState

	// RAG components
	vectorStore vectorstore.VectorStore
	retriever   *retrieval.Retriever
	augmenter   *retrieval.Augmenter

	// Prompt template for formatting inputs
	promptTemplate prompt.Template
}

// NewReactAgent creates a new executor-based agent with an injected LLM
// By default, creates a new session (no continuation)
func NewReactAgent(llm llms.Model) (*ReactAgent, error) {
	// Generate a unique session ID for new conversations
	// This ensures each agent has its own isolated memory
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	logger.Debug("Created new session ID: %s", sessionID)

	return NewReactAgentWithSession(llm, sessionID)
}

// NewReactAgentWithContinue creates a new executor-based agent with continuation support
func NewReactAgentWithContinue(llm llms.Model, continueHistory bool) (*ReactAgent, error) {
	return NewReactAgentWithOptions(llm, continueHistory, false)
}

// NewReactAgentWithOptions creates a new executor-based agent with full options
func NewReactAgentWithOptions(llm llms.Model, continueHistory, skipPermissions bool) (*ReactAgent, error) {
	var sessionID string
	if continueHistory {
		// Use a consistent session ID per project to maintain conversation context
		// This allows continuing conversations across ryan invocations
		sessionID = "default_project_session"
		logger.Debug("Using continued session ID: %s", sessionID)
	} else {
		// Generate a unique session ID for new conversations
		// This ensures each agent has its own isolated memory
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
		logger.Debug("Created new session ID: %s", sessionID)
	}

	return NewReactAgentWithSessionAndOptions(llm, sessionID, skipPermissions)
}

// NewReactAgentWithSession creates a new executor-based agent with a specific session ID
func NewReactAgentWithSession(llm llms.Model, sessionID string) (*ReactAgent, error) {
	return NewReactAgentWithSessionAndOptions(llm, sessionID, false)
}

// NewReactAgentWithSessionAndOptions creates a new executor-based agent with a specific session ID and options
func NewReactAgentWithSessionAndOptions(llm llms.Model, sessionID string, skipPermissions bool) (*ReactAgent, error) {
	logger.Info("Initializing ReactAgent with session: %s", sessionID)

	mem, err := memory.New(sessionID)
	if err != nil {
		logger.Error("Failed to create memory for session %s: %v", sessionID, err)
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}
	logger.Debug("Memory initialized for session: %s", sessionID)

	// Get configuration settings
	settings := config.Get()

	// Get enabled tools from registry based on configuration
	agentTools := registry.Global().GetEnabled(settings, skipPermissions)

	// Initialize RAG components if enabled
	var vectorStore vectorstore.VectorStore
	var retriever *retrieval.Retriever
	var augmenter *retrieval.Augmenter

	if settings.VectorStore.Enabled {
		logger.Info("Vector store is enabled, initializing RAG components")

		// Load vector store configuration
		vsConfig := vectorstore.LoadConfig()
		logger.Debug("Vector store config - Provider: %s, Collection: %s", vsConfig.Provider, vsConfig.CollectionName)

		// Create embedder based on configuration
		var embedder embeddings.Embedder
		if vsConfig.Embedding.Provider == "ollama" {
			embedConfig := embeddings.OllamaConfig{
				Endpoint: vsConfig.Embedding.Endpoint,
				Model:    vsConfig.Embedding.Model,
			}
			embedder, err = embeddings.NewOllamaEmbedder(embedConfig)
			if err != nil {
				logger.Warn("Could not initialize embedder: %v", err)
			} else {
				logger.Debug("Initialized Ollama embedder with model: %s", vsConfig.Embedding.Model)
			}
		}

		// Create vector store if embedder is available
		if embedder != nil {
			vectorStore, err = vectorstore.NewVectorStore(vsConfig, embedder)
			if err != nil {
				logger.Warn("Could not initialize vector store: %v", err)
			} else {
				logger.Info("Vector store initialized successfully")
				// Create retriever
				retriever = retrieval.NewRetriever(vectorStore, retrieval.Config{
					MaxDocuments:     vsConfig.Retrieval.K,
					ScoreThreshold:   vsConfig.Retrieval.ScoreThreshold,
					MaxContextLength: vsConfig.Retrieval.MaxContextLength,
				})
				logger.Debug("Retriever created with K=%d, threshold=%f", vsConfig.Retrieval.K, vsConfig.Retrieval.ScoreThreshold)

				// Create augmenter
				maxContextLength := vsConfig.Retrieval.MaxContextLength
				if maxContextLength == 0 {
					maxContextLength = 4000 // Default fallback
				}
				augmenter = retrieval.NewAugmenter(retriever, retrieval.AugmenterConfig{
					MaxContextLength: maxContextLength,
				})
				logger.Debug("Augmenter created with max context length: %d", maxContextLength)

				// Add retriever as a LangChain tool if available
				if retriever != nil {
					lcRetriever := retrieval.NewLangChainRetriever(retriever)
					// Note: We would need to create a tool wrapper here if we want to use it as a tool
					// For now, we'll use it directly in Execute/ExecuteStream
					_ = lcRetriever // Suppress unused variable warning
				}
			}
		}
	}

	// Create the agent - using a conversational agent with tools
	agent := agents.NewConversationalAgent(
		llm,
		agentTools,
	)

	// Create a LangChain memory wrapper around our SQLite history
	lcMem := lcmemory.NewConversationBuffer(
		lcmemory.WithChatHistory(mem.ChatMessageHistory()),
	)

	// Create executor with options
	maxIterations := settings.LangChain.Tools.MaxIterations
	if maxIterations == 0 {
		maxIterations = 10
	}

	executor := agents.NewExecutor(
		agent,
		agents.WithMaxIterations(maxIterations),
		agents.WithMemory(lcMem),
	)

	// Initialize token counter
	modelName := settings.Ollama.DefaultModel
	tokenCounter, err := tokens.NewTokenCounter(modelName)
	if err != nil {
		// Don't fail if token counter can't be initialized, just log warning
		logger.Warn("Could not initialize token counter: %v", err)
		tokenCounter = nil
	}

	return &ReactAgent{
		llm:          llm,
		executor:     executor,
		memory:       mem,
		tools:        agentTools,
		tokenCounter: tokenCounter,
		tokensSent:   0,
		tokensRecv:   0,
		state:        NewExecutionState(),
		vectorStore:  vectorStore,
		retriever:    retriever,
		augmenter:    augmenter,
	}, nil
}

// Execute handles a request and returns a response
func (e *ReactAgent) Execute(ctx context.Context, prompt string) (string, error) {
	logger.Debug("Execute called with prompt: %s", prompt)

	// Update state to thinking
	if e.state != nil {
		e.state.Reset()
		e.state.SetPhase(PhaseThinking)
	}

	// Format prompt using template if configured
	actualPrompt := prompt
	if e.promptTemplate != nil {
		formatted, err := e.FormatPrompt(prompt, nil)
		if err != nil {
			// Log but continue with original prompt
			logger.Warn("Could not format prompt with template: %v", err)
		} else {
			actualPrompt = formatted
			logger.Debug("Prompt formatted with template")
		}
	}

	// Augment prompt with retrieved context if RAG is enabled
	settings := config.Get()
	if e.augmenter != nil && settings.VectorStore.Enabled {
		logger.Debug("Attempting to augment prompt with RAG")
		augmented, err := e.augmenter.AugmentPrompt(ctx, actualPrompt)
		if err != nil {
			// Log but don't fail - continue without augmentation
			logger.Warn("Could not augment prompt: %v", err)
		} else {
			actualPrompt = augmented
			logger.Debug("Prompt augmented successfully")
		}
	}

	// Count input tokens
	if e.tokenCounter != nil {
		inputTokens := e.tokenCounter.CountTokens(actualPrompt)
		e.tokensMu.Lock()
		e.tokensSent += inputTokens
		e.tokensMu.Unlock()
		logger.Debug("Input tokens: %d (total sent: %d)", inputTokens, e.tokensSent)
	}

	// The executor will handle memory management now
	// Just pass the input through
	input := map[string]any{
		"input": actualPrompt,
	}

	// Execute through the agent (memory is handled by the executor)
	logger.Debug("Calling executor with input")
	result, err := e.executor.Call(ctx, input)
	if err != nil {
		logger.Error("Agent execution failed: %v", err)
		return "", fmt.Errorf("agent execution failed: %w", err)
	}
	logger.Debug("Executor call completed successfully")

	// Manually add to memory since LangChain executor might not be doing it properly
	// Add user message
	if err := e.memory.AddUserMessage(actualPrompt); err != nil {
		// Log but don't fail
		logger.Warn("Could not add user message to memory: %v", err)
	}

	// Extract the response
	response, ok := result["output"].(string)
	if !ok {
		// Try to get any string response from the result
		for _, v := range result {
			if str, ok := v.(string); ok {
				response = str
				break
			}
		}
		if response == "" {
			return "", fmt.Errorf("no valid response from agent")
		}
	}

	// Count output tokens
	if e.tokenCounter != nil {
		outputTokens := e.tokenCounter.CountTokens(response)
		e.tokensMu.Lock()
		e.tokensRecv += outputTokens
		e.tokensMu.Unlock()
		logger.Debug("Output tokens: %d (total received: %d)", outputTokens, e.tokensRecv)
	}

	// Add assistant message to memory
	if err := e.memory.AddAssistantMessage(response); err != nil {
		// Log but don't fail
		logger.Warn("Could not add assistant message to memory: %v", err)
	}

	return response, nil
}

// ExecuteStream handles a request with streaming response
func (e *ReactAgent) ExecuteStream(ctx context.Context, prompt string, handler core.Handler) error {
	logger.Debug("ExecuteStream called with prompt: %s", prompt)

	// Update state to thinking
	if e.state != nil {
		e.state.Reset()
		e.state.SetPhase(PhaseThinking)
	}

	// Format prompt using template if configured
	actualPrompt := prompt
	if e.promptTemplate != nil {
		formatted, err := e.FormatPrompt(prompt, nil)
		if err != nil {
			// Log but continue with original prompt
			logger.Warn("Could not format prompt with template: %v", err)
		} else {
			actualPrompt = formatted
			logger.Debug("Prompt formatted with template")
		}
	}

	// Augment prompt with retrieved context if RAG is enabled
	settings := config.Get()
	if e.augmenter != nil && settings.VectorStore.Enabled {
		logger.Debug("Attempting to augment prompt with RAG (streaming)")
		augmented, err := e.augmenter.AugmentPrompt(ctx, actualPrompt)
		if err != nil {
			// Log but don't fail - continue without augmentation
			logger.Warn("Could not augment prompt: %v", err)
		} else {
			actualPrompt = augmented
			logger.Debug("Prompt augmented successfully (streaming)")
		}
	}

	// Count input tokens
	if e.tokenCounter != nil {
		inputTokens := e.tokenCounter.CountTokens(actualPrompt)
		e.tokensMu.Lock()
		e.tokensSent += inputTokens
		e.tokensMu.Unlock()
		logger.Debug("Input tokens: %d (total sent: %d)", inputTokens, e.tokensSent)
	}

	// Create a LangChain streaming source using the agent's LLM
	source := providers.NewLangChainSource(e.llm)

	// Build conversation messages from memory
	messages := []core.Message{}

	// Add conversation history if available
	if e.memory != nil {
		llmMessages, err := e.memory.ConvertToLLMMessages()
		if err == nil {
			for _, msg := range llmMessages {
				messages = append(messages, core.Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
			logger.Debug("Added %d messages from memory to context", len(llmMessages))
		} else {
			logger.Warn("Could not convert memory to LLM messages: %v", err)
		}
	}

	// Add the current prompt
	messages = append(messages, core.Message{
		Role:    "user",
		Content: actualPrompt,
	})

	// Create a wrapper handler that tracks tokens and updates memory
	tokenAndMemoryHandler := &tokenAndMemoryHandler{
		inner:      handler,
		memory:     e.memory,
		prompt:     actualPrompt,
		agent:      e,
		buffer:     "",
		lastTokens: 0,
	}

	// Stream with full conversation history
	return source.StreamWithHistory(ctx, messages, tokenAndMemoryHandler)
}

// tokenAndMemoryHandler wraps a stream handler to track tokens and update memory
type tokenAndMemoryHandler struct {
	inner      core.Handler
	memory     *memory.Memory
	prompt     string
	agent      *ReactAgent
	buffer     string
	lastTokens int
}

func (h *tokenAndMemoryHandler) OnChunk(chunk []byte) error {
	// Accumulate chunks for memory and token tracking
	h.buffer += string(chunk)

	// Count tokens in accumulated buffer
	if h.agent.tokenCounter != nil {
		currentTokens := h.agent.tokenCounter.CountTokens(h.buffer)
		// Only update if token count changed
		if currentTokens > h.lastTokens {
			tokenDiff := currentTokens - h.lastTokens
			h.agent.tokensMu.Lock()
			h.agent.tokensRecv += tokenDiff
			h.agent.tokensMu.Unlock()
			h.lastTokens = currentTokens
		}
	}

	// Forward to original handler
	return h.inner.OnChunk(chunk)
}

func (h *tokenAndMemoryHandler) OnComplete(finalContent string) error {
	if finalContent == "" {
		finalContent = h.buffer
	}

	// Final token count (in case there's any discrepancy)
	if h.agent.tokenCounter != nil && finalContent != "" {
		finalTokens := h.agent.tokenCounter.CountTokens(finalContent)
		if finalTokens > h.lastTokens {
			tokenDiff := finalTokens - h.lastTokens
			h.agent.tokensMu.Lock()
			h.agent.tokensRecv += tokenDiff
			h.agent.tokensMu.Unlock()
		}
	}

	// Update memory with the exchange
	if h.memory != nil {
		// Add user message
		_ = h.memory.AddUserMessage(h.prompt)

		// Add assistant response
		_ = h.memory.AddAssistantMessage(finalContent)
	}

	return h.inner.OnComplete(finalContent)
}

func (h *tokenAndMemoryHandler) OnError(err error) {
	h.inner.OnError(err)
}

// GetLLM returns the underlying LLM for direct access if needed
func (e *ReactAgent) GetLLM() llms.Model {
	return e.llm
}

// GetMemory returns the memory instance
func (e *ReactAgent) GetMemory() *memory.Memory {
	return e.memory
}

// ClearMemory clears the conversation memory and resets token counts
func (e *ReactAgent) ClearMemory() error {
	// Reset token counts
	e.tokensMu.Lock()
	e.tokensSent = 0
	e.tokensRecv = 0
	e.tokensMu.Unlock()

	// Clear memory
	if e.memory != nil {
		return e.memory.Clear()
	}
	return nil
}

// GetTokenStats returns the cumulative token usage statistics
func (e *ReactAgent) GetTokenStats() (int, int) {
	e.tokensMu.RLock()
	defer e.tokensMu.RUnlock()
	return e.tokensSent, e.tokensRecv
}

// Close cleans up resources
func (e *ReactAgent) Close() error {
	var errs []error

	if e.memory != nil {
		if err := e.memory.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close memory: %w", err))
		}
	}

	if e.vectorStore != nil {
		if err := e.vectorStore.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close vector store: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}
	return nil
}

// AddDocuments adds documents to the vector store for retrieval
func (e *ReactAgent) AddDocuments(ctx context.Context, documents []vectorstore.Document) error {
	if e.vectorStore == nil {
		return fmt.Errorf("vector store not initialized")
	}
	return e.vectorStore.AddDocuments(ctx, documents)
}

// GetVectorStore returns the vector store instance
func (e *ReactAgent) GetVectorStore() vectorstore.VectorStore {
	return e.vectorStore
}

// GetRetriever returns the retriever instance
func (e *ReactAgent) GetRetriever() *retrieval.Retriever {
	return e.retriever
}

// SetPromptTemplate sets a custom prompt template for the agent
func (e *ReactAgent) SetPromptTemplate(template prompt.Template) {
	e.promptTemplate = template
}

// GetPromptTemplate returns the current prompt template
func (e *ReactAgent) GetPromptTemplate() prompt.Template {
	return e.promptTemplate
}

// FormatPrompt formats the input using the configured prompt template
func (e *ReactAgent) FormatPrompt(input string, additionalVars map[string]any) (string, error) {
	if e.promptTemplate == nil {
		// No template configured, return input as-is
		return input, nil
	}

	// Create variables map with the input
	vars := make(map[string]any)
	vars["input"] = input

	// Add any additional variables
	for k, v := range additionalVars {
		vars[k] = v
	}

	// Format the prompt
	return e.promptTemplate.Format(vars)
}

// GetExecutionState returns a snapshot of the current execution state
func (e *ReactAgent) GetExecutionState() ExecutionStateSnapshot {
	if e.state == nil {
		// Return empty state if not initialized
		return ExecutionStateSnapshot{
			Phase: PhaseIdle,
		}
	}
	return e.state.GetSnapshot()
}
