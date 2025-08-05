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

// Initialize from global config (reads from .ryan/settings.yaml)
manager, err := vectorstore.GetGlobalManager()
if err != nil {
    log.Fatal(err)
}

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

err = vectorstore.IndexDocumentsGlobal("documents", docs)

// Search
results, err := vectorstore.SearchGlobal("documents", "golang programming", 5)
```

### Configuration

Add to your `.ryan/settings.yaml`:

```yaml
vectorstore:
  enabled: true
  provider: chromem
  persistence_dir: ./.ryan/vectorstore
  enable_persistence: true

  embedder:
    provider: ollama         # ollama, openai, or mock
    model: nomic-embed-text
    base_url: http://localhost:11434
    # api_key: ""           # For OpenAI, set via OPENAI_API_KEY env var

  collections:
    - name: conversations
      metadata:
        type: chat_history
    - name: documents
      metadata:
        type: document_index

  indexer:
    chunk_size: 1000
    chunk_overlap: 200
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
results, err := vectorstore.SearchGlobal("documents", "search query", 10,
    vectorstore.WithFilter(map[string]interface{}{
        "type": "programming",
    }),
    vectorstore.WithMinScore(0.7),
)

// Get a specific collection for direct operations
collection, err := vectorstore.GetOrCreateCollectionGlobal("documents", nil)
results, err := collection.Query(ctx, "search query", 5)
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

## Document Indexer

The package includes a powerful document indexer for indexing various file types:

### Features
- **Smart Chunking**: Splits documents into overlapping chunks for better context
- **File Type Awareness**: Special handling for code files, structured data, and text
- **Metadata Extraction**: Automatically extracts file metadata (size, type, modification time)
- **Directory Indexing**: Recursively index entire directories with pattern matching

### Usage

```go
// Create indexer using global config
indexer, err := vectorstore.NewIndexerFromGlobalConfig("documents")

// Index a single file
err = indexer.IndexFile(ctx, "/path/to/document.txt")

// Index a directory
patterns := []string{"*.go", "*.md", "*.txt"}
err = indexer.IndexDirectory(ctx, "/path/to/docs", patterns)

// Search indexed documents
docs, err := indexer.SearchDocuments(ctx, "search query", 10)
```

### Supported File Types
- **Text Files**: `.txt`, `.md`, `.log`
- **Code Files**: `.go`, `.py`, `.js`, `.java`, `.cpp`, `.c`, `.rs`
- **Structured Data**: `.json`, `.yaml`, `.yml`, `.toml`

## Examples

See the `examples` directory for complete examples:
- `vectorstore-demo.go` - Basic vector store operations
- `document-indexer.go` - Command-line document indexing tool
- `langchain-vector-memory.go` - LangChain integration with vector memory

## Future Enhancements

- Additional vector store providers (SQLite-vec, FAISS)
- Advanced indexing strategies
- Hybrid search (keyword + semantic)
- Language-aware code chunking
- Collection backups and migrations
