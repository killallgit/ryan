# Tool System Quick Start Guide

## 5-Minute Setup

### 1. Try the Demo

```bash
# Clone the repo and run the demo
go run examples/tool_demo.go
```

This will show you:
- Available tools (`execute_bash`, `read_file`)
- Tool execution examples
- Provider format conversion (OpenAI, Anthropic, MCP)

### 2. Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/killallgit/ryan/pkg/tools"
)

func main() {
    // Create registry and add built-in tools
    registry := tools.NewRegistry()
    if err := registry.RegisterBuiltinTools(); err != nil {
        log.Fatal(err)
    }
    
    // Execute a bash command
    req := tools.ToolRequest{
        Name: "execute_bash",
        Parameters: map[string]interface{}{
            "command": "echo 'Hello, World!'",
        },
        Context: context.Background(),
    }
    
    result, err := registry.Execute(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }
    
    if result.Success {
        fmt.Printf("Output: %s", result.Content)
    } else {
        fmt.Printf("Error: %s", result.Error)
    }
}
```

### 3. Read a File

```go
// Read the first 10 lines of a file
req := tools.ToolRequest{
    Name: "read_file",
    Parameters: map[string]interface{}{
        "path":       "README.md",
        "start_line": float64(1),
        "end_line":   float64(10),
    },
    Context: context.Background(),
}

result, err := registry.Execute(context.Background(), req)
```

## Provider Integration

### For OpenAI

```go
// Get tools in OpenAI format
definitions, err := registry.GetDefinitions("openai")

// Use in API request
request := map[string]interface{}{
    "model":    "gpt-4",
    "messages": messages,
    "tools":    definitions,
}
```

### For Anthropic

```go
// Get tools in Anthropic format  
definitions, err := registry.GetDefinitions("anthropic")

// Use in API request
request := map[string]interface{}{
    "model":     "claude-3-sonnet-20240229",
    "messages":  messages,
    "tools":     definitions,
    "max_tokens": 1000,
}
```

### For Ollama

```go
// Ollama uses OpenAI-compatible format
definitions, err := registry.GetDefinitions("ollama")

request := map[string]interface{}{
    "model":    "llama3.1:8b",
    "messages": messages,
    "tools":    definitions,
}
```

## Creating Custom Tools

### Simple Custom Tool

```go
type GreetingTool struct{}

func (gt *GreetingTool) Name() string {
    return "greet"
}

func (gt *GreetingTool) Description() string {
    return "Generate a personalized greeting"
}

func (gt *GreetingTool) JSONSchema() map[string]interface{} {
    schema := tools.NewJSONSchema()
    tools.AddProperty(schema, "name", tools.JSONSchemaProperty{
        Type:        "string",
        Description: "Name of the person to greet",
    })
    tools.AddRequired(schema, "name")
    return schema
}

func (gt *GreetingTool) Execute(ctx context.Context, params map[string]interface{}) (tools.ToolResult, error) {
    startTime := time.Now()
    
    name, ok := params["name"].(string)
    if !ok {
        return tools.ToolResult{
            Success: false,
            Error:   "name parameter is required",
            Metadata: tools.ToolMetadata{
                ExecutionTime: time.Since(startTime),
                StartTime:     startTime,
                EndTime:       time.Now(),
                ToolName:      gt.Name(),
                Parameters:    params,
            },
        }, nil
    }
    
    greeting := fmt.Sprintf("Hello, %s! Nice to meet you.", name)
    
    return tools.ToolResult{
        Success: true,
        Content: greeting,
        Metadata: tools.ToolMetadata{
            ExecutionTime: time.Since(startTime),
            StartTime:     startTime,
            EndTime:       time.Now(),
            ToolName:      gt.Name(),
            Parameters:    params,
        },
    }, nil
}

// Register the tool
greetingTool := &GreetingTool{}
err := registry.Register(greetingTool)
```

## Safety and Security

### Built-in Protections

The tool system includes comprehensive safety features:

**BashTool Safety:**
```go
// These commands are automatically blocked:
// - sudo, su, rm -rf /, dd, mkfs, fdisk
// - shutdown, reboot, halt, poweroff
// - Dangerous patterns like "rm -rf", "> /dev/"

