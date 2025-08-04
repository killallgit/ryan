# Vector Store Debug View

The Vector Store debug view provides comprehensive information about your vector store, collections, embeddings, and debug statistics accessible through the command palette.

## Accessing the View

1. Start the application: `go run .`
2. Press `Ctrl+P` to open the command palette
3. Select **"Vector Store Debug View"** from the menu
4. The view will load and display your vector store information

## Features

### Main Table View
- **Collection Name**: Lists all collections in your vector store
- **Document Count**: Shows number of documents in each collection
- **Embedder Model**: Displays the embedding model being used
- **Last Updated**: Shows when the collection was last modified

### Navigation
- **↑/↓ Arrow Keys** or **j/k**: Navigate between collections
- **Page Up/Page Down**: Scroll through large lists
- **Enter**: View detailed information about selected collection
- **r**: Refresh data from vector store
- **Ctrl+P**: Return to command palette

### Collection Details View
When you press Enter on a collection, you'll see:
- Collection name and metadata
- Exact document count
- Embedder configuration
- Last update timestamp
- Press **ESC** to return to the main list

### Status Information
At the bottom of the view, you'll see:
- Vector store status (Enabled/Disabled)
- Provider information (e.g., "chromem")
- Total collections and documents
- Persistence directory (if configured)

## Configuration

The vector store view automatically detects your vector store configuration from your settings file. Make sure you have vector store enabled in your configuration:

```yaml
vectorstore:
  enabled: true
  provider: "chromem"
  persistence_dir: "/path/to/storage"
  embedder:
    provider: "ollama"
    model: "nomic-embed-text"
```

## Troubleshooting

### "Vector store is not enabled"
- This should not happen as vector store is enabled by default
- If you see this, check your configuration file
- Ensure `vectorstore.enabled: true`
- Check the logs for any initialization errors

### Empty collections list
- Check if vector store is properly initialized
- Verify persistence directory permissions
- Try refreshing with 'r' key

### Loading errors
- Check ollama service is running (if using ollama embedder)
- Verify network connectivity for external embedding services
- Check logs for detailed error messages

## UI Layout Example

```
┌─────────────────────────────────────────────────┐
│                Vector Store                     │
├─────────────────────────────────────────────────┤
│ COLLECTION          DOCS   EMBEDDER      UPDATED│
│ ─────────────────────────────────────────────── │
│ > chat_memory       1234   nomic-embed   5m ago │
│   code_index        5678   nomic-embed   1h ago │
│   documents          890   openai        2d ago │
│                                                 │
│ Vector Store: Enabled                           │
│ Provider: chromem | Collections: 3 | Docs: 7802│
│ ↑↓: Navigate  Enter: Details  r: Refresh       │
└─────────────────────────────────────────────────┘
```

This view is particularly useful for:
- **Debugging**: Understanding vector store state and issues
- **Monitoring**: Tracking collection sizes and growth over time
- **Performance**: Identifying large collections that might need optimization
- **Configuration**: Verifying embedder settings and connectivity