# Integration Tests

This directory contains integration tests that verify the chat functionality works correctly with a real Ollama deployment.

## Running Tests

### Unit Tests Only
```bash
task test
# or
go test ./pkg/...
```

### Integration Tests
```bash
task test:integration
# or
go test -v ./integration/...
```

### All Tests
```bash
task test:all
```

## Environment Variables

- `OLLAMA_URL`: Override the Ollama server URL (default: https://ollama.kitty-tetra.ts.net)
- `OLLAMA_TEST_MODEL`: Override the test model (default: qwen3:latest)
- `SKIP_INTEGRATION`: Set to any value to skip integration tests

## Test Results

Integration tests may have some failures due to the non-deterministic nature of LLM responses. The important aspects to verify:

1. **Connection**: Can connect to Ollama API
2. **Communication**: Can send prompts and receive responses
3. **Conversation**: Can maintain conversation context
4. **Error Handling**: Gracefully handles errors

## Notes

- Some tests have strict expectations that may fail with certain models
- The tests are designed to verify functionality, not exact responses
- Use `SKIP_INTEGRATION=1` in environments without Ollama access