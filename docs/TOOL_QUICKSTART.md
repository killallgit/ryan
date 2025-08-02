# Tool System Quick Start

## Current Status

Phase 3A in progress: Advanced tool execution engine with Claude Code parity goals.

**Available tools (current):**
- `execute_bash` - Shell commands with safety constraints  
- `read_file` - File reading with path validation

**What works now:**
- Tool execution and registry with basic concurrent support  
- Provider format conversion (OpenAI, Anthropic, Ollama)
- Security validation and sandboxing
- Streaming integration with TUI

**Phase 3B roadmap (comprehensive tool suite):**
- WebFetch, enhanced Grep, Glob, Git integration
- Directory operations, process management
- Batch execution with dependency resolution

## Basic Usage

```go
// Create registry and register built-in tools
registry := tools.NewRegistry()
registry.RegisterBuiltinTools()

// Execute bash command
req := tools.ToolRequest{
    Name: "execute_bash", 
    Parameters: map[string]interface{}{
        "command": "echo 'Hello'",
    },
}
result, err := registry.Execute(context.Background(), req)

// Read file
req = tools.ToolRequest{
    Name: "read_file",
    Parameters: map[string]interface{}{
        "path": "README.md",
    },
}
result, err = registry.Execute(context.Background(), req)
```

## Advanced Features (Phase 3A+)

### Batch Execution
Execute multiple tools concurrently:

```go
requests := []tools.ToolRequest{
    {Name: "read_file", Parameters: map[string]interface{}{"path": "config.json"}},
    {Name: "execute_bash", Parameters: map[string]interface{}{"command": "pwd"}},
}

batchResult, err := registry.ExecuteBatch(context.Background(), requests)
if err == nil {
    for toolName, result := range batchResult.Results {
        fmt.Printf("%s: %s\n", toolName, result.Content)
    }
}
```

### Asynchronous Execution
Execute tools without blocking:

```go
resultChan := registry.ExecuteAsync(context.Background(), req)
result := <-resultChan
```

## Provider Integration

Tools are automatically converted to provider-specific formats:

```go
// Get tools in specific provider format
definitions, err := registry.GetDefinitions("openai")     // OpenAI format
definitions, err := registry.GetDefinitions("anthropic")  // Anthropic format
definitions, err := registry.GetDefinitions("ollama")     // Ollama format
definitions, err := registry.GetDefinitions("mcp")        // MCP format
```

## Security Features

Built-in safety constraints:
- **BashTool**: Blocks dangerous commands (sudo, rm -rf, etc.), restricts paths
- **FileReadTool**: Prevents path traversal, limits file sizes, validates extensions

## Key Interfaces

```go
type Tool interface {
    Name() string
    Description() string  
    JSONSchema() map[string]interface{}
    Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
}

type ToolResult struct {
    Success  bool
    Content  string
    Error    string
    Metadata ToolMetadata
}
```

## Next Steps

**Phase 3A (current)**: Advanced execution engine with concurrent orchestration  
**Phase 3B**: Comprehensive tool suite (15+ tools) matching Claude Code coverage  
**Phase 3C**: Multi-provider integration with streaming tool execution in TUI

See [TOOL_PARITY_PLAN.md](TOOL_PARITY_PLAN.md) for detailed implementation roadmap.