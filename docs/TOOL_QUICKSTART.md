# Tool System Quick Start

## Current Status

Phase 1 complete: Universal tool foundation with two built-in tools.

**Available tools:**
- `execute_bash` - Shell commands with safety constraints  
- `read_file` - File reading with path validation

**What works now:**
- Tool execution and registry
- Provider format conversion (OpenAI, Anthropic, Ollama, MCP)
- Security validation

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

## Next Phase

Phase 2 will integrate tools with Ryan's Ollama client and TUI for real chat usage.