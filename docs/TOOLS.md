# Tool System Guide

✅ **Status**: Production ready with Docker tool calling validated

## Overview

Ryan's tool system provides secure, multi-provider LLM tool calling that works with OpenAI, Anthropic, Ollama, and MCP. The system enables AI assistants to execute commands, read files, and interact with the system safely while maintaining a responsive streaming experience.

## 🚀 Quick Start

### Prerequisites
- Ollama v0.4.0+ for tool calling support
- Compatible model (qwen3, deepseek-r1, llama3.1, mistral-small)
- Appropriate system permissions for tool execution

### Basic Usage

1. **Enable tools in configuration**:
```yaml
# ~/.ryan/settings.yaml
tools:
  enabled: true
  bash:
    enabled: true
    timeout: "90s"
  file_read:
    enabled: true
    max_file_size: "10MB"
```

2. **Example interaction**:
```
User: How many docker images are on the system?
Assistant: I'll check the Docker images for you.

[Tool Call: execute_bash]
Command: docker images | wc -l

[Tool Result: 34]

You have 34 Docker images on your system.
```

## Available Tools

### 1. Execute Bash (`execute_bash`)

Executes shell commands with comprehensive safety constraints.

**Parameters**:
- `command` (required): The bash command to execute
- `working_directory` (optional): Directory to run the command in

**Safety Features**:
- Forbidden command detection (sudo, rm -rf, etc.)
- Path restrictions to allowed directories
- Configurable timeout limits (default: 90s)
- Output size limitations

**Examples**:
```json
{
  "name": "execute_bash",
  "parameters": {
    "command": "docker images | wc -l"
  }
}

{
  "name": "execute_bash", 
  "parameters": {
    "command": "ls -la",
    "working_directory": "/tmp"
  }
}
```

### 2. Read File (`read_file`)

Reads text files with safety constraints and encoding detection.

**Parameters**:
- `path` (required): Path to the file to read
- `start_line` (optional): Line number to start reading from
- `end_line` (optional): Line number to stop reading at

**Safety Features**:
- File extension filtering (configurable whitelist)
- File size limits (default: 10MB)
- Path restrictions to allowed directories
- UTF-8 validation
- Line count limits (default: 10k lines)

**Examples**:
```json
{
  "name": "read_file",
  "parameters": {
    "path": "./README.md"
  }
}

{
  "name": "read_file",
  "parameters": {
    "path": "./config/settings.yaml",
    "start_line": 1,
    "end_line": 50
  }
}
```

## Configuration

### Settings File

```yaml
tools:
  enabled: true                    # Enable/disable tool calling
  truncate_output: true           # Truncate long tool outputs
  models:                         # Models that support tool calling
    - qwen3:latest
    - deepseek-r1:latest
    - llama3.1:latest
    - mistral-small:latest
  
  bash:
    enabled: true
    timeout: "90s"
    allowed_paths: [".", "/tmp", "~/"]
    forbidden_commands: ["sudo", "rm -rf", "dd", "mkfs"]
  
  file_read:
    enabled: true
    max_file_size: "10MB"
    max_lines: 10000
    allowed_extensions: [".txt", ".md", ".go", ".json", ".yaml", ".yml"]
    allowed_paths: [".", "/tmp", "~/"]
```

### Environment Variables

- `RYAN_TOOLS_ENABLED` - Override tool enabling (true/false)
- `RYAN_TOOLS_TIMEOUT` - Default timeout for tool execution
- `RYAN_TOOLS_DEBUG` - Enable debug logging for tool execution

## Provider Support

### Multi-Provider Architecture

Ryan's tool system uses a universal adapter pattern that supports multiple LLM providers through format conversion:

```go
// Universal tool interface
type Tool interface {
    Name() string
    Description() string
    JSONSchema() map[string]interface{}
    Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
}

// Provider-specific adapters
func ConvertToProvider(tool Tool, provider string) (map[string]interface{}, error)
```

### Supported Providers

| Provider | Format | Status | Notes |
|----------|--------|--------|-------|
| **Ollama** | OpenAI-compatible | ✅ Production | Primary integration |
| **OpenAI** | Native function calling | ✅ Compatible | Full API support |
| **Anthropic** | Native tool calling | ✅ Compatible | Claude integration |
| **MCP** | JSON-RPC protocol | 🚧 Planned | Model Context Protocol |

## Security & Safety

### Command Execution Safety

1. **Forbidden Commands**: Automatic detection and blocking of dangerous commands
2. **Path Restrictions**: Commands only execute in allowed directories
3. **Timeout Protection**: Configurable execution time limits
4. **Output Limits**: Prevent memory exhaustion from large outputs
5. **Resource Monitoring**: Track execution resource usage

### File Access Safety

1. **Extension Filtering**: Only allowed file types can be read
2. **Size Limits**: Prevent reading extremely large files
3. **Path Validation**: Files must be in allowed directories
4. **Content Validation**: UTF-8 validation prevents binary file issues

### Security Best Practices

- Use principle of least privilege for path restrictions
- Regularly review forbidden command lists
- Monitor tool execution logs for suspicious activity
- Configure appropriate timeouts for your use case
- Enable truncation for long outputs in production

## API Reference

### Core Interfaces

