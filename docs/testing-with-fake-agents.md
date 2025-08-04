# Testing with Fake Agents

This guide explains how to use the fake agent test helpers to write fast, reliable tests without depending on external LLM services.

## Overview

The fake agent system provides test doubles for LLM interactions, allowing you to:
- Write deterministic tests with predictable responses
- Test error conditions and edge cases
- Run tests without network dependencies
- Simulate streaming responses
- Test tool/function calling

## Basic Usage

### Simple Chat Client Testing

```go
import (
    "github.com/killallgit/ryan/pkg/testutil"
    "github.com/killallgit/ryan/pkg/chat"
)

func TestMyFeature(t *testing.T) {
    // Create a fake client with predefined responses
    fakeClient := testutil.NewFakeChatClient(
        "test-model",
        "First response",
        "Second response",
        "Third response",
    )
    
    // Use it like a regular chat client
    req := chat.ChatRequest{
        Model: "test-model",
        Messages: []chat.Message{
            chat.NewUserMessage("Hello"),
        },
    }
    
    resp, err := fakeClient.SendMessage(req)
    // resp.Content will be "First response"
}
```

### Controller Testing

```go
func TestChatController(t *testing.T) {
    fakeClient := testutil.NewFakeChatClient(
        "test-model",
        testutil.PredefinedResponses.SimpleChat...,
    )
    
    controller := controllers.NewChatController(
        fakeClient, 
        "test-model", 
        nil,
    )
    
    // Test conversation flow
    resp, err := controller.SendUserMessage("Hello")
    assert.NoError(t, err)
    assert.Equal(t, "Hello! How can I help you today?", resp.Content)
}
```

## Streaming Support

The fake streaming client simulates chunked responses:

```go
func TestStreaming(t *testing.T) {
    streamingClient := testutil.NewFakeStreamingChatClient(
        "test-model",
        "This will be streamed in chunks",
    )
    
    // Configure chunk behavior
    streamingClient.SetChunkSize(5)     // 5 chars per chunk
    streamingClient.SetChunkDelay(10 * time.Millisecond)
    
    // Stream the response
    ctx := context.Background()
    req := chat.ChatRequest{
        Model: "test-model",
        Messages: []chat.Message{
            chat.NewUserMessage("Test"),
        },
    }
    
    chunkChan, err := streamingClient.StreamMessage(ctx, req)
    require.NoError(t, err)
    
    // Collect chunks
    var chunks []chat.MessageChunk
    for chunk := range chunkChan {
        chunks = append(chunks, chunk)
    }
}
```

## Error Simulation

Test error handling by configuring failures:

```go
func TestErrorHandling(t *testing.T) {
    fakeClient := testutil.NewFakeChatClient("test-model", "success")
    
    // Fail on the second call
    fakeClient.GetFakeLLM().SetErrorOnCall(2, "API rate limit exceeded")
    
    // First call succeeds
    _, err := fakeClient.SendMessage(req)
    assert.NoError(t, err)
    
    // Second call fails
    _, err = fakeClient.SendMessage(req)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "API rate limit exceeded")
}
```

For streaming errors:

```go
streamingClient.SetFailAfter(3, "network error")
// Will successfully stream 3 chunks, then fail
```

## Tool/Function Calling

Test tool execution flows:

```go
func TestToolCalling(t *testing.T) {
    // Response with tool call
    toolResponse := `{"tool_calls": [{"name": "calculator", "arguments": {"a": 1, "b": 2}}]}`
    finalResponse := "The result is 3"
    
    fakeClient := testutil.NewFakeChatClient(
        "test-model",
        toolResponse,
        finalResponse,
    )
    
    // Register the tool
    toolRegistry := tools.NewRegistry()
    toolRegistry.Register(&calculatorTool{})
    
    controller := controllers.NewChatController(
        fakeClient,
        "test-model",
        toolRegistry,
    )
    
    // The controller will:
    // 1. Send user message
    // 2. Receive tool call response
    // 3. Execute the tool
    // 4. Send tool result back
    // 5. Receive final response
    resp, err := controller.SendUserMessage("What is 1 + 2?")
    assert.Equal(t, "The result is 3", resp.Content)
}
```

