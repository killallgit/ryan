# Vector Store Package

The `vectorstore` package provides embedded vector storage capabilities for Ryan, enabling semantic search and enhanced memory management without external dependencies.

## Features

- **Embedded Vector Database**: Uses chromem-go for pure Go, zero-dependency vector storage
- **Multiple Embedding Providers**: Supports Ollama (local), OpenAI, and mock embedders
- **Persistent Storage**: Optional file-based persistence for long-term memory
- **Semantic Search**: Find relevant documents based on meaning, not just keywords
- **Metadata Filtering**: Query documents with metadata constraints
- **Thread-Safe**: Concurrent access support with proper locking

## Usage

### Basic Example

```go
import "github.com/killallgit/ryan/pkg/vectorstore"

// Create configuration
config := vectorstore.DefaultConfig()

// Create manager
manager, err := vectorstore.NewManager(config)
if err != nil {
    log.Fatal(err)
}
defer manager.Close()

// Index documents
docs := []vectorstore.Document{
    {
        ID:      "1",
        Content: "Go is a statically typed, compiled language",
        Metadata: map[string]interface{}{
            "type": "programming",
        },
    },
}

err = manager.IndexDocuments(ctx, "documents", docs)

// Search
results, err := manager.Search(ctx, "documents", "golang programming", 5)
```

### Configuration

```go
config := vectorstore.Config{
    Provider:          "chromem",              // Vector store provider
    PersistenceDir:    ".ryan/vectorstore",    // Where to store data
    EnablePersistence: true,                   // Enable disk persistence
    
    EmbedderConfig: vectorstore.EmbedderConfig{
        Provider: "ollama",                    // Embedding provider
        Model:    "nomic-embed-text",          // Model to use
        BaseURL:  "http://localhost:11434",    // Ollama endpoint
    },
    
    Collections: []vectorstore.CollectionConfig{
        {Name: "conversations"},               // Pre-create collections
        {Name: "documents"},
    },
}
```

### Embedding Providers

1. **Ollama** (Recommended for local use)
   ```go
   config.EmbedderConfig = vectorstore.EmbedderConfig{
       Provider: "ollama",
       Model:    "nomic-embed-text",
       BaseURL:  "http://localhost:11434",
   }
   ```

2. **OpenAI**
   ```go
   config.EmbedderConfig = vectorstore.EmbedderConfig{
       Provider: "openai",
       Model:    "text-embedding-3-small",
       APIKey:   os.Getenv("OPENAI_API_KEY"),
   }
   ```

3. **Mock** (For testing)
   ```go
   config.EmbedderConfig = vectorstore.EmbedderConfig{
       Provider: "mock",
   }
   ```

### Advanced Queries

```go
// Query with filters
results, err := manager.Search(ctx, "documents", "search query", 10,
    vectorstore.WithFilter(map[string]interface{}{
        "type": "programming",
    }),
    vectorstore.WithMinScore(0.7),
)

// Query with pre-computed embedding
embedding, _ := manager.GetEmbedder().EmbedText(ctx, "query")
collection, _ := manager.GetCollection("documents")
results, err := collection.QueryWithEmbedding(ctx, embedding, 5)
```

## Architecture

The package follows a layered architecture:

1. **Manager Layer**: High-level operations and lifecycle management
2. **Store Layer**: Vector store implementation (currently chromem)
3. **Embedder Layer**: Text-to-vector conversion
4. **Collection Layer**: Document storage and retrieval

## Integration with Ryan

The vector store integrates with Ryan's components:

- **LangChain Memory**: Stores conversation history for semantic retrieval
- **Document Indexing Tool**: Indexes files and directories
- **Context Management**: Provides long-term memory across sessions

## Testing

Run integration tests:

```bash
INTEGRATION_TEST=true go test -v ./integration -run TestVectorStore
```

Test with real embeddings:

```bash
# Test with Ollama
INTEGRATION_TEST=true TEST_REAL_EMBEDDINGS=true go test -v ./integration -run TestRealEmbeddings

# Test with OpenAI
INTEGRATION_TEST=true TEST_REAL_EMBEDDINGS=true OPENAI_API_KEY=your-key go test -v ./integration -run TestRealEmbeddings
```

## Performance

- Query 1,000 documents: ~0.3ms
- Query 100,000 documents: ~40ms
- Concurrent document addition supported
- Memory-efficient chunked storage

## Future Enhancements

- Additional vector store providers (SQLite-vec, FAISS)
- Advanced indexing strategies
- Hybrid search (keyword + semantic)
- Automatic document chunking
- Collection backups and migrations