```go
type Tool interface {
    Name() string
    Description() string
    JSONSchema() map[string]interface{}
    Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
}

type ToolResult struct {
    Success  bool                   `json:"success"`
    Content  string                 `json:"content"`
    Data     map[string]interface{} `json:"data,omitempty"`
    Error    string                 `json:"error,omitempty"`
    Metadata ToolMetadata           `json:"metadata"`
}

type ToolRegistry struct {
    // Thread-safe tool management
}

func (r *Registry) Register(tool Tool) error
func (r *Registry) Execute(ctx context.Context, req ToolRequest) (ToolResult, error)
func (r *Registry) GetDefinitions(provider string) ([]map[string]interface{}, error)
```

### Usage Examples

```go
// Register built-in tools
registry := tools.NewRegistry()
err := registry.RegisterBuiltinTools()

// Execute a tool
req := tools.ToolRequest{
    Name: "execute_bash",
    Parameters: map[string]interface{}{
        "command": "docker ps --format 'table {{.Names}}\\t{{.Status}}'",
    },
}

result, err := registry.Execute(context.Background(), req)
if result.Success {
    fmt.Printf("Output: %s", result.Content)
} else {
    fmt.Printf("Error: %s", result.Error)
}
```

## Development

### Adding New Tools

1. **Implement Tool Interface**:
```go
type MyTool struct {
    config MyToolConfig
}

func (mt *MyTool) Name() string { return "my_tool" }
func (mt *MyTool) Description() string { return "Description of what this tool does" }
func (mt *MyTool) JSONSchema() map[string]interface{} { /* return parameter schema */ }
func (mt *MyTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
    // Implementation with proper error handling and validation
}
```

2. **Register Tool**:
```go
registry := tools.NewRegistry()
err := registry.Register(NewMyTool(config))
```

3. **Add Configuration Support**:
Update `pkg/config/config.go` and `examples/settings.example.yaml`

### Testing Tools

```go
func TestMyTool(t *testing.T) {
    tool := NewMyTool(defaultConfig)
    
    // Test successful execution
    result, err := tool.Execute(context.Background(), map[string]interface{}{
        "param": "valid_value",
    })
    
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.NotEmpty(t, result.Content)
    
    // Test error conditions
    result, err = tool.Execute(context.Background(), map[string]interface{}{
        "param": "invalid_value",
    })
    
    assert.NoError(t, err) // No execution error
    assert.False(t, result.Success) // But tool operation failed
    assert.NotEmpty(t, result.Error)
}
```

## Troubleshooting

### Common Issues

1. **"Tool not found"**
   - Ensure tool is properly registered: `registry.RegisterBuiltinTools()`
   - Check tool name spelling in configuration

2. **"Model not supported"**
   - Use a model with tool calling support (qwen3, deepseek-r1, llama3.1)
   - Check Ollama version (requires v0.4.0+)

3. **"Command forbidden"**
   - Command blocked by safety filters
   - Review `forbidden_commands` configuration
   - Use alternative safe commands

4. **"File not allowed"**
   - File extension not in allowed list
   - File path outside allowed directories
   - Check `allowed_extensions` and `allowed_paths` configuration

5. **Tool timeout**
   - Increase timeout in configuration
   - Optimize command for faster execution
   - Check system resource availability

### Debug Mode

Enable detailed logging:
```yaml
logging:
  level: "debug"
  
tools:
  debug: true
```

Or via environment:
```bash
RYAN_LOG_LEVEL=debug RYAN_TOOLS_DEBUG=true ryan
```

### Performance Optimization

1. **Timeout Tuning**: Adjust based on expected execution time
2. **Output Truncation**: Enable for better TUI performance
3. **Path Restrictions**: Limit to necessary directories only
4. **Extension Filtering**: Minimize allowed file types

## Future Enhancements

### Planned Features (Phase 3B+)

1. **Additional Tools**:
   - WebFetch with caching and rate limiting
   - Enhanced Grep with ripgrep integration
   - Glob pattern matching
   - Git operations (status, commit, diff)
   - Directory operations (ls, mkdir, tree)
   - Process management (ps, kill, monitor)

2. **Advanced Execution**:
   - Batch tool execution ("multiple tools in single response")
   - Concurrent orchestration with dependency resolution
   - Real-time progress tracking in TUI
   - Tool result streaming

3. **Enhanced UX**:
   - User consent prompts for dangerous operations
   - Tool execution history and replay
   - Result caching and optimization
   - Custom tool development SDK

4. **Security Enhancements**:
   - Sandboxing and resource limits
   - Audit logging for all operations
   - Advanced permission systems
   - Runtime security monitoring

## Model Compatibility

### Tested Models (Tool Calling Support)

**Tier 1: Excellent Support**
- Llama 3.1 (8B, 70B, 405B)
- Llama 3.2 (1B, 3B, 11B, 90B)
- Qwen 2.5 (1.5B-72B)
- Qwen 3 (8B+)
- DeepSeek-R1

**Tier 2: Good Support**
- Mistral/Mistral-Nemo
- Command-R/Command-R-Plus
- Granite 3.x

For detailed compatibility testing and benchmarks, see [TESTING.md](TESTING.md).

---

For complete implementation details and advanced usage patterns, see the source code in `pkg/tools/`.