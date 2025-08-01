package collections

import (
	"fmt"
	"sync"
	"time"
)

// RingBuffer is a thread-safe circular buffer with fixed capacity
// It prevents unbounded memory growth by overwriting oldest entries
type RingBuffer[T any] struct {
	data     []T
	head     int
	tail     int
	size     int
	capacity int
	
	// Thread safety
	mu sync.RWMutex
	
	// Memory tracking
	itemSize    func(T) int64
	totalMemory int64
	maxMemory   int64
	
	// Statistics
	totalPushes   int64
	totalOverwrites int64
	createdAt     time.Time
	
	// Callbacks
	onOverwrite func(T) // Called when an item is overwritten
	onPush      func(T) // Called when an item is pushed
}

// RingBufferConfig holds configuration for RingBuffer
type RingBufferConfig[T any] struct {
	Capacity    int
	MaxMemory   int64
	ItemSize    func(T) int64
	OnOverwrite func(T)
	OnPush      func(T)
}

// RingBufferStats contains usage statistics
type RingBufferStats struct {
	Size            int           `json:"size"`
	Capacity        int           `json:"capacity"`
	TotalMemory     int64         `json:"total_memory"`
	MaxMemory       int64         `json:"max_memory"`
	TotalPushes     int64         `json:"total_pushes"`
	TotalOverwrites int64         `json:"total_overwrites"`
	MemoryEfficiency float64      `json:"memory_efficiency"`
	CreatedAt       time.Time     `json:"created_at"`
	Age             time.Duration `json:"age"`
}

// NewRingBuffer creates a new ring buffer with the specified configuration
func NewRingBuffer[T any](config RingBufferConfig[T]) *RingBuffer[T] {
	if config.Capacity <= 0 {
		config.Capacity = 1000 // Default capacity
	}
	
	if config.ItemSize == nil {
		config.ItemSize = defaultItemSizeRing[T]
	}
	
	rb := &RingBuffer[T]{
		data:        make([]T, config.Capacity),
		capacity:    config.Capacity,
		itemSize:    config.ItemSize,
		maxMemory:   config.MaxMemory,
		onOverwrite: config.OnOverwrite,
		onPush:      config.OnPush,
		createdAt:   time.Now(),
	}
	
	return rb
}

// Push adds an item to the ring buffer
// If buffer is full, overwrites the oldest item
func (rb *RingBuffer[T]) Push(item T) bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	itemMemory := rb.itemSize(item)
	
	// Check memory limit before adding
	if rb.maxMemory > 0 && rb.totalMemory+itemMemory > rb.maxMemory {
		// Need to evict items to make room
		for rb.totalMemory+itemMemory > rb.maxMemory && rb.size > 0 {
			rb.evictOldest()
		}
	}
	
	// Check if we're overwriting an existing item
	var overwrittenItem T
	var isOverwrite bool
	
	if rb.size == rb.capacity {
		// Buffer is full, we'll overwrite the head
		overwrittenItem = rb.data[rb.head]
		isOverwrite = true
		
		// Subtract memory of overwritten item
		rb.totalMemory -= rb.itemSize(overwrittenItem)
		rb.totalOverwrites++
	}
	
	// Add the new item
	rb.data[rb.tail] = item
	rb.totalMemory += itemMemory
	rb.totalPushes++
	
	// Update pointers
	rb.tail = (rb.tail + 1) % rb.capacity
	
	if rb.size < rb.capacity {
		rb.size++
	} else {
		// Move head forward when buffer is full
		rb.head = (rb.head + 1) % rb.capacity
	}
	
	// Call callbacks without holding the lock
	rb.mu.Unlock()
	
	if isOverwrite && rb.onOverwrite != nil {
		rb.onOverwrite(overwrittenItem)
	}
	
	if rb.onPush != nil {
		rb.onPush(item)
	}
	
	rb.mu.Lock()
	return true
}

// Pop removes and returns the most recently added item
func (rb *RingBuffer[T]) Pop() (T, bool) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	var zero T
	
	if rb.size == 0 {
		return zero, false
	}
	
	// Get the last item (most recent)
	rb.tail = (rb.tail - 1 + rb.capacity) % rb.capacity
	item := rb.data[rb.tail]
	
	// Clear the slot
	rb.data[rb.tail] = zero
	
	// Update memory and size
	rb.totalMemory -= rb.itemSize(item)
	rb.size--
	
	return item, true
}

