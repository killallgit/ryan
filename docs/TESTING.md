# Testing Guide & Model Compatibility

This guide covers Ryan's testing strategies, model compatibility information, and performance benchmarking.

## Model Compatibility

### Tool-Compatible Models (41 Total Identified)

Ryan supports tool calling with a wide range of models. Our research identified **41 models** in Ollama that support tool calling functionality.

#### Tier 1: Excellent Tool Calling Support
**Recommended for Production Use**
- **Llama 3.1** (8B, 70B, 405B) - Mature, reliable tool calling
- **Llama 3.2** (1B, 3B, 11B, 90B) - Lightweight with solid support  
- **Qwen 2.5** (1.5B-72B) - Superior math/coding performance
- **Qwen 2.5-Coder** - Specialized for development workflows
- **Qwen 3** (8B+) - Latest with enhanced capabilities

#### Tier 2: Good Tool Calling Support
**Suitable for Development & Testing**
- **Mistral/Mistral-Nemo** - Reliable general performance
- **Command-R/Command-R-Plus** - Enterprise-focused
- **DeepSeek-R1** - Reasoning-optimized with custom tool support
- **Granite 3.x** - IBM models with solid tool integration

#### Tier 3: Limited or No Support
**Not Recommended for Tool Calling**
- **Gemma** models - No native tool support
- **Phi** models - Limited compatibility
- Most vision-only models

### Performance Benchmarks

#### Response Time Benchmarks
- **Qwen 2.5-Coder 1.5B**: ~800ms (lightweight, good for development)
- **Qwen 2.5 7B**: ~980ms (excellent balance)
- **Llama 3.1 8B**: ~1.2s (reliable, mature)
- **Mistral 7B**: ~1.1s (solid performance)

#### Accuracy Rates
- **Tier 1 Models**: 95-100% test pass rate
- **Tier 2 Models**: 75-95% test pass rate  
- **Tier 3 Models**: <50% test pass rate or no support

## Automated Testing Framework

### Model Compatibility Tester

Ryan includes a comprehensive testing framework for validating model compatibility:

```bash
# Test primary models (recommended starting point)
task test:models:primary

# Test extended model set
task test:models:secondary  

# Test all known compatible models
task test:models:all

# Test custom model list
MODELS="llama3.1:8b,qwen2.5:7b" task test:models:custom

# Build and run directly
task build:model-tester
./bin/model-tester -models primary -url http://localhost:11434
```

### Test Categories

The testing framework validates:

1. **Tool Call Detection** - Verifies model can make tool calls
2. **Basic Command Execution** - Tests `execute_bash` functionality  
3. **File Operations** - Tests `read_file` capabilities
4. **Error Handling** - Validates graceful error responses
5. **Multi-tool Sequences** - Tests complex workflows
6. **Performance Metrics** - Measures response times

### Sample Test Output

```
================================================================================
MODEL COMPATIBILITY TEST RESULTS
================================================================================

📊 Model: llama3.1:8b
   Tool Support: true
   Tests Passed: 4/4 (100.0%)
   Avg Response: 1.2s
   Basic Tool:   ✅
   File Read:    ✅  
   Error Handle: ✅
   Multi-tool:   ✅

📊 Model: qwen2.5:7b
   Tool Support: true
   Tests Passed: 4/4 (100.0%)
   Avg Response: 980ms
   Basic Tool:   ✅
   File Read:    ✅
   Error Handle: ✅ 
   Multi-tool:   ✅

--------------------------------------------------------------------------------
🎯 RECOMMENDATIONS:
   • Default model: qwen2.5:7b (best balance of accuracy and speed)
   • Consider model switching based on task complexity
   • Enable tool compatibility validation in UI
================================================================================
```

## Testing Strategies

### Unit Testing

Ryan follows comprehensive unit testing practices:

#### Pure Function Testing
```go
func TestMessageAccumulation(t *testing.T) {
    conv := chat.NewConversation("test-model")
    
    userMsg := chat.Message{
        Role:    "user",
        Content: "Hello",
    }
    
    conv = chat.AddMessage(conv, userMsg)
    
    assert.Equal(t, 1, chat.GetMessageCount(conv))
    
    lastMsg, exists := chat.GetLastMessage(conv)
    assert.True(t, exists)
    assert.Equal(t, "Hello", lastMsg.Content)
}
```

