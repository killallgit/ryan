package collections

import (
	"context"
	"sync"
	"time"
)

// TTLEntry represents a value with expiration time
type TTLEntry[V any] struct {
	Value     V
	ExpiresAt time.Time
	CreatedAt time.Time
	AccessCount int64
}

// TTLMap is a thread-safe map with automatic TTL-based cleanup
// It prevents memory leaks by automatically removing expired entries
type TTLMap[K comparable, V any] struct {
	data       sync.Map // K -> *TTLEntry[V]
	ttl        time.Duration
	maxSize    int
	
	// Cleanup management
	cleanupInterval time.Duration
	cleanupCh      chan K
	stopCleanup    chan struct{}
	cleanupWg      sync.WaitGroup
	
	// Statistics
	stats      TTLMapStats
	statsMu    sync.RWMutex
	
	// Callbacks
	onExpire   func(K, V)
	onEvict    func(K, V)
	
	// Context for lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	
	// Background cleanup control
	cleanupRunning bool
	cleanupMu      sync.Mutex
}

// TTLMapConfig holds configuration for TTLMap
type TTLMapConfig[K comparable, V any] struct {
	TTL             time.Duration
	MaxSize         int
	CleanupInterval time.Duration
	OnExpire        func(K, V)
	OnEvict         func(K, V)
}

// TTLMapStats contains usage statistics
type TTLMapStats struct {
	Size           int           `json:"size"`
	MaxSize        int           `json:"max_size"`
	Hits           int64         `json:"hits"`
	Misses         int64         `json:"misses"`
	Evictions      int64         `json:"evictions"`
	Expirations    int64         `json:"expirations"`
	LastCleanup    time.Time     `json:"last_cleanup"`
	CleanupDuration time.Duration `json:"cleanup_duration"`
}

// NewTTLMap creates a new TTL map with automatic cleanup
func NewTTLMap[K comparable, V any](config TTLMapConfig[K, V]) *TTLMap[K, V] {
	if config.TTL <= 0 {
		config.TTL = 5 * time.Minute // Default TTL
	}
	
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = config.TTL / 2 // Default to half of TTL
	}
	
	if config.MaxSize <= 0 {
		config.MaxSize = 10000 // Default max size
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	tm := &TTLMap[K, V]{
		ttl:             config.TTL,
		maxSize:         config.MaxSize,
		cleanupInterval: config.CleanupInterval,
		cleanupCh:       make(chan K, 100),
		stopCleanup:     make(chan struct{}),
		onExpire:        config.OnExpire,
		onEvict:         config.OnEvict,
		ctx:             ctx,
		cancel:          cancel,
		stats: TTLMapStats{
			MaxSize: config.MaxSize,
		},
	}
	
	// Start background cleanup
	tm.startCleanup()
	
	return tm
}

// Store adds or updates a key-value pair with TTL
func (tm *TTLMap[K, V]) Store(key K, value V) {
	now := time.Now()
	entry := &TTLEntry[V]{
		Value:       value,
		ExpiresAt:   now.Add(tm.ttl),
		CreatedAt:   now,
		AccessCount: 1,
	}
	
	// Check if we need to evict due to size limit
	tm.checkSizeLimit()
	
	// Store the entry
	if existing, loaded := tm.data.LoadOrStore(key, entry); loaded {
		// Update existing entry
		existingEntry := existing.(*TTLEntry[V])
		existingEntry.Value = value
		existingEntry.ExpiresAt = now.Add(tm.ttl)
		existingEntry.AccessCount++
		
		return
	}
	
	// New entry was stored
	tm.statsMu.Lock()
	tm.stats.Size++
	tm.statsMu.Unlock()
}

// StoreWithTTL adds a key-value pair with custom TTL
func (tm *TTLMap[K, V]) StoreWithTTL(key K, value V, ttl time.Duration) {
	now := time.Now()
	entry := &TTLEntry[V]{
		Value:       value,
		ExpiresAt:   now.Add(ttl),
		CreatedAt:   now,
		AccessCount: 1,
	}
	
	tm.checkSizeLimit()
	
	if existing, loaded := tm.data.LoadOrStore(key, entry); loaded {
		existingEntry := existing.(*TTLEntry[V])
		existingEntry.Value = value
		existingEntry.ExpiresAt = now.Add(ttl)
		existingEntry.AccessCount++
	} else {
		tm.statsMu.Lock()
		tm.stats.Size++
		tm.statsMu.Unlock()
	}
}

