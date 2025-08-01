package collections

import (
	"sync"
	"time"
)

// EvictFunc is called when an item is evicted from the BoundedMap
type EvictFunc[K comparable, V any] func(key K, value V)

// BoundedMap is a thread-safe map with LRU eviction and size limits
// It prevents unbounded memory growth while maintaining O(1) operations
type BoundedMap[K comparable, V any] struct {
	maxSize int
	evictFn EvictFunc[K, V]
	
	// Thread-safe data storage
	data sync.Map // K -> *lruNode[K, V]
	
	// LRU tracking with fine-grained locking
	lruMu    sync.Mutex
	lruHead  *lruNode[K, V]
	lruTail  *lruNode[K, V]
	size     int
	
	// Memory tracking
	memoryUsed    int64
	maxMemory     int64
	itemSizeFunc  func(K, V) int64
}

// lruNode represents a node in the LRU doubly-linked list
type lruNode[K comparable, V any] struct {
	key   K
	value V
	prev  *lruNode[K, V]
	next  *lruNode[K, V]
	
	// Metadata
	createdAt time.Time
	accessCount int64
}

// BoundedMapConfig holds configuration for BoundedMap
type BoundedMapConfig[K comparable, V any] struct {
	MaxSize      int
	MaxMemory    int64
	EvictFunc    EvictFunc[K, V]
	ItemSizeFunc func(K, V) int64
}

// NewBoundedMap creates a new bounded map with LRU eviction
func NewBoundedMap[K comparable, V any](config BoundedMapConfig[K, V]) *BoundedMap[K, V] {
	if config.MaxSize <= 0 {
		config.MaxSize = 1000 // Default max size
	}
	
	if config.ItemSizeFunc == nil {
		config.ItemSizeFunc = defaultItemSize[K, V]
	}
	
	bm := &BoundedMap[K, V]{
		maxSize:      config.MaxSize,
		maxMemory:    config.MaxMemory,
		evictFn:      config.EvictFunc,
		itemSizeFunc: config.ItemSizeFunc,
	}
	
	// Initialize LRU list with sentinel nodes
	bm.lruHead = &lruNode[K, V]{}
	bm.lruTail = &lruNode[K, V]{}
	bm.lruHead.next = bm.lruTail
	bm.lruTail.prev = bm.lruHead
	
	return bm
}

// Store adds or updates a key-value pair
func (bm *BoundedMap[K, V]) Store(key K, value V) {
	now := time.Now()
	
	// Check if key already exists
	if existing, loaded := bm.data.LoadOrStore(key, nil); loaded {
		// Update existing node
		node := existing.(*lruNode[K, V])
		
		// Calculate memory delta
		oldSize := bm.itemSizeFunc(node.key, node.value)
		newSize := bm.itemSizeFunc(key, value)
		
		node.value = value
		node.accessCount++
		
		bm.lruMu.Lock()
		bm.memoryUsed += newSize - oldSize
		bm.moveToFront(node)
		bm.lruMu.Unlock()
		
		return
	}
	
	// Create new node
	node := &lruNode[K, V]{
		key:         key,
		value:       value,
		createdAt:   now,
		accessCount: 1,
	}
	
	itemSize := bm.itemSizeFunc(key, value)
	
	bm.lruMu.Lock()
	defer bm.lruMu.Unlock()
	
	// Store the node (this might overwrite if another goroutine stored it)
	if actual, loaded := bm.data.LoadOrStore(key, node); loaded {
		// Another goroutine stored it first, update that one
		actualNode := actual.(*lruNode[K, V])
		oldSize := bm.itemSizeFunc(actualNode.key, actualNode.value)
		actualNode.value = value
		actualNode.accessCount++
		bm.memoryUsed += itemSize - oldSize
		bm.moveToFront(actualNode)
		return
	}
	
	// Add to LRU list
	bm.addToFront(node)
	bm.size++
	bm.memoryUsed += itemSize
	
	// Check for eviction
	bm.evictIfNeeded()
}

// Load retrieves a value by key and updates LRU position
func (bm *BoundedMap[K, V]) Load(key K) (V, bool) {
	var zero V
	
	value, ok := bm.data.Load(key)
	if !ok {
		return zero, false
	}
	
	node := value.(*lruNode[K, V])
	node.accessCount++
	
	// Update LRU position
	bm.lruMu.Lock()
	bm.moveToFront(node)
	bm.lruMu.Unlock()
	
	return node.value, true
}

// LoadAndDelete retrieves and deletes a value
func (bm *BoundedMap[K, V]) LoadAndDelete(key K) (V, bool) {
	var zero V
	
	value, loaded := bm.data.LoadAndDelete(key)
	if !loaded {
		return zero, false
	}
	
	node := value.(*lruNode[K, V])
	
	bm.lruMu.Lock()
	bm.removeFromList(node)
	bm.size--
	bm.memoryUsed -= bm.itemSizeFunc(node.key, node.value)
	bm.lruMu.Unlock()
	
	return node.value, true
}

