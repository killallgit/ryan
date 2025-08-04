# Chat Package

The `chat` package provides conversation management and memory capabilities for Ryan, including integration with LangChain Go.

## Features

- **Message Management**: Structured representation of chat messages with roles (user, assistant, system, tool, error)
- **Conversation Tracking**: Maintain conversation history with metadata
- **LangChain Integration**: Compatible with LangChain Go's memory system
- **Vector Memory**: Semantic search over conversation history using vector embeddings
- **Token Counting**: Track token usage for different models
- **Persistence**: Save and load conversations

## Vector Memory

The package includes `LangChainVectorMemory` which extends the basic conversation memory with vector store capabilities:

### Features
- **Semantic Search**: Find relevant messages based on meaning, not just keywords
- **Automatic Indexing**: Messages are automatically indexed when added
- **Context Retrieval**: Retrieve relevant conversation context for better responses
- **LangChain Compatible**: Works seamlessly with LangChain chains and agents

### Usage

```go
import (
    "github.com/killallgit/ryan/pkg/chat"
    "github.com/killallgit/ryan/pkg/vectorstore"
)

// Create vector store
store, err := vectorstore.NewChromemStore(embedder, persistDir, true)

// Configure vector memory
config := chat.VectorMemoryConfig{
    CollectionName: "conversations",
    MaxRetrieved:   10,
    ScoreThreshold: 0.7,
}

// Create vector memory
vm, err := chat.NewLangChainVectorMemory(store, config)

// Use with LangChain
adapter := &chat.VectorMemoryAdapter{LangChainVectorMemory: vm}
chain := chains.NewConversationChain(llm, adapter)

// Add messages
vm.AddMessage(ctx, chat.NewUserMessage("Hello!"))
vm.AddMessage(ctx, chat.NewAssistantMessage("Hi there!"))

// Search for relevant messages
relevant, err := vm.GetRelevantMessages(ctx, "greeting")
```

### Configuration Options

- **CollectionName**: Name of the vector store collection to use
- **MaxRetrieved**: Maximum number of messages to retrieve in searches
- **ScoreThreshold**: Minimum similarity score (0-1) for retrieved messages

## Message Types

The package supports various message roles:

```go
// User messages
msg := chat.NewUserMessage("What is machine learning?")

// Assistant responses
msg := chat.NewAssistantMessage("Machine learning is...")

// System messages
msg := chat.NewSystemMessage("You are a helpful assistant")

// Tool results
msg := chat.NewToolResultMessage("calculator", "Result: 42")

// Error messages
msg := chat.NewErrorMessage("Failed to process request")
```

## LangChain Integration

The package provides adapters to use Ryan's conversation system with LangChain:

```go
// Create basic memory
memory := chat.NewLangChainMemory()

// Create memory from existing conversation
memory, err := chat.NewLangChainMemoryWithConversation(conv)

// Use as LangChain memory
adapter := &chat.MemoryAdapter{LangChainMemory: memory}
```

## Examples

See the `examples` directory for complete examples:
- `langchain-memory.example.yaml` - Configuration example
- `langchain-vector-memory.go` - Vector memory with LangChain