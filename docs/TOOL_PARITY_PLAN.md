# Tool System Parity Implementation Plan

## Overview

This document provides a detailed, actionable implementation plan to achieve feature parity with Claude Code's sophisticated tool execution system. Based on analysis of the deobfuscated Claude Code bundle, we've identified key architectural patterns and capabilities that Ryan must implement.

## Current State Analysis

### Ryan's Tool System (As of Phase 2 Complete)
**Strengths**:
- ✅ Universal tool interface design (`Tool` interface)
- ✅ Provider adapter pattern established
- ✅ Basic registry system with thread-safe access
- ✅ Two working tools (BashTool, FileReadTool)
- ✅ TUI integration with event-driven updates
- ✅ Non-blocking UI architecture ready for tool execution

**Gaps Identified**:
- ❌ **Batch Execution**: Can only execute one tool at a time
- ❌ **Concurrent Orchestration**: No parallel tool execution capability
- ❌ **Tool Coverage**: Only 2 tools vs Claude Code's 15+ production tools
- ❌ **Advanced UX**: No progress tracking, cancellation, or result streaming
- ❌ **Provider Integration**: Only basic Ollama support, no multi-provider

### Claude Code's Advanced Capabilities
**Key Patterns Discovered**:
- **"Multiple tools in single response"**: Batch execution architecture
- **Concurrent orchestration**: Tools execute in parallel with result aggregation
- **Rich tool suite**: 15+ production-ready tools with comprehensive validation
- **Provider agnostic**: Universal interface across OpenAI, Anthropic, Ollama
- **Advanced UX**: Real-time progress, cancellation, result streaming

## Implementation Phases

### Phase 3A: Advanced Execution Engine (Weeks 1-2)

#### Week 1: Concurrent Tool Orchestrator

**Goal**: Build the foundation for parallel tool execution

**Core Components**:

1. **Goroutine Pool Manager** (`pkg/tools/executor.go`)
```go
type ExecutorPool struct {
    workers    chan chan ToolRequest
    workerPool chan *Worker
    maxWorkers int
    quit       chan bool
}

type Worker struct {
    ID         int
    workerPool chan chan ToolRequest
    jobChannel chan ToolRequest
    quit       chan bool
}

func NewExecutorPool(maxWorkers int) *ExecutorPool
func (p *ExecutorPool) Start()
func (p *ExecutorPool) Stop()
func (p *ExecutorPool) Submit(req ToolRequest) <-chan ToolResult
```

2. **Result Aggregator** (`pkg/tools/aggregator.go`)
```go
type ResultAggregator struct {
    results   map[string]ToolResult
    mu        sync.RWMutex
    callbacks map[string][]ResultCallback
}

type BatchResult struct {
    Results    map[string]ToolResult
    Errors     map[string]error
    StartTime  time.Time
    EndTime    time.Time
    Duration   time.Duration
}

func (ra *ResultAggregator) Collect(id string, result ToolResult)
func (ra *ResultAggregator) GetBatchResult() BatchResult
func (ra *ResultAggregator) OnResult(id string, callback ResultCallback)
```

3. **Progress Manager** (`pkg/tools/progress.go`)
```go
type ProgressManager struct {
    trackers map[string]*ProgressTracker
    mu       sync.RWMutex
    updates  chan ProgressUpdate
}

type ProgressTracker struct {
    ID          string
    Status      ProgressStatus
    Progress    float64    // 0.0 to 1.0
    Message     string
    StartTime   time.Time
    EstimatedEnd time.Time
}

type ProgressUpdate struct {
    ID       string
    Status   ProgressStatus
    Progress float64
    Message  string
}

func (pm *ProgressManager) Start(id string, description string)
func (pm *ProgressManager) Update(id string, progress float64, message string)
func (pm *ProgressManager) Complete(id string)
func (pm *ProgressManager) Subscribe() <-chan ProgressUpdate
```

**Tests Required**:
- Goroutine pool with maximum worker limits
- Result aggregation accuracy under concurrent load
- Progress tracking with realistic tool execution times
- Race condition testing with `-race` flag
- Resource cleanup and no goroutine leaks

**Exit Criteria**: Can execute 10 dummy tools concurrently with proper resource management

#### Week 2: Batch Execution System

