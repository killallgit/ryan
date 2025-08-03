# Tool Calling Integration Guide

## Overview

The Ryan chat application includes a sophisticated tool calling system that allows LLMs to execute commands and interact with the system safely. This guide explains how tool calling works, how to configure it, and provides usage examples.

## Architecture

### Core Components

1. **Tool Interface**: Standardized interface for all tools
2. **Tool Registry**: Thread-safe registry for managing available tools
3. **Provider Adapters**: Convert tool definitions for different LLM providers
4. **Chat Integration**: Seamless integration with streaming chat responses
5. **Safety Framework**: Security constraints and execution limits

### Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    JSONSchema() map[string]interface{}
    Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
}
```

## Available Tools

### 1. Execute Bash (`execute_bash`)

Executes shell commands with safety constraints.

**Parameters**:
- `command` (required): The bash command to execute
- `working_directory` (optional): Directory to run the command in

**Safety Features**:
- Forbidden command detection (sudo, rm -rf, etc.)
- Path restrictions to allowed directories
- Configurable timeout limits
- Output size limitations

**Example Usage**:
```json
{
  "name": "execute_bash",
  "parameters": {
    "command": "docker images | wc -l"
  }
}
```

### 2. Read File (`read_file`)

Reads text files with safety constraints.

**Parameters**:
- `path` (required): Path to the file to read
- `start_line` (optional): Line number to start reading from
- `end_line` (optional): Line number to stop reading at

**Safety Features**:
- File extension filtering
- File size limits (default: 10MB)
- Path restrictions to allowed directories
- UTF-8 validation
- Line count limits (default: 10k lines)

**Example Usage**:
```json
{
  "name": "read_file",
  "parameters": {
    "path": "./README.md",
    "start_line": 1,
    "end_line": 50
  }
}
```

## Configuration

### Settings File (`.ryan/settings.yaml`)

```yaml
tools:
  enabled: true                    # Enable/disable tool calling
  truncate_output: true           # Truncate long tool outputs
  models:                         # Models that support tool calling
    - qwen3:latest
    - deepseek-r1:latest
    - llama3.1:latest
  
  bash:
    enabled: true
    timeout: "90s"
    allowed_paths: [".", "/tmp"]
  
  file_read:
    enabled: true
    max_file_size: "10MB"
    allowed_extensions: [".txt", ".md", ".go", ".json"]
```

### Environment Requirements

1. **Ollama Version**: Requires Ollama v0.4.0+ for tool calling support
2. **Model Compatibility**: Use models that support function calling
3. **System Access**: Appropriate permissions for file/command execution

## Usage Examples

### Example 1: Docker System Information

**User Question**: "How many docker images are on the system?"

**Tool Call Flow**:
1. LLM receives question and determines it needs to execute a command
2. LLM calls `execute_bash` tool with `docker images | wc -l`
3. Tool executes command safely and returns count
4. LLM incorporates result into natural language response

**Expected Response**: "You have 34 Docker images on your system."

### Example 2: Code Analysis

**User Question**: "What's in the main.go file?"

**Tool Call Flow**:
1. LLM calls `read_file` tool with `{"path": "./main.go"}`
2. Tool reads file contents with safety checks
3. LLM analyzes the code and provides summary

### Example 3: System Monitoring

**User Question**: "Show me the current disk usage"

**Tool Call Flow**:
1. LLM calls `execute_bash` with `df -h`
2. Tool executes and returns disk usage information
3. LLM formats the output in a user-friendly way

## Provider Support

### Ollama (Primary)
- **Format**: OpenAI-compatible function calling
- **Models**: qwen3, deepseek-r1, llama3.1, mistral-small
- **Features**: Full tool calling support with streaming

### OpenAI (Compatible)
- **Format**: Native OpenAI function calling
- **Models**: GPT-4, GPT-3.5-turbo with function calling
- **Features**: Complete compatibility

### Anthropic (Compatible)
- **Format**: Native Anthropic tool calling
- **Models**: Claude models with tool support
- **Features**: Full integration support

## Safety and Security

### Command Execution Safety

1. **Forbidden Commands**: Automatic detection of dangerous commands
2. **Path Restrictions**: Commands only execute in allowed directories
3. **Timeout Protection**: Configurable execution time limits
4. **Output Limits**: Prevent memory exhaustion from large outputs

### File Access Safety

1. **Extension Filtering**: Only allowed file types can be read
2. **Size Limits**: Prevent reading extremely large files
3. **Path Validation**: Files must be in allowed directories
4. **Content Validation**: UTF-8 validation prevents binary file reading

### Resource Management

1. **Context Cancellation**: All tool executions are cancellable
2. **Memory Limits**: Built-in protection against memory leaks
3. **Concurrent Execution**: Safe parallel tool execution
4. **Error Recovery**: Graceful handling of tool failures

## Development

### Adding New Tools

1. **Implement Tool Interface**:
```go
type MyTool struct {
    // Tool configuration
}

func (mt *MyTool) Name() string { return "my_tool" }
func (mt *MyTool) Description() string { return "Description of what this tool does" }
func (mt *MyTool) JSONSchema() map[string]interface{} { /* return schema */ }
func (mt *MyTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
    // Implementation
}
```

2. **Register Tool**:
```go
registry := tools.NewRegistry()
err := registry.Register(NewMyTool())
```

3. **Add Configuration Support** (optional):
Update `pkg/config/config.go` and `examples/settings.example.yaml`

### Testing Tools

1. **Unit Tests**: Test tool execution in isolation
2. **Integration Tests**: Test with actual chat controllers
3. **Safety Tests**: Validate security constraints
4. **Provider Tests**: Test with different LLM providers

## Troubleshooting

### Common Issues

1. **"Tool not found"**: Ensure tool is properly registered
2. **"Model not supported"**: Use a model with tool calling support
3. **"Command forbidden"**: Command blocked by safety filters
4. **"File not allowed"**: File extension or path not permitted

### Debugging

1. **Enable Debug Logging**: Set log level to debug in configuration
2. **Check Tool Registry**: Verify tools are properly loaded
3. **Test Tool Directly**: Use direct tool execution for testing
4. **Validate Configuration**: Ensure all required settings are present

### Performance Optimization

1. **Timeout Tuning**: Adjust timeouts based on use case
2. **Output Truncation**: Enable for better performance
3. **Concurrent Limits**: Configure based on system resources
4. **Caching**: Consider result caching for repeated operations

## Best Practices

1. **Use Specific Commands**: Provide clear, specific commands for better results
2. **Handle Errors Gracefully**: Always check tool execution results
3. **Security First**: Follow principle of least privilege
4. **Test Thoroughly**: Validate tool behavior in different scenarios
5. **Monitor Performance**: Track tool execution times and resource usage

## Future Enhancements

Planned improvements include:

1. **Additional Tools**: Git operations, network tools, process management
2. **Enhanced Security**: User consent mechanisms, sandboxing
3. **Performance**: Result caching, batch operations
4. **UX Improvements**: Progress indicators, cancellation support
5. **Provider Expansion**: Support for additional LLM providers