// PopOldest removes and returns the oldest item
func (rb *RingBuffer[T]) PopOldest() (T, bool) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	var zero T
	
	if rb.size == 0 {
		return zero, false
	}
	
	// Get the oldest item
	item := rb.data[rb.head]
	
	// Clear the slot
	rb.data[rb.head] = zero
	
	// Update pointers and memory
	rb.head = (rb.head + 1) % rb.capacity
	rb.totalMemory -= rb.itemSize(item)
	rb.size--
	
	return item, true
}

// Peek returns the most recently added item without removing it
func (rb *RingBuffer[T]) Peek() (T, bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	var zero T
	
	if rb.size == 0 {
		return zero, false
	}
	
	// Get the last item (most recent)
	tail := (rb.tail - 1 + rb.capacity) % rb.capacity
	return rb.data[tail], true
}

// PeekOldest returns the oldest item without removing it
func (rb *RingBuffer[T]) PeekOldest() (T, bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	var zero T
	
	if rb.size == 0 {
		return zero, false
	}
	
	return rb.data[rb.head], true
}

// GetAll returns all items in chronological order (oldest first)
func (rb *RingBuffer[T]) GetAll() []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if rb.size == 0 {
		return nil
	}
	
	result := make([]T, rb.size)
	
	for i := 0; i < rb.size; i++ {
		idx := (rb.head + i) % rb.capacity
		result[i] = rb.data[idx]
	}
	
	return result
}

// GetRecent returns the N most recent items (newest first)
func (rb *RingBuffer[T]) GetRecent(n int) []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if rb.size == 0 || n <= 0 {
		return nil
	}
	
	if n > rb.size {
		n = rb.size
	}
	
	result := make([]T, n)
	
	for i := 0; i < n; i++ {
		idx := (rb.tail - 1 - i + rb.capacity) % rb.capacity
		result[i] = rb.data[idx]
	}
	
	return result
}

// GetOldest returns the N oldest items (oldest first)
func (rb *RingBuffer[T]) GetOldest(n int) []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if rb.size == 0 || n <= 0 {
		return nil
	}
	
	if n > rb.size {
		n = rb.size
	}
	
	result := make([]T, n)
	
	for i := 0; i < n; i++ {
		idx := (rb.head + i) % rb.capacity
		result[i] = rb.data[idx]
	}
	
	return result
}

// Size returns the current number of items
func (rb *RingBuffer[T]) Size() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

// Capacity returns the maximum capacity
func (rb *RingBuffer[T]) Capacity() int {
	return rb.capacity
}

// IsFull returns true if the buffer is at capacity
func (rb *RingBuffer[T]) IsFull() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size == rb.capacity
}

// IsEmpty returns true if the buffer is empty
func (rb *RingBuffer[T]) IsEmpty() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size == 0
}

// Clear removes all items from the buffer
func (rb *RingBuffer[T]) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	var zero T
	
	// Call overwrite callback for all items if provided
	if rb.onOverwrite != nil {
		for i := 0; i < rb.size; i++ {
			idx := (rb.head + i) % rb.capacity
			rb.onOverwrite(rb.data[idx])
		}
	}
	
	// Clear all data
	for i := range rb.data {
		rb.data[i] = zero
	}
	
	rb.head = 0
	rb.tail = 0
	rb.size = 0
	rb.totalMemory = 0
}

// ForEach iterates over all items in chronological order
func (rb *RingBuffer[T]) ForEach(fn func(int, T) bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	for i := 0; i < rb.size; i++ {
		idx := (rb.head + i) % rb.capacity
		if !fn(i, rb.data[idx]) {
			break
		}
	}
}

// ForEachReverse iterates over all items in reverse chronological order
func (rb *RingBuffer[T]) ForEachReverse(fn func(int, T) bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	for i := 0; i < rb.size; i++ {
		idx := (rb.tail - 1 - i + rb.capacity) % rb.capacity
		if !fn(i, rb.data[idx]) {
			break
		}
	}
}

// GetStats returns usage statistics
func (rb *RingBuffer[T]) GetStats() RingBufferStats {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	var memoryEfficiency float64
	if rb.maxMemory > 0 {
		memoryEfficiency = float64(rb.totalMemory) / float64(rb.maxMemory)
	} else if rb.capacity > 0 {
		memoryEfficiency = float64(rb.size) / float64(rb.capacity)
	}
	
	return RingBufferStats{
		Size:             rb.size,
		Capacity:         rb.capacity,
		TotalMemory:      rb.totalMemory,
		MaxMemory:        rb.maxMemory,
		TotalPushes:      rb.totalPushes,
		TotalOverwrites:  rb.totalOverwrites,
		MemoryEfficiency: memoryEfficiency,
		CreatedAt:        rb.createdAt,
		Age:              time.Since(rb.createdAt),
	}
}