**Goal**: Implement Claude Code's "multiple tools in single response" capability

**Core Components**:

1. **Dependency Resolution** (`pkg/tools/dependencies.go`)
```go
type DependencyGraph struct {
    nodes map[string]*ToolNode
    edges map[string][]string
}

type ToolNode struct {
    ID           string
    Tool         Tool
    Request      ToolRequest
    Dependencies []string
    Dependents   []string
    Status       NodeStatus
}

func (dg *DependencyGraph) AddTool(id string, tool Tool, deps []string)
func (dg *DependencyGraph) TopologicalSort() ([]string, error)
func (dg *DependencyGraph) GetExecutableNodes() []string
func (dg *DependencyGraph) MarkComplete(id string)
```

2. **Batch Executor** (`pkg/tools/batch.go`)
```go
type BatchExecutor struct {
    orchestrator  *ToolOrchestrator
    dependencies  *DependencyGraph
    maxConcurrent int
    timeout       time.Duration
    results       map[string]ToolResult
    errors        map[string]error
    progress      *ProgressManager
}

type BatchRequest struct {
    Tools        []ToolRequest
    Dependencies map[string][]string  // tool_id -> [dependency_ids]
    Timeout      time.Duration
    Context      context.Context
}

func (be *BatchExecutor) Execute(req BatchRequest) (BatchResult, error)
func (be *BatchExecutor) ExecuteParallel(ctx context.Context) error
func (be *BatchExecutor) handleResults(resultChan <-chan ToolResult)
```

3. **Context Management** (`pkg/tools/context.go`)
```go
type ExecutionContext struct {
    ID             string
    UserID         string
    SessionID      string
    RequestTime    time.Time
    Timeout        time.Duration
    CancelFunc     context.CancelFunc
    ProgressSink   chan<- ProgressUpdate
    ResourceLimits ResourceLimits
    Permissions    []Permission
}

type ResourceLimits struct {
    MaxMemoryMB    int
    MaxCPUPercent  float64
    MaxExecutionTime time.Duration
    MaxConcurrentTools int
}

func NewExecutionContext(userID, sessionID string) *ExecutionContext
func (ec *ExecutionContext) WithTimeout(timeout time.Duration) *ExecutionContext
func (ec *ExecutionContext) WithResourceLimits(limits ResourceLimits) *ExecutionContext
func (ec *ExecutionContext) Cancel()
```

**Tests Required**:
- Dependency resolution with complex graphs
- Batch execution with dependencies and failures
- Context cancellation and resource cleanup
- Timeout handling and partial results
- Error propagation and recovery

**Exit Criteria**: Can execute complex batch operations with dependencies, showing real-time progress in TUI

### Phase 3B: Comprehensive Tool Suite (Weeks 3-4)

#### Week 3: Core Tool Implementation

**Goal**: Implement 8 essential tools matching Claude Code patterns

1. **WebFetch Tool** (`pkg/tools/webfetch.go`)
```go
type WebFetchTool struct {
    client    *http.Client
    cache     *URLCache
    rateLimit *RateLimiter
}

func (wft *WebFetchTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)

// Parameters: url, method, headers, body, timeout, follow_redirects, cache_ttl
// Features: Rate limiting, caching, redirect handling, timeout control
```

2. **Enhanced Grep Tool** (`pkg/tools/grep.go`)
```go
type GrepTool struct {
    ripgrepBinary string
    highlighter   *SyntaxHighlighter
}

// Parameters: pattern, path, context_lines, case_sensitive, file_types, exclude_patterns
// Features: ripgrep integration, syntax highlighting, context display
```

3. **Glob Tool** (`pkg/tools/glob.go`)
```go
type GlobTool struct {
    maxResults int
    sorter     *ResultSorter
}

// Parameters: pattern, path, sort_by, max_results, include_hidden, file_only
// Features: Pattern matching, sorting by name/size/date, filtering
```

4. **Enhanced Read Tool** (`pkg/tools/read.go`)
```go
type EnhancedReadTool struct {
    encodingDetector *EncodingDetector
    pdfReader       PDFReader
    imageViewer     ImageViewer
}

// Parameters: file_path, encoding, max_size, offset, limit, format
// Features: Encoding detection, PDF text extraction, image description
```

