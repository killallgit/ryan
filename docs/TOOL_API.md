# Tool System API Reference

## Overview

Ryan's tool system provides a universal interface for LLM tool calling that works across all major providers. This document describes the public API for interacting with tools.

## Core Interfaces

### Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    JSONSchema() map[string]interface{}
    Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error)
}
```

**Methods:**
- `Name()` - Returns the unique identifier for the tool
- `Description()` - Returns human/LLM-readable description
- `JSONSchema()` - Returns JSON Schema for parameter validation
- `Execute()` - Runs the tool with provided parameters

### Registry Interface

```go
type Registry struct {
    // Private fields
}

func NewRegistry() *Registry
func (r *Registry) Register(tool Tool) error
func (r *Registry) Unregister(name string)
func (r *Registry) Get(name string) (Tool, bool)
func (r *Registry) List() []string
func (r *Registry) Execute(ctx context.Context, req ToolRequest) (ToolResult, error)
func (r *Registry) ExecuteAsync(ctx context.Context, req ToolRequest) <-chan ToolResult
func (r *Registry) RegisterBuiltinTools() error
```

## Data Types

### ToolRequest

```go
type ToolRequest struct {
    Name       string                 `json:"name"`
    Parameters map[string]interface{} `json:"parameters"`
    Context    context.Context        `json:"-"`
}
```

### ToolResult

```go
type ToolResult struct {
    Success  bool                   `json:"success"`
    Content  string                 `json:"content"`
    Data     map[string]interface{} `json:"data,omitempty"`
    Error    string                 `json:"error,omitempty"`
    Metadata ToolMetadata           `json:"metadata"`
}
```

### ToolMetadata

```go
type ToolMetadata struct {
    ExecutionTime time.Duration              `json:"execution_time"`
    StartTime     time.Time                  `json:"start_time"`
    EndTime       time.Time                  `json:"end_time"`
    ToolName      string                     `json:"tool_name"`
    Parameters    map[string]interface{}     `json:"parameters"`
}
```

## Built-in Tools

### BashTool

Executes shell commands with safety constraints.

**Name:** `execute_bash`

**Parameters:**
```json
{
  "command": "string (required) - The bash command to execute",
  "working_directory": "string (optional) - Working directory for execution"
}
```

**Example:**
```go
params := map[string]interface{}{
    "command": "echo 'Hello World'",
}

req := ToolRequest{
    Name:       "execute_bash",
    Parameters: params,
    Context:    context.Background(),
}

result, err := registry.Execute(context.Background(), req)
```

**Safety Features:**
- Forbidden command filtering (sudo, rm -rf /, etc.)
- Path restrictions to allowed directories
- Execution timeout (default 30s)
- Dangerous pattern detection

### FileReadTool

Reads file contents with security validation.

**Name:** `read_file`

**Parameters:**
```json
{
  "path": "string (required) - Path to the file to read",
  "start_line": "number (optional) - Starting line number (1-based)",
  "end_line": "number (optional) - Ending line number (1-based)"
}
```

**Example:**
```go
params := map[string]interface{}{
    "path":       "/path/to/file.txt",
    "start_line": float64(1),
    "end_line":   float64(10),
}

req := ToolRequest{
    Name:       "read_file", 
    Parameters: params,
    Context:    context.Background(),
}

result, err := registry.Execute(context.Background(), req)
```

**Safety Features:**
- Path traversal protection
- File extension whitelisting
- File size limits (default 10MB)
- Line count limits (default 10k lines)
- UTF-8 validation

## Provider Format Conversion

### Supported Providers

- **OpenAI** - Function calling format
- **Anthropic** - Tool use format
- **Ollama** - OpenAI compatible format
- **MCP** - Model Context Protocol format

### Converting Tools

```go
// Convert single tool to provider format
definition, err := ConvertToProvider(tool, "openai")

// Convert multiple tools
tools := []Tool{bashTool, fileReadTool}
definitions, err := BatchConvertToProvider(tools, "anthropic")

// Get all tools for a provider from registry
definitions, err := registry.GetDefinitions("mcp")
```

### Format Examples

**OpenAI Format:**
```json
{
  "type": "function",
  "function": {
    "name": "execute_bash",
    "description": "Execute a bash command...",
    "parameters": {
      "type": "object",
      "properties": {
        "command": {
          "type": "string",
          "description": "The bash command to execute"
        }
      },
      "required": ["command"]
    }
  }
}
```

**Anthropic Format:**
```json
{
  "name": "execute_bash",
  "description": "Execute a bash command...",
  "input_schema": {
    "type": "object",
    "properties": {
      "command": {
        "type": "string",
        "description": "The bash command to execute"
      }
    },
    "required": ["command"]
  }
}
```

**MCP Format:**
```json
{
  "name": "execute_bash",
  "description": "Execute a bash command...",
  "inputSchema": {
    "type": "object",
    "properties": {
      "command": {
        "type": "string",
        "description": "The bash command to execute"
      }
    },
    "required": ["command"]
  },
  "type": "tool"
}
```

## Error Handling

### ToolError Type

```go
type ToolError struct {
    ToolName string
    Message  string
    Cause    error
}

