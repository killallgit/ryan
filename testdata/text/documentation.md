# Vector Store Documentation

This document provides comprehensive information about the vector store implementation in Ryan.

## Overview

The vector store system enables semantic search and retrieval of documents, conversations, and other textual content. It supports multiple embedding providers and storage backends.

### Key Features

- **Multi-provider support**: OpenAI, Ollama, and mock embedders
- **Persistent storage**: ChromeDB-based vector storage with optional persistence
- **Automatic chunking**: Intelligent text chunking for large documents
- **Context-aware search**: Search within specific conversation contexts
- **Hybrid memory**: Combines working memory with vector-based retrieval

## Architecture

### Components

1. **Embedder Layer**
   - Generates vector embeddings from text
   - Supports batch processing for efficiency
   - Handles provider-specific optimizations

2. **Storage Layer**
   - Manages vector collections
   - Provides CRUD operations for documents
   - Handles similarity search queries

3. **Manager Layer**
   - Orchestrates embedder and storage
   - Manages collection lifecycle
   - Provides high-level API

### Usage Examples

```go
// Create a vector store manager
config := vectorstore.Config{
    Provider: "chromem",
    EmbedderConfig: vectorstore.EmbedderConfig{
        Provider: "ollama",
        Model: "nomic-embed-text",
    },
}

manager, err := vectorstore.NewManager(config)
if err != nil {
    log.Fatal(err)
}

// Index a document
doc := vectorstore.Document{
    ID: "doc1",
    Content: "This is a sample document for indexing",
    Metadata: map[string]interface{}{
        "type": "example",
        "source": "manual",
    },
}

err = manager.IndexDocument(ctx, "my-collection", doc)
if err != nil {
    log.Fatal(err)
}

// Search for similar documents
results, err := manager.Search(ctx, "my-collection", "sample document", 5)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("Found: %s (score: %.3f)\n", result.Document.ID, result.Score)
}
```

## Configuration

### Embedder Configuration

The embedder can be configured with various providers:

#### Ollama
```yaml
embedder:
  provider: "ollama"
  model: "nomic-embed-text"
  base_url: "http://localhost:11434"
  timeout: "30s"
```

#### OpenAI
```yaml
embedder:
  provider: "openai"
  model: "text-embedding-3-small"
  api_key: "your-api-key"
```

### Storage Configuration

```yaml
storage:
  provider: "chromem"
  persistence_dir: "./data/vectorstore"
  enable_persistence: true
```

## Best Practices

### Document Chunking

For large documents, use appropriate chunking strategies:

- **Code files**: Chunk by functions or classes
- **Text documents**: Chunk by paragraphs or sections
- **Structured data**: Keep related data together

### Collection Management

- Use meaningful collection names
- Group related documents in the same collection
- Consider collection size for optimal performance

### Search Optimization

- Use specific queries for better results
- Adjust the number of results based on use case
- Utilize metadata filters when appropriate

## Troubleshooting

### Common Issues

1. **Embedding failures**
   - Check provider connectivity
   - Verify API keys and configurations
   - Monitor rate limits

2. **Poor search results**
   - Review document chunking strategy
   - Check embedding model appropriateness
   - Verify query formulation

3. **Performance issues**
   - Monitor collection sizes
   - Consider index optimization
   - Review query patterns

### Debugging

Enable debug logging to troubleshoot issues:

```go
logger := logger.WithComponent("vectorstore")
logger.SetLevel(logger.Debug)
```

## API Reference

### Manager Interface

```go
type Manager interface {
    IndexDocument(ctx context.Context, collection string, doc Document) error
    IndexDocuments(ctx context.Context, collection string, docs []Document) error
    Search(ctx context.Context, collection string, query string, k int, opts ...QueryOption) ([]Result, error)
    GetCollection(name string) (Collection, error)
    ListCollections() ([]string, error)
    ClearCollection(ctx context.Context, name string) error
    Close() error
}
```

### Document Structure

```go
type Document struct {
    ID       string                 `json:"id"`
    Content  string                 `json:"content"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

## Performance Metrics

### Benchmarks

| Operation | Documents | Time | Memory |
|-----------|-----------|------|--------|
| Index | 1,000 | 30s | 150MB |
| Search | 1,000 | 50ms | 50MB |
| Batch Index | 10,000 | 180s | 500MB |

### Scalability

The vector store is designed to handle:
- Up to 100,000 documents per collection
- Sub-second search responses
- Concurrent read/write operations

## Future Enhancements

- Distributed vector storage
- Advanced chunking strategies  
- Multi-modal embeddings (text + code)
- Real-time indexing capabilities
- Enhanced metadata filtering