// Safe paths only:
// - User home directory
// - /tmp
// - Current working directory
```

**FileReadTool Safety:**
```go
// Automatic protections:
// - Path traversal prevention (no ../../../etc/passwd)
// - File extension whitelist (.txt, .md, .go, .json, etc.)
// - File size limits (10MB default)
// - Line count limits (10k lines default)
// - UTF-8 validation (no binary files)
```

### Custom Tool Security

When creating custom tools, follow these practices:

```go
func (ct *CustomTool) Execute(ctx context.Context, params map[string]interface{}) (tools.ToolResult, error) {
    startTime := time.Now()
    
    // 1. Always validate parameters
    input, ok := params["input"].(string)
    if !ok || strings.TrimSpace(input) == "" {
        return tools.ToolResult{
            Success: false,
            Error:   "input parameter is required and cannot be empty",
            // ... metadata
        }, nil
    }
    
    // 2. Sanitize input
    input = strings.TrimSpace(input)
    if len(input) > 1000 {
        return tools.ToolResult{
            Success: false,
            Error:   "input too long (max 1000 characters)",
            // ... metadata  
        }, nil
    }
    
    // 3. Use context for cancellation
    select {
    case <-ctx.Done():
        return tools.ToolResult{
            Success: false,
            Error:   "operation cancelled",
            // ... metadata
        }, nil
    default:
    }
    
    // 4. Implement timeouts
    done := make(chan string, 1)
    go func() {
        // Do work here
        done <- result
    }()
    
    select {
    case result := <-done:
        return tools.ToolResult{Success: true, Content: result, /* ... */}, nil
    case <-time.After(30 * time.Second):
        return tools.ToolResult{
            Success: false,
            Error:   "operation timed out",
            // ... metadata
        }, nil
    }
}
```

## Testing Your Tools

### Unit Testing

```go
func TestCustomTool(t *testing.T) {
    tool := &CustomTool{}
    
    // Test successful execution
    params := map[string]interface{}{
        "param1": "test_value",
    }
    
    result, err := tool.Execute(context.Background(), params)
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.NotEmpty(t, result.Content)
    
    // Test parameter validation
    result, err = tool.Execute(context.Background(), map[string]interface{}{})
    assert.NoError(t, err) // Tool doesn't error, but result shows failure
    assert.False(t, result.Success)
    assert.Contains(t, result.Error, "required")
}
```

### Integration Testing

```go
func TestToolIntegration(t *testing.T) {
    registry := tools.NewRegistry()
    customTool := &CustomTool{}
    
    err := registry.Register(customTool)
    require.NoError(t, err)
    
    req := tools.ToolRequest{
        Name: "custom_tool",
        Parameters: map[string]interface{}{
            "param1": "test",
        },
        Context: context.Background(),
    }
    
    result, err := registry.Execute(context.Background(), req)
    assert.NoError(t, err)
    assert.True(t, result.Success)
}
```

## Next Steps

1. **Read the full documentation:**
   - [TOOL_SYSTEM.md](./TOOL_SYSTEM.md) - Architecture overview
   - [TOOL_API.md](./TOOL_API.md) - Complete API reference
   - [TOOL_REGISTRY_TODO.md](./TOOL_REGISTRY_TODO.md) - Future roadmap

2. **Explore examples:**
   - `examples/tool_demo.go` - Complete working example
   - `pkg/tools/tools_test.go` - Unit test examples

3. **Contribute:**
   - Follow Ryan's functional programming principles
   - Write tests first
   - Keep it simple and secure
   - Use the existing architecture patterns

## Common Patterns

### Error Handling

```go
// Always return ToolResult, even for errors
if err := validateInput(params); err != nil {
    return tools.ToolResult{
        Success: false,
        Error:   err.Error(),
        Metadata: createMetadata(startTime, toolName, params),
    }, nil // Note: return nil error, put error info in result
}
```

### Metadata Creation Helper

```go
func createMetadata(startTime time.Time, toolName string, params map[string]interface{}) tools.ToolMetadata {
    endTime := time.Now()
    return tools.ToolMetadata{
        ExecutionTime: endTime.Sub(startTime),
        StartTime:     startTime,
        EndTime:       endTime,
        ToolName:      toolName,
        Parameters:    params,
    }
}
```

### Async Pattern

```go
// For long-running operations
resultChan := registry.ExecuteAsync(context.Background(), req)

// With timeout
select {
case result := <-resultChan:
    // Handle result
case <-time.After(60 * time.Second):
    // Handle timeout
}
```

---

*Ready to build amazing code co-pilot tools with Ryan!*