// Resize changes the capacity of the ring buffer
// This operation is expensive and should be used sparingly
func (rb *RingBuffer[T]) Resize(newCapacity int) error {
	if newCapacity <= 0 {
		return fmt.Errorf("capacity must be positive")
	}
	
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	if newCapacity == rb.capacity {
		return nil // No change needed
	}
	
	// Get all current items
	currentItems := make([]T, rb.size)
	for i := 0; i < rb.size; i++ {
		idx := (rb.head + i) % rb.capacity
		currentItems[i] = rb.data[idx]
	}
	
	// Create new data array
	newData := make([]T, newCapacity)
	
	// Copy items to new array
	itemsToCopy := rb.size
	if itemsToCopy > newCapacity {
		// If new capacity is smaller, keep only the most recent items
		itemsToCopy = newCapacity
		startIdx := rb.size - newCapacity
		currentItems = currentItems[startIdx:]
		
		// Call overwrite callback for discarded items
		if rb.onOverwrite != nil {
			for i := 0; i < startIdx; i++ {
				rb.onOverwrite(currentItems[i])
				rb.totalOverwrites++
			}
		}
	}
	
	copy(newData, currentItems)
	
	// Update buffer state
	rb.data = newData
	rb.capacity = newCapacity
	rb.head = 0
	rb.tail = itemsToCopy % newCapacity
	rb.size = itemsToCopy
	
	return nil
}

// Private methods

func (rb *RingBuffer[T]) evictOldest() {
	if rb.size == 0 {
		return
	}
	
	// Get the oldest item
	oldestItem := rb.data[rb.head]
	
	// Clear the slot
	var zero T
	rb.data[rb.head] = zero
	
	// Update state
	rb.head = (rb.head + 1) % rb.capacity
	rb.totalMemory -= rb.itemSize(oldestItem)
	rb.size--
	rb.totalOverwrites++
	
	// Call overwrite callback
	if rb.onOverwrite != nil {
		go rb.onOverwrite(oldestItem)
	}
}

// Default item size function
func defaultItemSizeRing[T any](item T) int64 {
	// Very rough estimate - in practice you'd implement this based on your data types
	return 64
}

// TimeBasedRingBuffer is a specialized ring buffer for time-series data
type TimeBasedRingBuffer[T any] struct {
	*RingBuffer[TimestampedItem[T]]
	maxAge time.Duration
}

// TimestampedItem wraps an item with a timestamp
type TimestampedItem[T any] struct {
	Item      T
	Timestamp time.Time
}

// NewTimeBasedRingBuffer creates a new time-based ring buffer
func NewTimeBasedRingBuffer[T any](capacity int, maxAge time.Duration) *TimeBasedRingBuffer[T] {
	config := RingBufferConfig[TimestampedItem[T]]{
		Capacity: capacity,
		ItemSize: func(item TimestampedItem[T]) int64 {
			return 64 + 8 // Item size + timestamp size (rough estimate)
		},
	}
	
	return &TimeBasedRingBuffer[T]{
		RingBuffer: NewRingBuffer(config),
		maxAge:     maxAge,
	}
}

// PushWithTime adds an item with the current timestamp
func (trb *TimeBasedRingBuffer[T]) PushWithTime(item T) {
	timestampedItem := TimestampedItem[T]{
		Item:      item,
		Timestamp: time.Now(),
	}
	trb.Push(timestampedItem)
	
	// Clean up expired items
	trb.cleanupExpired()
}

// GetRecent returns items newer than the specified duration
func (trb *TimeBasedRingBuffer[T]) GetItemsNewerThan(duration time.Duration) []T {
	cutoff := time.Now().Add(-duration)
	return trb.GetItemsSince(cutoff)
}

// GetItemsSince returns items newer than the specified time
func (trb *TimeBasedRingBuffer[T]) GetItemsSince(since time.Time) []T {
	allItems := trb.GetAll()
	var result []T
	
	for _, timestampedItem := range allItems {
		if timestampedItem.Timestamp.After(since) {
			result = append(result, timestampedItem.Item)
		}
	}
	
	return result
}

// cleanupExpired removes items older than maxAge
func (trb *TimeBasedRingBuffer[T]) cleanupExpired() {
	if trb.maxAge <= 0 {
		return
	}
	
	cutoff := time.Now().Add(-trb.maxAge)
	
	// Remove expired items from the front
	for !trb.IsEmpty() {
		oldest, ok := trb.PeekOldest()
		if !ok || oldest.Timestamp.After(cutoff) {
			break
		}
		trb.PopOldest()
	}
}