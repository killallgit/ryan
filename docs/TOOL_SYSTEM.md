# Tool System Architecture

## Overview

Ryan's tool system provides a universal interface for LLM tool calling that works across all major providers (OpenAI, Anthropic, Ollama, MCP) while maintaining Ryan's core principles of functional programming, incremental complexity, and comprehensive testing.

## Design Philosophy

### Universal Compatibility
Rather than building provider-specific implementations, Ryan implements a single tool interface that adapts to any provider's format through lightweight adapters. This "build once, support all" approach ensures consistency and reduces maintenance overhead.

### Industry Standard Adherence
Based on analysis of OpenAI, Anthropic, Ollama, and MCP protocols, all providers share common patterns:
- JSON Schema-based tool definitions
- Standard tool structure: name, description, parameters
- Request-response execution flow
- Tool result incorporation into LLM context

### Functional Architecture
Following Ryan's established patterns:
- Immutable tool definitions
- Pure functions for tool execution
- Channel-based communication for concurrent execution
- Event-driven integration with existing TUI

## Core Components

### Tool Interface
```go
type Tool interface {
    Name() string
    Description() string
    JSONSchema() map[string]interface{}
    Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
}
```

### Provider Adapters
- **OpenAI**: `{"type": "function", "function": {...}}`
- **Anthropic**: `{"name": ..., "input_schema": {...}}`
- **Ollama**: Compatible with OpenAI format
- **MCP**: JSON-RPC protocol wrapper

### Tool Registry
Manages available tools and their configurations:
```go
type ToolRegistry struct {
    tools map[string]Tool
    // Thread-safe access via sync.RWMutex
}
```

## Initial Tool Implementations

### BashTool
Executes shell commands with safety constraints:
- Command validation and sanitization
- Timeout and cancellation support
- Output capture and streaming
- Security restrictions (no sudo, limited paths)

### FileReadTool
Reads file contents with path validation:
- Path traversal protection
- File size limits
- Encoding detection
- Permission checking

## Integration with Ryan

### Ollama Client Extension
Extends existing `pkg/ollama/client.go`:
- Add tools parameter to chat requests
- Handle tool_calls in responses
- Maintain streaming architecture
- Preserve existing error handling

### TUI Integration
Leverages existing event system:
- Tool execution feedback via custom events
- Real-time progress updates
- Non-blocking UI during tool execution
- Error display in alert area

### Configuration
Extends existing viper configuration:
```yaml
tools:
  enabled: true
  bash:
    enabled: true
    timeout: "30s"
    allowed_paths: ["/home/user", "/tmp"]
  file_read:
    enabled: true
    max_file_size: "10MB"
    allowed_extensions: [".txt", ".md", ".go", ".json"]
```

## Security Model

### Safety First Approach
- Explicit user consent for tool execution
- Command validation and sanitization
- Resource usage limits
- Audit logging for all tool operations

### Sandboxing Strategy
- Restricted file system access
- Command whitelist/blacklist
- Timeout enforcement
- Resource monitoring

## Implementation Phases

### Phase 1: Foundation (Current)
- Universal tool interface
- Provider adapters
- Basic bash and file read tools
- Tool registry system

### Phase 2: Ollama Integration
- Extend existing client
- Tool execution loop
- TUI feedback system
- Error handling

### Phase 3: Multi-Provider Support
- OpenAI client with tools
- Anthropic client with tools
- Provider detection and switching
- Unified configuration

### Phase 4: MCP Protocol
- JSON-RPC transport
- Tool discovery
- Resource and prompt support
- Security consent model

## Testing Strategy

### Unit Tests
- Tool interface implementations
- Provider adapter conversions
- Parameter validation
- Error handling

### Integration Tests
- Tool execution with real commands
- Provider communication
- TUI event handling
- Configuration loading

### Security Tests
- Path traversal attempts
- Command injection prevention
- Resource limit enforcement
- Permission boundary testing

## Future Extensions

### Additional Tools
- Git operations
- Package manager interactions
- Development environment tools
- Network requests
- Database queries

### Advanced Features
- Tool composition and chaining
- Custom tool development
- Plugin system
- Performance monitoring

---

*This document evolves as we implement and learn from the tool system*