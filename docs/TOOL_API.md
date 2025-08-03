# Tool System API Reference

✅ **Status**: Production ready with Docker tool calling validated

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
- `JSONSchema()` - Returns parameter schema for provider conversion
- `Execute()` - Executes the tool with context and parameters

### ToolResult Structure

```go
type ToolResult struct {
    Success  bool                   `json:"success"`
    Content  string                 `json:"content"`
    Data     map[string]interface{} `json:"data,omitempty"`
    Error    string                 `json:"error,omitempty"`
    Metadata ToolMetadata           `json:"metadata"`
}
```
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
func (r *Registry) ExecuteBatch(ctx context.Context, reqs []ToolRequest) (BatchResult, error)
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

### BatchResult

```go
type BatchResult struct {
    Results   map[string]ToolResult `json:"results"`
    Errors    map[string]error      `json:"errors"`
    StartTime time.Time             `json:"start_time"`
    EndTime   time.Time             `json:"end_time"`
    Duration  time.Duration         `json:"duration"`
}
```

## Built-in Tools (Phase 3A: Basic Tools)

### BashTool (`execute_bash`)
- Parameters: `command` (required), `working_directory` (optional)
- Safety: Blocks dangerous commands, restricts paths, 30s timeout

### FileReadTool (`read_file`)  
- Parameters: `path` (required), `start_line`/`end_line` (optional)
- Safety: Path validation, extension whitelist, size limits

## Advanced Features (Phase 3B+)

### Batch Execution
Execute multiple tools concurrently with dependency resolution:

```go
requests := []ToolRequest{
    {Name: "read_file", Parameters: map[string]interface{}{"path": "config.json"}},
    {Name: "execute_bash", Parameters: map[string]interface{}{"command": "ls -la"}},
}

batchResult, err := registry.ExecuteBatch(ctx, requests)
```

### Progress Tracking
Monitor tool execution progress in real-time:

```go
// Progress updates sent via channels during execution
type ProgressUpdate struct {
    ToolID   string
    Progress float64  // 0.0 to 1.0
    Message  string
    Status   ProgressStatus
}
```

## Provider Format Conversion

Supports OpenAI, Anthropic, Ollama formats with universal adapter:

```go
// Get tools in provider-specific format
definitions, err := registry.GetDefinitions("openai")
definitions, err := registry.GetDefinitions("anthropic") 
definitions, err := registry.GetDefinitions("ollama")
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

## Usage Example

```go
registry := tools.NewRegistry()
registry.RegisterBuiltinTools()

req := tools.ToolRequest{
    Name: "execute_bash",
    Parameters: map[string]interface{}{
        "command": "ls -la",
    },
}

result, err := registry.Execute(context.Background(), req)
if result.Success {
    fmt.Printf("Output: %s", result.Content)
}
```

For complete implementation details, see `pkg/tools/` source code.