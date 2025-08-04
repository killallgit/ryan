# Context Management Design

*Integrating Claude CLI analysis with Ryan's advanced implementation*

## Overview

Ryan implements a sophisticated context management system inspired by Claude CLI's architecture but enhanced with additional features. The system manages conversation state, configuration hierarchy, and memory strategies through multiple integrated components.

## Architecture Pillars

### 1. Two-Tier Configuration System (Claude CLI Pattern)

Following Claude CLI's proven configuration architecture:

**Global vs Project Context**:
- **Global Configuration**: System-wide settings stored in `~/.ryan/.config.json` or `~/.ryan.json`
- **Project Configuration**: Project-specific settings stored within the global config under a `projects` key
- **Context Scoping**: Each project gets its own isolated context scope with inheritance from global settings

**Configuration Hierarchy** (Priority Order):
1. **Environment Variables** (highest priority)
2. **Project Configuration** 
3. **Global Configuration**
4. **System Defaults** (fallback)

### 2. Advanced Memory Management System

Ryan's implementation exceeds Claude CLI's capabilities with multiple memory strategies:

#### Memory Strategy Architecture
```go
// Multiple memory implementations for different use cases
type MemoryStrategy interface {
    AddMessage(msg Message) error
    GetContext(query string, maxTokens int) ([]Message, error)
    Clear() error
}

// Available implementations:
type VectorMemory struct {
    vectorStore *vectorstore.Manager
    embedder    *Embedder
    collection  string
}

type HybridMemory struct {
    workingMemory []Message          // Recent messages (LRU-style)
    vectorMemory  *VectorMemory      // Semantic search
    weights       MemoryWeights      // Relevance vs recency
}

type LangChainMemory struct {
    buffer       *memory.ConversationBuffer
    conversation *Conversation
}
```

#### Advanced Features Beyond Claude CLI
1. **Vector Context Manager**: Context-aware vector collections
2. **Document Indexer**: Automatic file indexing with configurable rules
3. **Hybrid Memory Strategies**: Working + vector + semantic weighting
4. **Graph-Aware Memory**: Relationship-based context assembly

### 3. Context Tree System (Ryan Innovation)

**Hierarchical Context Structure**:
```go
type Context struct {
    ID          string    `json:"id"`
    ParentID    *string   `json:"parent_id"`    // Context this branched from
    BranchPoint *string   `json:"branch_point"` // Message ID where branch occurred
    Title       string    `json:"title"`        // User-defined or auto-generated
    Created     time.Time `json:"created"`
    MessageIDs  []string  `json:"message_ids"`  // Linear message sequence
    IsActive    bool      `json:"is_active"`    // Current active context
}

type ContextTree struct {
    RootContextID string              `json:"root_context_id"`
    Contexts      map[string]*Context `json:"contexts"`
    Messages      map[string]*Message `json:"messages"`       // All messages by ID
    ActiveContext string              `json:"active_context"` // Currently active
}
```

**Interactive Navigation**: See [context-tree-ui.md](context-tree-ui.md) for UI implementation details.

## File Loading & Configuration Pipeline

### Configuration File Resolution (Claude CLI Pattern)
Following Claude CLI's robust file loading strategy:

1. **Path Resolution**: Check multiple locations in order
   - New format: `~/.ryan/.config.json`
   - Legacy format: `~/.ryan.json`
   - Environment override: `$RYAN_CONFIG_DIR`

2. **Project Root Detection**: 
   ```go
   func getProjectRoot() string {
       // Use git to find repository root (primary)
       if output, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
           return strings.TrimSpace(string(output))
       }
       // Fallback to current directory
       return filepath.Abs(".")
   }
   ```

3. **Loading Process**:
   - **Cache Check**: Check in-memory LRU cache first
   - **File Reading**: Read and parse JSON with comprehensive error handling
   - **Default Merging**: Merge with default configuration using deep merge
   - **Cache Storage**: Store result in cache for future use

### LRU Cache Implementation (Performance Optimization)