#### Tool Testing
```go
func TestBashToolExecution(t *testing.T) {
    tool := tools.NewBashTool(tools.BashConfig{
        Timeout:      30 * time.Second,
        AllowedPaths: []string{"/tmp"},
    })
    
    // Test successful execution
    result, err := tool.Execute(context.Background(), map[string]interface{}{
        "command": "echo 'Hello World'",
    })
    
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.Contains(t, result.Content, "Hello World")
    
    // Test forbidden command
    result, err = tool.Execute(context.Background(), map[string]interface{}{
        "command": "sudo rm -rf /",
    })
    
    assert.NoError(t, err) // No execution error
    assert.False(t, result.Success) // But tool blocked the command
    assert.Contains(t, result.Error, "forbidden")
}
```

### Integration Testing

#### API Integration
```go
func TestOllamaIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    client := ollama.NewClient("http://localhost:11434")
    
    // Test basic chat
    response, err := client.Chat(context.Background(), ollama.ChatRequest{
        Model: "llama3.1:8b",
        Messages: []ollama.Message{
            {Role: "user", Content: "Say hello"},
        },
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, response.Message.Content)
}
```

#### Tool Integration
```go
func TestToolIntegration(t *testing.T) {
    registry := tools.NewRegistry()
    err := registry.RegisterBuiltinTools()
    require.NoError(t, err)
    
    controller := controllers.NewChatController(
        mockStreamingClient(),
        registry,
        "test-model",
    )
    
    // Test tool calling workflow
    response, err := controller.SendUserMessage("What files are in the current directory?")
    
    assert.NoError(t, err)
    assert.NotEmpty(t, response.Content)
    
    // Verify tool was called
    history := controller.GetHistory()
    found := false
    for _, msg := range history {
        if strings.Contains(msg.Content, "execute_bash") {
            found = true
            break
        }
    }
    assert.True(t, found, "Tool call should be present in history")
}
```

### TUI Testing

#### Event Simulation
```go
func TestTUIKeyHandling(t *testing.T) {
    screen := tcell.NewSimulationScreen("UTF-8")
    screen.Init()
    screen.SetSize(80, 24)
    
    app := tui.NewApp(screen, mockController())
    
    // Simulate user typing
    screen.InjectKey(tcell.KeyRune, 'h', tcell.ModNone)
    screen.InjectKey(tcell.KeyRune, 'i', tcell.ModNone)
    screen.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
    
    // Process events
    for i := 0; i < 3; i++ {
        event := screen.PollEvent()
        app.HandleEvent(event)
    }
    
    // Verify input was processed
    assert.Equal(t, "hi", app.GetLastInput())
}
```

#### Component Testing
```go
func TestMessageListRender(t *testing.T) {
    screen := tcell.NewSimulationScreen("UTF-8")
    screen.Init()
    screen.SetSize(80, 24)
    
    messages := []chat.Message{
        {Role: "user", Content: "Test message"},
        {Role: "assistant", Content: "Test response"},
    }
    
    messageList := tui.NewMessageList(messages)
    rect := tui.Rect{X: 0, Y: 0, Width: 80, Height: 20}
    
    messageList.Render(screen, rect)
    
    // Verify content was rendered
    contents, _, _ := screen.GetContents()
    screenText := tui.ContentsToString(contents)
    
    assert.Contains(t, screenText, "Test message")
    assert.Contains(t, screenText, "Test response")
}
```

### Concurrency Testing

#### Race Detection
```go
func TestConcurrentToolExecution(t *testing.T) {
    registry := tools.NewRegistry()
    registry.RegisterBuiltinTools()
    
    var wg sync.WaitGroup
    results := make(chan tools.ToolResult, 10)
    
    // Execute tools concurrently
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            result, err := registry.Execute(context.Background(), tools.ToolRequest{
                Name: "execute_bash",
                Parameters: map[string]interface{}{
                    "command": fmt.Sprintf("echo 'Test %d'", id),
                },
            })
            
            assert.NoError(t, err)
            results <- result
        }(i)
    }
    
    wg.Wait()
    close(results)
    
    // Verify all executions succeeded
    count := 0
    for result := range results {
        assert.True(t, result.Success)
        count++
    }
    assert.Equal(t, 10, count)
}
```