func (e ToolError) Error() string
func (e ToolError) Unwrap() error
```

### Common Error Scenarios

1. **Tool Not Found**
   ```go
   result, err := registry.Execute(ctx, ToolRequest{Name: "nonexistent"})
   // err will be ToolError with "tool not found" message
   // result.Success will be false
   ```

2. **Parameter Validation**
   ```go
   // Missing required parameter
   result, err := registry.Execute(ctx, ToolRequest{
       Name: "execute_bash",
       Parameters: map[string]interface{}{}, // missing "command"
   })
   // result.Success will be false
   // result.Error will contain validation message
   ```

3. **Security Violations**
   ```go
   params := map[string]interface{}{
       "command": "sudo rm -rf /", // forbidden command
   }
   
   result, err := registry.Execute(ctx, req)
   // result.Success will be false
   // result.Error will contain security violation message
   ```

## Usage Patterns

### Basic Tool Execution

```go
// Create registry and register tools
registry := tools.NewRegistry()
registry.RegisterBuiltinTools()

// Execute a tool
req := tools.ToolRequest{
    Name: "execute_bash",
    Parameters: map[string]interface{}{
        "command": "ls -la",
    },
    Context: context.Background(),
}

result, err := registry.Execute(context.Background(), req)
if err != nil {
    log.Printf("Tool execution error: %v", err)
    return
}

if result.Success {
    fmt.Printf("Output: %s", result.Content)
} else {
    fmt.Printf("Tool failed: %s", result.Error)
}
```

### Asynchronous Execution

```go
// Execute tool asynchronously
resultChan := registry.ExecuteAsync(context.Background(), req)

// Handle result when available
select {
case result := <-resultChan:
    if result.Success {
        fmt.Printf("Async result: %s", result.Content)
    }
case <-time.After(30 * time.Second):
    fmt.Println("Tool execution timed out")
}
```

### Provider Integration

```go
// Get tools in OpenAI format for API request
definitions, err := registry.GetDefinitions("openai")
if err != nil {
    log.Fatal(err)
}

// Use definitions in API request to OpenAI
apiRequest := OpenAIRequest{
    Model: "gpt-4",
    Messages: messages,
    Tools: definitions, // Provider-specific format
}
```

## Custom Tool Development

### Implementing the Tool Interface

```go
type CustomTool struct {
    // Tool-specific fields
}

func (ct *CustomTool) Name() string {
    return "custom_tool"
}

func (ct *CustomTool) Description() string {
    return "A custom tool that does something specific"
}

func (ct *CustomTool) JSONSchema() map[string]interface{} {
    schema := tools.NewJSONSchema()
    tools.AddProperty(schema, "param1", tools.JSONSchemaProperty{
        Type:        "string",
        Description: "First parameter",
    })
    tools.AddRequired(schema, "param1")
    return schema
}

func (ct *CustomTool) Execute(ctx context.Context, params map[string]interface{}) (tools.ToolResult, error) {
    startTime := time.Now()
    
    // Validate parameters
    param1, ok := params["param1"].(string)
    if !ok {
        return tools.ToolResult{
            Success: false,
            Error:   "param1 is required and must be a string",
            Metadata: tools.ToolMetadata{
                ExecutionTime: time.Since(startTime),
                StartTime:     startTime,
                EndTime:       time.Now(),
                ToolName:      ct.Name(),
                Parameters:    params,
            },
        }, nil
    }
    
    // Tool implementation here
    result := doSomething(param1)
    
    return tools.ToolResult{
        Success: true,
        Content: result,
        Metadata: tools.ToolMetadata{
            ExecutionTime: time.Since(startTime),
            StartTime:     startTime,
            EndTime:       time.Now(),
            ToolName:      ct.Name(),
            Parameters:    params,
        },
    }, nil
}

// Register the custom tool
customTool := &CustomTool{}
err := registry.Register(customTool)
```

## Security Considerations

### Built-in Safety Features

1. **Command Validation** - Dangerous commands are automatically blocked
2. **Path Restrictions** - File access is limited to allowed directories  
3. **Resource Limits** - Execution time and memory usage are bounded
4. **Input Sanitization** - All parameters are validated before execution

### Best Practices

1. **Always validate parameters** in custom tools
2. **Use context for cancellation** and timeout handling
3. **Implement proper error handling** with descriptive messages
4. **Follow the principle of least privilege** when accessing system resources
5. **Log security-sensitive operations** for audit purposes

## Performance Considerations

### Execution Timing

Tools track detailed execution metrics:
- Queue time (time waiting for execution)
- Execution time (actual tool runtime)
- Total duration (end-to-end)

### Concurrent Execution

- Tools can execute concurrently using `ExecuteAsync`
- Registry is thread-safe for concurrent access
- Individual tools should be designed to be stateless

### Resource Management

- File size limits prevent memory exhaustion
- Execution timeouts prevent runaway processes
- Path restrictions limit file system access

---

*This API reference is maintained alongside the tool system implementation and updated with each release.*