**Cache Design** (Following Claude CLI patterns):
```go
type ConfigCache struct {
    maxSize int                    // Default: 50 items
    cache   map[string]*CacheEntry
    order   *list.List            // LRU ordering
    mutex   sync.RWMutex          // Thread safety
}

type CacheEntry struct {
    Key        string
    Value      interface{}
    Timestamp  time.Time
    AccessTime time.Time
    element    *list.Element      // For LRU management
}
```

**Cache Behaviors**:
- **LRU Eviction**: Least recently used items removed when cache is full
- **TTL Support**: Time-based expiration for dynamic content
- **Thread Safety**: Read-write mutex for concurrent access
- **Memory Efficiency**: Configurable size limits with monitoring

### State Persistence & Atomicity

**Atomic Configuration Updates** (Claude CLI Safety Pattern):
```go
type AtomicWriter struct {
    lockfile *LockFile
    backup   *BackupManager
    target   string
}

// Atomic update process:
// 1. Acquire file lock for concurrent protection
// 2. Create backup file (.backup extension)
// 3. Write to temporary file
// 4. Atomic rename (temp -> actual)
// 5. Release lock and cleanup
```

**Delta Storage Optimization**:
- Only persist changed values to minimize file size
- Deep comparison with defaults to identify deltas
- Efficient serialization with JSON marshaling
- Backup restoration on corruption detection

## Memory Strategy Implementations

### 1. Vector Context Manager

**Context-Aware Collections**:
```go
type VectorContextManager struct {
    manager            *vectorstore.Manager
    tree               *ContextTree
    globalCollection   string                    // Shared knowledge
    contextCollections map[string]string         // contextID -> collectionName
    config             VectorContextConfig
}

type VectorContextConfig struct {
    GlobalCollection    string  // e.g., "global_knowledge"
    ContextPrefix       string  // e.g., "ctx_"
    EnableCrossSearch   bool    // Search across contexts
    MaxContextsPerQuery int     // Performance limit
    ScoreThreshold      float32 // Relevance filtering
    MaxRetrieved        int     // Result count limit
}
```

**Advanced Features**:
- **Semantic Search**: Vector-based similarity matching
- **Context Isolation**: Separate collections per conversation branch
- **Cross-Context Search**: Optional search across all contexts
- **Performance Tuning**: Configurable thresholds and limits

### 2. Hybrid Memory System

**Multi-Strategy Approach**:
```go
type HybridMemory struct {
    workingMemory   []Message          // Recent messages (configurable size)
    vectorMemory    *VectorMemory      // Semantic retrieval
    contextTree     *ContextTree       // Conversation branching
    config          HybridMemoryConfig
}

type HybridMemoryConfig struct {
    WorkingMemorySize   int     // Number of recent messages
    MaxContextTokens    int     // Token limit for assembled context
    SemanticWeight      float32 // Weight for semantic relevance (0.0-1.0)
    RecencyWeight       float32 // Weight for recency (0.0-1.0)
    DeduplicationWindow int     // Avoid duplicate recent messages
    EnableToolIndexing  bool    // Index tool outputs separately
}
```

**Context Assembly Algorithm**:
1. **Working Memory**: Always include recent messages
2. **Semantic Retrieval**: Vector search for relevant historical content
3. **Weighted Scoring**: Combine semantic relevance + recency scores
4. **Deduplication**: Remove duplicates within window
5. **Token Management**: Trim to fit within token limits

### 3. Document Indexer

**Automatic File Indexing**:
```go
type DocumentIndexer struct {
    manager      *vectorstore.Manager
    config       DocumentIndexerConfig
    indexedFiles map[string]time.Time  // Track indexed files and timestamps
}

type DocumentIndexerConfig struct {
    CollectionName       string        // Collection to store documents
    AutoIndexFiles       bool          // Auto-index when accessed
    AutoIndexDirectories bool          // Auto-index directory contents
    MaxFileSize          int64         // Maximum file size (bytes)
    SupportedExtensions  []string      // File extensions to index
    ExcludePatterns      []string      // Patterns to exclude
    ChunkSize            int           // Text chunk size for indexing
    UpdateInterval       time.Duration // File change detection
}
```

