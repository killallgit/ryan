# Vector Store Integration

## Overview

The application now automatically stores and indexes chat conversations in a vector store, enabling semantic search and context retrieval capabilities. This integration is enabled by default and works transparently in the background.

## What's Happening

### Automatic Chat Indexing
- **All chat conversations** are now automatically stored in the vector store
- **Messages are embedded** using the configured embedder (default: ollama/nomic-embed-text)
- **Semantic search** capabilities are available for retrieving relevant context
- **Memory is preserved** across sessions when persistence is enabled

### Vector Store Configuration
The vector store is enabled by default with these settings:
```yaml
vectorstore:
  enabled: true                    # Vector store is active
  provider: chromem               # In-memory with optional persistence
  persistence_dir: ./.ryan/vectorstore  # Local storage directory
  enable_persistence: true        # Save data between sessions
  embedder:
    provider: ollama              # Use local Ollama for embeddings
    model: nomic-embed-text       # High-quality embedding model
    base_url: http://localhost:11434
```

### Collections Created
- **conversations**: Stores chat messages and conversations
- **documents**: Available for document indexing (future use)

## Debugging and Monitoring

### Vector Store Debug View
Access comprehensive vector store information via the command palette:
1. Press `Ctrl+P` to open command palette
2. Select "Vector Store Debug View"
3. View collections, document counts, embedder info, and more

### What You'll See
- **Collection statistics**: Document counts per collection
- **Embedder information**: Model being used, dimensions
- **Storage details**: Provider, persistence status
- **Real-time data**: Refresh with 'r' key

## Technical Details

### Memory System
- Chat controllers now use `LangChainVectorMemory` instead of basic memory
- Vector memory provides both traditional conversation history and semantic retrieval
- Fallback to regular memory if vector store initialization fails

### Storage Backend
- **chromem**: In-memory vector database with optional persistence
- **Embeddings**: Generated using Ollama's nomic-embed-text model (with automatic fallback to mock embedder)
- **Persistence**: Data saved to `./.ryan/vectorstore` directory
- **Collections**: Organized by conversation type and document type
- **Fallback**: If Ollama is not available, automatically uses mock embedder for development

### Performance
- **Lazy initialization**: Vector store initializes only when needed
- **Efficient storage**: Only unique conversation content is embedded
- **Configurable limits**: Memory window size and retrieval limits can be tuned

## Benefits

1. **Enhanced Context**: AI can access semantically relevant past conversations
2. **Better Continuity**: Conversations maintain context across sessions
3. **Debugging Tools**: Easy visibility into what's being stored and indexed
4. **Scalable Storage**: Handles large conversation histories efficiently
5. **Local Control**: All data stays on your machine

## Troubleshooting

### Vector store not working?
- Check if Ollama is running: `ollama list`
- Verify embedder model is available: `ollama pull nomic-embed-text`
- Check logs for initialization errors
- Use vector store debug view to diagnose issues

### No collections showing?
- Start a chat conversation to trigger indexing
- Check `./.ryan/vectorstore` directory exists and is writable
- Verify configuration in settings file

### Performance issues?
- Reduce `vectorstore.indexer.chunk_size` for faster processing
- Disable persistence temporarily: `enable_persistence: false`
- Check available disk space in persistence directory

The vector store integration enhances your chat experience by providing intelligent context retrieval while maintaining full local control of your data.