// Delete removes a key-value pair
func (bm *BoundedMap[K, V]) Delete(key K) {
	value, loaded := bm.data.LoadAndDelete(key)
	if !loaded {
		return
	}
	
	node := value.(*lruNode[K, V])
	
	bm.lruMu.Lock()
	bm.removeFromList(node)
	bm.size--
	bm.memoryUsed -= bm.itemSizeFunc(node.key, node.value)
	bm.lruMu.Unlock()
	
	// Call eviction function if provided
	if bm.evictFn != nil {
		bm.evictFn(node.key, node.value)
	}
}

// Range iterates over all key-value pairs
func (bm *BoundedMap[K, V]) Range(fn func(K, V) bool) {
	bm.data.Range(func(key, value any) bool {
		node := value.(*lruNode[K, V])
		return fn(node.key, node.value)
	})
}

// Size returns the current number of items
func (bm *BoundedMap[K, V]) Size() int {
	bm.lruMu.Lock()
	defer bm.lruMu.Unlock()
	return bm.size
}

// MemoryUsed returns current memory usage in bytes
func (bm *BoundedMap[K, V]) MemoryUsed() int64 {
	bm.lruMu.Lock()
	defer bm.lruMu.Unlock()
	return bm.memoryUsed
}

// Clear removes all items
func (bm *BoundedMap[K, V]) Clear() {
	// Collect all items for eviction callback
	var items []struct {
		key   K
		value V
	}
	
	if bm.evictFn != nil {
		bm.data.Range(func(key, value any) bool {
			node := value.(*lruNode[K, V])
			items = append(items, struct {
				key   K
				value V
			}{node.key, node.value})
			return true
		})
	}
	
	// Clear the map
	bm.data.Range(func(key, value any) bool {
		bm.data.Delete(key)
		return true
	})
	
	bm.lruMu.Lock()
	bm.lruHead.next = bm.lruTail
	bm.lruTail.prev = bm.lruHead
	bm.size = 0
	bm.memoryUsed = 0
	bm.lruMu.Unlock()
	
	// Call eviction function for all items
	if bm.evictFn != nil {
		for _, item := range items {
			bm.evictFn(item.key, item.value)
		}
	}
}

// GetStats returns usage statistics
func (bm *BoundedMap[K, V]) GetStats() BoundedMapStats {
	bm.lruMu.Lock()
	defer bm.lruMu.Unlock()
	
	return BoundedMapStats{
		Size:        bm.size,
		MaxSize:     bm.maxSize,
		MemoryUsed:  bm.memoryUsed,
		MaxMemory:   bm.maxMemory,
		LoadFactor:  float64(bm.size) / float64(bm.maxSize),
	}
}

// BoundedMapStats contains usage statistics
type BoundedMapStats struct {
	Size       int     `json:"size"`
	MaxSize    int     `json:"max_size"`
	MemoryUsed int64   `json:"memory_used"`
	MaxMemory  int64   `json:"max_memory"`
	LoadFactor float64 `json:"load_factor"`
}

// Private methods for LRU management

func (bm *BoundedMap[K, V]) addToFront(node *lruNode[K, V]) {
	node.next = bm.lruHead.next
	node.prev = bm.lruHead
	bm.lruHead.next.prev = node
	bm.lruHead.next = node
}

func (bm *BoundedMap[K, V]) removeFromList(node *lruNode[K, V]) {
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
}

func (bm *BoundedMap[K, V]) moveToFront(node *lruNode[K, V]) {
	bm.removeFromList(node)
	bm.addToFront(node)
}

func (bm *BoundedMap[K, V]) evictIfNeeded() {
	// Check size limit
	for bm.size > bm.maxSize {
		bm.evictLRU()
	}
	
	// Check memory limit
	if bm.maxMemory > 0 {
		for bm.memoryUsed > bm.maxMemory && bm.size > 0 {
			bm.evictLRU()
		}
	}
}

func (bm *BoundedMap[K, V]) evictLRU() {
	if bm.size == 0 {
		return
	}
	
	// Get LRU node
	lru := bm.lruTail.prev
	if lru == bm.lruHead {
		return // Empty list
	}
	
	// Remove from map and list
	bm.data.Delete(lru.key)
	bm.removeFromList(lru)
	bm.size--
	bm.memoryUsed -= bm.itemSizeFunc(lru.key, lru.value)
	
	// Call eviction function
	if bm.evictFn != nil {
		// Release lock before calling eviction function to prevent deadlocks
		bm.lruMu.Unlock()
		bm.evictFn(lru.key, lru.value)
		bm.lruMu.Lock()
	}
}

// Default item size function (rough estimate)
func defaultItemSize[K comparable, V any](key K, value V) int64 {
	// This is a very rough estimate - in practice you'd want a more sophisticated approach
	return 64 // Base overhead + rough estimate for key/value
}