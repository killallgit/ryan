# LangChain Integration Strategy

## Current State Analysis

Ryan currently has **partial LangChain integration** but is not leveraging the full power of LangChain Go for streaming, tool usage, and context management. We have custom implementations where LangChain provides superior, battle-tested solutions.

### What We Have ✅

- ✅ Basic LangChain memory wrapper (`LangChainMemory`)
- ✅ LangChain chat controller that extends base controller
- ✅ LangChain streaming client using LLM interface
- ✅ Conversion utilities between our types and LangChain types

### What We're Missing 🚧

- 🚧 **Streaming**: Using custom logic instead of LangChain's `llms.WithStreamingFunc`
- 🚧 **Tool Integration**: Separate tool system instead of LangChain's `tools.Tool` interface
- 🚧 **Agent System**: No use of LangChain's agent framework for autonomous tool calling
- 🚧 **Chains**: No use of conversation chains for workflow orchestration
- 🚧 **Advanced Memory**: Only basic memory, missing Window/Summary memory types
- 🚧 **Prompt Templates**: Basic string templates instead of LangChain's prompt system

## Integration Strategy

### Phase 1: Enhanced Streaming with LangChain ⚡

**Objective**: Replace custom streaming with LangChain Go's streaming capabilities

**Current Approach**:
```go
// Custom streaming implementation
func (sc *StreamingClient) StreamMessage(ctx context.Context, req ChatRequest) (<-chan MessageChunk, error) {
    // Custom HTTP streaming logic
    // Manual chunk processing
    // Custom error handling
}
```

**LangChain Approach**:
```go
// Use LangChain's built-in streaming
response, err := llm.GenerateContent(ctx, messages, 
    llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
        select {
        case resultChan <- chunk:
        case <-ctx.Done():
            return ctx.Err()
        }
        return nil
    }),
)
```

**Benefits**:
- ✅ Better error handling and cancellation
- ✅ Optimized channel patterns
- ✅ Provider-agnostic streaming
- ✅ Built-in backpressure handling

### Phase 2: Tool System Integration 🛠️

**Objective**: Integrate our tools with LangChain's tool framework

**Current Approach**:
```go
type Tool interface {
    Name() string
    Description() string
    JSONSchema() map[string]interface{}
    Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
}
```

**LangChain Approach**:
```go
type LangChainTool struct {
    name        string
    description string
}

func (t *LangChainTool) Name() string { return t.name }
func (t *LangChainTool) Description() string { return t.description }
func (t *LangChainTool) Call(ctx context.Context, input string) (string, error) {
    // Tool logic with LangChain integration
}
```

**Integration Strategy**:
```go
// Adapter pattern to bridge our tools with LangChain
type ToolAdapter struct {
    ryanTool tools.Tool
}

func (ta *ToolAdapter) Call(ctx context.Context, input string) (string, error) {
    // Parse input to parameters
    params := parseToolInput(input)
    
    // Execute Ryan tool
    result, err := ta.ryanTool.Execute(ctx, params)
    if err != nil {
        return "", err
    }
    
    // Return result as string
    return result.Content, nil
}
```

### Phase 3: Agent Framework Integration 🤖

**Objective**: Enable autonomous tool calling with LangChain agents

**Implementation**:
```go
// Convert Ryan tools to LangChain tools
langchainTools := make([]tools.Tool, len(ryanTools))
for i, tool := range ryanTools {
    langchainTools[i] = &ToolAdapter{ryanTool: tool}
}

// Create conversational agent
agent := agents.NewConversationalAgent(llm, langchainTools, 
    agents.WithMemory(memory))

// Create executor for autonomous operation
executor := agents.NewExecutor(agent)

// Agent can now automatically decide when to use tools
result, err := executor.Call(ctx, map[string]any{
    "input": "How many docker images are on the system and what's the latest one?",
})
```

**Benefits**:
- ✅ Autonomous tool selection and execution
- ✅ Multi-step reasoning with tools
- ✅ Better tool orchestration
- ✅ ReAct pattern implementation

### Phase 4: Advanced Memory and Chains 🧠

**Objective**: Leverage LangChain's advanced memory and conversation chains

**Current Memory**:
```go
// Basic conversation buffer only
memory := chat.NewLangChainMemory()
```

**Enhanced Memory Options**:
```go
// Window memory for large conversations
windowMemory := memory.NewConversationWindowBuffer(10) // Last 10 messages

// Summary memory for very long conversations
summaryMemory := memory.NewConversationSummaryBuffer(llm, 1000) // Summarize when > 1000 tokens

// Token-based memory management
tokenMemory := memory.NewConversationTokenBuffer(llm, maxTokens)
```

