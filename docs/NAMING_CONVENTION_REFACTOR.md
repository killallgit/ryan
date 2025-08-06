# Naming Convention Refactor Plan

## Current Issues

### 1. "Enhanced" Prefix Overuse
The word "enhanced" appears 35+ times in the codebase, mostly in comments and occasionally in function/variable names:
- `langchain_enhanced` logger component
- "Enhanced LangChain client failed" log messages
- "Enhanced agent framework" comments
- `enhancedMessages` variable in `ollama_tools.go`
- "Enhanced constructors with metadata support" comments

**Problem**: "Enhanced" doesn't convey specific functionality. If something is "enhanced", what was it before? What specific enhancement does it provide?

### 2. Redundant Type Suffixes
Many types include their kind in the name when it's already clear from context:
- `ChatController` (the package is `controllers`, so just `Chat` would suffice)
- `BashTool`, `TreeTool`, `GitTool` (in `tools` package, could be `Bash`, `Tree`, `Git`)
- `VectorStore` interface (in `vectorstore` package)
- `DocumentIndexer`, `DocumentProcessor` (in `vectorstore` package)
- `LangChainChatController` (redundant "Controller" suffix)
- `ChatControllerInterface` (redundant "Interface" suffix)

### 3. Inconsistent Hierarchy Naming
The codebase has overlapping concepts without clear hierarchy:
- `ChatController` vs `LangChainChatController` vs `LangChainController`
- `LangChainVectorMemory` vs `HybridMemory` vs `GraphAwareMemory`
- `Client` vs `ChatClient` vs `StreamingClient`

### 4. Unclear Abstraction Levels
Some names don't clearly indicate their abstraction level:
- Is `Manager` a high-level orchestrator or a low-level resource handler?
- What's the difference between a `Controller` and an `Orchestrator`?
- When should something be a `Handler` vs a `Processor`?

## Proposed Naming Convention

### 1. Remove Redundant Suffixes in Package Context
When a type is in a package that already describes its domain, avoid repeating that domain in the type name:

```go
// BEFORE
package controllers
type ChatController struct {}
type ChatControllerInterface interface {}

// AFTER
package controllers
type Chat struct {}
type Controller interface {}  // Common interface
```

### 2. Use Specific Descriptors Instead of "Enhanced"
Replace vague terms with specific functionality:

```go
// BEFORE
// Enhanced LangChain client with autonomous reasoning
type Client struct {}

// AFTER
// LangChain client with autonomous reasoning and ReAct pattern
type AutonomousClient struct {}
// OR
type ReActClient struct {}
```

### 3. Clear Implementation vs Interface Naming
Interfaces should describe capabilities, implementations should be specific:

```go
// BEFORE
type ChatControllerInterface interface {}
type ChatController struct {}
type LangChainChatController struct {}

// AFTER
type Controller interface {}         // Base interface
type BasicChat struct {}            // Basic implementation
type LangChainChat struct {}        // LangChain-powered implementation
```

### 4. Hierarchical Naming Pattern
Establish clear patterns for different abstraction levels:

```
High-level coordination: Orchestrator, Coordinator
Mid-level control:       Controller, Manager
Low-level operations:    Handler, Processor, Executor
Data structures:         Store, Repository, Registry
Utilities:              Helper, Builder, Factory
```

### 5. Memory System Naming
Clarify the memory hierarchy:

```go
// BEFORE
type LangChainVectorMemory struct {}
type HybridMemory struct {}
type GraphAwareMemory struct {}

// AFTER
type VectorMemory struct {}       // Base vector-based memory
type HybridMemory struct {}       // Combines vector + conversation buffer
type GraphMemory struct {}        // Graph-based contextual memory
```

## Refactoring Priority

### Phase 1: High-Impact, Low-Risk (Week 1)
1. **Remove "enhanced" from comments and logs** (35+ occurrences)
   - Simple find-replace with specific descriptions
   - No code changes required

2. **Rename variables with "enhanced" prefix** (3 occurrences)
   - `enhancedMessages` → `messagesWithContext`
   - Local scope changes only

### Phase 2: Interface Cleanup (Week 2)
1. **Standardize interface naming**
   - `ChatControllerInterface` → `Controller`
   - Move to top of package for clarity

2. **Remove redundant suffixes from interfaces**
   - Keep interfaces focused on capabilities
   - Use embedding for composition

### Phase 3: Package-Level Refactoring (Week 3-4)
1. **controllers package**
   - `ChatController` → `Basic`
   - `LangChainChatController` → `LangChain`
   - `LangChainController` → `Agent` (if it's agent-focused)

2. **tools package**
   - `BashTool` → `Bash`
   - `TreeTool` → `Tree`
   - `GitTool` → `Git`
   - Keep `Tool` interface as-is

3. **vectorstore package**
   - `VectorStore` interface → `Store` interface
   - `DocumentProcessor` → `Processor`
   - `DocumentIndexer` → `Indexer`

### Phase 4: Cross-Package Coordination (Week 5)
1. **Establish naming guidelines document**
2. **Update all imports and references**
3. **Run comprehensive test suite**
4. **Update documentation**

## Migration Strategy

### Step 1: Create Aliases (Backward Compatibility)
```go
// Temporarily maintain old names as aliases
type ChatController = Basic
type LangChainChatController = LangChain
```

### Step 2: Update Internal Usage
- Update all internal references to new names
- Keep public API stable initially

### Step 3: Deprecate Old Names
- Add deprecation notices
- Provide migration guide

### Step 4: Remove Aliases
- After grace period, remove old names
- Major version bump if needed

## Naming Guidelines Going Forward

### DO:
- Use the package name to provide context
- Be specific about functionality
- Follow Go naming conventions (exported vs unexported)
- Use clear, descriptive names without redundancy
- Consider the reader who doesn't know the codebase

### DON'T:
- Add redundant suffixes (Controller in controllers package)
- Use vague enhancers (enhanced, improved, better)
- Mix abstraction levels in the same package
- Create deeply nested type names
- Use abbreviations unless well-known

## Code Examples

### Before:
```go
package controllers

type ChatControllerInterface interface {
    SendUserMessage(content string) (chat.Message, error)
}

type ChatController struct {
    client chat.ChatClient
}

type LangChainChatController struct {
    *ChatController
    memory *chat.LangChainVectorMemory
}

// Enhanced LangChain agent response
func (lc *LangChainController) SendEnhancedMessage() {}
```

### After:
```go
package controllers

type Controller interface {
    SendUserMessage(content string) (chat.Message, error)
}

type Basic struct {
    client chat.ChatClient
}

type LangChain struct {
    *Basic
    memory *chat.VectorMemory
}

// Send message with ReAct reasoning loop
func (lc *Agent) SendWithReasoning() {}
```

## Testing Strategy

1. **Comprehensive test coverage before refactoring**
2. **Parallel testing with aliases**
3. **Integration tests for all renamed components**
4. **Performance benchmarks to ensure no regression**
5. **Documentation tests for all examples**

## Success Metrics

- Reduced cognitive load when reading code
- Clearer separation of concerns
- Easier onboarding for new developers
- Consistent naming across all packages
- No "enhanced" or vague descriptors remaining