5. **Write Tool** (`pkg/tools/write.go`)
```go
type WriteTool struct {
    backupManager *BackupManager
    validator     *ContentValidator
}

// Parameters: file_path, content, encoding, create_backup, permissions
// Features: Backup creation, content validation, permission handling
```

6. **Directory Operations Tool** (`pkg/tools/ls.go`)
```go
type DirectoryTool struct {
    maxDepth int
    sorter   *DirectorySorter
}

// Parameters: path, recursive, max_depth, sort_by, filter, show_hidden
// Features: Tree view, filtering, sorting, size calculation
```

7. **Git Integration Tool** (`pkg/tools/git.go`)
```go
type GitTool struct {
    gitBinary string
    validator *GitValidator
}

// Operations: status, diff, commit, log, branch, remote
// Features: Repository validation, conflict detection, safe operations
```

8. **Process Management Tool** (`pkg/tools/process.go`)
```go
type ProcessTool struct {
    allowedCommands []string
    resourceLimits  ProcessLimits
}

// Operations: ps, kill, monitor, resource_usage
// Features: Command whitelisting, resource monitoring, safe termination
```

**Tests Required**:
- Each tool with comprehensive parameter validation
- Error handling for network failures, file permissions, etc.
- Security testing for path traversal, command injection
- Performance testing with large files and datasets
- Integration testing with real external systems

**Exit Criteria**: 8 production-ready tools with comprehensive error handling and security

#### Week 4: Advanced Tool Features

**Goal**: Add sophisticated features matching Claude Code's polish

1. **Tool Result Caching** (`pkg/tools/cache.go`)
```go
type ToolResultCache struct {
    storage    map[string]CachedResult
    ttl        map[string]time.Time
    maxSize    int64
    currentSize int64
    mu         sync.RWMutex
}

type CachedResult struct {
    Result    ToolResult
    Timestamp time.Time
    TTL       time.Duration
    Size      int64
}

func (trc *ToolResultCache) Get(key string) (ToolResult, bool)
func (trc *ToolResultCache) Set(key string, result ToolResult, ttl time.Duration)
func (trc *ToolResultCache) Invalidate(pattern string)
func (trc *ToolResultCache) Cleanup()
```

2. **User Consent Manager** (`pkg/tools/consent.go`)
```go
type ConsentManager struct {
    policies    map[string]ConsentPolicy
    userChoices map[string]UserChoice
    prompter    ConsentPrompter
}

type ConsentPolicy struct {
    ToolName    string
    Operation   string
    RiskLevel   RiskLevel
    RequiresConsent bool
    Description string
}

func (cm *ConsentManager) RequiresConsent(tool string, params map[string]interface{}) bool
func (cm *ConsentManager) RequestConsent(ctx context.Context, policy ConsentPolicy) (bool, error)
func (cm *ConsentManager) RememberChoice(tool string, choice UserChoice)
```

3. **Resource Monitor** (`pkg/tools/monitor.go`)
```go
type ResourceMonitor struct {
    limits      ResourceLimits
    current     ResourceUsage
    alerts      chan ResourceAlert
    shutdown    chan bool
}

type ResourceUsage struct {
    MemoryMB     int64
    CPUPercent   float64
    ActiveTools  int
    ExecutionTime time.Duration
}

func (rm *ResourceMonitor) CheckLimits() error
func (rm *ResourceMonitor) RecordUsage(toolID string, usage ResourceUsage)
func (rm *ResourceMonitor) GetCurrentUsage() ResourceUsage
func (rm *ResourceMonitor) EnforceLimits(toolID string) error
```

**Tests Required**:
- Cache hit/miss ratios and memory management
- Consent flow with user interaction simulation
- Resource limit enforcement and violation handling
- Integration testing with all tools
- Performance benchmarking with realistic workloads

**Exit Criteria**: Production-ready tool system with caching, consent, and resource management

### Phase 3C: Multi-Provider Integration (Weeks 5-6)

#### Week 5: Provider Abstraction Layer

**Goal**: Universal tool calling interface for multiple LLM providers