**Conversation Chains**:
```go
// Replace manual conversation management with chains
chain := chains.NewConversationChain(llm, memory)

// Chain automatically manages memory and context
result, err := chains.Run(ctx, chain, userInput)
```

### Phase 5: Prompt Templates and Advanced Features 📝

**Objective**: Use LangChain's prompt templating and advanced features

**Current Prompts**:
```go
// Basic string concatenation
systemPrompt := "You are a helpful assistant..."
userPrompt := userInput
```

**LangChain Prompts**:
```go
// Structured prompt templates
template := prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
    prompts.NewSystemMessagePromptTemplate(
        "You are a helpful AI assistant with access to tools. Current context: {context}",
        []string{"context"}
    ),
    prompts.NewHumanMessagePromptTemplate(
        "{input}",
        []string{"input"}
    ),
})

// Format with variables
messages, err := template.Format(map[string]any{
    "context": getContextualInfo(),
    "input": userInput,
})
```

## Implementation Roadmap

### Week 1: Streaming Integration
- [ ] Implement LangChain-based streaming client
- [ ] Update TUI to use new streaming patterns
- [ ] Maintain backward compatibility
- [ ] Add configuration flag `langchain.enhanced_streaming`

### Week 2: Tool System Bridge
- [ ] Create tool adapter pattern
- [ ] Implement bridge between Ryan tools and LangChain tools
- [ ] Add unit tests for adapter
- [ ] Update tool registry to support both systems

### Week 3: Agent Integration
- [ ] Implement conversational agent setup
- [ ] Create agent executor for TUI integration
- [ ] Add configuration for agent vs. direct tool calling
- [ ] Test autonomous tool calling scenarios

### Week 4: Memory and Chains Enhancement
- [ ] Implement advanced memory types
- [ ] Add conversation chain integration
- [ ] Create memory configuration options
- [ ] Add prompt template support

### Week 5: Production Optimization
- [ ] Performance testing and optimization
- [ ] Documentation updates
- [ ] Configuration consolidation
- [ ] Migration guide for existing users

## Configuration Strategy

```yaml
langchain:
  enabled: true
  
  streaming:
    provider_optimization: true  # Provider-specific optimizations
  
  tools:
    use_agent_framework: true    # Use agents vs direct tool calling
    autonomous_execution: true   # Allow multi-step tool execution
    max_iterations: 5           # Limit agent iterations
  
  memory:
    type: "buffer"              # buffer, window, summary, token
    window_size: 10             # For window memory
    max_tokens: 4000           # For token memory
    summary_threshold: 1000     # When to summarize
  
  prompts:
    use_templates: true         # Use LangChain prompt templates
    context_injection: true     # Auto-inject context
```

## Benefits of Full Integration

### For Users 👥
- ✅ **Better Performance**: Optimized streaming and memory management
- ✅ **Smarter Tool Usage**: Autonomous agents that can reason about tool usage
- ✅ **Longer Conversations**: Advanced memory management for extended chats
- ✅ **Multi-step Operations**: Agents can execute complex workflows automatically

### For Developers 👨‍💻
- ✅ **Less Custom Code**: Leverage battle-tested LangChain implementations
- ✅ **Better Patterns**: Industry-standard agent and tool patterns
- ✅ **Provider Agnostic**: Easy switching between LLM providers
- ✅ **Extensibility**: Access to LangChain's ecosystem of tools and integrations

### For Project 🚀
- ✅ **Future-Proof**: Aligned with LangChain ecosystem evolution
- ✅ **Community**: Access to LangChain Go community and contributions
- ✅ **Standards**: Following established patterns for LLM applications
- ✅ **Maintenance**: Reduced custom code maintenance burden

## Migration Strategy

### Backward Compatibility 🔄
- Maintain existing interfaces during transition
- Use feature flags to enable LangChain features gradually
- Provide migration path for existing configurations
- Keep existing tool implementations working

### Phased Rollout 📊
1. **Opt-in**: New features behind flags
2. **Testing**: Comprehensive testing with both systems
3. **Default**: Make LangChain default for new installations
4. **Deprecation**: Gradual deprecation of custom implementations
5. **Removal**: Remove custom code after proven stability

## Success Metrics

- **Performance**: Streaming latency and memory usage
- **Reliability**: Error rates and crash frequency  
- **User Experience**: Tool calling success rates
- **Developer Experience**: Code complexity and maintenance burden
- **Feature Parity**: Maintain all existing functionality

This strategy positions Ryan as a modern, LangChain-native application while preserving the simplicity and safety principles that define the project.