// Load retrieves a value if it exists and hasn't expired
func (tm *TTLMap[K, V]) Load(key K) (V, bool) {
	var zero V
	
	value, ok := tm.data.Load(key)
	if !ok {
		tm.statsMu.Lock()
		tm.stats.Misses++
		tm.statsMu.Unlock()
		return zero, false
	}
	
	entry := value.(*TTLEntry[V])
	now := time.Now()
	
	// Check if entry has expired
	if now.After(entry.ExpiresAt) {
		// Expired, remove it
		tm.deleteEntry(key, entry)
		
		tm.statsMu.Lock()
		tm.stats.Misses++
		tm.stats.Expirations++
		tm.statsMu.Unlock()
		
		return zero, false
	}
	
	// Update access count
	entry.AccessCount++
	
	tm.statsMu.Lock()
	tm.stats.Hits++
	tm.statsMu.Unlock()
	
	return entry.Value, true
}

// LoadOrStore gets an existing value or stores a new one
func (tm *TTLMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	if existing, loaded := tm.Load(key); loaded {
		return existing, true
	}
	
	tm.Store(key, value)
	return value, false
}

// LoadAndDelete retrieves and removes a value
func (tm *TTLMap[K, V]) LoadAndDelete(key K) (V, bool) {
	var zero V
	
	value, loaded := tm.data.LoadAndDelete(key)
	if !loaded {
		return zero, false
	}
	
	entry := value.(*TTLEntry[V])
	
	tm.statsMu.Lock()
	tm.stats.Size--
	tm.statsMu.Unlock()
	
	return entry.Value, true
}

// Delete removes a key-value pair
func (tm *TTLMap[K, V]) Delete(key K) {
	if value, loaded := tm.data.LoadAndDelete(key); loaded {
		entry := value.(*TTLEntry[V])
		
		tm.statsMu.Lock()
		tm.stats.Size--
		tm.statsMu.Unlock()
		
		// Call eviction callback
		if tm.onEvict != nil {
			go tm.onEvict(key, entry.Value)
		}
	}
}

// Extend extends the TTL for a key
func (tm *TTLMap[K, V]) Extend(key K, extension time.Duration) bool {
	value, ok := tm.data.Load(key)
	if !ok {
		return false
	}
	
	entry := value.(*TTLEntry[V])
	now := time.Now()
	
	// Check if already expired
	if now.After(entry.ExpiresAt) {
		return false
	}
	
	// Extend the expiration time
	entry.ExpiresAt = entry.ExpiresAt.Add(extension)
	return true
}

// Touch updates the access time and extends TTL
func (tm *TTLMap[K, V]) Touch(key K) bool {
	value, ok := tm.data.Load(key)
	if !ok {
		return false
	}
	
	entry := value.(*TTLEntry[V])
	now := time.Now()
	
	// Check if already expired
	if now.After(entry.ExpiresAt) {
		return false
	}
	
	// Reset TTL and update access count
	entry.ExpiresAt = now.Add(tm.ttl)
	entry.AccessCount++
	
	return true
}

// Range iterates over all non-expired entries
func (tm *TTLMap[K, V]) Range(fn func(K, V) bool) {
	now := time.Now()
	var expiredKeys []K
	
	tm.data.Range(func(key, value any) bool {
		k := key.(K)
		entry := value.(*TTLEntry[V])
		
		// Check if expired
		if now.After(entry.ExpiresAt) {
			expiredKeys = append(expiredKeys, k)
			return true // Continue iteration
		}
		
		// Call the function with non-expired entry
		return fn(k, entry.Value)
	})
	
	// Clean up expired entries found during iteration
	for _, key := range expiredKeys {
		tm.data.Delete(key)
		tm.statsMu.Lock()
		tm.stats.Size--
		tm.stats.Expirations++
		tm.statsMu.Unlock()
	}
}

// Size returns the current number of entries (including potentially expired ones)
func (tm *TTLMap[K, V]) Size() int {
	tm.statsMu.RLock()
	defer tm.statsMu.RUnlock()
	return tm.stats.Size
}

// GetStats returns usage statistics
func (tm *TTLMap[K, V]) GetStats() TTLMapStats {
	tm.statsMu.RLock()
	defer tm.statsMu.RUnlock()
	return tm.stats
}

// Clear removes all entries
func (tm *TTLMap[K, V]) Clear() {
	// Collect all entries for callbacks
	var entries []struct {
		key   K
		value V
	}
	
	if tm.onEvict != nil {
		tm.data.Range(func(key, value any) bool {
			entry := value.(*TTLEntry[V])
			entries = append(entries, struct {
				key   K
				value V
			}{key.(K), entry.Value})
			return true
		})
	}
	
	// Clear the map
	tm.data.Range(func(key, value any) bool {
		tm.data.Delete(key)
		return true
	})
	
	tm.statsMu.Lock()
	tm.stats.Size = 0
	tm.statsMu.Unlock()
	
	// Call eviction callbacks
	if tm.onEvict != nil {
		go func() {
			for _, entry := range entries {
				tm.onEvict(entry.key, entry.value)
			}
		}()
	}
}

// Close stops the background cleanup and releases resources
func (tm *TTLMap[K, V]) Close() error {
	tm.cancel()
	
	// Stop cleanup goroutines
	close(tm.stopCleanup)
	tm.cleanupWg.Wait()
	
	return nil
}