**Indexing Process**:
- **File Watching**: Monitor file system changes
- **Content Chunking**: Split large files into manageable chunks
- **Extension Filtering**: Only index supported file types
- **Incremental Updates**: Only re-index changed files
- **Metadata Storage**: Track file paths, timestamps, and relationships

## Integration with Tool System

### Tool Output Indexing

**Enhanced Tool Results**:
```go
type ToolResult struct {
    Content     string                 // Tool output content
    Metadata    map[string]interface{} // Tool-specific metadata
    ContextID   string                 // Associated conversation context
    Timestamp   time.Time              // Execution time
    Indexable   bool                   // Should be indexed for future retrieval
}
```

**Tool Memory Integration**:
- **Automatic Indexing**: Tool outputs indexed in vector memory
- **Context Association**: Tool results linked to conversation contexts
- **Semantic Search**: Tool outputs searchable via vector similarity
- **Performance Tracking**: Tool execution metrics stored for analysis

### Configuration-Driven Tool Behavior

**Tool Configuration Hierarchy**:
```yaml
# Global tool configuration
tools:
  enabled: true
  timeout: "30s"
  resource_limits:
    max_memory_mb: 512
    max_cpu_percent: 80.0
  
  # Tool-specific settings
  bash:
    enabled: true
    allowed_paths: ["/home/user", "/tmp"]
    forbidden_commands: ["sudo", "rm -rf"]
  
  webfetch:
    enabled: true
    cache_ttl: "1h"
    rate_limit: "10/minute"
    allowed_hosts: ["github.com", "stackoverflow.com"]

# Project-specific overrides
projects:
  "/path/to/project":
    tools:
      bash:
        allowed_paths: ["/path/to/project", "/tmp"]
      webfetch:
        allowed_hosts: ["project-specific-api.com"]
```

## Performance Characteristics

### Memory Management
- **LRU Cache**: O(1) configuration lookups with 50-item default limit
- **Vector Search**: Optimized embeddings with configurable score thresholds
- **Working Memory**: Bounded size with automatic pruning
- **Context Switching**: Efficient context activation with minimal overhead

### Storage Efficiency
- **Delta Persistence**: Only store changes from defaults
- **Atomic Operations**: Prevent corruption with backup/restore
- **Compression**: Optional gzip compression for large configurations
- **Backup Management**: Automatic cleanup of old backup files

### Concurrent Access
- **File Locking**: Prevent corruption during concurrent access
- **Thread Safety**: All cache operations protected with RW mutex
- **Context Isolation**: Separate goroutines for different contexts
- **Resource Limits**: Configurable limits to prevent resource exhaustion

## Error Handling & Recovery

### Graceful Degradation
1. **Configuration Corruption**: Automatic fallback to backup files
2. **Cache Invalidation**: Clear corrupted cache entries automatically
3. **Memory Overflow**: Intelligent pruning when approaching limits
4. **Context Switching Failures**: Maintain previous context on errors

### Error Recovery Patterns
```go
type ErrorRecovery struct {
    MaxRetries      int           // Maximum retry attempts
    BackoffStrategy BackoffType   // Exponential, linear, constant
    FallbackConfig  *Config       // Safe configuration to fall back to
    RecoveryActions []RecoveryAction // Ordered recovery steps
}
```

## Future Enhancements

### Planned Improvements
1. **Distributed Caching**: Support for Redis/memcached backends
2. **Configuration Validation**: JSON schema validation for all config files
3. **Migration System**: Automatic migration between configuration versions
4. **Performance Monitoring**: Real-time metrics for memory and cache usage
5. **Context Merging**: Advanced algorithms for combining conversation branches

### Integration Opportunities
1. **External Vector Stores**: Support for Pinecone, Weaviate, Chroma
2. **Cloud Storage**: S3/GCS backends for configuration and context storage
3. **Real-time Sync**: Multi-device context synchronization
4. **Analytics Integration**: Context usage patterns and optimization insights

---

*This context management system represents a sophisticated evolution beyond Claude CLI's capabilities, incorporating advanced memory strategies, interactive context navigation, and enterprise-grade configuration management while maintaining the reliability and performance characteristics of the original design.*