1. **Provider Interface** (`pkg/providers/interface.go`)
```go
type Provider interface {
    Name() string
    ConvertTool(tool tools.Tool) (ProviderTool, error)
    ParseToolCall(response []byte) ([]ToolCall, error)
    FormatToolResult(result tools.ToolResult) (ProviderResult, error)
    SupportsStreaming() bool
    SupportsBatchExecution() bool
}

type ProviderTool struct {
    Name        string
    Description string
    Definition  interface{} // Provider-specific format
}

type ToolCall struct {
    ID         string
    Name       string
    Parameters map[string]interface{}
}

type ProviderResult struct {
    Content interface{} // Provider-specific result format
    Error   string      // If execution failed
}
```

2. **OpenAI Adapter** (`pkg/providers/openai.go`)
```go
type OpenAIProvider struct {
    apiKey string
    client *openai.Client
}

// Converts to OpenAI function calling format:
// {"type": "function", "function": {"name": "...", "description": "...", "parameters": {...}}}

func (oai *OpenAIProvider) ConvertTool(tool tools.Tool) (ProviderTool, error)
func (oai *OpenAIProvider) ParseToolCall(response []byte) ([]ToolCall, error)
func (oai *OpenAIProvider) FormatToolResult(result tools.ToolResult) (ProviderResult, error)
```

3. **Anthropic Adapter** (`pkg/providers/anthropic.go`)
```go
type AnthropicProvider struct {
    apiKey string
    client *anthropic.Client
}

// Converts to Anthropic tool format:
// {"name": "...", "description": "...", "input_schema": {...}}

func (ap *AnthropicProvider) ConvertTool(tool tools.Tool) (ProviderTool, error)
func (ap *AnthropicProvider) ParseToolCall(response []byte) ([]ToolCall, error)
func (ap *AnthropicProvider) FormatToolResult(result tools.ToolResult) (ProviderResult, error)
```

4. **Ollama Adapter** (`pkg/providers/ollama.go`)
```go
type OllamaProvider struct {
    baseURL string
    client  *ollama.Client
}

// Uses OpenAI-compatible format for tool calling
// Extends existing client with tool support

func (op *OllamaProvider) ConvertTool(tool tools.Tool) (ProviderTool, error)
func (op *OllamaProvider) ParseToolCall(response []byte) ([]ToolCall, error)
func (op *OllamaProvider) FormatToolResult(result tools.ToolResult) (ProviderResult, error)
```

**Tests Required**:
- Tool definition conversion accuracy for each provider
- Tool call parsing with malformed inputs
- Result formatting consistency across providers
- Error handling for provider-specific failures
- Performance comparison between providers

**Exit Criteria**: Universal tool interface works with OpenAI, Anthropic, and Ollama

#### Week 6: Streaming Tool Integration & TUI Enhancement

**Goal**: Real-time tool execution feedback in streaming responses

1. **Streaming Tool Execution** (`pkg/tools/streaming.go`)
```go
type StreamingToolExecutor struct {
    orchestrator *ToolOrchestrator
    streamSink   chan<- StreamingUpdate
    progressMgr  *ProgressManager
}

type ToolStreamingUpdate struct {
    Type        UpdateType
    ToolID      string
    ToolName    string
    Progress    float64
    Message     string
    Result      *ToolResult
    Error       error
    Timestamp   time.Time
}

func (ste *StreamingToolExecutor) ExecuteWithStreaming(
    tools []ToolRequest,
    stream chan<- ToolStreamingUpdate,
) error
```

2. **TUI Tool Integration** (`pkg/tui/tool_display.go`)
```go
type ToolDisplay struct {
    executingTools map[string]ToolProgress
    completedTools []ToolResult
    erroredTools   []ToolError
    width          int
    height         int
}

type ToolProgress struct {
    ID          string
    Name        string
    Progress    float64
    Message     string
    StartTime   time.Time
    ElapsedTime time.Duration
}

func (td *ToolDisplay) UpdateProgress(id string, progress float64, message string)
func (td *ToolDisplay) Completetool(id string, result ToolResult)
func (td *ToolDisplay) RenderTools(screen tcell.Screen, area Rect)
```