// Private methods

func (tm *TTLMap[K, V]) startCleanup() {
	tm.cleanupMu.Lock()
	defer tm.cleanupMu.Unlock()
	
	if tm.cleanupRunning {
		return
	}
	
	tm.cleanupRunning = true
	
	// Start periodic cleanup
	tm.cleanupWg.Add(1)
	go tm.periodicCleanup()
	
	// Start on-demand cleanup
	tm.cleanupWg.Add(1)
	go tm.onDemandCleanup()
}

func (tm *TTLMap[K, V]) periodicCleanup() {
	defer tm.cleanupWg.Done()
	
	ticker := time.NewTicker(tm.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			tm.cleanup()
		case <-tm.stopCleanup:
			return
		case <-tm.ctx.Done():
			return
		}
	}
}

func (tm *TTLMap[K, V]) onDemandCleanup() {
	defer tm.cleanupWg.Done()
	
	for {
		select {
		case key := <-tm.cleanupCh:
			if value, ok := tm.data.Load(key); ok {
				entry := value.(*TTLEntry[V])
				if time.Now().After(entry.ExpiresAt) {
					tm.deleteEntry(key, entry)
				}
			}
		case <-tm.stopCleanup:
			return
		case <-tm.ctx.Done():
			return
		}
	}
}

func (tm *TTLMap[K, V]) cleanup() {
	start := time.Now()
	var expiredCount int64
	
	now := time.Now()
	var expiredKeys []K
	
	// Collect expired keys
	tm.data.Range(func(key, value any) bool {
		entry := value.(*TTLEntry[V])
		if now.After(entry.ExpiresAt) {
			expiredKeys = append(expiredKeys, key.(K))
		}
		return true
	})
	
	// Remove expired entries
	for _, key := range expiredKeys {
		if value, loaded := tm.data.LoadAndDelete(key); loaded {
			entry := value.(*TTLEntry[V])
			expiredCount++
			
			// Call expiration callback
			if tm.onExpire != nil {
				go tm.onExpire(key, entry.Value)
			}
		}
	}
	
	// Update statistics
	tm.statsMu.Lock()
	tm.stats.Size -= int(expiredCount)
	tm.stats.Expirations += expiredCount
	tm.stats.LastCleanup = start
	tm.stats.CleanupDuration = time.Since(start)
	tm.statsMu.Unlock()
}

func (tm *TTLMap[K, V]) deleteEntry(key K, entry *TTLEntry[V]) {
	tm.data.Delete(key)
	
	tm.statsMu.Lock()
	tm.stats.Size--
	tm.statsMu.Unlock()
	
	if tm.onExpire != nil {
		go tm.onExpire(key, entry.Value)
	}
}

func (tm *TTLMap[K, V]) checkSizeLimit() {
	tm.statsMu.RLock()
	size := tm.stats.Size
	tm.statsMu.RUnlock()
	
	if size >= tm.maxSize {
		tm.evictOldest()
	}
}

func (tm *TTLMap[K, V]) evictOldest() {
	var oldestKey K
	var oldestEntry *TTLEntry[V]
	var found bool
	
	// Find the oldest entry
	tm.data.Range(func(key, value any) bool {
		entry := value.(*TTLEntry[V])
		if !found || entry.CreatedAt.Before(oldestEntry.CreatedAt) {
			oldestKey = key.(K)
			oldestEntry = entry
			found = true
		}
		return true
	})
	
	if found {
		if tm.data.CompareAndDelete(oldestKey, oldestEntry) {
			tm.statsMu.Lock()
			tm.stats.Size--
			tm.stats.Evictions++
			tm.statsMu.Unlock()
			
			if tm.onEvict != nil {
				go tm.onEvict(oldestKey, oldestEntry.Value)
			}
		}
	}
}

// Utility functions

// GetExpiration returns the expiration time for a key
func (tm *TTLMap[K, V]) GetExpiration(key K) (time.Time, bool) {
	value, ok := tm.data.Load(key)
	if !ok {
		return time.Time{}, false
	}
	
	entry := value.(*TTLEntry[V])
	return entry.ExpiresAt, true
}

// GetTTL returns the remaining TTL for a key
func (tm *TTLMap[K, V]) GetTTL(key K) (time.Duration, bool) {
	expiration, ok := tm.GetExpiration(key)
	if !ok {
		return 0, false
	}
	
	remaining := time.Until(expiration)
	if remaining < 0 {
		return 0, false
	}
	
	return remaining, true
}

// Keys returns all non-expired keys
func (tm *TTLMap[K, V]) Keys() []K {
	var keys []K
	now := time.Now()
	
	tm.data.Range(func(key, value any) bool {
		entry := value.(*TTLEntry[V])
		if now.Before(entry.ExpiresAt) {
			keys = append(keys, key.(K))
		}
		return true
	})
	
	return keys
}