## Advanced Features

### Tracking Call History

```go
fakeLLM := fakeClient.GetFakeLLM()

// Check call count
assert.Equal(t, 3, fakeLLM.GetCallCount())

// Inspect last prompt
lastPrompt := fakeLLM.GetLastPrompt()
assert.Contains(t, lastPrompt, "important context")

// Reset state
fakeLLM.Reset()
```

### Dynamic Response Addition

```go
fakeLLM := fakeClient.GetFakeLLM()

// Add responses during test
fakeLLM.AddResponse("New response")
```

### Predefined Response Sets

```go
// Use built-in response patterns
fakeClient := testutil.NewFakeChatClient(
    "test-model",
    testutil.PredefinedResponses.SimpleChat...,
)

// Available sets:
// - PredefinedResponses.SimpleChat
// - PredefinedResponses.ErrorResponse  
// - PredefinedResponses.ToolCalling
```

## Best Practices

1. **Use fake agents for unit tests** - They're fast and deterministic
2. **Keep integration tests for critical paths** - Test against real services sparingly
3. **Test error conditions** - Fake agents make it easy to simulate failures
4. **Verify prompts** - Use `GetLastPrompt()` to ensure correct context is sent
5. **Reset between tests** - Call `Reset()` to avoid test pollution

## Migration Guide

To migrate existing tests:

1. Replace mock implementations with fake clients:
   ```go
   // Before
   mockClient := &MockChatClient{}
   mockClient.On("SendMessage", req).Return(response, nil)
   
   // After
   fakeClient := testutil.NewFakeChatClient("model", "response")
   ```

2. Update assertions to use fake client methods:
   ```go
   // Before
   mockClient.AssertExpectations(t)
   
   // After
   assert.Equal(t, 1, fakeClient.GetFakeLLM().GetCallCount())
   ```

3. For streaming tests, use the streaming client:
   ```go
   // Create streaming-capable fake
   streamingClient := testutil.NewFakeStreamingChatClient(...)
   ```

## Complete Example

Here's a complete test suite using fake agents:

```go
package mypackage_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/killallgit/ryan/pkg/chat"
    "github.com/killallgit/ryan/pkg/controllers"
    "github.com/killallgit/ryan/pkg/testutil"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestChatFeature(t *testing.T) {
    t.Run("handles conversation", func(t *testing.T) {
        client := testutil.NewFakeChatClient(
            "gpt-4",
            "I understand your question.",
            "Here's the answer.",
        )
        
        controller := controllers.NewChatController(client, "gpt-4", nil)
        
        // First exchange
        resp1, err := controller.SendUserMessage("Explain quantum computing")
        require.NoError(t, err)
        assert.Equal(t, "I understand your question.", resp1.Content)
        
        // Second exchange  
        resp2, err := controller.SendUserMessage("Can you elaborate?")
        require.NoError(t, err)
        assert.Equal(t, "Here's the answer.", resp2.Content)
        
        // Verify conversation history
        assert.Equal(t, 4, controller.GetMessageCount())
    })
    
    t.Run("streams responses", func(t *testing.T) {
        client := testutil.NewFakeStreamingChatClient(
            "gpt-4",
            "Streaming response text",
        )
        client.SetChunkSize(4)
        
        controller := controllers.NewChatController(client, "gpt-4", nil)
        
        ctx := context.Background()
        updates, err := controller.StartStreaming(ctx, "Stream this")
        require.NoError(t, err)
        
        var chunks []string
        for update := range updates {
            if update.Type == controllers.ChunkReceived {
                chunks = append(chunks, update.Content)
            }
        }
        
        // Should receive: "Stre", "amin", "g re", "spon", "se t", "ext"
        assert.Len(t, chunks, 6)
    })
}
```

## Conclusion

The fake agent test helpers provide a powerful way to test LLM-integrated code without external dependencies. They support all major chat client features including streaming, tool calling, and error simulation, making it possible to achieve high test coverage with fast, reliable tests.