3. **Enhanced Controller** (`pkg/controllers/tool_controller.go`)
```go
type ToolController struct {
    orchestrator *tools.ToolOrchestrator
    providers    map[string]providers.Provider
    progress     *tools.ProgressManager
    streamSink   chan<- tui.StreamingUpdate
}

func (tc *ToolController) ExecuteToolsWithStreaming(
    provider string,
    toolCalls []providers.ToolCall,
    stream chan<- tui.StreamingUpdate,
) error
```

**Tests Required**:
- Real-time progress updates during tool execution
- TUI responsiveness with multiple concurrent tools
- Stream handling with tool results mixed with chat responses
- Error recovery and display in streaming context
- Memory efficiency with large tool results

**Exit Criteria**: TUI shows real-time tool execution progress with streaming chat responses

## Success Metrics & Validation

### Performance Benchmarks
**Execution Performance**:
- Tool startup time: < 100ms (matching Claude Code responsiveness)
- Concurrent tool limit: 10+ simultaneous executions
- Memory efficiency: < 50MB base + 10MB per concurrent tool
- Result streaming latency: < 50ms from tool completion

**Resource Management**:
- No memory leaks during extended tool execution sessions
- Proper goroutine cleanup with zero leaks
- CPU usage stays under 80% during peak tool execution
- Network requests respect rate limiting and timeout controls

### Feature Parity Validation
**Tool Coverage**: 15+ production-ready tools matching Claude Code capabilities
- ✅ File operations (read, write, glob, ls)
- ✅ Search and analysis (grep with syntax highlighting)
- ✅ Network operations (webfetch with caching)
- ✅ Development tools (git integration)
- ✅ System integration (process management)

**Execution Patterns**: 
- ✅ Batch execution ("multiple tools in single response")
- ✅ Concurrent orchestration with dependency resolution
- ✅ Real-time progress tracking and cancellation
- ✅ Provider-agnostic tool calling (OpenAI, Anthropic, Ollama)

**User Experience**:
- ✅ Non-blocking TUI during tool execution
- ✅ Real-time progress indicators with time estimates
- ✅ Graceful error handling and recovery
- ✅ Tool result streaming and display

### Security & Safety Validation
**Security Model**: 
- ✅ User consent for potentially dangerous operations
- ✅ Resource limits and sandboxing
- ✅ Path traversal and command injection prevention
- ✅ Audit logging for all tool operations

**Error Handling**:
- ✅ Network failures and timeout handling
- ✅ Resource exhaustion and cleanup
- ✅ Malformed input validation and sanitization
- ✅ Graceful degradation when tools fail

## Risk Mitigation

### Technical Risks
1. **Concurrency Complexity**: Mitigate with comprehensive testing using `-race` flag and deadlock detection
2. **Memory Leaks**: Implement resource monitoring and automated leak detection in CI
3. **Provider Compatibility**: Build extensive test suites for each provider with real API calls
4. **Performance Regression**: Establish benchmarking pipeline with performance alerts

### Architectural Risks
1. **Over-Engineering**: Follow incremental development with working functionality at each step
2. **Tool Quality**: Implement comprehensive security and validation testing for each tool
3. **Integration Complexity**: Build integration tests that validate end-to-end workflows
4. **Maintainability**: Document all architectural decisions and maintain clear code organization

## Next Steps

### Immediate Actions (Week 1)
1. **Setup Development Environment**: Create feature branch for tool parity work
2. **Architecture Review**: Validate proposed architecture with stakeholders
3. **Test Infrastructure**: Setup benchmarking and performance testing framework
4. **Begin Implementation**: Start with goroutine pool and basic orchestration

### Weekly Milestones
- **Week 1**: Concurrent execution engine with progress tracking
- **Week 2**: Batch execution system with dependency resolution
- **Week 3**: 8 core tools implemented and tested
- **Week 4**: Advanced features (caching, consent, monitoring)
- **Week 5**: Multi-provider integration (OpenAI, Anthropic, Ollama)
- **Week 6**: Streaming integration and TUI enhancements

### Success Gates
Each week has clear exit criteria that must be met before proceeding to the next phase. This ensures we build solid foundations before adding complexity, maintaining Ryan's incremental development philosophy while achieving Claude Code's sophisticated capabilities.

---

*Implementation plan created: 2025-08-02*  
*Target completion: 6 weeks from start date*  
*Success metric: Full feature parity with Claude Code's tool execution system*