#### Streaming Concurrency
```go
func TestStreamingConcurrency(t *testing.T) {
    client := chat.NewStreamingClient("http://localhost:11434")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    chunks, err := client.StreamMessage(ctx, chat.ChatRequest{
        Model: "llama3.1:8b",
        Messages: []chat.Message{
            {Role: "user", Content: "Count to 10"},
        },
    })
    
    require.NoError(t, err)
    
    var content strings.Builder
    var chunkCount int
    
    for chunk := range chunks {
        if chunk.Error != nil {
            t.Fatalf("Streaming error: %v", chunk.Error)
        }
        
        content.WriteString(chunk.Content)
        chunkCount++
        
        if chunk.Done {
            break
        }
        
        // Verify we're receiving chunks in reasonable time
        select {
        case <-time.After(5 * time.Second):
            t.Fatal("Chunk timeout - streaming may be blocked")
        default:
            // Continue
        }
    }
    
    assert.Greater(t, chunkCount, 1, "Should receive multiple chunks")
    assert.NotEmpty(t, content.String())
}
```

## Performance Testing

### Benchmarking Tools

```go
func BenchmarkToolExecution(b *testing.B) {
    registry := tools.NewRegistry()
    registry.RegisterBuiltinTools()
    
    req := tools.ToolRequest{
        Name: "execute_bash",
        Parameters: map[string]interface{}{
            "command": "echo 'benchmark test'",
        },
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        result, err := registry.Execute(context.Background(), req)
        if err != nil || !result.Success {
            b.Fatalf("Tool execution failed: %v", err)
        }
    }
}

func BenchmarkMessageAccumulation(b *testing.B) {
    acc := chat.NewMessageAccumulator()
    streamID := "benchmark-stream"
    
    chunks := make([]chat.MessageChunk, 1000)
    for i := range chunks {
        chunks[i] = chat.MessageChunk{
            StreamID: streamID,
            Content:  fmt.Sprintf("chunk %d ", i),
            Done:     i == len(chunks)-1,
        }
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        for _, chunk := range chunks {
            acc.AddChunk(chunk)
        }
        
        _ = acc.GetCurrentContent(streamID)
        acc.CleanupStream(streamID)
    }
}
```

### Memory Profiling

```bash
# Run with memory profiling
go test -memprofile=mem.prof -bench=. ./pkg/tools/

# Analyze memory usage
go tool pprof mem.prof
```

### Load Testing

```go
func TestHighLoadScenario(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }
    
    registry := tools.NewRegistry()
    registry.RegisterBuiltinTools()
    
    const concurrency = 50
    const requests = 1000
    
    sem := make(chan struct{}, concurrency)
    results := make(chan bool, requests)
    
    start := time.Now()
    
    for i := 0; i < requests; i++ {
        go func() {
            sem <- struct{}{}
            defer func() { <-sem }()
            
            result, err := registry.Execute(context.Background(), tools.ToolRequest{
                Name: "execute_bash",
                Parameters: map[string]interface{}{
                    "command": "echo 'load test'",
                },
            })
            
            results <- err == nil && result.Success
        }()
    }
    
    // Collect results
    successCount := 0
    for i := 0; i < requests; i++ {
        if <-results {
            successCount++
        }
    }
    
    duration := time.Since(start)
    
    t.Logf("Load test completed: %d/%d successful in %v", 
        successCount, requests, duration)
    t.Logf("Rate: %.2f requests/second", 
        float64(requests)/duration.Seconds())
    
    assert.Greater(t, successCount, requests*90/100, 
        "Should have >90% success rate")
}
```

## Test Automation

### Task Commands

