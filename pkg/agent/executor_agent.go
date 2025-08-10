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
	ryantools "github.com/killallgit/ryan/pkg/tools"
	"github.com/killallgit/ryan/pkg/vectorstore"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
	lcmemory "github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/tools"
)

// ExecutorAgent is a LangChain executor-based agent implementation
// It wraps LangChain's conversational agent with an executor for handling requests
type ExecutorAgent struct {
	llm          llms.Model
	executor     *agents.Executor
	memory       *memory.Memory
	tools        []tools.Tool
	tokenCounter *tokens.TokenCounter
	tokensSent   int
	tokensRecv   int
	tokensMu     sync.RWMutex

	// RAG components
	vectorStore vectorstore.VectorStore
	retriever   *retrieval.Retriever
	augmenter   *retrieval.Augmenter

	// Prompt template for formatting inputs
	promptTemplate prompt.Template
}

// NewExecutorAgent creates a new executor-based agent with an injected LLM
func NewExecutorAgent(llm llms.Model) (*ExecutorAgent, error) {
	var sessionID string
	settings := config.Get()
	if settings.Continue {
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

	return NewExecutorAgentWithSession(llm, sessionID)
}

// NewExecutorAgentWithSession creates a new executor-based agent with a specific session ID
func NewExecutorAgentWithSession(llm llms.Model, sessionID string) (*ExecutorAgent, error) {
	logger.Info("Initializing ExecutorAgent with session: %s", sessionID)

	mem, err := memory.New(sessionID)
	if err != nil {
		logger.Error("Failed to create memory for session %s: %v", sessionID, err)
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}
	logger.Debug("Memory initialized for session: %s", sessionID)

	// Initialize tools with permission checking
	agentTools := []tools.Tool{}

	// Get configuration settings
	settings := config.Get()

	// Only add tools if enabled in config
	if settings.Tools.Enabled {
		logger.Debug("Tools are enabled, initializing available tools")

		// Add file tools
		if settings.Tools.File.Read.Enabled {
			agentTools = append(agentTools, ryantools.NewFileReadTool())
			logger.Debug("Added FileReadTool")
		}
		if settings.Tools.File.Write.Enabled {
			agentTools = append(agentTools, ryantools.NewFileWriteTool())
			logger.Debug("Added FileWriteTool")
		}

		// Add git tool
		if settings.Tools.Git.Enabled {
			agentTools = append(agentTools, ryantools.NewGitTool())
			logger.Debug("Added GitTool")
		}

		// Add search tool
		if settings.Tools.Search.Enabled {
			agentTools = append(agentTools, ryantools.NewRipgrepTool())
			logger.Debug("Added RipgrepTool")
		}

		// Add web fetch tool
		if settings.Tools.Web.Enabled {
			agentTools = append(agentTools, ryantools.NewWebFetchTool())
			logger.Debug("Added WebFetchTool")
		}

		logger.Info("Initialized %d tools", len(agentTools))
	} else {
		logger.Info("Tools are disabled in configuration")
	}

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
					MaxDocuments:   vsConfig.Retrieval.K,
					ScoreThreshold: vsConfig.Retrieval.ScoreThreshold,
				})
				logger.Debug("Retriever created with K=%d, threshold=%f", vsConfig.Retrieval.K, vsConfig.Retrieval.ScoreThreshold)

				// Create augmenter
				augmenter = retrieval.NewAugmenter(retriever, retrieval.AugmenterConfig{
					MaxContextLength: 4000, // TODO: Add to settings if needed
				})
				logger.Debug("Augmenter created with max context length: %d", 4000)

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

	return &ExecutorAgent{
		llm:          llm,
		executor:     executor,
		memory:       mem,
		tools:        agentTools,
		tokenCounter: tokenCounter,
		tokensSent:   0,
		tokensRecv:   0,
		vectorStore:  vectorStore,
		retriever:    retriever,
		augmenter:    augmenter,
	}, nil
}

// Execute handles a request and returns a response
func (e *ExecutorAgent) Execute(ctx context.Context, prompt string) (string, error) {
	logger.Debug("Execute called with prompt: %s", prompt)

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

	// Manually add to memory since LangChain executor might not be doing it
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
func (e *ExecutorAgent) ExecuteStream(ctx context.Context, prompt string, handler core.Handler) error {
	logger.Debug("ExecuteStream called with prompt: %s", prompt)

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
	agent      *ExecutorAgent
	buffer     string
	lastTokens int
}

func (h *tokenAndMemoryHandler) OnChunk(chunk string) error {
	// Accumulate chunks for memory and token tracking
	h.buffer += chunk

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
func (e *ExecutorAgent) GetLLM() llms.Model {
	return e.llm
}

// GetMemory returns the memory instance
func (e *ExecutorAgent) GetMemory() *memory.Memory {
	return e.memory
}

// ClearMemory clears the conversation memory and resets token counts
func (e *ExecutorAgent) ClearMemory() error {
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
func (e *ExecutorAgent) GetTokenStats() (int, int) {
	e.tokensMu.RLock()
	defer e.tokensMu.RUnlock()
	return e.tokensSent, e.tokensRecv
}

// Close cleans up resources
func (e *ExecutorAgent) Close() error {
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
func (e *ExecutorAgent) AddDocuments(ctx context.Context, documents []vectorstore.Document) error {
	if e.vectorStore == nil {
		return fmt.Errorf("vector store not initialized")
	}
	return e.vectorStore.AddDocuments(ctx, documents)
}

// GetVectorStore returns the vector store instance
func (e *ExecutorAgent) GetVectorStore() vectorstore.VectorStore {
	return e.vectorStore
}

// GetRetriever returns the retriever instance
func (e *ExecutorAgent) GetRetriever() *retrieval.Retriever {
	return e.retriever
}

// SetPromptTemplate sets a custom prompt template for the agent
func (e *ExecutorAgent) SetPromptTemplate(template prompt.Template) {
	e.promptTemplate = template
}

// GetPromptTemplate returns the current prompt template
func (e *ExecutorAgent) GetPromptTemplate() prompt.Template {
	return e.promptTemplate
}

// FormatPrompt formats the input using the configured prompt template
func (e *ExecutorAgent) FormatPrompt(input string, additionalVars map[string]any) (string, error) {
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