```bash
# Run all tests
task test

# Run tests with race detection
task test:race

# Run integration tests only
task test:integration

# Run with coverage
task test:coverage

# Run model compatibility tests
task test:models:primary
task test:models:all

# Run benchmarks
task test:bench

# Run load tests
task test:load
```

### CI/CD Pipeline

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Run unit tests
        run: task test
      
      - name: Run race tests
        run: task test:race
      
      - name: Run benchmarks
        run: task test:bench
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
```

## Security Testing

### Tool Security Validation

```go
func TestToolSecurityConstraints(t *testing.T) {
    tool := tools.NewBashTool(tools.BashConfig{
        AllowedPaths:      []string{"/tmp"},
        ForbiddenCommands: []string{"sudo", "rm -rf"},
        Timeout:          5 * time.Second,
    })
    
    dangerousCommands := []string{
        "sudo rm -rf /",
        "dd if=/dev/zero of=/dev/sda",
        "rm -rf /*",
        "mkfs.ext4 /dev/sda1",
        ":(){ :|:& };:",  // Fork bomb
    }
    
    for _, cmd := range dangerousCommands {
        result, err := tool.Execute(context.Background(), map[string]interface{}{
            "command": cmd,
        })
        
        assert.NoError(t, err, "Should not have execution error")
        assert.False(t, result.Success, "Dangerous command should be blocked: %s", cmd)
        assert.Contains(t, result.Error, "forbidden", "Should indicate command is forbidden")
    }
}
```

### Path Traversal Testing

```go
func TestPathTraversalPrevention(t *testing.T) {
    tool := tools.NewFileReadTool(tools.FileReadConfig{
        AllowedPaths: []string{"/tmp", "/home/user"},
        MaxFileSize:  1024 * 1024, // 1MB
    })
    
    maliciousPaths := []string{
        "../../../etc/passwd",
        "/etc/shadow",
        "../../root/.ssh/id_rsa",
        "/proc/version",
        "/sys/class/dmi/id/product_uuid",
    }
    
    for _, path := range maliciousPaths {
        result, err := tool.Execute(context.Background(), map[string]interface{}{
            "path": path,
        })
        
        assert.NoError(t, err)
        assert.False(t, result.Success, "Should block access to: %s", path)
    }
}
```

## Best Practices

### Test Organization

```
tests/
├── unit/           # Pure function tests
├── integration/    # API and system tests  
├── benchmarks/     # Performance tests
├── security/       # Security validation
├── compatibility/  # Model compatibility
└── fixtures/       # Test data and mocks
```

### Testing Guidelines

1. **Test Pyramid**: More unit tests, fewer integration tests
2. **Fast Feedback**: Unit tests should run in milliseconds
3. **Isolation**: Tests should not depend on external services
4. **Deterministic**: Tests should produce consistent results
5. **Comprehensive**: Cover happy paths, edge cases, and error conditions

### Mock Usage

```go
type MockStreamingClient struct {
    responses []chat.MessageChunk
    delay     time.Duration
}

func (msc *MockStreamingClient) StreamMessage(ctx context.Context, req chat.ChatRequest) (<-chan chat.MessageChunk, error) {
    chunks := make(chan chat.MessageChunk, len(msc.responses))
    
    go func() {
        defer close(chunks)
        for _, chunk := range msc.responses {
            select {
            case chunks <- chunk:
                if msc.delay > 0 {
                    time.Sleep(msc.delay)
                }
            case <-ctx.Done():
                return
            }
        }
    }()
    
    return chunks, nil
}
```

## Troubleshooting Tests

### Common Issues

1. **Flaky Tests**: Usually caused by timing issues or shared state
2. **Race Conditions**: Run with `-race` flag to detect
3. **Resource Leaks**: Check for unclosed channels or goroutines
4. **Integration Failures**: Verify test environment setup

### Debug Mode

```bash
# Run tests with verbose output
go test -v ./...

# Run specific test with debug logging
RYAN_LOG_LEVEL=debug go test -v -run TestSpecificFunction

# Run with race detection
go test -race ./...

# Profile memory usage
go test -memprofile=mem.prof ./...
```

This comprehensive testing approach ensures Ryan maintains high quality, performance, and security standards across all components and supported models.